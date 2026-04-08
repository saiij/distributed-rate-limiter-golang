package limiter

import (
	"context"
	"sync"
	"time"

	"saiij.distributed.rate.limiter/internal/ratelimiter"
)

var _ ratelimiter.RateLimiter = (*SlidingWindow)(nil)

type SlidingWindowRequest struct {
	CurrentCounter int
	PrevCounter    int
	Window         int
}

type SlidingWindow struct {
	windows             map[string]SlidingWindowRequest
	mu                  sync.RWMutex
	windowsSize         int64
	maxRequestPerWindow int
	now                 func() time.Time
}

func NewSlidingWindow(ws int64, maxReq int) *SlidingWindow {
	return &SlidingWindow{
		windows:             make(map[string]SlidingWindowRequest),
		windowsSize:         ws,
		maxRequestPerWindow: maxReq,
		now:                 time.Now,
	}
}

func (s *SlidingWindow) Allow(ctx context.Context, key string) (*ratelimiter.AllowResponse, error) {
	response := ratelimiter.NewAllowResponse()

	currentTime := s.now()
	currentWindow := currentTime.Unix() / s.windowsSize
	s.mu.Lock()
	defer s.mu.Unlock()
	window, exists := s.windows[key]
	if !exists {
		// create
		slidingWindowRequest := SlidingWindowRequest{
			CurrentCounter: 1,
			PrevCounter:    0,
			Window:         int(currentWindow),
		}
		s.windows[key] = slidingWindowRequest

	} else {
		if window.Window == int(currentWindow) {
			window.CurrentCounter++
			s.windows[key] = window
		} else {
			windowDiff := int(currentWindow) - window.Window
			if windowDiff == 1 {
				window.PrevCounter = window.CurrentCounter
			} else {
				window.PrevCounter = 0
			}
			window.Window = int(currentWindow)
			window.CurrentCounter = 1
			s.windows[key] = window
		}
	}
	window = s.windows[key]
	secElapsed := currentTime.Unix() - (currentWindow * s.windowsSize)
	perElapsed := float64(secElapsed) / float64(s.windowsSize)
	counter := int(float64(window.PrevCounter)*(1-perElapsed) + float64(window.CurrentCounter))

	if counter > s.maxRequestPerWindow {
		response.CanAccess = false
		response.RequestRemain = max(0, s.maxRequestPerWindow-counter)
	} else {
		response.CanAccess = true
		response.RequestRemain = max(0, s.maxRequestPerWindow-counter)
	}

	response.ResetRequestAt = currentTime.Add(time.Duration(s.windowsSize) * time.Second)
	diffTimer := response.ResetRequestAt.Sub(currentTime)
	response.RetryIn = diffTimer

	return response, nil
}
