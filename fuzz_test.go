package dashboard

import (
	"bytes"
	"encoding/json/v2"
	"maps"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

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
		f.Add(
			s.status,
			s.version,
			s.uptime,
			s.shuttingDown,
			s.latency,
			s.checkName,
			s.checkStatus,
			s.checkErr,
		)
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
		// to U+FFFD, so strict equality stays a meaningful invariant.
		resp := health.Response{
			Status:         health.Status(strings.ToValidUTF8(status, "\uFFFD")),
			Version:        strings.ToValidUTF8(version, "\uFFFD"),
			Uptime:         strings.ToValidUTF8(uptime, "\uFFFD"),
			ShuttingDown:   shuttingDown,
			TotalLatencyMs: latency,
			Checks: map[string]health.Check{
				strings.ToValidUTF8(checkName, "\uFFFD"): {
					Status: health.Status(strings.ToValidUTF8(checkStatus, "\uFFFD")),
					Error:  strings.ToValidUTF8(checkErr, "\uFFFD"),
				},
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

		if !reflect.DeepEqual(decoded, resp) {
			t.Fatalf("round-trip lost data:\nwant %+v\ngot  %+v\npayload %s", resp, decoded, data)
		}

		reencoded, err := json.Marshal(decoded)
		if err != nil {
			t.Fatalf("re-Marshal failed: %v", err)
		}

		if !bytes.Equal(reencoded, data) {
			t.Errorf("encoding not idempotent:\nfirst:  %s\nsecond: %s", data, reencoded)
		}
	})
}

// unescapeLabelValue reverses escapeLabelValue in a single left-to-right
// pass, mirroring how a Prometheus exposition parser decodes escapes.
func unescapeLabelValue(v string) string {
	return strings.NewReplacer(`\\`, `\`, `\"`, `"`, `\n`, "\n").Replace(v)
}

// FuzzEscapeLabelValue exercises the Prometheus label-value escaper with
// arbitrary input. Invariants: escaping never panics, is deterministic,
// never emits a raw newline, and round-trips — unescaping the output
// yields the original input, so scrape parsers see lossless values.
func FuzzEscapeLabelValue(f *testing.F) {
	for _, seed := range []string{
		"",
		"plain",
		`\`,
		`"`,
		"\n",
		`\n`,
		`\"`,
		`\\`,
		`\"`,
		"line1\nline2",
		`back\slash"quote`,
		`\"`,
		"🚀 \"quoted\" \\ path\nnext",
		"\r\t\x00",
		strings.Repeat(`\"\n`, 50),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		escaped := escapeLabelValue(value)

		if again := escapeLabelValue(value); escaped != again {
			t.Fatalf("escapeLabelValue not deterministic for %q: %q then %q", value, escaped, again)
		}

		if strings.Contains(escaped, "\n") {
			t.Errorf("escaped value still contains a raw newline: %q -> %q", value, escaped)
		}

		if restored := unescapeLabelValue(escaped); restored != value {
			t.Errorf("round-trip lost data: %q -> %q -> %q", value, escaped, restored)
		}
	})
}

