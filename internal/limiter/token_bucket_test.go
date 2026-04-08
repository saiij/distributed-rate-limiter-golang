package limiter

import (
	"sync"
	"testing"
	"time"
)

func TestTokenBucketRequestLimit(t *testing.T) {
	var key string = "ip:1.111.111"
	tokenBucket := NewTokenBucket(10, 1)
	response, err := tokenBucket.Allow(t.Context(), key)
	if err != nil {
		t.Error(err.Error())
	}
	if !response.CanAccess {
		t.Error("cannot access")
	}

	if response.RequestRemain != tokenBucket.maxTokenCapacity-1 {
		t.Errorf("invalid value for request remain : %d", response.RequestRemain)
	}
}

func TestTokenBucketRequestOutOfLimit(t *testing.T) {
	var key string = "ip:1.111.111"
	tokenBucket := NewTokenBucket(10, 1)
	limit := tokenBucket.maxTokenCapacity + 1

	for i := 1; i <= limit; i++ {
		response, err := tokenBucket.Allow(t.Context(), key)
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

func TestTokenBucketRequestAfterLimitIsOut(t *testing.T) {
	var key string = "ip:1.111.111"
	fakeTime := time.Now()

	tokenBucket := &TokenBucket{
		keyStates:        make(map[string]TokenBucketRequest),
		maxTokenCapacity: 10,
		rechargeRate:     1,
		now:              func() time.Time { return fakeTime },
	}
	limit := tokenBucket.maxTokenCapacity + 1

	for i := 1; i <= limit; i++ {
		_, err := tokenBucket.Allow(t.Context(), key)
		if err != nil {
			t.Error(err.Error())
		}
	}
	// avanzar el tiempo
	fakeTime = fakeTime.Add(60 * time.Second)
	response, err := tokenBucket.Allow(t.Context(), key)
	if err != nil {
		t.Error(err.Error())
	}
	if !response.CanAccess {
		t.Error("cannot access")
	}

	if response.RequestRemain != tokenBucket.maxTokenCapacity-1 {
		t.Errorf("invalid value for request remain : %d", response.RequestRemain)
	}
}

func TestTokenBucketRaceConditions(t *testing.T) {
	var key string = "ip:1.111.111"
	tokenBucket := NewTokenBucket(10, 1)

	var wg sync.WaitGroup

	goroutinesLimit := 100
	for i := 0; i <= goroutinesLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := tokenBucket.Allow(t.Context(), key)
			if err != nil {
				t.Error(err.Error())
			}
		}()
	}

	wg.Wait()
}
