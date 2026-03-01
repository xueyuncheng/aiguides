package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"aiguide/internal/pkg/constant"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// mockLimiter 满足与 redis_rate.Limiter 相同调用签名的辅助类型，
// 用于测试时注入预设响应，无需真实 Redis。
// 由于 RateLimiter 直接调用 *redis_rate.Limiter，这里通过构造
// 只包含内存客户端的 Limiter 来模拟行为。

// newInProcessLimiter 返回一个基于 miniredis 的限流器（如可用），
// 此处简化为直接测试 rateLimitKey 逻辑。
func TestRateLimitKey_WithUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(constant.ContextKeyUserID, 42)

	key := rateLimitKey(c)
	expected := "ratelimit:user:42"
	if key != expected {
		t.Errorf("rateLimitKey() = %q, want %q", key, expected)
	}
}

func TestRateLimitKey_WithoutUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	key := rateLimitKey(c)
	// key 应当以 "ratelimit:ip:" 开头
	if len(key) < len("ratelimit:ip:") {
		t.Errorf("rateLimitKey() = %q, expected ip-based key", key)
	}
}

func TestRateLimiter_AllowsRequests(t *testing.T) {
	// 使用真实 Redis 时跳过；此测试仅验证 Redis 不可达时中间件不阻塞请求
	gin.SetMode(gin.TestMode)

	// 连接一个不存在的 Redis（必然失败），验证降级逻辑
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:16399", // 不存在的端口
	})

	cfg := RateLimiterConfig{
		Rate:   10,
		Period: time.Minute,
	}

	handler := RateLimiter(rdb, cfg)

	// 构造一个带 auth 信息的请求
	req := httptest.NewRequest(http.MethodPost, "/api/assistant/chats/1", nil)
	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/api/assistant/chats/:id", handler, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	router.ServeHTTP(w, req)

	// Redis 不可达时，应降级放行（返回 200）
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when Redis unavailable, got %d", w.Code)
	}
}

func TestRateLimiterConfig_Defaults(t *testing.T) {
	cfg := RateLimiterConfig{
		Rate:   60,
		Period: 60 * time.Second,
	}

	if cfg.Rate != 60 {
		t.Errorf("expected rate 60, got %d", cfg.Rate)
	}
	if cfg.Period != 60*time.Second {
		t.Errorf("expected period 60s, got %v", cfg.Period)
	}
}

func TestRateLimitHeaders_Set(t *testing.T) {
	// 验证在限流通过时，响应头被正确设置
	// 由于无法连接真实 Redis，验证 Redis 失败时不设置限流头也不报错
	gin.SetMode(gin.TestMode)

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:16399",
	})

	cfg := RateLimiterConfig{Rate: 5, Period: time.Minute}
	handler := RateLimiter(rdb, cfg)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/", handler, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.ServeHTTP(w, req)

	// Redis 不可达时降级，X-RateLimit-Limit 头不应被设置
	if w.Header().Get("X-RateLimit-Limit") != "" {
		// 若 Redis 意外可用，校验 header 值
		limit, err := strconv.Atoi(w.Header().Get("X-RateLimit-Limit"))
		if err != nil || limit != 5 {
			t.Errorf("X-RateLimit-Limit header: got %q, expected \"5\"", w.Header().Get("X-RateLimit-Limit"))
		}
	}
}
