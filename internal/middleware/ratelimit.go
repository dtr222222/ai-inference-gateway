package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// 别忘了要加应有的注释

func RateLimitMiddleWare(rdb *redis.Client, limit, window int) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "rate_limit:" + c.ClientIP()
		luaScript := `
			local current = redis.call('INCR', KEYS[1])
			
			if tonumber(current) == 1 then
				redis.call('EXPIRE', KEYS[1], ARGV[2])
			end
			
			if tonumber(current) > tonumber(ARGV[1]) then
				return 0
			else 
				return 1
			end
		`

		ctx, stop := context.WithTimeout(c.Request.Context(), 200*time.Millisecond)
		defer stop()
		cmd := rdb.Eval(ctx, luaScript, []string{key}, limit, window)

		// 这里要先检查 cmd.Err()，因为 int64() 函数返回的 error，同样会检查 cmd.Err()
		// 会对错误的排查产生干扰
		// 不为空说明是 redis 挂了或者超时
		if cmd.Err() != nil {
			slog.Error("Rate limit failed, bypassing...", "error", cmd.Err())
			c.Next()
			return
		}

		res, err := cmd.Int64()
		// 转换失败
		if err != nil {
			slog.Error("transforming to int64 failed", "error", err)
			c.Next()
			return
		}

		// 请求超限
		if res == 0 {
			slog.Warn("rate limit exceeded", "ip", c.ClientIP())
			c.AbortWithStatusJSON(429, gin.H{"error": "Too Many Requests"})
			return
		}

		c.Next()

	}
}
