package dashboard

import (
	"sync"
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
)

// sampleWith builds a sample at a fixed time with the given status value.
func sampleWith(v float64) sample {
	return sample{At: time.Unix(0, 0), Value: v}
}

func TestHistoryBuffer_SnapshotEmpty(t *testing.T) {
	t.Parallel()

	h := newHistoryBuffer(10)
	got := h.snapshot()
	if got != nil {
		t.Errorf("empty buffer snapshot: want nil, got %v", got)
	}
}

func TestHistoryBuffer_PreservesChronologicalOrder(t *testing.T) {
	t.Parallel()

	h := newHistoryBuffer(4)
	for _, v := range []float64{1, 0.5, 0, 1} {
		h.record(sampleWith(v))
	}

	got := h.snapshot()
	values := samplesToValues(got)
	want := []float64{1, 0.5, 0, 1}

	if len(got) != len(want) {
		t.Fatalf("snapshot length: want %d, got %d", len(want), len(got))
	}

	for i := range want {
		if values[i] != want[i] {
			t.Errorf("snapshot[%d]: want %v, got %v", i, want[i], values[i])
		}
	}
}

func TestHistoryBuffer_WrapsAtCapacity(t *testing.T) {
	t.Parallel()

	h := newHistoryBuffer(3)
	for _, v := range []float64{1, 1, 1, 0.5, 0} {
		h.record(sampleWith(v))
	}

	got := h.snapshot()
	values := samplesToValues(got)
	want := []float64{1, 0.5, 0}

	if len(got) != len(want) {
		t.Fatalf("snapshot length: want %d, got %d (%v)", len(want), len(got), got)
	}

	for i := range want {
		if values[i] != want[i] {
			t.Errorf("snapshot[%d]: want %v, got %v", i, want[i], values[i])
		}
	}
}

func TestHistoryBuffer_MinCapacityOne(t *testing.T) {
	t.Parallel()

	buf := newHistoryBuffer(0)
	buf.record(sampleWith(1))

	if got := buf.snapshot(); len(got) != 1 || got[0].Value != 1 {
		t.Errorf("capacity-0 buffer: want [1], got %v", got)
	}
}

func TestHistoryBuffer_SnapshotReturnsCopy(t *testing.T) {
	t.Parallel()

	buf := newHistoryBuffer(2)
	buf.record(sampleWith(1))

	first := buf.snapshot()
	first[0].Value = 99

	if got := buf.snapshot(); got[0].Value != 1 {
		t.Errorf("snapshot not isolated from mutations: want 1, got %v", got[0].Value)
	}
}

func TestHistoryBuffer_ConcurrentRecordAndSnapshot(t *testing.T) {
	t.Parallel()

	buf := newHistoryBuffer(8)

	var wg sync.WaitGroup

	for i := range 16 {
		wg.Add(2)

		go func() {
			defer wg.Done()
			buf.record(sampleWith(float64(i % 2)))
		}()

		go func() {
			defer wg.Done()
			_ = buf.snapshot()
		}()
	}

	wg.Wait()

	if got := buf.snapshot(); len(got) != 8 {
		t.Errorf("final snapshot length: want 8, got %d", len(got))
	}
}

func TestStatusValue_MapsTrendScale(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status health.Status
		want   float64
	}{
		{health.StatusPass, 1},
		{health.StatusWarn, 0.5},
		{health.StatusFail, 0},
		{health.Status("bogus"), 0},
		{health.Status(""), 0},
	}

	for _, tt := range tests {
		if got := statusValue(tt.status); got != tt.want {
			t.Errorf("statusValue(%q): want %v, got %v", tt.status, tt.want, got)
		}
	}
}

func samplesToValues(samples []sample) []float64 {
	values := make([]float64, 0, len(samples))
	for _, s := range samples {
		values = append(values, s.Value)
	}

	return values
}

func TestPopulateHistory_LastUpdatedIsLastSampleTime(t *testing.T) {
	t.Parallel()

	observed := time.Date(2026, 9, 4, 12, 30, 15, 0, time.UTC)

	history := newHistoryBuffer(4)
	history.record(
		sample{
			At:     observed.Add(-2 * time.Second),
			Value:  trendPassValue,
			Status: string(health.StatusPass),
		},
	)
	history.record(sample{At: observed, Value: trendWarnValue, Status: string(health.StatusWarn)})

	vm := viewModel{LastUpdated: "would-be-render-time"}
	populateHistory(&vm, history, 0)

	want := observed.UTC().Format(updatedStampFormat)
	if vm.LastUpdated != want {
		t.Errorf(
			"LastUpdated: want %q (last sample observation time), got %q",
			want,
			vm.LastUpdated,
		)
	}
}

func TestPopulateHistory_EmptyBufferKeepsRenderTime(t *testing.T) {
	t.Parallel()

	history := newHistoryBuffer(4)

	vm := viewModel{LastUpdated: "render-time"}
	populateHistory(&vm, history, 0)

	if vm.LastUpdated != "render-time" {
		t.Errorf("empty history must not touch LastUpdated, got %q", vm.LastUpdated)
	}
}

func TestPopulateHistory_TimelineTotalDisclosesCap(t *testing.T) {
	t.Parallel()

	history := newHistoryBuffer(16)
	firstAt := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)

	// Eight alternating samples produce seven transitions — more than the
	// five the timeline keeps, so the note's pre-cap count must be set.
	statuses := []string{
		string(health.StatusPass), string(health.StatusWarn),
		string(health.StatusPass), string(health.StatusWarn),
		string(health.StatusPass), string(health.StatusWarn),
		string(health.StatusPass), string(health.StatusWarn),
	}

	for i, status := range statuses {
		value := float64(trendPassValue)
		if status == string(health.StatusWarn) {
			value = trendWarnValue
		}

		history.record(
			sample{At: firstAt.Add(time.Duration(i) * time.Second), Value: value, Status: status},
		)
	}

	vm := viewModel{}
	populateHistory(&vm, history, 0)

	if len(vm.Timeline) != maxTimelineEntries {
		t.Fatalf("timeline entries = %d, want capped at %d", len(vm.Timeline), maxTimelineEntries)
	}

	if vm.TimelineTotal != len(statuses)-1 {
		t.Errorf(
			"TimelineTotal = %d, want %d (pre-cap transition count)",
			vm.TimelineTotal,
			len(statuses)-1,
		)
	}
}

func TestPopulateHistory_TimelineTotalZeroWhenAllFit(t *testing.T) {
	t.Parallel()

	history := newHistoryBuffer(8)
	firstAt := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)

	for i := range 3 {
		status := string(health.StatusPass)
		value := float64(trendPassValue)
		if i%2 == 1 {
			status = string(health.StatusWarn)
			value = trendWarnValue
		}

		history.record(
			sample{At: firstAt.Add(time.Duration(i) * time.Second), Value: value, Status: status},
		)
	}

	vm := viewModel{}
	populateHistory(&vm, history, 0)

	if vm.TimelineTotal != 0 {
		t.Errorf("TimelineTotal = %d, want 0 when nothing was dropped", vm.TimelineTotal)
	}
}
