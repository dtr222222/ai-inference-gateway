package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimitMiddleWare 限流中间件
func RateLimitMiddleWare(rdb *redis.Client, limit, window int) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "rate_limit:" + c.ClientIP()
		// lua 脚本，
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

func RateLimitBucketMiddleWare(rdb *redis.Client, volume, rate, window int) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "rate_limit:" + c.ClientIP()

		luaScript := `
			local time_arr = redis.call('TIME')
			local now = tonumber(time_arr[1])
			local data = redis.call('HMGET', KEYS[1], 'tokens', 'last_time')
			local tokens = tonumber(data[1])
			local last_time = tonumber(data[2])
		
			if not tokens then
				tokens = tonumber(ARGV[1])
				last_time = now
			end
		
			local delta = math.max(now - last_time, 0)
			local generated = delta * tonumber(ARGV[2])

			local final_tokens = math.min(tokens + generated, tonumber(ARGV[1]))
			if final_tokens >= 1 then 
				redis.call('HSET', KEYS[1], 'tokens', final_tokens-1, 'last_time', now)
				redis.call('EXPIRE', KEYS[1], ARGV[3])
				return 1
			else
				return 0
			end
		`

		ctx, stop := context.WithTimeout(c.Request.Context(), 200*time.Millisecond)
		defer stop()
		cmd := rdb.Eval(ctx, luaScript, []string{key}, volume, rate, window)

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
