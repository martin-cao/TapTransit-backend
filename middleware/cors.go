// Package middleware 提供 Gin 中间件（日志、跨域等）。
package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORS 跨域中间件，优先允许白名单域名，其次放行开发环境。
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 允许的源（可根据部署环境调整）
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"http://localhost:8080",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:5173",
			"http://127.0.0.1:8080",
		}

		// 检查是否在允许列表中
		allowOrigin := ""
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				allowOrigin = origin
				break
			}
		}

		// 如果不在列表中，默认允许所有（开发环境）。
		// 生产环境应明确指定允许的域名。
		if allowOrigin == "" {
			allowOrigin = origin // 开发环境允许所有源
		}

		c.Writer.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		// 预检请求直接返回，避免进入业务路由。
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
