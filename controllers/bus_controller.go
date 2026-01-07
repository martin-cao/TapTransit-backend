package controllers

import (
	"TapTransit-backend/services"
	"TapTransit-backend/utils"

	"github.com/gin-gonic/gin"
)

type BusController struct {
	// 依赖上传服务，用于处理网关批量记录
	uploadService *services.UploadService
}

func NewBusController(uploadService *services.UploadService) *BusController {
	return &BusController{
		uploadService: uploadService,
	}
}

// UploadBatchRecords 批量上传乘车记录
// @Summary 批量上传乘车记录
// @Description 网关上传批量乘车记录
// @Tags 公交数据
// @Accept json
// @Produce json
// @Param records body []services.BatchRecordRequest true "批量记录"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/bus/batchRecords [post]
func (c *BusController) UploadBatchRecords(ctx *gin.Context) {
	var records []services.BatchRecordRequest
	if err := ctx.ShouldBindJSON(&records); err != nil {
		// JSON 结构不合法或字段缺失
		utils.BadRequest(ctx, "请求参数错误: "+err.Error())
		return
	}

	// 交给服务层做业务处理（计费、写库、幂等）
	successCount, err := c.uploadService.UploadBatchRecords(records)
	if err != nil {
		utils.InternalServerError(ctx, "处理记录失败: "+err.Error())
		return
	}

	// 返回接收数量与总数量，便于网关统计
	utils.Success(ctx, gin.H{
		"received": successCount,
		"total":    len(records),
	})
}
