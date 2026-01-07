package services

import (
	"TapTransit-backend/models"
	"TapTransit-backend/utils"
	"fmt"
	"math"
	"time"
)

// CalculateFareV2 计算单次乘车费用（按 TapTransit 设计文档规则）。
// 计算顺序：基础票价 → 罚款计费 → 特殊票种 → 换乘优惠 → 月度折扣
func (s *FareService) CalculateFareV2(cardID string, routeID uint, startStationID uint, endStationID *uint, boardTime time.Time, isPenaltyFare bool) (*FareCalculationResult, error) {
	result := &FareCalculationResult{
		BaseFare:       0,
		DiscountAmount: 0,
		DiscountType:   "",
		ActualFare:     0,
		PenaltyFare:    isPenaltyFare,
	}

	// 获取线路信息
	var route models.Route
	if err := s.db.First(&route, routeID).Error; err != nil {
		return nil, fmt.Errorf("线路不存在: %w", err)
	}

	// 1. 计算基础票价
	baseFare, err := s.calculateBaseFareV2(&route, startStationID, endStationID)
	if err != nil {
		return nil, fmt.Errorf("计算基础票价失败: %w", err)
	}
	result.BaseFare = baseFare
	result.ActualFare = baseFare

	// 2. 如果是罚款计费，直接使用 max_fare，不享受任何优惠
	if isPenaltyFare {
		if route.MaxFare > 0 {
			result.BaseFare = route.MaxFare
			result.ActualFare = route.MaxFare
		} else {
			result.ActualFare = baseFare
		}
		result.DiscountAmount = 0
		result.DiscountType = ""
		result.ActualFare = s.roundDown(result.ActualFare, 2)
		return result, nil
	}

	// 3. 特殊票种折扣（优先级最高）
	var card models.Card
	if err := s.db.Where("card_id = ?", cardID).First(&card).Error; err == nil {
		cardDiscount, cardType, isFree := s.checkCardTypeDiscountV2(card.CardType, result.ActualFare)
		if cardDiscount > 0 {
			result.ActualFare -= cardDiscount
			if result.ActualFare < 0 {
				result.ActualFare = 0
			}
			result.DiscountAmount += cardDiscount
			result.DiscountType = cardType
			// 若为全免，直接返回，跳过后续折扣
			if isFree {
				result.ActualFare = s.roundDown(result.ActualFare, 2)
				if route.MaxFare > 0 && result.ActualFare > route.MaxFare {
					result.ActualFare = route.MaxFare
				}
				return result, nil
			}
		}
	}

	// 4. 换乘优惠
	transferDiscount, transferType := s.checkTransferDiscountV2(cardID, routeID, startStationID, boardTime, result.ActualFare)
	if transferDiscount > 0 {
		result.ActualFare -= transferDiscount
		if result.ActualFare < 0 {
			result.ActualFare = 0
		}
		result.DiscountAmount += transferDiscount
		if result.DiscountType != "" {
			result.DiscountType += "," + transferType
		} else {
			result.DiscountType = transferType
		}
	}

	// 5. 月度累计折扣
	monthlyDiscountRate, monthlyType := s.checkMonthlyDiscountV2(cardID, result.ActualFare)
	if monthlyDiscountRate > 0 {
		discountAmount := result.ActualFare * monthlyDiscountRate
		result.ActualFare -= discountAmount
		if result.ActualFare < 0 {
			result.ActualFare = 0
		}
		result.DiscountAmount += discountAmount
		if result.DiscountType != "" {
			result.DiscountType += "," + monthlyType
		} else {
			result.DiscountType = monthlyType
		}
	}

	// 6. 边界处理：向下保留 2 位小数，确保不超过 max_fare
	result.ActualFare = s.roundDown(result.ActualFare, 2)
	if route.MaxFare > 0 && result.ActualFare > route.MaxFare {
		result.ActualFare = route.MaxFare
	}

	return result, nil
}

