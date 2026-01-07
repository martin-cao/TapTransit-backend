package models

import (
	"time"

	"gorm.io/gorm"
)

// Route 公交线路信息与计费模式配置。
type Route struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	RouteID       string  `gorm:"uniqueIndex;not null;size:50" json:"route_id"` // 线路编号，如"A"、"1"、"K1"
	Name          string  `gorm:"size:100;not null" json:"name"`                // 线路名称
	Status        string  `gorm:"size:20;default:'active'" json:"status"`       // 状态：active, inactive
	FareType      string  `gorm:"size:50;default:'uniform'" json:"fare_type"`   // 计价模式：uniform(统一), segment(分段), distance(距离)
	TapMode       string  `gorm:"size:20;default:'single_tap'" json:"tap_mode"` // 刷卡模式：single_tap(单次刷卡), tap_in_out(进出站刷卡)
	MaxFare       float64 `gorm:"type:decimal(10,2);default:0" json:"max_fare"` // 最高无优惠票价（用于罚款计费）
	DirectionMode string  `gorm:"size:20;default:'both'" json:"direction_mode"` // 方向模式：single(单向), both(双向)
}

// TableName 指定表名，避免自动复数推断不一致。
func (Route) TableName() string {
	return "routes"
}
