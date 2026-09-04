package dashboard

import (
	"testing"
	"time"

	health "github.com/larsartmann/go-health"
)

// TestShouldBroadcast_TTLReAsserts verifies PushOnChangeTTL: unchanged
// state broadcasts every n-th tick, and a real change resets the counter.
func TestShouldBroadcast_TTLReAsserts(t *testing.T) {
	t.Parallel()

	p := &pusher{pushMode: PushOnChange, ttl: 3}
	resp := health.Response{Status: health.StatusPass}

	// First observation is always a change.
	if !p.shouldBroadcast(resp) {
		t.Fatal("first tick should broadcast (initial fingerprint)")
	}

	// Two silent ticks (threshold 3 not reached).
	if p.shouldBroadcast(resp) {
		t.Error("tick 2 should be silent (ttl=3)")
	}
	if p.shouldBroadcast(resp) {
		t.Error("tick 3 should be silent (ttl=3)")
	}

	// Third unchanged tick re-asserts.
	if !p.shouldBroadcast(resp) {
		t.Error("tick 4 should re-assert (ttl=3)")
	}

	// The counter restarts after a re-assertion.
	if p.shouldBroadcast(resp) {
		t.Error("tick 5 should be silent again")
	}

	// A real change resets the cycle and broadcasts immediately.
	changed := health.Response{Status: health.StatusFail}
	if !p.shouldBroadcast(changed) {
		t.Error("changed state should broadcast")
	}
	if p.shouldBroadcast(changed) {
		t.Error("first unchanged tick after a change should be silent")
	}
}

// TestShouldBroadcast_TTLDisabled pins the default: without the option,
// unchanged state never broadcasts.
func TestShouldBroadcast_TTLDisabled(t *testing.T) {
	t.Parallel()

	p := &pusher{pushMode: PushOnChange}
	resp := health.Response{Status: health.StatusPass}

	if !p.shouldBroadcast(resp) {
		t.Fatal("first tick should broadcast")
	}

	for i := 0; i < 10; i++ {
		if p.shouldBroadcast(resp) {
			t.Fatalf("tick %d should be silent without TTL", i+2)
		}
	}
}

// TestPopulateHistory_TimelineMaxAge verifies the age cap drops entries
// older than the configured duration while keeping recent ones.
func TestPopulateHistory_TimelineMaxAge(t *testing.T) {
	t.Parallel()

	now := time.Now()
	buffer := newHistoryBuffer(8)

	buffer.record(sample{At: now.Add(-3 * time.Hour), Value: 1, Status: "pass"})
	buffer.record(sample{At: now.Add(-2 * time.Hour), Value: 0, Status: "fail"}) // too old
	buffer.record(sample{At: now.Add(-30 * time.Second), Value: 0.5, Status: "warn"})
	buffer.record(sample{At: now.Add(-5 * time.Second), Value: 1, Status: "pass"})

	var vm viewModel
	populateHistory(&vm, buffer, time.Hour)

	if len(vm.Timeline) != 2 {
		t.Fatalf("want 2 recent timeline entries after 1h cap, got %d: %+v", len(vm.Timeline), vm.Timeline)
	}

	if vm.Timeline[0].Status != "warn" || vm.Timeline[1].Status != "pass" {
		t.Errorf("recent entries wrong: %+v", vm.Timeline)
	}

	// Without a cap, all three transitions survive.
	var full viewModel
	populateHistory(&full, buffer, 0)

	if len(full.Timeline) != 3 {
		t.Errorf("without cap want 3 entries, got %d", len(full.Timeline))
	}
}
