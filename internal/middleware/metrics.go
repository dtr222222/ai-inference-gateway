package middleware

import (
	"ai-inference-gateway/internal/metrics"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		path := c.Request.URL.Path
		if path == "/metrics" {
			// 跳过自我监控
			return
		}
		latency := time.Since(start)
		method := c.Request.Method
		status := strconv.Itoa(c.Writer.Status())

		metrics.RecordRequest(method, path, status)
		metrics.ObserveDuration(method, path, latency.Seconds())
	}
}
