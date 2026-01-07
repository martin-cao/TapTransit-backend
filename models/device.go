package models

import (
	"time"

	"gorm.io/gorm"
)

// Device 设备信息（网关/读卡器）。
type Device struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	DeviceID      string     `gorm:"uniqueIndex;not null;size:50" json:"device_id"` // 设备 ID
	DeviceType    string     `gorm:"size:50;default:'gateway'" json:"device_type"`  // 设备类型：gateway/reader
	VehicleNumber string     `gorm:"size:50" json:"vehicle_number"`                 // 车辆编号
	Status        string     `gorm:"size:20;default:'active'" json:"status"`        // 状态：active/inactive/maintenance
	LastSeen      *time.Time `json:"last_seen"`                                     // 最后在线时间
}

// TableName 指定表名，避免自动复数推断不一致。
func (Device) TableName() string {
	return "devices"
}
