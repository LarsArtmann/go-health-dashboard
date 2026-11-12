package dashboard

import (
	"strings"
	"testing"
	"time"
)

// The 2026-09-16 triage sized the buckets array from the bounds at the
// type level ([len(latencyBucketBounds)]atomic.Uint64); this test guards
// the pairing at the value level so a bounds edit without a matching
// histogram resize fails loudly instead of silently dropping or
// misreporting observations.
func TestLatencyHistogram_BucketCountMatchesBounds(t *testing.T) {
	t.Parallel()

	hist := newLatencyHistogram()

	if got := len(hist.buckets); got != len(latencyBucketBounds) {
		t.Fatalf(
			"histogram has %d buckets for %d latencyBucketBounds — resize the buckets array with the bounds",
			got,
			len(latencyBucketBounds),
		)
	}

	var b strings.Builder
	hist.renderPrometheus(&b)

	if got := strings.Count(b.String(), "_bucket{le="); got != len(latencyBucketBounds)+1 {
		t.Errorf(
			"exposition emits %d bucket lines, want %d (one per bound + +Inf)",
			got,
			len(latencyBucketBounds)+1,
		)
	}
}

// TestLatencyHistogram_ObservePlacesBuckets pins the cumulative semantics
// the exposition relies on: an observation increments every bucket whose
// bound it fits under, and nothing else.
func TestLatencyHistogram_ObservePlacesBuckets(t *testing.T) {
	t.Parallel()

	hist := newLatencyHistogram()
	hist.observe(float64(50*time.Millisecond) / float64(time.Second))

	for i, bound := range latencyBucketBounds {
		want := uint64(0)
		if 0.05 <= bound {
			want = 1
		}

		if got := hist.buckets[i].Load(); got != want {
			t.Errorf("bucket[%d] (le=%g): want %d, got %d", i, bound, want, got)
		}
	}

	if got := hist.count.Load(); got != 1 {
		t.Errorf("count: want 1, got %d", got)
	}
}

// TestJitterLifetime pins the M15 spread: values stay within ±10% of the
// cap, zero stays zero, and the spread actually varies (an un-jittered
// cap reconnects herds).
func TestJitterLifetime(t *testing.T) {
	t.Parallel()

	const base = time.Minute

	seen := make(map[time.Duration]bool)

	for range 200 {
		jittered := jitterLifetime(base)

		low := base - base/10
		high := base + base/10

		if jittered < low || jittered > high {
			t.Fatalf("jitterLifetime(%s) = %s, want within [%s, %s]", base, jittered, low, high)
		}

		seen[jittered] = true
	}

	if len(seen) < 10 {
		t.Errorf(
			"jitter produced only %d distinct values in 200 draws; the spread is too narrow",
			len(seen),
		)
	}

	if got := jitterLifetime(0); got != 0 {
		t.Errorf("jitterLifetime(0) = %s, want 0", got)
	}
}
