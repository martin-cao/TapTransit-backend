package controllers

import (
	"TapTransit-backend/models"
	"TapTransit-backend/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type TransactionController struct {
}

func NewTransactionController() *TransactionController {
	return &TransactionController{}
}

// GetTransactions 查询交易记录
// @Summary 查询交易记录
// @Description 根据条件查询交易记录
// @Tags 交易记录
// @Produce json
// @Param date query string false "日期（格式：2006-01-02）"
// @Param route_id query int false "线路ID"
// @Param card_id query string false "卡片ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/transactions [get]
func (c *TransactionController) GetTransactions(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	dateStr := ctx.Query("date")
	routeIDStr := ctx.Query("route_id")
	cardID := ctx.Query("card_id")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 构建基础查询
	query := utils.DB.Model(&models.Transaction{})

	// 日期筛选（按上车时间）
	if dateStr != "" {
		date, err := time.Parse("2006-01-02", dateStr)
		if err == nil {
			startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
			endOfDay := startOfDay.Add(24 * time.Hour)
			query = query.Where("board_time >= ? AND board_time < ?", startOfDay, endOfDay)
		}
	}

	// 线路筛选
	if routeIDStr != "" {
		if routeID, err := strconv.ParseUint(routeIDStr, 10, 32); err == nil {
			query = query.Where("route_id = ?", routeID)
		}
	}

	// 卡片筛选
	if cardID != "" {
		query = query.Where("card_id = ?", cardID)
	}

	var total int64
	// 统计总数，用于分页
	query.Count(&total)

	var transactions []models.Transaction
	offset := (page - 1) * pageSize
	// 预加载关联信息，避免前端二次查询
	query.Preload("Card").Preload("Route").
		Order("board_time DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&transactions)

	// 返回数据和分页信息
	utils.Success(ctx, gin.H{
		"data": transactions,
		"pagination": gin.H{
			"page":      page,
			"page_size": pageSize,
			"total":     total,
			"pages":     (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}
