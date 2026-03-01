package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"aiguide/internal/pkg/constant"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

// RateLimiterConfig 令牌桶限流配置
type RateLimiterConfig struct {
	// Rate 每个周期允许的请求数（令牌桶容量）
	Rate int
	// Period 令牌补充周期
	Period time.Duration
}

// RateLimiter 基于 Redis 的令牌桶限流中间件
// 按用户 ID 进行限流，未登录请求按 IP 限流
func RateLimiter(rdb *redis.Client, cfg RateLimiterConfig) gin.HandlerFunc {
	limiter := redis_rate.NewLimiter(rdb)

	return func(c *gin.Context) {
		// 优先使用用户 ID 作为限流 key，否则回退到 IP 地址
		key := rateLimitKey(c)

		res, err := limiter.Allow(c.Request.Context(), key, redis_rate.Limit{
			Rate:   cfg.Rate,
			Burst:  cfg.Rate,
			Period: cfg.Period,
		})
		if err != nil {
			slog.Error("rate limiter Allow() error", "key", key, "err", err)
			// Redis 不可用时放行请求，避免影响正常服务
			c.Next()
			return
		}

		// 返回限流相关响应头
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.Rate))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", res.Remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(res.ResetAfter).Unix()))

		if res.Allowed == 0 {
			retryAfter := res.RetryAfter.Seconds()
			c.Header("Retry-After", fmt.Sprintf("%.0f", retryAfter))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "请求过于频繁，请稍后重试",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// rateLimitKey 生成限流 key：已认证用户用 user:<id>，否则用 ip:<addr>
func rateLimitKey(c *gin.Context) string {
	if userID, exists := c.Get(constant.ContextKeyUserID); exists {
		return fmt.Sprintf("ratelimit:user:%v", userID)
	}
	return fmt.Sprintf("ratelimit:ip:%s", c.ClientIP())
}
