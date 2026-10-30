package middleware

import (
	"log/slog"
	"time"
	"uuid"

	"github.com/gin-gonic/gin"
)

func SlogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		reqID := uuid.New().String()
		c.Set("req_id", reqID)

		// 这里需要把 latency 变成字符串，方便 slog 打印时能够打印出 ms
		// 这里也要过滤 path，避免不必要的日志

		// 注意这个函数必须得放在 c.next 前面，防止 panic 崩溃
		defer func() {
			path := c.FullPath()
			if path == "" {
				path = "unmatched"
			}
			if path == "/metrics" {
				return
			}

			latency := time.Since(start).Milliseconds()
			method := c.Request.Method
			status := c.Writer.Status()
			ip := c.ClientIP()
			slog.Info("access log", "latency_ms", latency,
				"method", method, "url_path", path,
				"status", status, "client_ip", ip,
				"request_id", reqID,
			)
		}()

		c.Next()
	}
}
