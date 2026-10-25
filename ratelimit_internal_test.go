package dashboard

import (
	"sync/atomic"
	"testing"
	"time"
)

// TestRateLimiter_RefillAndCeiling pins the refill half of the token
// bucket deterministically (the fuzz target pins the burst half): backdate
// the last refill stamp by whole windows and the bucket must return
// exactly the window's worth of tokens, capped at max — never above, and
// never a partial grant below one token.
func TestRateLimiter_RefillAndCeiling(t *testing.T) {
	t.Parallel()

	limiter := newRateLimiter(3, 2*time.Second)

	// Drain the burst.
	for i := 0; i < 3; i++ {
		if !limiter.Allow() {
			t.Fatalf("initial burst request %d denied", i+1)
		}
	}

	if limiter.Allow() {
		t.Fatal("empty bucket admitted a request")
	}

	// Backdate by exactly one window: three tokens return (capped at max=3).
	limiter.mu.Lock()
	limiter.last = limiter.last.Add(-2 * time.Second)
	limiter.mu.Unlock()

	for i := 0; i < 3; i++ {
		if !limiter.Allow() {
			t.Fatalf("refilled request %d denied", i+1)
		}
	}

	if limiter.Allow() {
		t.Fatal("bucket exceeded its ceiling after refill")
	}

	// Backdate by ten windows: still exactly max tokens — refill is capped.
	limiter.mu.Lock()
	limiter.last = limiter.last.Add(-20 * time.Second)
	limiter.mu.Unlock()

	granted := 0
	for limiter.Allow() && granted < 10 {
		granted++
	}

	if granted != 3 {
		t.Fatalf("long idle refill granted %d requests, want exactly the max of 3", granted)
	}
}

// TestRateLimiter_ClampsInvalidConfig pins the constructor's guards: a
// non-positive request count or window falls back to sane values instead
// of a divide-by-zero refill rate or a zero-burst lockout.
func TestRateLimiter_ClampsInvalidConfig(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		maxRequests int
		window      time.Duration
	}{
		"zero requests":    {maxRequests: 0, window: time.Second},
		"negative requests": {
			maxRequests: -5,
			window:      time.Second,
		},
		"zero window":      {maxRequests: 5, window: 0},
		"negative window":  {maxRequests: 5, window: -time.Second},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			limiter := newRateLimiter(tc.maxRequests, tc.window)

			if limiter.max < 1 || limiter.refill <= 0 {
				t.Fatalf(
					"clamped config invalid: max=%v refill=%v",
					limiter.max,
					limiter.refill,
				)
			}

			if !limiter.Allow() {
				t.Fatal("clamped limiter denied its first request")
			}
		})
	}
}

// TestRateLimiter_ConcurrentAllowIsRaceSafe drives Allow from parallel
// goroutines: the mutex must keep token accounting consistent (no more
// admissions than burst + accrued refill could ever justify).
func TestRateLimiter_ConcurrentAllowIsRaceSafe(t *testing.T) {
	t.Parallel()

	const goroutines = 16

	limiter := newRateLimiter(8, time.Hour) // refill negligible during the test

	var admitted atomic.Int64

	start := make(chan struct{})

	done := make(chan struct{}, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			<-start

			if limiter.Allow() {
				admitted.Add(1)
			}

			done <- struct{}{}
		}()
	}

	close(start)

	for i := 0; i < goroutines; i++ {
		<-done
	}

	if got := admitted.Load(); got > 8 {
		t.Fatalf("admitted %d of %d concurrent requests, burst is 8", got, goroutines)
	}
}
