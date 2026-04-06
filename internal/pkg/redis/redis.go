package redis

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/cenkalti/backoff/v4"
	goredis "github.com/redis/go-redis/v9"
)

// Client 是对 go-redis 客户端的轻量封装。
type Client struct {
	inner *goredis.Client
}

// NewFromGoRedis 使用现有 go-redis 客户端创建封装客户端。
func NewFromGoRedis(client *goredis.Client) *Client {
	return &Client{inner: client}
}

// Raw 返回底层 go-redis 客户端。
func (c *Client) Raw() *goredis.Client {
	if c == nil {
		return nil
	}
	return c.inner
}

// Addr 返回客户端连接地址。
func (c *Client) Addr() string {
	if c == nil || c.inner == nil {
		return ""
	}
	return c.inner.Options().Addr
}

// Close 关闭底层连接。
func (c *Client) Close() error {
	if c == nil || c.inner == nil {
		return nil
	}
	return c.inner.Close()
}

// Config Redis YAML 配置
type Config struct {
	// Addr Redis 地址，例如 "localhost:6379"
	Addr string `yaml:"addr" validate:"required"`
	// Password Redis 密码（可选）
	Password string `yaml:"password"`
}

// New 创建并校验 Redis 客户端连接。
func New(ctx context.Context, cfg Config) (*Client, error) {
	addr := strings.TrimSpace(cfg.Addr)
	if addr == "" {
		return nil, fmt.Errorf("redis.addr is required")
	}

	rdb := goredis.NewClient(&goredis.Options{
		Addr:     addr,
		Password: cfg.Password,
	})

	bo := backoff.WithContext(backoff.NewExponentialBackOff(), ctx)

	operation := func() error {
		if err := rdb.Ping(ctx).Err(); err != nil {
			slog.Error("rdb.Ping() error", "err", err)
			return fmt.Errorf("rdb.Ping() error, err = %w", err)
		}

		return nil
	}

	if err := backoff.Retry(operation, bo); err != nil {
		slog.Error("backoff.Retry() error", "err", err)

		rdb.Close()
		return nil, fmt.Errorf("backoff.Retry() error, err = %w", err)
	}

	return &Client{inner: rdb}, nil
}
