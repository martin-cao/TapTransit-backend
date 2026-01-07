// TapTransit 后端服务启动入口。
package main

import (
	"TapTransit-backend/config"
	"TapTransit-backend/middleware"
	"TapTransit-backend/routes"
	"TapTransit-backend/services"
	"TapTransit-backend/utils"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置（包含服务端口、数据库与 Redis 等信息）
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 设置 Gin 运行模式（开发/生产）
	gin.SetMode(cfg.Server.Mode)

	// 初始化数据库连接，并挂载到全局 utils.DB
	db, err := utils.InitDatabase(cfg)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 验证数据库连接（db 变量已存储在 utils.DB 中，供 routes 使用）
	if db == nil {
		log.Fatalf("数据库连接为空")
	}
	log.Println("数据库连接成功")

	// 初始化 Redis（失败则降级为直接查库）
	_, err = utils.InitRedis(cfg)
	if err != nil {
		log.Printf("初始化Redis失败（将使用数据库替代）: %v", err)
	} else {
		log.Println("Redis连接成功")
	}

	// 初始化服务层（计费、罚款、缓存刷新、数据清理）
	fareService := services.NewFareService(db)
	penaltyService := services.NewPenaltyService(db, fareService)
	cacheService := services.NewCacheService(db)
	cleanupService := services.NewCleanupService(db)

	// 启动配置缓存刷新定时任务（每 5 分钟刷新一次）
	cacheService.StartCacheRefreshTask(5)
	log.Println("配置缓存服务已启动（每5分钟刷新）")

	// 启动罚款计费定时任务（每 5 分钟检查一次，2 小时超时）
	penaltyService.StartPenaltyProcessor(5, 120)
	log.Println("罚款计费定时任务已启动（每5分钟检查，2小时超时）")

	// 启动数据清理定时任务（每 24 小时执行一次，保留 7 天）
	cleanupService.StartCleanupTask(24, 7)
	log.Println("数据清理定时任务已启动（每24小时执行，保留7天）")

	// 创建 Gin 引擎并挂载中间件
	r := gin.Default()

	// 配置 CORS（必须在路由之前）
	r.Use(middleware.CORS())

	// 设置业务路由（API 分组、鉴权等）
	routes.SetupRoutes(r)

	// 健康检查接口（便于部署探活）
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
			"status":  "ok",
		})
	})

	// 启动 HTTP 服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("服务器启动在 %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
