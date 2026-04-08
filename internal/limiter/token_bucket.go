package limiter

import (
	"context"
	"sync"
	"time"

	"saiij.distributed.rate.limiter/internal/ratelimiter"
)

var _ ratelimiter.RateLimiter = (*TokenBucket)(nil)

type TokenBucketRequest struct {
	totalTokens int
	lastRequest time.Time
}

type TokenBucket struct {
	keyStates        map[string]TokenBucketRequest
	maxTokenCapacity int
	rechargeRate     int
	mu               sync.RWMutex
	now              func() time.Time
}

func NewTokenBucket(mc int, rr int) *TokenBucket {
	return &TokenBucket{
		keyStates:        make(map[string]TokenBucketRequest),
		maxTokenCapacity: mc,
		rechargeRate:     rr,
		now:              time.Now,
	}
}

func (t *TokenBucket) Allow(ctx context.Context, key string) (*ratelimiter.AllowResponse, error) {
	response := ratelimiter.NewAllowResponse()

	currentTime := t.now()

	t.mu.Lock()
	defer t.mu.Unlock()

	state, exists := t.keyStates[key]
	if !exists {
		request := TokenBucketRequest{
			totalTokens: t.maxTokenCapacity,
			lastRequest: currentTime,
		}
		t.keyStates[key] = request
	}

	state = t.keyStates[key]

	timePass := int(currentTime.Sub(state.lastRequest).Seconds())
	newTokens := timePass * t.rechargeRate
	currentTokens := min(t.maxTokenCapacity, newTokens+state.totalTokens)

	if currentTokens >= 1 {
		state.totalTokens = currentTokens - 1
		state.lastRequest = currentTime
		t.keyStates[key] = state
		response.CanAccess = true
	} else {
		response.CanAccess = false
	}
	response.RequestRemain = min(t.maxTokenCapacity, state.totalTokens)

	return response, nil
}
