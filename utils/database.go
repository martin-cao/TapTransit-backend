// Package utils 提供数据库、Redis、响应等通用工具函数。
package utils

import (
	"TapTransit-backend/config"
	"TapTransit-backend/models"
	"fmt"
	"log"

	"gorm.io/gorm"
)

// DB 为全局数据库连接，供路由与服务层复用。
var DB *gorm.DB

// InitDatabase 初始化数据库并执行自动迁移（按依赖顺序创建表）。
func InitDatabase(cfg *config.Config) (*gorm.DB, error) {
	db, err := config.InitDB(&cfg.Database)
	if err != nil {
		return nil, err
	}

	// 按依赖顺序逐个迁移表，确保被引用的表先创建。
	// 注意：Route 模型中的 Stations 关联不会影响表创建，因为外键在 RouteStation 表中。

	// 第一阶段：基础表（无外键依赖，只创建表结构）
	log.Println("开始迁移数据库表...")

	// 定义迁移表列表，包含表名和模型（便于日志记录）。
	migrations := []struct {
		name  string
		model interface{}
	}{
		// 第一阶段：基础表（无外键依赖）
		{"cards", &models.Card{}},
		{"routes", &models.Route{}},
		{"stations", &models.Station{}},
		{"devices", &models.Device{}},
		{"users", &models.User{}},
		{"discount_policies", &models.DiscountPolicy{}},
		// 第二阶段：关联表（依赖基础表）
		{"route_stations", &models.RouteStation{}},
		{"fares", &models.Fare{}},
		{"transfers", &models.Transfer{}},
		// 第三阶段：交易表和扩展表（依赖基础表）
		{"transactions", &models.Transaction{}},
		{"monthly_aggregates", &models.MonthlyAggregate{}},
		{"tap_events", &models.TapEvent{}},
	}

	// 逐个迁移表，失败则立即返回错误。
	for _, m := range migrations {
		if err := db.AutoMigrate(m.model); err != nil {
			return nil, fmt.Errorf("迁移表 %s 失败: %w", m.name, err)
		}
		log.Printf("✓ %s 表创建完成", m.name)
	}

	log.Println("✅ 所有数据库表迁移完成")

	DB = db
	return db, nil
}
