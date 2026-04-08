package ratelimiter

import (
	"context"
	"fmt"
)

var _ RateLimiter = (*RedisRateLimiter)(nil)

type RedisRateLimiter struct {
	store Store
	cfg   *RateLimiterConfig
}

func NewRedisRateLimiter(store Store, cfg *RateLimiterConfig) *RedisRateLimiter {
	return &RedisRateLimiter{
		store: store,
		cfg:   cfg,
	}
}

func (l *RedisRateLimiter) Allow(ctx context.Context, key string) (*AllowResponse, error) {
	switch l.cfg.Type {
	case FixedWindow:
		response, err := l.store.FixedWindow(ctx, key, l.cfg.WindowSize, l.cfg.MaxRequestPerWindow)
		if err != nil {
			return nil, err
		}
		return response, nil
	case SlidingWindow:
		response, err := l.store.SlidingWindow(ctx, key, l.cfg.WindowSize, l.cfg.MaxRequestPerWindow)
		if err != nil {
			return nil, err
		}
		return response, nil
	case TokenBucket:
		response, err := l.store.TokenBucket(ctx, key, l.cfg.MaxTokenCapacity, l.cfg.RechargeRate)
		if err != nil {
			return nil, err
		}
		return response, nil
	default:
		return nil, fmt.Errorf("invalid operation")
	}
}
