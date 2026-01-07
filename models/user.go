package models

import (
	"time"

	"gorm.io/gorm"
)

// User 用户信息（司机或运营人员账户）。
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Username string `gorm:"uniqueIndex;not null;size:50" json:"username"` // 用户名
	Password string `gorm:"not null;size:255" json:"-"`                   // 密码（哈希后）
	RealName string `gorm:"size:100" json:"real_name"`                    // 真实姓名
	Role     string `gorm:"size:50;default:'driver'" json:"role"`         // 角色：admin, driver, operator
	Status   string `gorm:"size:20;default:'active'" json:"status"`       // 状态：active, inactive
}

// TableName 指定表名，避免自动复数推断不一致。
func (User) TableName() string {
	return "users"
}
