package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 兜底函数
		defer func() {
			if r := recover(); r != nil {
				// 服务端输出
				slog.Error("server error",
					slog.Any("panic", r),
					"stack_info", string(debug.Stack()),
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"client_ip", c.ClientIP(),
				)
				// 客户端输出
				c.AbortWithStatusJSON(http.StatusInternalServerError,
					gin.H{"error": "internal server error"})
			}
		}()

		// 中间件不能没有 next 函数，不然 gin 框架无法继续传递下去
		c.Next()
	}
}
