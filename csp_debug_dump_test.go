package dashboard_test

import (
	"strings"
	"testing"

)

func TestDebugDumpNonceHTML(t *testing.T) {
	s := setupDashboard(t)
	defer s.cleanup()

	w := doRequest(t, s.mux, "/health")
	body := w.Body.String()

	if i := strings.Index(body, `nonce=""`); i >= 0 {
		start := i - 400
		if start < 0 {
			start = 0
		}

		t.Logf("CONTEXT: %s", body[start:i+200])
	} else {
		t.Log("no empty nonce found")
	}
}