// FuzzFingerprintChecks exercises the change-detection fingerprint with
// arbitrary field values. Invariants: never panics, is deterministic,
// distinguishes any single-field mutation, and never aliases a name
// containing delimiter characters with a different field split.
func FuzzFingerprintChecks(
	f *testing.F,
) {
	for _, s := range []struct{ n1, s1, e1, n2, s2, e2 string }{
		{"db", "pass", "", "cache", "warn", "slow"},
		{"a", "b:c", "", "a:b", "c", ""},
		{"", "", "", "", "", ""},
		{"svc:1", "pass", "e:1;x", "svc:1", "fail", "e:1;x"},
		{"", "pass", "", "pass", "", ""},
		{"🚀", "pass", "\n", "🚀", "pass", "\n"},
		{strings.Repeat("k", 300), "pass", "", "k", "pass", ""},
	} {
		f.Add(s.n1, s.s1, s.e1, s.n2, s.s2, s.e2)
	}

	//nolint:lll // long parameter list is inherent to fuzz corpus tuples
	f.Fuzz(func(t *testing.T, n1, s1, e1, n2, s2, e2 string) {
		nameA, statusA, errA := validUTF8(n1), validUTF8(s1), validUTF8(e1)
		nameB, statusB, errB := validUTF8(n2), validUTF8(s2), validUTF8(e2)

		checks := map[string]health.Check{
			nameA: {Status: health.Status(statusA), Error: errA},
			nameB: {Status: health.Status(statusB), Error: errB},
		}

		fp := fingerprintChecks(checks)

		if again := fingerprintChecks(checks); fp != again {
			t.Fatalf("fingerprintChecks not deterministic:\n  fp1=%q\n  fp2=%q", fp, again)
		}

		mutatedStatus := maps.Clone(checks)

		mutated := mutatedStatus[nameA]
		mutated.Status = health.Status(string(mutated.Status) + "x")
		mutatedStatus[nameA] = mutated

		if fingerprintChecks(mutatedStatus) == fp {
			t.Errorf("fingerprint unchanged after status mutation of %q", nameA)
		}

		mutatedError := maps.Clone(checks)

		mutated = mutatedError[nameA]
		mutated.Error += "x"
		mutatedError[nameA] = mutated

		if fingerprintChecks(mutatedError) == fp {
			t.Errorf("fingerprint unchanged after error mutation of %q", nameA)
		}

		mutatedName := maps.Clone(checks)
		mutatedName[nameA+"x"] = mutatedName[nameA]
		delete(mutatedName, nameA)

		if fingerprintChecks(mutatedName) == fp {
			t.Errorf("fingerprint unchanged after name mutation of %q", nameA)
		}
	})
}

// validUTF8 rewrites invalid UTF-8 to U+FFFD so equality invariants stay
// meaningful across encoders that legitimately transcode invalid bytes.
func validUTF8(s string) string {
	return strings.ToValidUTF8(s, "\uFFFD")
}

// fuzzProber is a minimal internal Prober stub for fuzz harnesses that
// only exercise response-derived surfaces (no goroutines, no handlers).
type fuzzProber struct{}

func (fuzzProber) CachedResponse() health.Response    { return health.Response{} }
func (fuzzProber) RefreshInterval() time.Duration     { return time.Second }
func (fuzzProber) LivenessHandler() http.HandlerFunc  { return nil }
func (fuzzProber) ReadinessHandler() http.HandlerFunc { return nil }
func (fuzzProber) StartupHandler() http.HandlerFunc   { return nil }

// csvExportFor renders the current trend buffer through ExportHandler as
// CSV. The pusher is never started, so no goroutines participate.
func csvExportFor(t *testing.T, status string) string {
	t.Helper()

	d := New(fuzzProber{}, WithTrend(4))
	d.push.Store(newPusher(d))

	buf := d.push.Load().history
	buf.record(sample{At: time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC), Value: 1, Status: "pass"})
	buf.record(
		sample{At: time.Date(2026, 9, 4, 12, 0, 30, 0, time.UTC), Value: 0.5, Status: status},
	)

	rec := httptest.NewRecorder()
	req := fuzzRequest("text/csv")
	d.ExportHandler().ServeHTTP(rec, req)

	return rec.Body.String()
}

// FuzzCSVExport exercises the CSV exporter with arbitrary status payloads.
// Invariants: never panics, deterministic, always starts with the fixed
// header, and clean statuses (no comma, quote, CR, or LF) render as exactly
// three round-tripping fields per row.
func FuzzCSVExport(f *testing.F) {
	for _, seed := range []string{
		"pass",
		"warn",
		"fail",
		"unknown",
		"",
		"PASS",
		"pass,extra",
		"pass\nfail",
		"pass\r\nfail",
		`"quoted"`,
		"unicode ✓ café",
		"pass;injection",
		strings.Repeat("x", 300),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, status string) {
		first := csvExportFor(t, status)
		if second := csvExportFor(t, status); first != second {
			t.Fatalf("CSV export not deterministic for status %q", status)
		}

		lines := strings.Split(first, "\n")
		if len(lines) == 0 || lines[0] != "timestamp,value,status" {
			t.Fatalf("CSV header missing or wrong for status %q: %q", status, first)
		}

		clean := !strings.ContainsAny(status, ",\"\r\n")
		if clean {
			if len(lines) != 4 { // header + 2 rows + trailing newline
				t.Fatalf(
					"expected 4 lines for clean status %q, got %d: %q",
					status,
					len(lines),
					first,
				)
			}
			fields := strings.Split(lines[2], ",")
			if len(fields) != 3 || fields[2] != status {
				t.Fatalf("row does not round-trip clean status %q: %q", status, lines[2])
			}
		}
	})
}

