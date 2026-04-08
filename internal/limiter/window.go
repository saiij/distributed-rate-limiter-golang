package limiter

import (
	"context"
	"sync"
	"time"

	"saiij.distributed.rate.limiter/internal/ratelimiter"
)

var _ ratelimiter.RateLimiter = (*FixedWindow)(nil)

type UserRequest struct {
	RequestCounter int
	Window         int
}

type FixedWindow struct { // no se si esta bien este nombre
	windows             map[string]UserRequest
	mu                  sync.RWMutex
	windowsSize         int64
	maxRequestPerWindow int
	now                 func() time.Time
}

func NewFixedWindow(windowSize int64, maxRequestPerWindow int) *FixedWindow {
	return &FixedWindow{
		windows:             make(map[string]UserRequest),
		windowsSize:         windowSize,
		maxRequestPerWindow: maxRequestPerWindow,
		now:                 time.Now,
	}
}

func (f *FixedWindow) Allow(ctx context.Context, key string) (*ratelimiter.AllowResponse, error) {
	response := ratelimiter.NewAllowResponse()

	currentTime := f.now()
	currentTimeWindow := currentTime.Unix() / f.windowsSize

	f.mu.Lock()
	defer f.mu.Unlock()
	l, exists := f.windows[key]
	// no existe , crear registro
	if !exists {

		request := UserRequest{
			RequestCounter: 1,
			Window:         int(currentTimeWindow),
		}
		f.windows[key] = request

		response.CanAccess = true
		response.RequestRemain = max(0, f.maxRequestPerWindow-request.RequestCounter)
	} else {
		// existe validar hora
		if l.Window == int(currentTimeWindow) {
			l.RequestCounter++
			f.windows[key] = l
		} else {
			l.Window = int(currentTimeWindow)
			l.RequestCounter = 1
			f.windows[key] = l
		}

		if l.RequestCounter > f.maxRequestPerWindow {
			response.CanAccess = false
			response.RequestRemain = max(0, f.maxRequestPerWindow-l.RequestCounter)
		} else {
			response.CanAccess = true
			response.RequestRemain = max(0, f.maxRequestPerWindow-l.RequestCounter)
		}

	}

	response.ResetRequestAt = currentTime.Add(time.Duration(f.windowsSize) * time.Second)
	diffTimer := response.ResetRequestAt.Sub(currentTime)
	response.RetryIn = diffTimer

	return response, nil
}
