package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// JSONB 用于 PostgreSQL 的 JSONB 类型映射。
type JSONB map[string]interface{}

// Value 实现 driver.Valuer 接口，写入数据库。
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan 实现 sql.Scanner 接口，从数据库读取。
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// TapEvent 刷卡事件记录（保留原始事件，便于稽核与问题排查）。
type TapEvent struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	RecordID    string    `gorm:"uniqueIndex;not null;size:100;index" json:"record_id"` // 记录ID（网关幂等键）
	CardID      string    `gorm:"index;not null;size:32" json:"card_id"`                // 卡片ID
	RouteID     uint      `gorm:"index" json:"route_id"`                                // 线路ID
	StationID   uint      `gorm:"index" json:"station_id"`                              // 站点ID
	StationName string    `gorm:"size:100" json:"station_name"`                         // 站点名称（冗余字段）
	TapType     string    `gorm:"size:20;not null" json:"tap_type"`                     // 刷卡类型：tap_in(进站), tap_out(出站), unknown(未知)
	TapTime     time.Time `gorm:"index;not null" json:"tap_time"`                       // 刷卡时间
	GatewayID   string    `gorm:"size:50" json:"gateway_id"`                            // 网关设备ID
	RawPayload  JSONB     `gorm:"type:jsonb" json:"raw_payload,omitempty"`              // 原始数据（JSON格式，可选）
	CreatedAt   time.Time `json:"created_at"`
}

// TableName 指定表名，避免自动复数推断不一致。
func (TapEvent) TableName() string {
	return "tap_events"
}
