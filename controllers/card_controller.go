package controllers

import (
	"TapTransit-backend/models"
	"TapTransit-backend/services"
	"TapTransit-backend/utils"
	"errors"
	"math"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

// CardRegistrationRequest 网关注册卡片请求。
type CardRegistrationRequest struct {
	CardID       string `json:"card_id" binding:"required"`
	BalanceCents uint64 `json:"balance_cents"`
	Status       string `json:"status"`
	RegisteredAt uint64 `json:"registered_at"`
	GatewayID    string `json:"gateway_id"`
}

// CardStateSnapshotRequest 卡片状态快照请求。
type CardStateSnapshotRequest struct {
	CardID             string  `json:"card_id"`
	BalanceCents       uint64  `json:"balance_cents"`
	CardStatus         string  `json:"card_status"`
	EntryStationID     *uint   `json:"entry_station_id"`
	LastRouteID        *uint   `json:"last_route_id"`
	LastDirection      *string `json:"last_direction"`
	LastBoardStationID *uint   `json:"last_board_station_id"`
	LastAlightStationID *uint  `json:"last_alight_station_id"`
	UpdatedAt          uint64  `json:"updated_at"`
	Source             string  `json:"source"`
}

// CardStateReject 状态校验失败项。
type CardStateReject struct {
	CardID string `json:"card_id"`
	Reason string `json:"reason,omitempty"`
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

// RegisterCard 注册卡片（来自网关）。
func (c *CardController) RegisterCard(ctx *gin.Context) {
	var req CardRegistrationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(ctx, "请求参数错误: "+err.Error())
		return
	}
	status, ok := normalizeCardStatus(req.Status)
	if !ok {
		utils.BadRequest(ctx, "卡片状态不合法")
		return
	}
	balance := centsToBalance(req.BalanceCents)

	card, err := c.cardService.GetCardByID(req.CardID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			card = &models.Card{
				CardID:   req.CardID,
				CardType: "normal",
				Status:   status,
				Balance:  balance,
			}
			if err := c.cardService.CreateCard(card); err != nil {
				utils.InternalServerError(ctx, "创建卡片失败: "+err.Error())
				return
			}
		} else {
			utils.InternalServerError(ctx, "查询卡片失败: "+err.Error())
			return
		}
	} else {
		updates := map[string]interface{}{
			"status":  status,
			"balance": balance,
		}
		if err := c.cardService.UpdateCard(req.CardID, updates); err != nil {
			utils.InternalServerError(ctx, "更新卡片失败: "+err.Error())
			return
		}
	}

	utils.Success(ctx, gin.H{
		"card_id": req.CardID,
		"status":  status,
	})
}

// UploadCardStateBatch 批量校验卡片状态快照。
func (c *CardController) UploadCardStateBatch(ctx *gin.Context) {
	var snapshots []CardStateSnapshotRequest
	if err := ctx.ShouldBindJSON(&snapshots); err != nil {
		utils.BadRequest(ctx, "请求参数错误: "+err.Error())
		return
	}
	accepted := make([]string, 0, len(snapshots))
	rejected := make([]CardStateReject, 0)

	for _, snap := range snapshots {
		if snap.CardID == "" {
			rejected = append(rejected, CardStateReject{Reason: "missing card_id"})
			continue
		}
		if !validSnapshotStatus(snap.CardStatus) {
			rejected = append(rejected, CardStateReject{CardID: snap.CardID, Reason: "invalid status"})
			continue
		}
		card, err := c.cardService.GetCardByID(snap.CardID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if snap.Source == "register" {
					card = &models.Card{
						CardID:   snap.CardID,
						CardType: "normal",
						Status:   "active",
						Balance:  centsToBalance(snap.BalanceCents),
					}
					if err := c.cardService.CreateCard(card); err != nil {
						rejected = append(rejected, CardStateReject{CardID: snap.CardID, Reason: "create failed"})
						continue
					}
					accepted = append(accepted, snap.CardID)
					continue
				}
				rejected = append(rejected, CardStateReject{CardID: snap.CardID, Reason: "card not found"})
				continue
			}
			rejected = append(rejected, CardStateReject{CardID: snap.CardID, Reason: "query failed"})
			continue
		}

		if card.Status == "blocked" || card.Status == "lost" {
			rejected = append(rejected, CardStateReject{CardID: snap.CardID, Reason: "card blocked"})
			continue
		}
		if snap.CardStatus == "in_trip" && snap.EntryStationID == nil {
			rejected = append(rejected, CardStateReject{CardID: snap.CardID, Reason: "missing entry station"})
			continue
		}
		if snap.CardStatus == "idle" && snap.EntryStationID != nil {
			rejected = append(rejected, CardStateReject{CardID: snap.CardID, Reason: "unexpected entry station"})
			continue
		}
		if snap.CardStatus == "blocked" {
			_ = c.cardService.UpdateCard(snap.CardID, map[string]interface{}{"status": "blocked"})
			accepted = append(accepted, snap.CardID)
			continue
		}

		switch snap.Source {
		case "register":
			balance := centsToBalance(snap.BalanceCents)
			_ = c.cardService.UpdateCard(snap.CardID, map[string]interface{}{
				"status":  "active",
				"balance": balance,
			})
			accepted = append(accepted, snap.CardID)
		case "recharge":
			balance := centsToBalance(snap.BalanceCents)
			if balance+0.005 < card.Balance {
				rejected = append(rejected, CardStateReject{CardID: snap.CardID, Reason: "balance mismatch"})
				continue
			}
			_ = c.cardService.UpdateCard(snap.CardID, map[string]interface{}{"balance": balance})
			accepted = append(accepted, snap.CardID)
		default:
			if !balanceMatches(card.Balance, snap.BalanceCents) {
				rejected = append(rejected, CardStateReject{CardID: snap.CardID, Reason: "balance mismatch"})
				continue
			}
			accepted = append(accepted, snap.CardID)
		}
	}

	utils.Success(ctx, gin.H{
		"accepted": accepted,
		"rejected": rejected,
	})
}

func normalizeCardStatus(status string) (string, bool) {
	switch status {
	case "", "active":
		return "active", true
	case "blocked", "lost":
		return status, true
	default:
		return "", false
	}
}

func validSnapshotStatus(status string) bool {
	switch status {
	case "idle", "in_trip", "blocked":
		return true
	default:
		return false
	}
}

func centsToBalance(cents uint64) float64 {
	return float64(cents) / 100.0
}

func balanceMatches(balance float64, cents uint64) bool {
	return int64(math.Round(balance*100.0)) == int64(cents)
}
