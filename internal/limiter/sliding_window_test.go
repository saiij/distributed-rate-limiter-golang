package limiter

import (
	"sync"
	"testing"
	"time"
)

func TestSlidingWindowRequestLimit(t *testing.T) {
	var key string = "ip:1.111.111"

	slidingWindow := NewSlidingWindow(60, 3)
	response, err := slidingWindow.Allow(t.Context(), key)
	if err != nil {
		t.Error(err.Error())
	}
	if !response.CanAccess {
		t.Error("cannot access")
	}

	if response.RequestRemain != slidingWindow.maxRequestPerWindow-1 {
		t.Errorf("invalid value for request remain : %d", response.RequestRemain)
	}
}

func TestSlidingWindowRequestOutOfLimit(t *testing.T) {
	var key string = "ip:1.111.111"

	slidingWindow := NewSlidingWindow(60, 3)

	limit := slidingWindow.maxRequestPerWindow + 1

	for i := 1; i <= limit; i++ {
		response, err := slidingWindow.Allow(t.Context(), key)
		if err != nil {
			t.Error(err.Error())
		}

		if i == limit {
			if response.CanAccess {
				t.Errorf("invalid response, access is :%t", response.CanAccess)
			}

			if response.RequestRemain != 0 {
				t.Errorf("invalid response, request remain is : %d", response.RequestRemain)
			}
		}
	}
}

func TestSlidingWindowRequestAfterWindowLimitIsOut(t *testing.T) {
	var key string = "ip:1.111.111"
	fakeTime := time.Now()
	slidingWindow := &SlidingWindow{
		windows:             make(map[string]SlidingWindowRequest),
		windowsSize:         60,
		maxRequestPerWindow: 3,
		now:                 func() time.Time { return fakeTime },
	}

	limit := slidingWindow.maxRequestPerWindow + 1

	for i := 0; i <= limit; i++ {
		_, err := slidingWindow.Allow(t.Context(), key)
		if err != nil {
			t.Error(err.Error())
		}
	}
	// avanzar el tiempo
	fakeTime = fakeTime.Add(90 * time.Second)
	response, err := slidingWindow.Allow(t.Context(), key)
	if err != nil {
		t.Error(err.Error())
		t.Logf("CanAccess: %v, RequestRemain: %d", response.CanAccess, response.RequestRemain)
	}
	if !response.CanAccess {
		t.Error("cannot access")
		t.Logf("CanAccess: %v, RequestRemain: %d", response.CanAccess, response.RequestRemain)
	}

	if response.RequestRemain != slidingWindow.maxRequestPerWindow-1 {
		t.Errorf("invalid value for request remain : %d", response.RequestRemain)
	}
}

func TestSlidingWindowRaceConditions(t *testing.T) {
	var key string = "ip:1.111.111"
	slidingWindow := NewSlidingWindow(60, 3)

	var wg sync.WaitGroup

	goroutinesLimit := 100
	for i := 0; i <= goroutinesLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := slidingWindow.Allow(t.Context(), key)
			if err != nil {
				t.Error(err.Error())
			}
		}()
	}

	wg.Wait()
}
