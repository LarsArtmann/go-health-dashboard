package dashboard

import (
	"bytes"
	"encoding/csv"
	"encoding/json/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	health "github.com/larsartmann/go-health"
)

// jsonSample is the wire form of one recorded status sample, shared by the
// trend and export endpoints so both always agree on the mapping.
type jsonSample struct {
	At     string  `json:"at"`
	Value  float64 `json:"value"`
	Status string  `json:"status"`
}

// jsonTransition is the wire form of one status flip, shared by the trend
// endpoint and any future consumers of the derived history.
type jsonTransition struct {
	At   string `json:"at"`
	From string `json:"from"`
	To   string `json:"to"`
}

// jsonCheckMeta is the wire form of one check's go-health v0.2.0 metadata.
// Unknown facts are omitted (omitzero), never rendered as zero — mirroring
// go-health's own omitzero discipline on the /health response.
type jsonCheckMeta struct {
	Since      string `json:"since,omitzero"`
	DurationNs int64  `json:"duration_ns,omitzero"`
}

// jsonExportPayload is the export endpoint's JSON document: the recorded
// samples plus the CURRENT per-check metadata snapshot. Export is
// dashboard-owned (unlike the go-health-shaped /health response), so it is
// the one endpoint where per-check since/duration ride along — freshness
// and timing data for JSON consumers without touching HTML.
type jsonExportPayload struct {
	Samples []jsonSample `json:"samples"`
	// Checks maps every check in the current response to its known
	// metadata; fields absent when the source does not report them.
	Checks map[string]jsonCheckMeta `json:"checks"`
}

// buildExportPayload assembles the export document from the trend buffer
// and the current (sanitized) response. Checks is never nil so the payload
// always decodes with a present, iterable checks object.
func buildExportPayload(samples []sample, resp health.Response) jsonExportPayload {
	checks := make(map[string]jsonCheckMeta, len(resp.Checks))
	for name, check := range resp.Checks {
		meta := jsonCheckMeta{DurationNs: check.DurationNanos}
		if !check.Since.IsZero() {
			meta.Since = check.Since.UTC().Format(time.RFC3339)
		}

		checks[name] = meta
	}

	return jsonExportPayload{
		Samples: jsonSamples(samples),
		Checks:  checks,
	}
}

// jsonSamples maps history samples to their JSON wire form. Timestamps are
// RFC3339 UTC. Never returns nil, so payloads encode as [] not null.
func jsonSamples(samples []sample) []jsonSample {
	out := make([]jsonSample, 0, len(samples))
	for _, s := range samples {
		out = append(out, jsonSample{
			At:     s.At.UTC().Format(time.RFC3339),
			Value:  s.Value,
			Status: s.Status,
		})
	}

	return out
}

// jsonTransitions maps derived status transitions to their JSON wire form.
// Never returns nil, so payloads encode as [] not null.
func jsonTransitions(transitions []statusTransition) []jsonTransition {
	out := make([]jsonTransition, 0, len(transitions))
	for _, tr := range transitions {
		out = append(out, jsonTransition{
			At:   tr.At.UTC().Format(time.RFC3339),
			From: tr.From,
			To:   tr.To,
		})
	}

	return out
}

// notActiveMessage maps a nil-push / nil-history state to the right 503
// message: a nil pusher means the dashboard was never started (or was shut
// down), while a nil history means trend recording is not enabled.
func trendUnavailable(w http.ResponseWriter, history *historyBuffer) bool {
	if history == nil {
		http.Error(
			w,
			"dashboard: trend history is not enabled (set WithTrend)",
			http.StatusServiceUnavailable,
		)

		return true
	}

	return false
}

