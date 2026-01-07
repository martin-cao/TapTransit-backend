package services

import (
	"TapTransit-backend/models"
	"TapTransit-backend/utils"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
)

type UploadService struct {
	db          *gorm.DB
	fareService *FareService
}

// NewUploadService 创建记录上传服务。
func NewUploadService(db *gorm.DB, fareService *FareService) *UploadService {
	return &UploadService{
		db:          db,
		fareService: fareService,
	}
}

// BatchRecordRequest 网关上传的批量记录请求。
type BatchRecordRequest struct {
	RecordID      string       `json:"record_id"`                   // 记录 ID（网关幂等键，可选）
	CardID        string       `json:"card_id" binding:"required"`   // 卡片 UID
	BoardTime     FlexibleTime `json:"board_time" binding:"required"`// 上车时间
	BoardStation  string       `json:"board_station" binding:"required"` // 上车站点（名称或编号）
	AlightTime    *FlexibleTime `json:"alight_time"`                // 下车时间（可选）
	AlightStation string       `json:"alight_station"`              // 下车站点（可选）
	RouteID       uint         `json:"route_id"`                    // 线路 ID（可选）
	GatewayID     string       `json:"gateway_id"`                  // 网关设备 ID
}

// FlexibleTime 支持 UNIX 秒（数字或字符串）与 RFC3339 时间格式。
type FlexibleTime struct {
	time.Time
}

// UnmarshalJSON 兼容多种时间格式的反序列化。
func (t *FlexibleTime) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	if string(b) == "null" {
		t.Time = time.Time{}
		return nil
	}
	var s string
	if b[0] == '"' {
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		return t.parseString(s)
	}
	return t.parseString(string(b))
}

// parseString 解析时间字符串，支持 UNIX 秒与 RFC3339。
func (t *FlexibleTime) parseString(value string) error {
	if value == "" {
		t.Time = time.Time{}
		return nil
	}
	if secs, err := strconv.ParseInt(value, 10, 64); err == nil {
		t.Time = time.Unix(secs, 0).UTC()
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		t.Time = parsed
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		t.Time = parsed
		return nil
	}
	return fmt.Errorf("invalid time format: %s", value)
}

// UploadBatchRecords 批量上传乘车记录。
func (s *UploadService) UploadBatchRecords(records []BatchRecordRequest) (int, error) {
	var successCount int

	for _, record := range records {
		// 单条失败不影响整批处理
		if err := s.processSingleRecord(record); err != nil {
			// 记录错误但继续处理下一条
			fmt.Printf("处理记录失败: %v, 错误: %v\n", record, err)
			continue
		}
		successCount++
	}

	return successCount, nil
}

