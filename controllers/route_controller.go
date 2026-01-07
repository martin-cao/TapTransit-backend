package controllers

import (
	"TapTransit-backend/models"
	"TapTransit-backend/utils"

	"github.com/gin-gonic/gin"
)

type RouteController struct {
}

func NewRouteController() *RouteController {
	return &RouteController{}
}

// GetRoutes 获取所有线路列表
// @Summary 获取线路列表
// @Description 获取所有线路信息
// @Tags 线路管理
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/routes [get]
func (c *RouteController) GetRoutes(ctx *gin.Context) {
	var routes []models.Route
	// 仅返回启用线路，避免前端展示无效数据
	if err := utils.DB.Where("status = 'active'").Find(&routes).Error; err != nil {
		utils.InternalServerError(ctx, "查询失败")
		return
	}

	utils.Success(ctx, routes)
}
