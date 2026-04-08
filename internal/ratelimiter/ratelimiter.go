package ratelimiter

import (
	"context"
	"time"
)

type AllowResponse struct {
	CanAccess      bool
	RequestRemain  int
	ResetRequestAt time.Time
	RetryIn        time.Duration
}

func NewAllowResponse() *AllowResponse {
	return &AllowResponse{}
}

type RateLimiter interface {
	Allow(ctx context.Context, key string) (*AllowResponse, error)
}

type Store interface {
	FixedWindow(ctx context.Context, key string, windowSize int64, maxRequestPerWindow int) (*AllowResponse, error)
	SlidingWindow(ctx context.Context, key string, windowSize int64, maxRequestPerWindow int) (*AllowResponse, error)
	TokenBucket(ctx context.Context, key string, maxTokenCapacity int, rechargeRate int) (*AllowResponse, error)
}

type RateLimiterConfig struct {
	Type                Algorithm
	WindowSize          int64
	MaxRequestPerWindow int
	MaxTokenCapacity    int
	RechargeRate        int
	Now                 func() time.Time
}
