package models

import (
	"time"

	"gorm.io/gorm"
)

// Transaction 乘车交易记录（可能包含未完成行程）。
type Transaction struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	RecordID         string     `gorm:"uniqueIndex;size:100" json:"record_id"`               // 记录 ID（网关幂等键）
	CardID           string     `gorm:"index;not null;size:32" json:"card_id"`               // 卡 ID
	RouteID          uint       `gorm:"index" json:"route_id"`                               // 线路 ID
	StartStation     uint       `gorm:"index" json:"start_station"`                          // 上车站点 ID
	EndStation       *uint      `gorm:"index" json:"end_station,omitempty"`                  // 下车站点 ID（可为空）
	StartStationName string     `gorm:"size:100" json:"start_station_name"`                  // 上车站点名称（冗余便于查询）
	EndStationName   string     `gorm:"size:100" json:"end_station_name"`                    // 下车站点名称
	BoardTime        time.Time  `gorm:"index;not null" json:"board_time"`                    // 上车时间
	AlightTime       *time.Time `gorm:"index" json:"alight_time,omitempty"`                  // 下车时间（空表示未下车）
	Fare             float64    `gorm:"type:decimal(10,2);not null" json:"fare"`             // 应收金额（基础票价）
	ActualFare       float64    `gorm:"type:decimal(10,2);not null" json:"actual_fare"`      // 实收金额（优惠后）
	DiscountType     string     `gorm:"size:50" json:"discount_type"`                        // 优惠类型：transfer/monthly/student
	DiscountAmount   float64    `gorm:"type:decimal(10,2);default:0" json:"discount_amount"` // 优惠金额
	PenaltyFare      bool       `gorm:"default:false" json:"penalty_fare"`                   // 是否罚款计费
	Status           string     `gorm:"size:20;default:'completed'" json:"status"`           // 状态：pending/completed/cancelled
	GatewayID        string     `gorm:"size:50" json:"gateway_id"`                           // 网关设备 ID（记录来源）

	// 预加载关联信息（卡片、线路）
	Card  Card  `gorm:"foreignKey:CardID;references:CardID" json:"card,omitempty"`
	Route Route `gorm:"foreignKey:RouteID" json:"route,omitempty"`
}

// TableName 指定表名，避免自动复数推断不一致。
func (Transaction) TableName() string {
	return "transactions"
}
