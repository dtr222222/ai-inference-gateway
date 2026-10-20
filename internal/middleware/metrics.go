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

		// WARNING: no defer here. If a downstream handler panics, c.Next()
		// unwinds past this point and neither RecordRequest nor ObserveDuration
		// runs. Combined with the same issue in SlogMiddleware, every panic
		// (i.e. every 500) is invisible in BOTH metrics and access logs.
		// Fix: wrap everything below in `defer func(){ ... }()`.
		c.Next()

		// CONFIRMED BUG: using the raw URL path as a Prometheus label value.
		// As soon as any route has a path param (e.g. /results/:id) or query
		// (e.g. ?page=2), this creates unbounded label cardinality and can OOM
		// Prometheus. Use c.FullPath() (route template) instead; for unmatched
		// routes it returns "" -> map that to a constant like "unmatched".
		path := c.Request.URL.Path
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
	}
}
