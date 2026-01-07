package controllers

import (
	"TapTransit-backend/models"
	"TapTransit-backend/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ConfigController struct {
}

func NewConfigController() *ConfigController {
	return &ConfigController{}
}

// GetRouteConfig 获取线路配置信息
// @Summary 获取线路配置
// @Description 获取线路的站点列表、票价表、换乘优惠等信息
// @Tags 配置
// @Produce json
// @Param route_id query int true "线路ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/bus/config [get]
func (c *ConfigController) GetRouteConfig(ctx *gin.Context) {
	routeIDStr := ctx.Query("route_id")
	if routeIDStr == "" {
		utils.BadRequest(ctx, "缺少route_id参数")
		return
	}

	// 解析并校验 route_id
	routeID, err := strconv.ParseUint(routeIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(ctx, "route_id参数格式错误")
		return
	}

	var route models.Route
	if err := utils.DB.Where("id = ?", routeID).First(&route).Error; err != nil {
		utils.NotFound(ctx, "线路不存在")
		return
	}

	// 获取票价规则
	var fares []models.Fare
	utils.DB.Where("route_id = ? AND status = 'active'", routeID).Find(&fares)

	// 获取换乘优惠规则
	var transfers []models.Transfer
	utils.DB.Where("(from_route_id = ? OR to_route_id = ?) AND status = 'active'", routeID, routeID).Find(&transfers)

	// 获取站点列表（通过 RouteStation 关联）
	var routeStations []models.RouteStation
	utils.DB.Where("route_id = ?", routeID).
		Order("sequence ASC").
		Find(&routeStations)

	// 拉取站点基础信息，并按 ID 建索引
	stationIDs := make([]uint, 0, len(routeStations))
	for _, rs := range routeStations {
		stationIDs = append(stationIDs, rs.StationID)
	}

	stationsByID := make(map[uint]models.Station, len(stationIDs))
	if len(stationIDs) > 0 {
		var stations []models.Station
		utils.DB.Where("id IN ?", stationIDs).Find(&stations)
		for _, station := range stations {
			stationsByID[station.ID] = station
		}
	}

	// 输出结构对齐网关侧配置格式
	stations := make([]map[string]interface{}, 0, len(routeStations))
	for _, rs := range routeStations {
		station, ok := stationsByID[rs.StationID]
		if !ok {
			continue
		}
		stations = append(stations, map[string]interface{}{
			"id":          station.ID,
			"station_id":  station.StationID,
			"name":        station.Name,
			"sequence":    rs.Sequence,
			"is_transfer": station.IsTransfer,
		})
	}

	// 返回整条线路配置（站点、票价、换乘规则）
	utils.Success(ctx, gin.H{
		"route_id":   route.ID,
		"route_name": route.Name,
		"fare_type":  route.FareType,
		"tap_mode":   route.TapMode,
		"max_fare":   route.MaxFare,
		"stations":   stations,
		"fares":      fares,
		"transfers":  transfers,
	})
}
