package services

import (
	"TapTransit-backend/models"
	"TapTransit-backend/utils"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type FareService struct {
	db *gorm.DB
}

// NewFareService 创建计费服务。
func NewFareService(db *gorm.DB) *FareService {
	return &FareService{db: db}
}

// CalculateFare 计算单次乘车费用（基础票价 + 各类优惠）。
func (s *FareService) CalculateFare(cardID string, routeID uint, startStationID, endStationID uint, boardTime time.Time) (*FareCalculationResult, error) {
	result := &FareCalculationResult{
		BaseFare:       0,
		DiscountAmount: 0,
		DiscountType:   "",
		ActualFare:     0,
	}

	// 1. 计算基础票价
	baseFare, err := s.calculateBaseFare(routeID, startStationID, endStationID)
	if err != nil {
		return nil, fmt.Errorf("计算基础票价失败: %w", err)
	}
	result.BaseFare = baseFare
	result.ActualFare = baseFare

	// 2. 检查换乘优惠（仅在换乘规则命中时生效）
	transferDiscount, transferType, err := s.checkTransferDiscount(cardID, routeID, startStationID, boardTime, baseFare)
	if err == nil && transferDiscount > 0 {
		result.DiscountAmount += transferDiscount
		result.DiscountType = transferType
		result.ActualFare -= transferDiscount
		if result.ActualFare < 0 {
			result.ActualFare = 0
		}
	}

	// 3. 检查月度累计折扣（按基础票价计算折扣比例）
	monthlyDiscount, monthlyType, err := s.checkMonthlyDiscount(cardID, baseFare)
	if err == nil && monthlyDiscount > 0 {
		// 如果已有换乘优惠，月度折扣仍在基础票价上计算，实际金额叠加到优惠总额中。
		discountAmount := baseFare * monthlyDiscount
		result.DiscountAmount += discountAmount
		if result.DiscountType != "" {
			result.DiscountType += "," + monthlyType
		} else {
			result.DiscountType = monthlyType
		}
		result.ActualFare -= discountAmount
		if result.ActualFare < 0 {
			result.ActualFare = 0
		}
	}

	// 4. 检查其他优惠（学生卡、老人卡等）
	cardDiscount, cardType, err := s.checkCardTypeDiscount(cardID, baseFare)
	if err == nil && cardDiscount > 0 {
		discountAmount := baseFare * cardDiscount
		result.DiscountAmount += discountAmount
		if result.DiscountType != "" {
			result.DiscountType += "," + cardType
		} else {
			result.DiscountType = cardType
		}
		result.ActualFare -= discountAmount
		if result.ActualFare < 0 {
			result.ActualFare = 0
		}
	}

	return result, nil
}

// calculateBaseFare 计算基础票价（不含任何优惠）。
func (s *FareService) calculateBaseFare(routeID uint, startStationID, endStationID uint) (float64, error) {
	// 先查找特定线路和站点的票价规则
	var fare models.Fare
	err := s.db.Where("route_id = ? AND ((start_station = ? AND end_station = ?) OR (start_station = 0 AND end_station = 0)) AND status = 'active'",
		routeID, startStationID, endStationID).
		Order("start_station DESC"). // 优先使用具体站点规则
		First(&fare).Error

	if err == nil {
		if fare.FareType == "uniform" {
			return fare.BasePrice, nil
		} else if fare.FareType == "segment" {
			// 分段计价：需要计算站点间区段数
			segmentCount := s.calculateSegmentCount(routeID, startStationID, endStationID)
			if segmentCount <= 0 {
				return fare.BasePrice, nil
			}
			if segmentCount == 1 {
				return fare.BasePrice, nil
			}
			// 起步价 + (段数-1) * 续程价
			return fare.BasePrice + float64(segmentCount-1)*fare.ExtraPrice, nil
		}
	}

	// 如果找不到特定规则，使用线路的默认统一票价
	err = s.db.Where("route_id = ? AND start_station = 0 AND end_station = 0 AND status = 'active'", routeID).
		First(&fare).Error
	if err == nil {
		return fare.BasePrice, nil
	}

	// 如果还没有，使用系统默认票价
	return 2.0, nil // 默认2元
}

// calculateSegmentCount 计算两个站点间的区段数。
func (s *FareService) calculateSegmentCount(routeID uint, startStationID, endStationID uint) int {
	var startRS, endRS models.RouteStation
	s.db.Where("route_id = ? AND station_id = ?", routeID, startStationID).First(&startRS)
	s.db.Where("route_id = ? AND station_id = ?", routeID, endStationID).First(&endRS)

	if startRS.ID == 0 || endRS.ID == 0 {
		return 0
	}

	diff := endRS.Sequence - startRS.Sequence
	if diff < 0 {
		diff = -diff
	}
	return diff
}

// checkTransferDiscount 检查换乘优惠（基于上一次下车信息）。
func (s *FareService) checkTransferDiscount(cardID string, routeID uint, stationID uint, boardTime time.Time, baseFare float64) (float64, string, error) {
	// 从 Redis 获取最近一次下车信息
	onboardInfo, err := utils.GetCardOnboardInfo(cardID)
	if err != nil {
		return 0, "", nil // 没有上车记录，不是换乘
	}

	// 解析上车信息（格式：routeID:stationID:timestamp）
	var prevRouteID, prevStationID uint
	var prevTimestamp int64
	_, err = fmt.Sscanf(onboardInfo, "%d:%d:%d", &prevRouteID, &prevStationID, &prevTimestamp)
	if err != nil {
		return 0, "", nil
	}

	prevAlightTime := time.Unix(prevTimestamp, 0)

	// 查找换乘规则
	var transfer models.Transfer
	err = s.db.Where("from_route_id = ? AND from_station_id = ? AND to_route_id = ? AND to_station_id = ? AND status = 'active'",
		prevRouteID, prevStationID, routeID, stationID).
		First(&transfer).Error

	if err != nil {
		return 0, "", nil // 没有匹配的换乘规则
	}

	// 检查时间窗口
	if boardTime.Sub(prevAlightTime).Minutes() > float64(transfer.TimeWindow) {
		return 0, "", nil // 超过时间窗口
	}

	// 计算优惠金额
	if transfer.DiscountRate > 0 {
		// 使用折扣比例
		discountAmount := baseFare * transfer.DiscountRate
		return discountAmount, "transfer", nil
	}
	return transfer.DiscountAmount, "transfer", nil
}

// checkMonthlyDiscount 检查月度累计折扣（使用数据库）。
func (s *FareService) checkMonthlyDiscount(cardID string, currentAmountAfterDiscounts float64) (float64, string, error) {
	// 从数据库获取当月累计金额（不包含本次交易）
	currentAmount, err := utils.GetCurrentMonthAggregate(s.db, cardID)
	if err != nil {
		// 查询失败时返回0折扣，不影响主流程
		return 0, "", nil
	}

	// 查询折扣策略
	var policies []models.DiscountPolicy
	s.db.Where("policy_type = 'monthly_accumulate' AND status = 'active'").
		Order("threshold DESC"). // 从高到低排序
		Find(&policies)

	// 注意：设计文档要求按“扣除特殊票种和换乘优惠后的金额”累计。
	// 这里传入的 currentAmountAfterDiscounts 是本次交易扣除优惠后的金额，用于判断是否触发阈值。
	// currentAmount 是之前已累计的金额。

	// 计算累计金额（之前累计 + 本次应付金额）
	totalAmount := currentAmount + currentAmountAfterDiscounts

	for _, policy := range policies {
		if totalAmount >= policy.Threshold {
			return policy.DiscountRate, "monthly_discount", nil
		}
	}

	return 0, "", nil
}

// checkCardTypeDiscount 检查卡类型折扣（学生卡、老人卡等）。
func (s *FareService) checkCardTypeDiscount(cardID string, baseFare float64) (float64, string, error) {
	var card models.Card
	err := s.db.Where("card_id = ?", cardID).First(&card).Error
	if err != nil {
		return 0, "", nil
	}

	var policy models.DiscountPolicy
	err = s.db.Where("policy_type = ? AND (card_type_filter = ? OR card_type_filter = '') AND status = 'active'",
		card.CardType, card.CardType).
		First(&policy).Error

	if err != nil {
		return 0, "", nil
	}
	return policy.DiscountRate, card.CardType + "_discount", nil
}

// FareCalculationResult 计费结果（最终返回给业务层/前端）。
type FareCalculationResult struct {
	BaseFare       float64 `json:"base_fare"`       // 基础票价
	DiscountAmount float64 `json:"discount_amount"` // 优惠金额
	DiscountType   string  `json:"discount_type"`   // 优惠类型
	ActualFare     float64 `json:"actual_fare"`     // 实收金额
	PenaltyFare    bool    `json:"penalty_fare"`    // 是否为罚款计费
}
