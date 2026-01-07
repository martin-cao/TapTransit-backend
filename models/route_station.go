package models

import (
	"time"

	"gorm.io/gorm"
)

// RouteStation 线路与站点的关联关系（定义线路经过的站点顺序与方向）。
type RouteStation struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	RouteID    uint     `gorm:"not null;index" json:"route_id"`                  // 线路ID
	StationID  uint     `gorm:"not null;index" json:"station_id"`                // 站点ID
	Sequence   int      `gorm:"not null" json:"sequence"`                        // 站点在线路上的顺序（从1开始）
	Direction  string   `gorm:"size:20" json:"direction"`                        // 方向：up, down
	ZoneID     *string  `gorm:"size:50;index" json:"zone_id,omitempty"`          // 分区ID（用于分区/分段计价）
	DistanceKm *float64 `gorm:"type:decimal(10,3)" json:"distance_km,omitempty"` // 到线路起点的累计距离（公里）

	// 预加载用的关联结构，避免额外查询。
	Route   Route   `gorm:"foreignKey:RouteID;references:ID" json:"route,omitempty"`
	Station Station `gorm:"foreignKey:StationID;references:ID" json:"station,omitempty"`
}

// TableName 指定表名，避免自动复数推断不一致。
func (RouteStation) TableName() string {
	return "route_stations"
}