// TrendHandler serves the recorded status history as JSON. Enabled together
// with WithTrend at Routes.Trend (default /health/trend). The payload
// contains the raw samples plus the derived status transitions, so
// consumers can render their own timelines or alert on flips:
//
//	{"samples":[{"at":"2026-09-03T10:00:00Z","value":1,"status":"pass"}],
//	 "transitions":[{"at":"...","from":"pass","to":"warn"}]}
func (d *Dashboard) TrendHandler() http.HandlerFunc {
	type trendPayload struct {
		Samples     []jsonSample     `json:"samples"`
		Transitions []jsonTransition `json:"transitions"`
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		if trendUnavailable(w, d.history) {
			return
		}

		out := trendPayload{
			Samples:     jsonSamples(d.history.snapshot()),
			Transitions: jsonTransitions(d.history.transitions()),
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")

		// Deterministic encoding: the payload is slices-only today, so map
		// order cannot bite yet — the option future-proofs the endpoint
		// against a future map-bearing field silently regressing
		// byte-stability for diff-based scrapers.
		if err := json.MarshalWrite(w, out, json.Deterministic(true)); err != nil {
			http.Error(w, "dashboard: failed to encode trend", http.StatusInternalServerError)
		}
	}
}

// ExportHandler serves the recorded status history as JSON (default), CSV
// (?format=csv or Accept: text/csv), or newline-delimited JSON
// (?format=ndjson — one sample object per line, streamed for consumers
// that tail the export). Enabled together with WithTrend at Routes.Export
// (default /health/export).
//
// The JSON document is dashboard-owned and carries the current per-check
// metadata (go-health v0.2.0 since/duration_ns) next to the samples:
//
//	{"samples":[{"at":"...","value":1,"status":"pass"}],
//	 "checks":{"api/db":{"since":"...","duration_ns":42000000}}}
//
// CSV and NDJSON stay per-sample rows (check-level fields have no sample
// column to live in); the /health JSON contract stays go-health-shaped by
// design — export is the one endpoint that extends the shape.
func (d *Dashboard) ExportHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if trendUnavailable(w, d.history) {
			return
		}

		format := strings.ToLower(r.URL.Query().Get("format"))
		if format == "" {
			if strings.Contains(r.Header.Get("Accept"), "text/csv") {
				format = "csv"
			} else {
				format = "json"
			}
		}

		samples := d.history.snapshot()

		switch format {
		case "csv":
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")

			// encoding/csv provides RFC-4180 quoting (commas, quotes, and
			// newlines in adversarial status strings cannot break columns)
			// and writeCSVRow's discards are safe because bytes.Buffer
			// writes cannot fail — the document stays one writeBody call.
			var b bytes.Buffer

			csvWriter := csv.NewWriter(&b)

			writeCSVRow(csvWriter, "timestamp", "value", "status")

			for _, s := range samples {
				writeCSVRow(csvWriter,
					s.At.UTC().Format(time.RFC3339),
					strconv.FormatFloat(s.Value, 'g', -1, 64),
					exportCSVField(s.Status),
				)
			}

			csvWriter.Flush()
			writeBody(w, b.Bytes())
		case "json":
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-cache")

			out := buildExportPayload(samples, d.currentResponse())

			// Checks is a Go map: without the deterministic option the key
			// order changes per scrape, and export consumers diff payloads.
			if err := json.MarshalWrite(w, out, json.Deterministic(true)); err != nil {
				http.Error(w, "dashboard: failed to encode export", http.StatusInternalServerError)
			}
		case "ndjson":
			w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")

			// One self-contained JSON object per line: consumers can split on
			// newlines without parsing the whole payload first.
			for _, sample := range jsonSamples(samples) {
				line, err := json.Marshal(sample)
				if err != nil {
					http.Error(
						w,
						"dashboard: failed to encode export",
						http.StatusInternalServerError,
					)

					return
				}

				writeBody(w, line)
				writeBody(w, []byte{'\n'})
			}
		default:
			http.Error(w, "dashboard: unsupported export format "+format, http.StatusBadRequest)
		}
	}
}

// csvFormulaStarters are the leading bytes that turn a spreadsheet cell
// into a formula (the OWASP CSV-injection class: =, +, -, @, tab, CR).
// Export CSV is operator-consumed — analysts open it in Excel or Sheets —
// so an adversarial status string surfacing through a hostile or buggy
// prober must not execute there.
const csvFormulaStarters = "=+-@\t\r"

// exportCSVField neutralizes one CSV field against formula injection: a
// formula-starter first byte gets a leading single quote (the displayed
// value is unchanged, the formula is dead). RFC-4180 quoting for commas,
// quotes, and newlines is owned by the csv.Writer in ExportHandler.
func exportCSVField(field string) string {
	if field != "" && strings.ContainsRune(csvFormulaStarters, rune(field[0])) {
		return "'" + field
	}

	return field
}

// writeCSVRow appends one CSV record. The discard is provably safe: the
// writer wraps a bytes.Buffer, whose writes cannot fail.
//
//nolint:erraudit // bytes.Buffer-backed csv.Writer writes cannot fail
func writeCSVRow(w *csv.Writer, fields ...string) {
	_ = w.Write(fields)
}
