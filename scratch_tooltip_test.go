package dashboard_test

// SCRATCH reproduction for the "hover one tooltip, all tooltips light up"
// report. Drives the REAL rendered dashboard page with headless Chrome in
// two CSS modes — the Play CDN (no-CSSPath default) and a locally compiled
// Tailwind v4 build (the consumer recipe) — hovers the first badge
// disclosure, focuses it, and reports every role="tooltip" computed
// visibility. Delete when diagnosed.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/remote"

	dashboard "github.com/larsartmann/go-health-dashboard"
)

const compiledV4CSS = "/tmp/dash-v4.css"

type tooltipState struct {
	Text    string `json:"text"`
	Display string `json:"display"`
}

const tooltipProbeJS = `(() => {
	const tips = [...document.querySelectorAll('[role="tooltip"]')];
	return JSON.stringify(tips.map((tip) => ({
		text: tip.textContent.trim().slice(0, 40),
		display: getComputedStyle(tip).display,
	})));
})()`

func TestScratch_TooltipHoverRealPage(t *testing.T) {
	chromePath := usableChrome(t)

	for _, mode := range []string{"playcdn", "compiled-v4"} {
		t.Run(mode, func(t *testing.T) {
			opts := []dashboard.Option{}
			static := http.NewServeMux()
			if mode == "compiled-v4" {
				css, err := os.ReadFile(compiledV4CSS)
				if err != nil {
					t.Skipf("no compiled v4 CSS at %s: %v", compiledV4CSS, err)
				}
				static.HandleFunc("/static/app.css", func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "text/css")
					_, _ = w.Write(css)
				})
				opts = append(opts,
					dashboard.WithCSSPath("/static/app.css"),
					dashboard.WithDatastarSrc("/static/datastar.js"),
				)
			}

			s := setupDashboard(t, opts...)
			defer s.cleanup()

			mux := http.NewServeMux()
			mux.Handle("/static/", static)
			mux.Handle("/", s.mux)

			server := httptest.NewServer(mux)
			t.Cleanup(server.Close)

			devtools, stopChrome := startHeadlessChrome(t, chromePath)
			defer stopChrome()

			runCtx, runCancel := context.WithTimeout(context.Background(), browserRunTimeout)
			defer runCancel()

			allocCtx, allocCancel := remote.NewAllocator(runCtx, devtools)
			defer allocCancel()

			ctx, cancel := chromedp.NewContext(allocCtx)
			defer cancel()

			if err := chromedp.Do(ctx, chromedp.Navigate(server.URL+"/health")); err != nil {
				t.Fatalf("navigate: %v", err)
			}

			var report string
			deadline := time.Now().Add(45 * time.Second)
			for {
				if err := chromedp.Do(
					ctx,
					evalInto(
						`(() => { const t = document.querySelector('[role="tooltip"]'); return t ? getComputedStyle(t).display : "no-tip"; })()`,
						&report,
					),
				); err != nil {
					t.Fatalf("probe display: %v", err)
				}
				if report == "none" || time.Now().After(deadline) {
					break
				}
				time.Sleep(250 * time.Millisecond)
			}
			t.Logf("pre-hover first tooltip display = %q", report)

			var centerJSON string
			if err := chromedp.Do(ctx, evalInto(`(() => {
				// A badge disclosure INSIDE the healthy CollapsibleSection: the
				// section <details> itself carries class "group" and wraps every
				// healthy row — the suspected shared ancestor.
				const b = document.querySelector('details.group span[tabindex="0"].group') || document.querySelector('span[tabindex="0"].group');
				if (!b) { return "{}"; }
				const inSection = !!b.closest('details.group');
				b.scrollIntoView({block: "center"});
				const r = b.getBoundingClientRect();
				return JSON.stringify({x: r.x + r.width/2, y: r.y + r.height/2, inSection});
			})()`, &centerJSON)); err != nil {
				t.Fatalf("badge center: %v", err)
			}
			var center struct {
				X         float64 `json:"x"`
				Y         float64 `json:"y"`
				InSection bool    `json:"inSection"`
			}
			if err := json.Unmarshal([]byte(centerJSON), &center); err != nil || center.X == 0 {
				t.Fatalf("no badge found or bad center %q: %v", centerJSON, err)
			}
			t.Logf(
				"hover target inSection=%v center=(%.1f,%.1f)",
				center.InSection,
				center.X,
				center.Y,
			)

			probeVisible := func(label string) {
				if err := chromedp.Do(ctx, evalInto(tooltipProbeJS, &report)); err != nil {
					t.Fatalf("probe tooltips: %v", err)
				}
				var states []tooltipState
				if err := json.Unmarshal([]byte(report), &states); err != nil {
					t.Fatalf("parse states: %v", err)
				}
				visible := 0
				for _, st := range states {
					if st.Display != "none" {
						visible++
						t.Logf("%s: tooltip %q display=%s", label, st.Text, st.Display)
					}
				}
				t.Logf("%s: visible tooltips %d of %d", label, visible, len(states))
				if len(states) > 1 && visible != 1 {
					t.Errorf("%s REPRODUCED: %d tooltips visible (want exactly 1)", label, visible)
				}
			}

			if err := chromedp.Do(ctx,
				chromedp.WaitVisible("details.group span[tabindex=\"0\"].group"),
				chromedp.MouseEvent(chromedp.MouseMoved, center.X, center.Y),
			); err != nil {
				t.Fatalf("hover: %v", err)
			}
			time.Sleep(300 * time.Millisecond)
			probeVisible("HOVER")

			// Focus a DIFFERENT badge inside the section: the focus path must
			// stay isolated too.
			if err := chromedp.Do(ctx, evalDiscard(`(() => {
				const badges = [...document.querySelectorAll('details.group span[tabindex="0"].group')];
				if (badges.length < 2) { return "need-two"; }
				badges[1].focus();
				return "ok";
			})()`)); err != nil {
				t.Fatalf("focus: %v", err)
			}
			time.Sleep(200 * time.Millisecond)
			probeVisible("FOCUS")
		})
	}
}
