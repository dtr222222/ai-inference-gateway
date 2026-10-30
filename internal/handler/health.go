package handler

import (
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

func HealthHandler(isShuttingDown *atomic.Bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isShuttingDown.Load() {
			c.JSON(200, gin.H{"status": "OK"})
		} else {
			c.JSON(503, gin.H{"status": "Service Unavailable"})
		}
	}

}
