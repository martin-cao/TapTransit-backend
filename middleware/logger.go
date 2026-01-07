package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger 简单的日志中间件，格式化输出常用请求信息。
func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// 自定义格式：IP、时间、方法、路径、协议、状态码、耗时、UA、错误
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	})
}