// roundDown 向下保留 n 位小数（防止浮点舍入导致超收）。
func (s *FareService) roundDown(value float64, decimals int) float64 {
	multiplier := math.Pow(10, float64(decimals))
	return math.Floor(value*multiplier) / multiplier
}

// calculateBaseFareV2 计算基础票价（支持新的计费规则）。
func (s *FareService) calculateBaseFareV2(route *models.Route, startStationID uint, endStationID *uint) (float64, error) {
	switch route.FareType {
	case "uniform":
		// 统一票价
		return s.getUniformFareV2(route.ID, route.MaxFare), nil
	case "segment":
		if endStationID != nil && *endStationID > 0 {
			// 进出站模式：优先站点对定价
			fare := s.getStationPairFare(route.ID, startStationID, *endStationID)
			if fare > 0 {
				return fare, nil
			}
			// 站数阶梯计费
			return s.calculateSegmentFareByStations(route.ID, startStationID, *endStationID), nil
		} else {
			// 单次刷卡：按上车站分区计价
			return s.calculateSegmentFareByZone(route.ID, startStationID, route.MaxFare), nil
		}
	case "distance":
		if endStationID == nil {
			// 缺少下车站时退化为统一票价
			return s.getUniformFareV2(route.ID, route.MaxFare), nil
		}
		return s.calculateSegmentFareByStations(route.ID, startStationID, *endStationID), nil
	default:
		return s.getUniformFareV2(route.ID, route.MaxFare), nil
	}
}

// getUniformFareV2 获取统一票价（无匹配则用 max_fare 兜底）。
func (s *FareService) getUniformFareV2(routeID uint, maxFare float64) float64 {
	var fare models.Fare
	err := s.db.Where("route_id = ? AND fare_type = 'uniform' AND status = 'active'", routeID).First(&fare).Error
	if err == nil {
		return fare.BasePrice
	}
	if maxFare > 0 {
		return maxFare
	}
	return 2.0
}

// getStationPairFare 获取站点对定价（优先匹配）。
func (s *FareService) getStationPairFare(routeID uint, startStationID, endStationID uint) float64 {
	var fare models.Fare
	err := s.db.Where("route_id = ? AND start_station = ? AND end_station = ? AND status = 'active'",
		routeID, startStationID, endStationID).First(&fare).Error
	if err == nil {
		return fare.BasePrice
	}
	return 0
}

// calculateSegmentFareByZone 分段计价（single_tap 模式，按上车站 zone 定价）。
func (s *FareService) calculateSegmentFareByZone(routeID uint, startStationID uint, maxFare float64) float64 {
	var routeStation models.RouteStation
	err := s.db.Where("route_id = ? AND station_id = ?", routeID, startStationID).First(&routeStation).Error
	if err != nil || routeStation.ZoneID == nil {
		return s.getUniformFareV2(routeID, maxFare)
	}
	var fare models.Fare
	err = s.db.Where("route_id = ? AND start_station = ? AND status = 'active'", routeID, startStationID).First(&fare).Error
	if err == nil {
		return fare.BasePrice
	}
	if maxFare > 0 {
		return maxFare
	}
	return s.getUniformFareV2(routeID, maxFare)
}

// calculateSegmentFareByStations 分段计价（tap_in_out 模式，按站数阶梯计费）。
// 具体阶梯由 fare_rules 中的 base_price/extra_price/segment_count 决定。
func (s *FareService) calculateSegmentFareByStations(routeID uint, startStationID, endStationID uint) float64 {
	segmentCount := s.calculateSegmentCountV2(routeID, startStationID, endStationID)
	if segmentCount <= 0 {
		return 2.0
	}
	var fare models.Fare
	err := s.db.Where(
		"route_id = ? AND fare_type = 'segment' AND status = 'active' AND start_station = 0 AND end_station = 0",
		routeID,
	).First(&fare).Error
	if err != nil {
		return 2.0
	}
	base := fare.BasePrice
	if base <= 0 {
		base = 2.0
	}
	extra := fare.ExtraPrice
	included := fare.SegmentCount
	if included <= 0 {
		included = 1
	}
	if segmentCount <= included || extra <= 0 {
		return base
	}
	return base + float64(segmentCount-included)*extra
}

