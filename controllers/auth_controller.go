// Package controllers 存放 HTTP 控制器，负责参数校验与响应封装。
package controllers

import (
	"TapTransit-backend/models"
	"TapTransit-backend/utils"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// AuthController 处理登录/登出相关接口。
type AuthController struct{}

func NewAuthController() *AuthController {
	return &AuthController{}
}

// loginRequest 登录请求体。
type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// loginUser 登录响应中的用户信息（脱敏字段）。
type loginUser struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

// loginResponse 登录响应体，包含临时 token。
type loginResponse struct {
	Token string    `json:"token"`
	User  loginUser `json:"user"`
}

// Login 简单登录（开发用，明文密码对比）。
func (a *AuthController) Login(ctx *gin.Context) {
	var req loginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(ctx, "缺少用户名或密码")
		return
	}

	var user models.User
	if err := utils.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		utils.Unauthorized(ctx, "用户名或密码错误")
		return
	}

	// 开发阶段使用明文密码比对，生产环境需替换为哈希校验。
	if user.Password != req.Password {
		utils.Unauthorized(ctx, "用户名或密码错误")
		return
	}

	// 生成临时 token（开发占位）
	resp := loginResponse{
		Token: fmt.Sprintf("dev_token_%d", time.Now().Unix()),
		User: loginUser{
			ID:       user.ID,
			Username: user.Username,
			Name:     user.RealName,
			Role:     user.Role,
		},
	}
	utils.Success(ctx, resp)
}

// Logout 登出（占位），前端可清理本地状态。
func (a *AuthController) Logout(ctx *gin.Context) {
	utils.Success(ctx, gin.H{"ok": true})
}
