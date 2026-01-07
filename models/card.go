// Package models 定义数据库实体模型与 GORM 映射。
package models

import (
	"time"

	"gorm.io/gorm"
)

// Card IC 卡信息与票种状态。
type Card struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	CardID     string  `gorm:"uniqueIndex;not null;size:32" json:"card_id"` // 卡片 UID
	HolderName string  `gorm:"size:100" json:"holder_name"`                // 持有人姓名（可选）
	CardType   string  `gorm:"size:50;default:'normal'" json:"card_type"`   // 卡类型：normal/student/elder/disabled
	Status     string  `gorm:"size:20;default:'active'" json:"status"`      // 状态：active/blocked/lost
	Balance    float64 `gorm:"default:0" json:"balance"`                   // 卡内余额（如支持电子钱包）
}

// TableName 指定表名，避免自动复数推断不一致。
func (Card) TableName() string {
	return "cards"
}
