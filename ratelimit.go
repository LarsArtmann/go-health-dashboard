package dashboard

import (
	"encoding/json/v2"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// rateLimiter is a hand-rolled token bucket. The module keeps zero runtime
// dependencies, so golang.org/x/time/rate is deliberately not used.
type rateLimiter struct {
	mu     sync.Mutex
	tokens float64
	max    float64
	refill float64 // tokens per second
	last   time.Time
}

// newRateLimiter allows bursts up to maxRequests, refilling continuously at
// maxRequests per window.
func newRateLimiter(maxRequests int, window time.Duration) *rateLimiter {
	if maxRequests < 1 {
		maxRequests = 1
	}

	if window <= 0 {
		window = time.Second
	}

	return &rateLimiter{
		tokens: float64(maxRequests),
		max:    float64(maxRequests),
		refill: float64(maxRequests) / window.Seconds(),
		last:   time.Now(),
	}
}

// Allow consumes one token, reporting whether the request is admitted.
func (l *rateLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	l.tokens = min(l.max, l.tokens+now.Sub(l.last).Seconds()*l.refill)
	l.last = now

	if l.tokens < 1 {
		return false
	}

	l.tokens--

	return true
}

// retryAfter returns the whole seconds until the next token is available.
func (l *rateLimiter) retryAfter() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return int(math.Ceil((1 - l.tokens) / l.refill))
}

// applyRateLimit gates a dashboard-owned handler behind the token bucket
// when WithRateLimit is configured; otherwise it returns the handler
// unchanged. Requests beyond the budget receive 429 with a Retry-After
// hint so well-behaved clients back off.
func (d *Dashboard) applyRateLimit(next http.Handler) http.Handler {
	if d.limiter == nil {
		return next
	}

	limiter := d.limiter

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			w.Header().Set("Retry-After", strconv.Itoa(limiter.retryAfter()))
			if wantsJSON(r) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)

				body, _ := json.Marshal(
					rateLimitBody{
						Error:      "dashboard: rate limit exceeded",
						RetryAfter: limiter.retryAfter(),
					},
					json.Deterministic(true),
				)
				_, _ = w.Write(body)

				return
			}

			http.Error(w, "dashboard: rate limit exceeded", http.StatusTooManyRequests)

			return
		}

		next.ServeHTTP(w, r)
	})
}

// rateLimitBody is the JSON document served to JSON-negotiating clients on
// 429 (WithRateLimit). retry_after mirrors the Retry-After header in
// seconds.
type rateLimitBody struct {
	Error      string `json:"error"`
	RetryAfter int    `json:"retry_after"`
}
