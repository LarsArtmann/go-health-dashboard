package dashboard

import (
	"encoding/json/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// TrendHandler serves the recorded status history as JSON. Enabled together
// with WithTrend at Routes.Trend (default /health/trend). The payload
// contains the raw samples plus the derived status transitions, so
// consumers can render their own timelines or alert on flips:
//
//	{"samples":[{"at":"2026-09-03T10:00:00Z","value":1,"status":"pass"}],
//	 "transitions":[{"at":"...","from":"pass","to":"warn"}]}
func (d *Dashboard) TrendHandler() http.HandlerFunc {
	type jsonSample struct {
		At     string  `json:"at"`
		Value  float64 `json:"value"`
		Status string  `json:"status"`
	}

	type jsonTransition struct {
		At   string `json:"at"`
		From string `json:"from"`
		To   string `json:"to"`
	}

	type trendPayload struct {
		Samples     []jsonSample     `json:"samples"`
		Transitions []jsonTransition `json:"transitions"`
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		push := d.push.Load()

		if push == nil || push.history == nil {
			http.Error(w, "dashboard: trend history is not enabled", http.StatusServiceUnavailable)

			return
		}

		samples := push.history.snapshot()
		out := trendPayload{
			Samples:     make([]jsonSample, 0, len(samples)),
			Transitions: []jsonTransition{},
		}

		for _, s := range samples {
			out.Samples = append(out.Samples, jsonSample{
				At:     s.At.UTC().Format(time.RFC3339),
				Value:  s.Value,
				Status: s.Status,
			})
		}

		for _, tr := range push.history.transitions() {
			out.Transitions = append(out.Transitions, jsonTransition{
				At:   tr.At.UTC().Format(time.RFC3339),
				From: tr.From,
				To:   tr.To,
			})
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

		if push == nil || push.history == nil {
			http.Error(w, "dashboard: trend history is not enabled", http.StatusServiceUnavailable)

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
			type jsonSample struct {
				At     string  `json:"at"`
				Value  float64 `json:"value"`
				Status string  `json:"status"`
			}

			out := make([]jsonSample, 0, len(samples))
			for _, s := range samples {
				out = append(out, jsonSample{
					At:     s.At.UTC().Format(time.RFC3339),
					Value:  s.Value,
					Status: s.Status,
				})
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-cache")

			if err := json.MarshalWrite(w, out); err != nil {
				http.Error(w, "dashboard: failed to encode export", http.StatusInternalServerError)
			}
		default:
			http.Error(w, "dashboard: unsupported export format "+format, http.StatusBadRequest)
		}
	}
}
