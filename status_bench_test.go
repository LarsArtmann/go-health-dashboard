package dashboard

import (
	"fmt"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
)

// benchMetadataChecks builds a severity-flat response of n checks with full
// go-health v0.2.0 metadata (state-entry time + execution duration) — the
// worst case for the per-tick stamping loop in buildViewModelAt.
func benchMetadataChecks(n int, withMetadata bool) health.Response {
	since := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

	checks := make(map[string]health.Check, n)
	for i := range n {
		check := health.Check{Status: health.StatusPass}
		if withMetadata {
			check.Since = since
			check.DurationNanos = int64(823 * time.Microsecond)
		}

		checks[fmt.Sprintf("svc-%02d.DatabaseService", i)] = check
	}

	return health.Response{Status: health.StatusPass, Checks: checks}
}

// BenchmarkBuildViewModelAt_Stamping isolates the per-tick stamping loop:
// derive SinceText/DurationText for every row, 60 checks with full
// metadata versus the same table with none. The delta is the cost the
// pusher pays per tick for the v0.2.0 metadata line.
func BenchmarkBuildViewModelAt_Stamping(b *testing.B) {
	now := time.Date(2026, 9, 17, 12, 17, 0, 0, time.UTC)

	for _, testCase := range []struct {
		name         string
		withMetadata bool
	}{
		{name: "no_metadata", withMetadata: false},
		{name: "with_metadata", withMetadata: true},
	} {
		resp := benchMetadataChecks(60, testCase.withMetadata)

		b.Run(testCase.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				vm := buildViewModelAt(resp, "bench", "/health/sse", GroupBySeverity, now)
				if len(vm.Groups) == 0 {
					b.Fatal("no groups built")
				}
			}
		})
	}
}
