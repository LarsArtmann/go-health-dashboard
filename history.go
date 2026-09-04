package dashboard

import (
	"sync"
	"time"

	health "github.com/larsartmann/go-health"
)

// sample is one recorded health observation: the numeric value the
// sparkline plots plus the raw status and the observation time. Timestamps
// power the status timeline, the trend JSON endpoint, and the CSV export.
type sample struct {
	At     time.Time
	Value  float64
	Status string
}

// historyBuffer is a fixed-capacity ring buffer of status samples in
// chronological order. The pusher goroutine records on every tick; the SSE
// handler snapshots from other goroutines when rendering initial state, so
// all access is mutex-guarded.
type historyBuffer struct {
	mu      sync.Mutex
	samples []sample
	next    int
	full    bool
}

func newHistoryBuffer(capacity int) *historyBuffer {
	if capacity < 1 {
		capacity = 1
	}

	return &historyBuffer{samples: make([]sample, capacity)}
}

// record appends a sample, overwriting the oldest once at capacity.
func (h *historyBuffer) record(s sample) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.samples[h.next] = s
	h.next = (h.next + 1) % len(h.samples)

	if h.next == 0 {
		h.full = true
	}
}

// snapshot returns the recorded samples oldest-first, or nil when empty.
func (h *historyBuffer) snapshot() []sample {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.full && h.next == 0 {
		return nil
	}

	out := make([]sample, 0, len(h.samples))
	if h.full {
		out = append(out, h.samples[h.next:]...)
	}

	return append(out, h.samples[:h.next]...)
}

// latest returns the most recent sample, or false when nothing is recorded.
func (h *historyBuffer) latest() (sample, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.full && h.next == 0 {
		return sample{}, false
	}

	idx := h.next - 1
	if idx < 0 {
		idx = len(h.samples) - 1
	}

	return h.samples[idx], true
}

// statusTransition is one flip in the recorded status history.
type statusTransition struct {
	At   time.Time
	From string
	To   string
}

// transitions derives the status changes from the recorded samples,
// oldest-first. The first sample has no predecessor and never produces a
// transition.
func (h *historyBuffer) transitions() []statusTransition {
	samples := h.snapshot()

	var out []statusTransition

	for i := 1; i < len(samples); i++ {
		if samples[i].Status != samples[i-1].Status {
			out = append(out, statusTransition{
				At:   samples[i].At,
				From: samples[i-1].Status,
				To:   samples[i].Status,
			})
		}
	}

	return out
}

// populateHistory fills the sparkline values and the recent status-change
// timeline from the trend history. Shared by the initial HTML render and
// the SSE patches so both always agree. The Updated stamp is set to the
// last sample's observation time — the moment the health state was actually
// seen — rather than the render time, which can lag by up to one push
// interval (and is arbitrarily stale for a freshly connected browser).
const maxTimelineEntries = 5

func populateHistory(vm *viewModel, buffer *historyBuffer) {
	samples := buffer.snapshot()

	values := make([]float64, 0, len(samples))
	for _, s := range samples {
		values = append(values, s.Value)
	}

	vm.History = values

	if len(samples) > 0 {
		vm.LastUpdated = samples[len(samples)-1].At.UTC().Format(updatedStampFormat)
	}

	transitions := buffer.transitions()
	if len(transitions) > maxTimelineEntries {
		transitions = transitions[len(transitions)-maxTimelineEntries:]
	}

	for _, tr := range transitions {
		vm.Timeline = append(vm.Timeline, TimelineEntry{
			At:       tr.At.Format("15:04:05"),
			Status:   tr.To,
			Degraded: tr.To != string(health.StatusPass),
		})
	}
}
