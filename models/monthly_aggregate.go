package models

import (
	"time"
)

// MonthlyAggregate 月度累计金额（用于优化月度累计折扣计算）。
type MonthlyAggregate struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CardID      string    `gorm:"uniqueIndex:idx_card_month;not null;size:32;index" json:"card_id"` // 卡片ID
	Month       string    `gorm:"uniqueIndex:idx_card_month;not null;size:7" json:"month"`          // 月份（YYYY-MM格式）
	TotalAmount float64   `gorm:"type:decimal(10,2);default:0;not null" json:"total_amount"`        // 当月累计金额
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名，避免自动复数推断不一致。
func (MonthlyAggregate) TableName() string {
	return "monthly_aggregates"
}
