package dashboard

import (
	"sync"

	"github.com/larsartmann/go-health"
)

// changeDetector remembers the last judged response — overall status plus
// the check fingerprint — and reports whether a new response differs from
// it. The SSE pusher (PushOnChange broadcasting) and the webhook notifier
// (change-only fires) share one transition rule: announce when the status
// or the per-check fingerprint differs from the previously judged one, and
// the first observation always counts as a change so receivers learn the
// initial state.
type changeDetector struct {
	mu              sync.Mutex
	seen            bool
	lastStatus      health.Status
	lastFingerprint string
}

// changed reports whether resp differs from the remembered state, in which
// case resp becomes the new remembered state. Calls serialize on the
// internal mutex; each response is judged against the previous judgment,
// so concurrent callers see one consistent change stream.
func (c *changeDetector) changed(resp health.Response) bool {
	fp := fingerprintChecks(resp.Checks)

	c.mu.Lock()
	defer c.mu.Unlock()

	changed := !c.seen || resp.Status != c.lastStatus || fp != c.lastFingerprint
	if changed {
		c.seen = true
		c.lastStatus = resp.Status
		c.lastFingerprint = fp
	}

	return changed
}
