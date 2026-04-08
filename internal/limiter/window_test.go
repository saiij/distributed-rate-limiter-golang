package limiter

import (
	"sync"
	"testing"
	"time"
)

// casos que hay que testear
// 1. request dentro de la ventana - si
// 2. request fuera de la ventana - si
// 3. request con limite agotado - no
// 4. request con limite no agotado - si
// 5. request en una ventana nueva despues de agotar el limite - si
// 6. concurrencia

// 1. Request dentro del rango permitido -> permitida

func TestRequestInLimit(t *testing.T) {
	var key string = "ip:1.111.111"
	fixedWindow := NewFixedWindow(60, 3)

	response, err := fixedWindow.Allow(t.Context(), key)
	if err != nil {
		t.Error(err.Error())
	}
	if !response.CanAccess {
		t.Error("cannot access")
	}

	if response.RequestRemain != fixedWindow.maxRequestPerWindow-1 {
		t.Errorf("invalid value for request remain : %d", response.RequestRemain)
	}
}

func TestRequestOutOfLimit(t *testing.T) {
	var key string = "ip:1.111.111"
	fixedWindow := NewFixedWindow(60, 3)

	limit := fixedWindow.maxRequestPerWindow + 1
	for i := 1; i <= limit; i++ {
		response, err := fixedWindow.Allow(t.Context(), key)
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

func TestRequestAfterWindowLimitIsOut(t *testing.T) {
	var key string = "ip:1.111.111"
	fakeTime := time.Now()
	fixedWindow := &FixedWindow{
		windows:             make(map[string]UserRequest),
		windowsSize:         60,
		maxRequestPerWindow: 3,
		now:                 func() time.Time { return fakeTime },
	}

	limit := fixedWindow.maxRequestPerWindow + 1
	for i := 1; i <= limit; i++ {
		_, err := fixedWindow.Allow(t.Context(), key)
		if err != nil {
			t.Error(err.Error())
		}
	}

	// avanzar el tiempo
	fakeTime = fakeTime.Add(60 * time.Second)
	response, err := fixedWindow.Allow(t.Context(), key)
	if err != nil {
		t.Error(err.Error())
	}
	if !response.CanAccess {
		t.Error("cannot access")
	}

	if response.RequestRemain != fixedWindow.maxRequestPerWindow-1 {
		t.Errorf("invalid value for request remain : %d", response.RequestRemain)
	}
}

func TestRaceConditions(t *testing.T) {
	var key string = "ip:1.111.111"
	fixedWindow := NewFixedWindow(60, 3)
	var wg sync.WaitGroup

	goroutinesLimit := 100

	for i := 0; i <= goroutinesLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := fixedWindow.Allow(t.Context(), key)
			if err != nil {
				t.Error(err.Error())
			}
		}()
	}

	wg.Wait()
}