// FuzzRecommendedCSP exercises the CSP builder with arbitrary nonce input.
// Invariants: never panics, deterministic, always contains exactly the eight
// directives, and the nonce appears only inside a 'nonce-…' source and only
// when it matches the CSP base64 alphabet.
func FuzzRecommendedCSP(f *testing.F) {
	for _, seed := range []string{
		"",
		"abc123",
		"4fpV0zKBFnDClJKBYXsinDHXOsB9DlCVxcwSoYQIEn0",
		"with space",
		"quote\"inside",
		"semi;colon",
		"new\nline",
		"nonce-game",
		"'nonce-evil'",
		"<>",
		"base64+/",
		"a=b",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, nonce string) {
		first := RecommendedCSP(nonce)
		if second := RecommendedCSP(nonce); first != second {
			t.Fatalf("RecommendedCSP not deterministic for nonce %q", nonce)
		}

		directives := strings.Split(first, "; ")
		if len(directives) != 8 {
			t.Fatalf(
				"expected 8 directives for nonce %q, got %d: %q",
				nonce,
				len(directives),
				first,
			)
		}

		matches := cspNonceValue.MatchString(nonce)
		if !matches && strings.Contains(first, "'nonce-"+nonce+"'") {
			t.Fatalf("invalid nonce %q leaked into CSP as a nonce-source: %q", nonce, first)
		}

		if matches && strings.Count(first, "'nonce-"+nonce+"'") != 1 {
			t.Fatalf(
				"nonce %q does not appear exactly once as a quoted nonce-source: %q",
				nonce,
				first,
			)
		}
	})
}

// FuzzWebhookPayload exercises webhook payload construction and its
// deterministic JSON encoding with arbitrary check names, errors, and
// statuses. Invariants: never panics, encoding is valid JSON that
// round-trips, masking holds in public mode, and ChangedAt is RFC3339.
func FuzzWebhookPayload(f *testing.F) {
	for _, seed := range [][3]string{
		{"api", "connection refused", "fail"},
		{"", "", "pass"},
		{"check\"quote", "err\nwith newline", "warn"},
		{"café", "utf8 ✓", "pass"},
		{"a,b", `{"json":"injection"}`, "fail"},
		{strings.Repeat("n", 200), strings.Repeat("e", 500), "warn"},
	} {
		f.Add(seed[0], seed[1], seed[2], false)
	}

	f.Fuzz(func(t *testing.T, name, errText, status string, public bool) {
		notifier := newWebhookNotifier(
			Config{WebhookURL: "http://fuzz.invalid/hook", PublicMode: public},
		)
		resp := health.Response{
			Status: health.Status(validUTF8(status)),
			Checks: map[string]health.Check{
				validUTF8(name): {
					Status: health.Status(validUTF8(status)),
					Error:  validUTF8(errText),
				},
			},
		}

		payload := notifier.buildPayload(resp)
		body, err := json.Marshal(payload, json.Deterministic(true))
		if err != nil {
			t.Fatalf(
				"marshal failed for name=%q err=%q status=%q public=%v: %v",
				name,
				errText,
				status,
				public,
				err,
			)
		}

		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatalf("payload is not valid JSON: %v\n%s", err, body)
		}

		if _, err := time.Parse(time.RFC3339, payload.ChangedAt); err != nil {
			t.Fatalf("ChangedAt not RFC3339: %q", payload.ChangedAt)
		}

		if public {
			checks, ok := decoded["checks"].(map[string]any)
			if !ok {
				t.Fatalf("public payload missing checks map: %s", body)
			}
			for key, raw := range checks {
				if key == validUTF8(name) {
					t.Fatalf("public mode kept original check name %q: %s", name, body)
				}
				fields, _ := raw.(map[string]any)
				if errText != "" && fields["error"] == validUTF8(errText) {
					t.Fatalf("public mode kept error text for %q: %s", key, body)
				}
			}
		}
	})
}