// calculateSegmentCountV2 计算两个站点间的站数。
func (s *FareService) calculateSegmentCountV2(routeID uint, startStationID, endStationID uint) int {
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

// checkCardTypeDiscountV2 检查卡类型折扣（默认值：学生 8 折、长者 5 折、爱心 0 元）。
func (s *FareService) checkCardTypeDiscountV2(cardType string, currentFare float64) (float64, string, bool) {
	if cardType == "normal" {
		return 0, "", false
	}
	var policy models.DiscountPolicy
	err := s.db.Where("policy_type = ? AND (card_type_filter = ? OR card_type_filter = '') AND status = 'active'",
		cardType, cardType).First(&policy).Error
	if err != nil {
		return s.getDefaultCardDiscount(cardType, currentFare)
	}
	var discountAmount float64
	if policy.DiscountAmount > 0 {
		discountAmount = policy.DiscountAmount
	} else if policy.DiscountRate >= 0 {
		discountAmount = currentFare * policy.DiscountRate
	}
	isFree := discountAmount >= currentFare
	return discountAmount, cardType + "_discount", isFree
}

// getDefaultCardDiscount 获取默认卡类型折扣。
func (s *FareService) getDefaultCardDiscount(cardType string, currentFare float64) (float64, string, bool) {
	switch cardType {
	case "student":
		return currentFare * 0.2, "student_discount", false
	case "elder":
		return currentFare * 0.5, "elder_discount", false
	case "disabled":
		return currentFare, "disabled_discount", true
	default:
		return 0, "", false
	}
}

// checkTransferDiscountV2 检查换乘优惠（优先级：fixed_fare > discount_amount > discount_rate）。
func (s *FareService) checkTransferDiscountV2(cardID string, routeID uint, stationID uint, boardTime time.Time, baseFare float64) (float64, string) {
	var lastTransaction models.Transaction
	// 查询最近一次已完成且有下车时间的行程
	err := s.db.Where("card_id = ? AND status = 'completed' AND alight_time IS NOT NULL", cardID).
		Order("alight_time DESC").First(&lastTransaction).Error
	if err != nil {
		return 0, ""
	}
	if lastTransaction.AlightTime == nil || lastTransaction.EndStation == nil {
		return 0, ""
	}
	var transfer models.Transfer
	// 匹配换乘规则（线路 + 站点）
	err = s.db.Where("from_route_id = ? AND from_station_id = ? AND to_route_id = ? AND to_station_id = ? AND status = 'active'",
		lastTransaction.RouteID, *lastTransaction.EndStation, routeID, stationID).First(&transfer).Error
	if err != nil {
		return 0, ""
	}
	// 时间窗口校验
	timeWindowMinutes := transfer.TimeWindow
	if timeWindowMinutes == 0 {
		timeWindowMinutes = 60
	}
	if boardTime.Sub(*lastTransaction.AlightTime).Minutes() > float64(timeWindowMinutes) {
		return 0, ""
	}
	var discountAmount float64
	if transfer.DiscountAmount > 0 {
		discountAmount = transfer.DiscountAmount
	} else if transfer.DiscountRate >= 0 {
		discountAmount = baseFare * transfer.DiscountRate
	}
	return discountAmount, "transfer"
}

// checkMonthlyDiscountV2 检查月度累计折扣（阈值：≥200 元 8 折，≥500 元 5 折）。
func (s *FareService) checkMonthlyDiscountV2(cardID string, currentAmountAfterDiscounts float64) (float64, string) {
	currentAmount, err := utils.GetCurrentMonthAggregate(s.db, cardID)
	if err != nil {
		return 0, ""
	}
	totalAmount := currentAmount + currentAmountAfterDiscounts
	if totalAmount >= 500 {
		return 0.5, "monthly_discount"
	} else if totalAmount >= 200 {
		return 0.2, "monthly_discount"
	}
	return 0, ""
}
