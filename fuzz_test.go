package dashboard

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	health "github.com/larsartmann/go-health"
)

// fuzzRequest builds a GET /health request carrying the given Accept header.
func fuzzRequest(accept string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.Header.Set("Accept", accept)

	return r
}

// FuzzWantsJSON exercises the Accept-header parser with arbitrary input.
// Invariants: the parser never panics, is deterministic, and treats an
// absent or empty Accept header as a request for HTML.
func FuzzWantsJSON(f *testing.F) {
	for _, seed := range []string{
		"",
		"application/json",
		"text/html",
		"*/*",
		"application/*",
		"text/*",
		"application/json;q=0.9, text/html;q=0.8",
		"text/html;q=0.8, application/json;q=0.9",
		"application/json;q=1.0, */*;q=0.1",
		"APPLICATION/JSON",
		"application/json; q=abc",
		"application/json; q=",
		"text/html;q=0.8;level=1",
		"*",
		";;;",
		",,,",
		"application/json;q=2.0",
		"application/json;q=-1",
		"garbage",
		"application/json garbage text/html",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, accept string) {
		r := fuzzRequest(accept)

		first := wantsJSON(r)
		if second := wantsJSON(r); first != second {
			t.Fatalf("wantsJSON not deterministic for Accept %q: %v then %v", accept, first, second)
		}

		if accept == "" && first {
			t.Fatal("empty Accept header must not want JSON")
		}
	})
}

// FuzzHealthResponseSerialization round-trips arbitrary health responses
// through the same JSON codec the /health endpoint uses. Invariants:
// marshalling never fails, the payload decodes back losslessly, and
// re-encoding is byte-identical (idempotent).
func FuzzHealthResponseSerialization(
	f *testing.F,
) {
	type seedCase struct {
		status, version, uptime          string
		shuttingDown                     bool
		latency                          int64
		checkName, checkStatus, checkErr string
	}

	for _, s := range []seedCase{
		{status: "pass", version: "1.0.0", uptime: "1h2m", latency: 12, checkName: "database", checkStatus: "pass"},
		{status: "fail", checkName: "database", checkStatus: "fail", checkErr: "connection refused"},
		{status: "warn", checkName: "cache", checkStatus: "warn", checkErr: "slow"},
		{status: "pass", shuttingDown: true},
		{status: "pass", checkName: "", checkStatus: "pass"},
		{version: "emoji-🚀", checkErr: " ERR\n \"quoted\" \\ path"},
		{latency: -5},
	} {
		f.Add(s.status, s.version, s.uptime, s.shuttingDown, s.latency, s.checkName, s.checkStatus, s.checkErr)
	}

	//nolint:lll // long parameter list is inherent to fuzz corpus tuples
	f.Fuzz(func(
		t *testing.T,
		status, version, uptime string,
		shuttingDown bool,
		latency int64,
		checkName, checkStatus, checkErr string,
	) {
		// Replace invalid UTF-8, which JSON transcoding legitimately rewrites
		// to U+FFFD, so strict field equality stays a meaningful invariant.
		status = strings.ToValidUTF8(status, "\uFFFD")
		version = strings.ToValidUTF8(version, "\uFFFD")
		uptime = strings.ToValidUTF8(uptime, "\uFFFD")
		checkName = strings.ToValidUTF8(checkName, "\uFFFD")
		checkStatus = strings.ToValidUTF8(checkStatus, "\uFFFD")
		checkErr = strings.ToValidUTF8(checkErr, "\uFFFD")

		resp := health.Response{
			Status:         health.Status(status),
			Version:        version,
			Uptime:         uptime,
			ShuttingDown:   shuttingDown,
			TotalLatencyMs: latency,
			Checks: map[string]health.Check{
				checkName: {Status: health.Status(checkStatus), Error: checkErr},
			},
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("Marshal failed for %+v: %v", resp, err)
		}

		var decoded health.Response
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Unmarshal failed for payload %q: %v", data, err)
		}

		if decoded.Status != resp.Status {
			t.Errorf("status: want %q, got %q", resp.Status, decoded.Status)
		}

		if decoded.Version != resp.Version {
			t.Errorf("version: want %q, got %q", resp.Version, decoded.Version)
		}

		if decoded.Uptime != resp.Uptime {
			t.Errorf("uptime: want %q, got %q", resp.Uptime, decoded.Uptime)
		}

		if decoded.ShuttingDown != resp.ShuttingDown {
			t.Errorf("shuttingDown: want %v, got %v", resp.ShuttingDown, decoded.ShuttingDown)
		}

		if decoded.TotalLatencyMs != resp.TotalLatencyMs {
			t.Errorf("latency: want %d, got %d", resp.TotalLatencyMs, decoded.TotalLatencyMs)
		}

		if len(decoded.Checks) != 1 {
			t.Fatalf("checks: want 1 entry, got %d", len(decoded.Checks))
		}

		gotCheck, ok := decoded.Checks[checkName]
		if !ok {
			t.Fatalf("checks: key %q missing from decoded payload", checkName)
		}

		if gotCheck.Status != health.Status(checkStatus) {
			t.Errorf("check status: want %q, got %q", checkStatus, gotCheck.Status)
		}

		if gotCheck.Error != checkErr {
			t.Errorf("check error: want %q, got %q", checkErr, gotCheck.Error)
		}

		reencoded, err := json.Marshal(decoded)
		if err != nil {
			t.Fatalf("re-Marshal failed: %v", err)
		}

		if string(reencoded) != string(data) {
			t.Errorf("encoding not idempotent:\nfirst:  %s\nsecond: %s", data, reencoded)
		}
	})
}
