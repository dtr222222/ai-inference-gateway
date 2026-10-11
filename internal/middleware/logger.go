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
		latency := time.Since(start).String()
		method := c.Request.Method
		urlPath := c.Request.URL.Path
		status := c.Writer.Status()
		ip := c.ClientIP()

		slog.Info("access log", "latency", latency,
			"method", method, "url_path", urlPath,
			"status", status, "client_ip", ip)
	}
}
