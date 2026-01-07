package controllers

import (
	"TapTransit-backend/models"
	"TapTransit-backend/services"
	"TapTransit-backend/utils"

	"github.com/gin-gonic/gin"
)

type CardController struct {
	// 卡片相关业务封装
	cardService *services.CardService
}

// CardProfileResponse 卡片信息 + 对应折扣策略（便于前端展示）。
type CardProfileResponse struct {
	models.Card
	DiscountRate   *float64 `json:"discount_rate,omitempty"`
	DiscountAmount *float64 `json:"discount_amount,omitempty"`
}

func NewCardController(cardService *services.CardService) *CardController {
	return &CardController{
		cardService: cardService,
	}
}

// GetCard 查询卡片信息
// @Summary 查询卡片信息
// @Description 根据卡ID查询卡片状态和信息
// @Tags 卡片管理
// @Produce json
// @Param id path string true "卡片ID"
// @Success 200 {object} models.Card
// @Router /api/v1/card/{id} [get]
func (c *CardController) GetCard(ctx *gin.Context) {
	cardID := ctx.Param("id")
	if cardID == "" {
		utils.BadRequest(ctx, "缺少卡片ID")
		return
	}

	// 查询卡片基础信息
	card, err := c.cardService.GetCardByID(cardID)
	if err != nil {
		utils.NotFound(ctx, "卡片不存在")
		return
	}

	// 叠加对应票种优惠信息
	profile := CardProfileResponse{Card: *card}
	if discount, err := c.cardService.GetCardDiscount(card.CardType); err == nil {
		profile.DiscountRate = &discount.DiscountRate
		profile.DiscountAmount = &discount.DiscountAmount
	}
	utils.Success(ctx, profile)
}

// ListCards 查询卡片列表
// @Summary 查询卡片列表
// @Description 支持按卡号、姓名、状态筛选
// @Tags 卡片管理
// @Produce json
// @Param card_id query string false "卡片ID（精确）"
// @Param cardNo query string false "卡号（模糊）"
// @Param userName query string false "持有人姓名（模糊）"
// @Param status query string false "状态 active/blocked/lost"
// @Success 200 {array} models.Card
// @Router /api/v1/cards [get]
func (c *CardController) ListCards(ctx *gin.Context) {
	cardID := ctx.Query("card_id")
	cardNo := ctx.Query("cardNo")
	holderName := ctx.Query("userName")
	status := ctx.Query("status")

	// 组装筛选条件
	filter := services.CardFilter{
		CardID:     cardID,
		CardNoLike: cardNo,
		HolderName: holderName,
		Status:     status,
	}

	cards, err := c.cardService.ListCards(filter)
	if err != nil {
		utils.InternalServerError(ctx, "查询卡片失败")
		return
	}
	// 逐张卡附加折扣信息（可能为空）
	profiles := make([]CardProfileResponse, 0, len(cards))
	for _, card := range cards {
		profile := CardProfileResponse{Card: card}
		if discount, err := c.cardService.GetCardDiscount(card.CardType); err == nil {
			profile.DiscountRate = &discount.DiscountRate
			profile.DiscountAmount = &discount.DiscountAmount
		}
		profiles = append(profiles, profile)
	}
	utils.Success(ctx, profiles)
}
