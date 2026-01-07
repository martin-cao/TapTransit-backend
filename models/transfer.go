package models

import (
	"time"

	"gorm.io/gorm"
)

// Transfer 换乘优惠规则（起始线路/站点 -> 目标线路/站点）。
type Transfer struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	FromRouteID    uint    `gorm:"index" json:"from_route_id"`                             // 起始线路 ID
	FromStationID  uint    `gorm:"index" json:"from_station_id"`                           // 起始站点 ID（换乘站）
	ToRouteID      uint    `gorm:"index" json:"to_route_id"`                               // 换乘后线路 ID
	ToStationID    uint    `gorm:"index" json:"to_station_id"`                             // 换乘后站点 ID
	DiscountAmount float64 `gorm:"type:decimal(10,2);default:0" json:"discount_amount"`   // 优惠金额
	DiscountRate   float64 `gorm:"type:decimal(5,4);default:0" json:"discount_rate"`      // 优惠比例（0-1）
	TimeWindow     int     `gorm:"default:60" json:"time_window"`                         // 时间窗口（分钟）
	Status         string  `gorm:"size:20;default:'active'" json:"status"`                // 状态：active/inactive
}

// TableName 指定表名，避免自动复数推断不一致。
func (Transfer) TableName() string {
	return "transfers"
}