// processSingleRecord 处理单条记录（解析站点、幂等、计费与入库）。
func (s *UploadService) processSingleRecord(record BatchRecordRequest) error {
	if record.BoardTime.IsZero() {
		return fmt.Errorf("上车时间缺失")
	}
	// 解析站点信息（支持站点名称或站点编号）
	startStationID, startStationName, err := s.parseStation(record.BoardStation)
	if err != nil {
		return fmt.Errorf("解析上车站点失败: %w", err)
	}

	endStationID := uint(0)
	endStationName := ""
	if record.AlightStation != "" {
		endStationID, endStationName, err = s.parseStation(record.AlightStation)
		if err != nil {
			return fmt.Errorf("解析下车站点失败: %w", err)
		}
	}

	// 如果没有路由 ID，尝试从站点推断
	routeID := record.RouteID
	if routeID == 0 {
		routeID, _ = s.inferRouteFromStation(startStationID)
	}

	// 获取线路信息（用于判断刷卡模式）
	var route models.Route
	if err := s.db.First(&route, routeID).Error; err != nil {
		return fmt.Errorf("线路不存在: %w", err)
	}

	// 检查卡片是否存在，不存在则创建
	var card models.Card
	err = s.db.Where("card_id = ?", record.CardID).First(&card).Error
	if err == gorm.ErrRecordNotFound {
		card = models.Card{
			CardID:   record.CardID,
			CardType: "normal",
			Status:   "active",
		}
		if err := s.db.Create(&card).Error; err != nil {
			return fmt.Errorf("创建卡片失败: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("查询卡片失败: %w", err)
	}

	// 检查卡片状态
	if card.Status != "active" {
		return fmt.Errorf("卡片状态异常: %s", card.Status)
	}

	// 检查重复刷卡（冷却时间 10-30 秒）
	var recentTransaction models.Transaction
	boardTime := record.BoardTime.Time
	var alightTime *time.Time
	if record.AlightTime != nil && !record.AlightTime.IsZero() {
		at := record.AlightTime.Time
		alightTime = &at
	}
	if alightTime == nil {
		err = s.db.Where("card_id = ? AND route_id = ? AND board_time > ?",
			record.CardID, routeID, boardTime.Add(-30*time.Second)).
			Order("board_time DESC").
			First(&recentTransaction).Error
		if err == nil {
			// 如果在 30 秒内有记录，检查是否在 10 秒内（认为是重复刷卡）
			if boardTime.Sub(recentTransaction.BoardTime).Seconds() < 10 {
				return nil // 重复刷卡，跳过
			}
		}
	}

	// 生成或使用 RecordID（幂等键）
	recordID := record.RecordID
	if recordID == "" {
		// 如果网关没有提供 RecordID，自动生成一个（格式：gatewayID_cardID_timestamp）
		recordID = fmt.Sprintf("%s_%s_%d", record.GatewayID, record.CardID, boardTime.Unix())
	}

	// 检查 RecordID 是否已存在（幂等性检查）
	var existingTransaction models.Transaction
	err = s.db.Where("record_id = ?", recordID).First(&existingTransaction).Error
	if err == nil {
		// RecordID 已存在，跳过处理（幂等性）
		return nil
	}

	// 创建交易记录（默认 pending，等待后续处理）
	transaction := models.Transaction{
		RecordID:         recordID,
		CardID:           record.CardID,
		RouteID:          routeID,
		StartStation:     startStationID,
		StartStationName: startStationName,
		EndStationName:   endStationName,
		BoardTime:        boardTime,
		GatewayID:        record.GatewayID,
		Status:           "pending",
	}

	// 根据线路的 TapMode 判断处理方式
	if route.TapMode == "tap_in_out" {
		// tap_in_out 模式：需要下车刷卡
		return s.processTapInOutMode(record, route, transaction, startStationID, startStationName, endStationID, endStationName, recordID, boardTime, alightTime)
	} else {
		// single_tap 模式：上车即计费（默认或明确指定）
		return s.processSingleTapMode(record, route, transaction, startStationID, startStationName, endStationID, endStationName, recordID, boardTime)
	}
}

// parseStation 解析站点信息（简化版，实际可扩展更多规则）。
func (s *UploadService) parseStation(stationInfo string) (uint, string, error) {
	// 尝试查找站点（通过station_id或name匹配）
	var station models.Station
	err := s.db.Where("station_id = ? OR name = ?", stationInfo, stationInfo).First(&station).Error
	if err == nil {
		return station.ID, station.Name, nil
	}

	// 如果找不到，返回错误（调用方会使用原始字符串作为名称）
	return 0, stationInfo, fmt.Errorf("站点不存在: %s", stationInfo)
}

// processSingleTapMode 处理 single_tap 模式（上车即计费）。
func (s *UploadService) processSingleTapMode(record BatchRecordRequest, route models.Route, transaction models.Transaction, startStationID uint, startStationName string, endStationID uint, endStationName string, recordID string, boardTime time.Time) error {
	// single_tap 模式：上车即完成计费，不需要下车站点
	var endStationPtr *uint
	if endStationID > 0 {
		endStationPtr = &endStationID
		transaction.EndStation = endStationPtr
		transaction.EndStationName = endStationName
	}

	// 记录 TapEvent（上车刷卡）
	tapEventID := fmt.Sprintf("%s_tap_%d", recordID, boardTime.UnixNano())
	if err := s.createTapEvent(tapEventID, record.CardID, route.ID, startStationID, startStationName, "tap_in", boardTime, record.GatewayID); err != nil {
		fmt.Printf("记录TapEvent失败: %v\n", err)
	}

	// 使用新的计费逻辑 CalculateFareV2
	fareResult, err := s.fareService.CalculateFareV2(
		record.CardID,
		route.ID,
		startStationID,
		endStationPtr, // single_tap模式下可能为nil
		boardTime,
		false, // 不是罚款计费
	)
	if err != nil {
		return fmt.Errorf("计算费用失败: %w", err)
	}

	transaction.Fare = fareResult.BaseFare
	transaction.ActualFare = fareResult.ActualFare
	transaction.DiscountType = fareResult.DiscountType
	transaction.DiscountAmount = fareResult.DiscountAmount
	transaction.PenaltyFare = fareResult.PenaltyFare
	transaction.Status = "completed"

	// 更新数据库中的月度累计金额
	// 注意：罚款计费不计入月度累计
	if !fareResult.PenaltyFare {
		if err := utils.IncrementMonthlyAggregate(s.db, record.CardID, fareResult.ActualFare); err != nil {
			fmt.Printf("更新月度累计失败: %v\n", err)
		}
	}

	// 保存交易记录
	if err := s.db.Create(&transaction).Error; err != nil {
		return fmt.Errorf("保存交易记录失败: %w", err)
	}

	return nil
}

// processTapInOutMode 处理 tap_in_out 模式（需要下车刷卡）。
func (s *UploadService) processTapInOutMode(record BatchRecordRequest, route models.Route, transaction models.Transaction, startStationID uint, startStationName string, endStationID uint, endStationName string, recordID string, boardTime time.Time, alightTime *time.Time) error {
	// tap_in_out 模式：有下车站点则完成 pending；否则生成 pending 交易
	if alightTime != nil && endStationID > 0 {
		// 有下车站点，查找该卡的 pending 交易
		var pendingTransaction models.Transaction
		err := s.db.Where("card_id = ? AND status = ? AND route_id = ?", record.CardID, "pending", route.ID).
			Order("board_time DESC").
			First(&pendingTransaction).Error

		if err == nil {
			// 找到 pending 交易，更新为完成状态
			// 记录 TapEvent（下车刷卡，匹配 pending 交易）
			tapEventID := fmt.Sprintf("%s_tapout_%d", pendingTransaction.RecordID, alightTime.UnixNano())
			if err := s.createTapEvent(tapEventID, record.CardID, route.ID, endStationID, endStationName, "tap_out", *alightTime, record.GatewayID); err != nil {
				fmt.Printf("记录TapEvent失败: %v\n", err)
			}

			pendingTransaction.EndStation = &endStationID
			pendingTransaction.EndStationName = endStationName
			pendingTransaction.AlightTime = alightTime

			// 使用新的计费逻辑 CalculateFareV2（使用 pending 交易的上车站点信息）
			fareResult, err := s.fareService.CalculateFareV2(
				record.CardID,
				route.ID,
				pendingTransaction.StartStation,
				&endStationID,
				pendingTransaction.BoardTime,
				false, // 不是罚款计费（有下车站点）
			)
			if err != nil {
				return fmt.Errorf("计算费用失败: %w", err)
			}

			pendingTransaction.Fare = fareResult.BaseFare
			pendingTransaction.ActualFare = fareResult.ActualFare
			pendingTransaction.DiscountType = fareResult.DiscountType
			pendingTransaction.DiscountAmount = fareResult.DiscountAmount
			pendingTransaction.PenaltyFare = fareResult.PenaltyFare
			pendingTransaction.Status = "completed"

			// 更新数据库中的月度累计金额
			if !fareResult.PenaltyFare {
				if err := utils.IncrementMonthlyAggregate(s.db, record.CardID, fareResult.ActualFare); err != nil {
					fmt.Printf("更新月度累计失败: %v\n", err)
				}
			}

			// 更新 pending 交易为完成状态
			if err := s.db.Save(&pendingTransaction).Error; err != nil {
				return fmt.Errorf("更新交易记录失败: %w", err)
			}

			return nil
		} else {
			// 没有找到 pending 交易，可能是新的一次完整的上下车记录
			// 记录 TapEvent（上车和下车，一次性上报）
			tapEventInID := fmt.Sprintf("%s_tapin_%d", recordID, boardTime.UnixNano())
			if err := s.createTapEvent(tapEventInID, record.CardID, route.ID, startStationID, startStationName, "tap_in", boardTime, record.GatewayID); err != nil {
				fmt.Printf("记录TapEvent失败: %v\n", err)
			}
			tapEventOutID := fmt.Sprintf("%s_tapout_%d", recordID, alightTime.UnixNano())
			if err := s.createTapEvent(tapEventOutID, record.CardID, route.ID, endStationID, endStationName, "tap_out", *alightTime, record.GatewayID); err != nil {
				fmt.Printf("记录TapEvent失败: %v\n", err)
			}

			// 创建新的完成交易
			transaction.EndStation = &endStationID
			transaction.EndStationName = endStationName
			transaction.AlightTime = alightTime

			// 使用新的计费逻辑 CalculateFareV2
			fareResult, err := s.fareService.CalculateFareV2(
				record.CardID,
				route.ID,
				startStationID,
				&endStationID,
				boardTime,
				false, // 不是罚款计费（有下车站点）
			)
			if err != nil {
				return fmt.Errorf("计算费用失败: %w", err)
			}

			transaction.Fare = fareResult.BaseFare
			transaction.ActualFare = fareResult.ActualFare
			transaction.DiscountType = fareResult.DiscountType
			transaction.DiscountAmount = fareResult.DiscountAmount
			transaction.PenaltyFare = fareResult.PenaltyFare
			transaction.Status = "completed"

			// 更新数据库中的月度累计金额
			if !fareResult.PenaltyFare {
				if err := utils.IncrementMonthlyAggregate(s.db, record.CardID, fareResult.ActualFare); err != nil {
					fmt.Printf("更新月度累计失败: %v\n", err)
				}
			}

			// 保存交易记录
			if err := s.db.Create(&transaction).Error; err != nil {
				return fmt.Errorf("保存交易记录失败: %w", err)
			}

			return nil
		}
	} else {
		// 只有上车记录，生成 pending 交易（等待下车刷卡）
		// 记录 TapEvent（上车刷卡）
		tapEventID := fmt.Sprintf("%s_tapin_%d", recordID, boardTime.UnixNano())
		if err := s.createTapEvent(tapEventID, record.CardID, route.ID, startStationID, startStationName, "tap_in", boardTime, record.GatewayID); err != nil {
			fmt.Printf("记录TapEvent失败: %v\n", err)
		}

		transaction.Status = "pending"
		transaction.Fare = 0
		transaction.ActualFare = 0

		// 检查是否已有 pending 交易（同一张卡）
		var existingPending models.Transaction
		err := s.db.Where("card_id = ? AND status = ? AND route_id = ?", record.CardID, "pending", route.ID).
			Order("board_time DESC").
			First(&existingPending).Error

		if err == nil {
			// 如果已有 pending 交易，可能是重复刷卡
			// 这里简化处理：如果时间间隔很短（如 30 秒内），跳过
			if boardTime.Sub(existingPending.BoardTime).Seconds() < 30 {
				return nil // 重复刷卡，跳过
			}
		}

		// 保存 pending 交易
		if err := s.db.Create(&transaction).Error; err != nil {
			return fmt.Errorf("保存pending交易失败: %w", err)
		}

		return nil
	}
}

// createTapEvent 创建 TapEvent 记录。
func (s *UploadService) createTapEvent(recordID string, cardID string, routeID uint, stationID uint, stationName string, tapType string, tapTime time.Time, gatewayID string) error {
	tapEvent := models.TapEvent{
		RecordID:    recordID,
		CardID:      cardID,
		RouteID:     routeID,
		StationID:   stationID,
		StationName: stationName,
		TapType:     tapType,
		TapTime:     tapTime,
		GatewayID:   gatewayID,
	}

	if err := s.db.Create(&tapEvent).Error; err != nil {
		return fmt.Errorf("创建TapEvent失败: %w", err)
	}

	return nil
}

// inferRouteFromStation 从站点推断线路。
func (s *UploadService) inferRouteFromStation(stationID uint) (uint, error) {
	var routeStation models.RouteStation
	err := s.db.Where("station_id = ?", stationID).First(&routeStation).Error
	if err != nil {
		return 0, err
	}
	return routeStation.RouteID, nil
}
