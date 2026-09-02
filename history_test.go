package dashboard

import (
	"sync"
	"testing"

	health "github.com/larsartmann/go-health"
)

func TestHistoryBuffer_SnapshotEmpty(t *testing.T) {
	t.Parallel()

	h := newHistoryBuffer(10)
	if got := h.snapshot(); got != nil {
		t.Errorf("empty buffer snapshot: want nil, got %v", got)
	}
}

func TestHistoryBuffer_PreservesChronologicalOrder(t *testing.T) {
	t.Parallel()

	h := newHistoryBuffer(4)
	for _, v := range []float64{1, 0.5, 0, 1} {
		h.record(v)
	}

	got := h.snapshot()
	want := []float64{1, 0.5, 0, 1}

	if len(got) != len(want) {
		t.Fatalf("snapshot length: want %d, got %d", len(want), len(got))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("snapshot[%d]: want %v, got %v", i, want[i], got[i])
		}
	}
}

func TestHistoryBuffer_WrapsAtCapacity(t *testing.T) {
	t.Parallel()

	h := newHistoryBuffer(3)
	for _, v := range []float64{1, 1, 1, 0.5, 0} {
		h.record(v)
	}

	got := h.snapshot()
	want := []float64{1, 0.5, 0}

	if len(got) != len(want) {
		t.Fatalf("snapshot length: want %d, got %d (%v)", len(want), len(got), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("snapshot[%d]: want %v, got %v", i, want[i], got[i])
		}
	}
}

func TestHistoryBuffer_MinCapacityOne(t *testing.T) {
	t.Parallel()

	h := newHistoryBuffer(0)
	h.record(1)

	if got := h.snapshot(); len(got) != 1 || got[0] != 1 {
		t.Errorf("capacity-0 buffer: want [1], got %v", got)
	}
}

func TestHistoryBuffer_SnapshotReturnsCopy(t *testing.T) {
	t.Parallel()

	h := newHistoryBuffer(2)
	h.record(1)

	first := h.snapshot()
	first[0] = 99

	if got := h.snapshot(); got[0] != 1 {
		t.Errorf("snapshot not isolated from mutations: want 1, got %v", got[0])
	}
}

func TestHistoryBuffer_ConcurrentRecordAndSnapshot(t *testing.T) {
	t.Parallel()

	h := newHistoryBuffer(8)

	var wg sync.WaitGroup

	for i := range 16 {
		wg.Add(2)

		go func() {
			defer wg.Done()
			h.record(float64(i % 2))
		}()

		go func() {
			defer wg.Done()
			_ = h.snapshot()
		}()
	}

	wg.Wait()

	if got := h.snapshot(); len(got) != 8 {
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
