package models

import (
	"time"

	"gorm.io/gorm"
)

// Station 站点基础信息。
type Station struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	StationID  string  `gorm:"uniqueIndex;not null;size:50" json:"station_id"` // 站点编号
	Name       string  `gorm:"size:100;not null" json:"name"`                  // 站点名称
	Latitude   float64 `gorm:"type:decimal(10,8)" json:"latitude"`             // 纬度
	Longitude  float64 `gorm:"type:decimal(11,8)" json:"longitude"`            // 经度
	Address    string  `gorm:"size:200" json:"address"`                        // 地址
	IsTransfer bool    `gorm:"default:false" json:"is_transfer"`               // 是否为换乘站
}

// TableName 指定表名，避免自动复数推断不一致。
func (Station) TableName() string {
	return "stations"
}
