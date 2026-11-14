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
	maxRequests, window = normalizeRateLimit(maxRequests, window)

	return &rateLimiter{
		tokens: float64(maxRequests),
		max:    float64(maxRequests),
		refill: float64(maxRequests) / window.Seconds(),
		last:   time.Now(),
	}
}

// normalizeRateLimit clamps configured rate-limit parameters to their safe
// floors: fewer than one request per window would admit nothing, and a
// non-positive window would never refill. Both rate-limit constructors
// pass configuration through here so the limiters share one floor.
func normalizeRateLimit(maxRequests int, window time.Duration) (int, time.Duration) {
	if maxRequests < 1 {
		maxRequests = 1
	}

	if window <= 0 {
		window = time.Second
	}

	return maxRequests, window
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
			// One retry value per rejection: the header, the JSON body, and
			// the observability record (M14) must agree.
			retry := limiter.retryAfter()
			w.Header().Set("Retry-After", strconv.Itoa(retry))
			d.ratelimit429Total.Add(1)
			d.ratelimitRetryAfter.observe(float64(retry))

			serveRateLimited(w, r, retry)

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

// serveRateLimited answers a rejection: JSON body for JSON-negotiating
// clients (the retry_after field mirrors the Retry-After header), plain
// text otherwise. Both limiter kinds (shared and per-IP) share the shape.
func serveRateLimited(w http.ResponseWriter, r *http.Request, retry int) {
	if wantsJSON(r) {
		body, err := json.Marshal(
			rateLimitBody{
				Error:      "dashboard: rate limit exceeded",
				RetryAfter: retry,
			},
			json.Deterministic(true),
		)
		if err != nil {
			http.Error(w, "dashboard: rate limit exceeded", http.StatusTooManyRequests)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		writeBody(w, body)

		return
	}

	http.Error(w, "dashboard: rate limit exceeded", http.StatusTooManyRequests)
}
