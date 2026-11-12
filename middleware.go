package dashboard

import (
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// maxPerIPBuckets bounds the per-address map before a sweep: an attacker
// cycling source ports (or a NAT gate) must not grow memory without limit.
// On overflow, buckets idle for a full window are dropped — a bucket that
// has refilled fully is indistinguishable from a fresh one.
const maxPerIPBuckets = 4096

// perIPRateLimiter hands each remote address its own token bucket. The
// sweep is O(n) on the rare overflow path only — normal requests pay one
// mutex-protected map lookup.
type perIPRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateLimiter
	max     int
	window  time.Duration
}

func newPerIPRateLimiter(maxRequests int, window time.Duration) *perIPRateLimiter {
	if maxRequests < 1 {
		maxRequests = 1
	}

	if window <= 0 {
		window = time.Second
	}

	return &perIPRateLimiter{
		buckets: make(map[string]*rateLimiter),
		max:     maxRequests,
		window:  window,
	}
}

// clientKey reduces RemoteAddr to host:port's host part. Proxy headers are
// deliberately ignored (attacker-controlled).
func clientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

func (l *perIPRateLimiter) allow(r *http.Request) (bool, int) {
	key := clientKey(r)

	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.buckets) >= maxPerIPBuckets {
		l.sweepLocked()
	}

	bucket, ok := l.buckets[key]
	if !ok {
		bucket = newRateLimiter(l.max, l.window)
		l.buckets[key] = bucket
	}

	admitted := bucket.Allow()

	return admitted, bucket.retryAfter()
}

// sweepLocked drops buckets idle for a full window. Callers hold l.mu.
func (l *perIPRateLimiter) sweepLocked() {
	threshold := time.Now().Add(-l.window)

	for key, bucket := range l.buckets {
		bucket.mu.Lock()
		idle := bucket.last.Before(threshold)
		bucket.mu.Unlock()

		if idle {
			delete(l.buckets, key)
		}
	}
}

// applyPerIPRateLimit gates dashboard-owned handlers per remote address
// when WithPerIPRateLimit is configured; otherwise it returns the handler
// unchanged. The 429 shape matches the shared limiter: Retry-After header,
// JSON body for JSON-negotiating clients, and the same rejection metrics.
func (d *Dashboard) applyPerIPRateLimit(next http.Handler) http.Handler {
	if d.perIPLimiter == nil {
		return next
	}

	limiter := d.perIPLimiter

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admitted, retry := limiter.allow(r)
		if !admitted {
			w.Header().Set("Retry-After", strconv.Itoa(retry))
			d.ratelimit429Total.Add(1)
			d.ratelimitRetryAfter.observe(float64(retry))

			serveRateLimited(w, r, retry)

			return
		}

		next.ServeHTTP(w, r)
	})
}

// applyRequestLogging emits one structured line per dashboard-owned
// request when WithRequestLogging is configured; otherwise the handler is
// returned unchanged. The status recorder wraps writes so the log carries
// what the client actually received.
func (d *Dashboard) applyRequestLogging(next http.Handler) http.Handler {
	if d.cfg.RequestLogger == nil {
		return next
	}

	logger := d.cfg.RequestLogger

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		logger.InfoContext(r.Context(), "dashboard request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", recorder.status),
			slog.Duration("duration", time.Since(start)),
			slog.String("remote_addr", r.RemoteAddr),
		)
	})
}

// statusRecorder captures the response status for request logging. The
// default mirrors net/http: a handler that never calls WriteHeader
// implicitly answered 200. Writes pass through the embedded
// ResponseWriter unchanged.
type statusRecorder struct {
	http.ResponseWriter

	status int
	wrote  bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wrote {
		r.status = code
		r.wrote = true
	}

	r.ResponseWriter.WriteHeader(code)
}

// Flush forwards so SSE streaming (go-sse heartbeats and patches) keeps
// working through the recorder.
func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
