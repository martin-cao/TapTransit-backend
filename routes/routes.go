// Package routes 负责定义 Gin 的路由分组与控制器绑定。
package routes

import (
	"TapTransit-backend/controllers"
	"TapTransit-backend/services"
	"TapTransit-backend/utils"

	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置所有路由与 API 分组。
func SetupRoutes(r *gin.Engine) {
	// 初始化服务层（依赖全局 DB）
	fareService := services.NewFareService(utils.DB)
	uploadService := services.NewUploadService(utils.DB, fareService)
	cardService := services.NewCardService(utils.DB)

	// 初始化控制器（封装业务与响应）
	busController := controllers.NewBusController(uploadService)
	cardController := controllers.NewCardController(cardService)
	configController := controllers.NewConfigController()
	transactionController := controllers.NewTransactionController()
	routeController := controllers.NewRouteController()
	authController := controllers.NewAuthController()

	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		// 认证相关
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authController.Login)
			auth.POST("/logout", authController.Logout)
		}

		// 公交数据相关
		bus := v1.Group("/bus")
		{
			bus.POST("/batchRecords", busController.UploadBatchRecords) // 批量上传记录
			bus.GET("/config", configController.GetRouteConfig)         // 获取线路配置
		}

		// 卡片相关
		card := v1.Group("/card")
		{
			card.GET("/:id", cardController.GetCard) // 查询卡片信息
		}

		cards := v1.Group("/cards")
		{
			cards.GET("", cardController.ListCards) // 查询卡片列表
			cards.POST("/register", cardController.RegisterCard)
			cards.POST("/state/batch", cardController.UploadCardStateBatch)
		}

		// 交易记录相关
		transactions := v1.Group("/transactions")
		{
			transactions.GET("", transactionController.GetTransactions) // 查询交易记录
		}

		// 线路相关
		routes := v1.Group("/routes")
		{
			routes.GET("", routeController.GetRoutes) // 获取线路列表
		}
	}
}
