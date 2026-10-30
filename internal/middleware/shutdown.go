package middleware

import (
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

func ShutdownMiddleware(isShuttingDown *atomic.Bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 最后的阶段 prometheus 仍可能访问 /metrics
		if isShuttingDown.Load() && c.Request.URL.Path != "/healthz" && c.Request.URL.Path != "/metrics" {
			c.AbortWithStatusJSON(503, gin.H{"error": "service shutting down"})
			return
		}
		c.Next()
	}
}
