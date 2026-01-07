package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构，保持前端解析一致性。
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// Success 成功响应，仅返回数据。
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// SuccessWithMessage 成功响应（带提示信息）。
func SuccessWithMessage(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// Error 错误响应，允许自定义 HTTP 状态码。
func Error(c *gin.Context, code int, message string) {
	c.JSON(code, Response{
		Success: false,
		Message: message,
	})
}

// BadRequest 400 错误响应。
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, message)
}

// Unauthorized 401 错误响应。
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message)
}

// NotFound 404 错误响应。
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, message)
}

// InternalServerError 500 错误响应。
func InternalServerError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, message)
}
