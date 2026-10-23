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

		defer func() {
			// path := c.Request.URL.Path 这里不能这么写
			// c.Request.URL.Path 是客户端原本发来的字符串
			// 这里得用 c.FullPath(), 这个会去尝试匹配占位符
			// 会把 result/123 和 result/456 都映射成 result/:id
			// 避免标签基数爆炸
			path := c.FullPath()
			// 边缘处理：如果根本没查到，FullPath 会返回“”，
			// 因此得特殊处理一下
			if path == "" {
				path = "unmatched"
			}
			if path == "/metrics" {
				// 跳过自我监控
				// Skipping self-monitoring is correct, but the hardcoded string will
				// need updating for every new infra endpoint (/healthz, /readyz, ...).
				// Could also be skipped implicitly by filtering on c.FullPath().
				return
			}
			latency := time.Since(start)
			method := c.Request.Method
			status := strconv.Itoa(c.Writer.Status())

			// NOTE: duration has no "status" label (see metrics.go). You can't
			// compute p99 for 5xx separately -- the exact case you care most about.
			metrics.RecordRequest(method, path, status)
			// latency.Seconds() is correct: Prometheus histograms expect float64 seconds.
			// (Contrast with SlogMiddleware which logs latency.String() -- inconsistent.)
			metrics.ObserveDuration(method, path, latency.Seconds())
		}()

		c.Next()
	}
}
