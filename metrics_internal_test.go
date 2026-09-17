package dashboard

import (
	"strings"
	"testing"
)

// The 2026-09-16 triage sized the buckets array from the bounds at the
// type level ([len(latencyBucketBounds)]atomic.Uint64); this test guards
// the pairing at the value level so a bounds edit without a matching
// histogram resize fails loudly instead of silently dropping or
// misreporting observations.
func TestLatencyHistogram_BucketCountMatchesBounds(t *testing.T) {
	t.Parallel()

	h := newLatencyHistogram()

	if got := len(h.buckets); got != len(latencyBucketBounds) {
		t.Fatalf(
			"histogram has %d buckets for %d latencyBucketBounds — resize the buckets array with the bounds",
			got,
			len(latencyBucketBounds),
		)
	}

	var b strings.Builder
	h.renderPrometheus(&b)

	if got := strings.Count(b.String(), "_bucket{le="); got != len(latencyBucketBounds)+1 {
		t.Errorf(
			"exposition emits %d bucket lines, want %d (one per bound + +Inf)",
			got,
			len(latencyBucketBounds)+1,
		)
	}
}
