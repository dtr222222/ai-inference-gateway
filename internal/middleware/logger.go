package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func SlogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		// 这里需要把 latency 变成字符串，方便 slog 打印时能够打印出 ms
		// 这里也要过滤 path，避免不必要的日志
		// WARNING: latency is logged as a string (e.g. "1.234s"), which cannot be
		// range-queried in Loki/ELK. Prefer a numeric field: "latency_ms": 1234.
		// NOTE: this check runs AFTER c.Next(), so a panic in a downstream handler
		// skips the access log entirely (see below).

		defer func() {
			path := c.Request.URL.Path
			if path == "/metrics" {
				return
			}

			latency := time.Since(start).String()
			method := c.Request.Method
			status := c.Writer.Status()
			ip := c.ClientIP()
			slog.Info("access log", "latency", latency,
				"method", method, "url_path", path,
				"status", status, "client_ip", ip)
		}()
	}
}
