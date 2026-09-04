package dashboard

import (
	"encoding/json/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
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
func trendUnavailable(w http.ResponseWriter, push *pusher, history *historyBuffer) bool {
	if push == nil {
		http.Error(w, "dashboard: SSE pusher is not active (call Start before serving traffic)", http.StatusServiceUnavailable)

		return true
	}

	if history == nil {
		http.Error(w, "dashboard: trend history is not enabled (set WithTrend)", http.StatusServiceUnavailable)

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
		push := d.push.Load()
		if trendUnavailable(w, push, pushHistory(push)) {
			return
		}

		out := trendPayload{
			Samples:     jsonSamples(push.history.snapshot()),
			Transitions: jsonTransitions(push.history.transitions()),
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")

		if err := json.MarshalWrite(w, out); err != nil {
			http.Error(w, "dashboard: failed to encode trend", http.StatusInternalServerError)
		}
	}
}

// ExportHandler serves the recorded status history as JSON (default) or CSV
// (?format=csv or Accept: text/csv). Enabled together with WithTrend at
// Routes.Export (default /health/export). CSV columns: timestamp, value,
// status.
func (d *Dashboard) ExportHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		push := d.push.Load()
		if trendUnavailable(w, push, pushHistory(push)) {
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

		samples := push.history.snapshot()

		switch format {
		case "csv":
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")

			var b strings.Builder

			b.WriteString("timestamp,value,status\n")

			for _, s := range samples {
				b.WriteString(s.At.UTC().Format(time.RFC3339))
				b.WriteByte(',')
				b.WriteString(strconv.FormatFloat(s.Value, 'g', -1, 64))
				b.WriteByte(',')
				b.WriteString(s.Status)
				b.WriteByte('\n')
			}

			_, _ = w.Write([]byte(b.String()))
		case "json":
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-cache")

			if err := json.MarshalWrite(w, jsonSamples(samples)); err != nil {
				http.Error(w, "dashboard: failed to encode export", http.StatusInternalServerError)
			}
		default:
			http.Error(w, "dashboard: unsupported export format "+format, http.StatusBadRequest)
		}
	}
}

// pushHistory returns the pusher's trend buffer, or nil when either is
// absent. A helper so the two trend endpoints share one nil-handling path.
func pushHistory(push *pusher) *historyBuffer {
	if push == nil {
		return nil
	}

	return push.history
}
