package dashboard_test

// SCRATCH reproduction for the "hover one tooltip, all tooltips light up"
// report. Drives the REAL rendered dashboard page (Play CDN path, the
// no-CSSPath default the demos use) with headless Chrome, hovers the first
// badge disclosure, and reports every role="tooltip" computed visibility.
// Delete when diagnosed.

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/remote"
)

type tooltipState struct {
	Text    string `json:"text"`
	Display string `json:"display"`
}

const realPageProbeJS = `(() => {
	const tips = [...document.querySelectorAll('[role="tooltip"]')];
	return JSON.stringify(tips.map((tip) => ({
		text: tip.textContent.trim().slice(0, 40),
		display: getComputedStyle(tip).display,
	})));
})()`

func TestScratch_TooltipHoverRealPage(t *testing.T) {
	chromePath := usableChrome(t)

	s := setupDashboard(t)
	defer s.cleanup()

	server := httptest.NewServer(s.mux)
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

	// The healthy fixture yields unproven-green disclosures; wait until the
	// first tooltip computes hidden (CDN compiled) and page is interactive.
	var report string
	deadline := time.Now().Add(45 * time.Second)
	for {
		if err := chromedp.Do(ctx, evalInto(`(() => { const t = document.querySelector('[role="tooltip"]'); return t ? getComputedStyle(t).display : "no-tip"; })()`, &report)); err != nil {
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
		const b = document.querySelector('span[tabindex="0"].group');
		if (!b) { return "{}"; }
		b.scrollIntoView({block: "center"});
		const r = b.getBoundingClientRect();
		return JSON.stringify({x: r.x + r.width/2, y: r.y + r.height/2, w: window.innerWidth, h: window.innerHeight});
	})()`, &centerJSON)); err != nil {
		t.Fatalf("badge center: %v", err)
	}
	var center struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
		W int     `json:"w"`
		H int     `json:"h"`
	}
	if err := json.Unmarshal([]byte(centerJSON), &center); err != nil || center.X == 0 {
		t.Fatalf("no badge found or bad center %q: %v", centerJSON, err)
	}
	t.Logf("hover target center=(%.1f,%.1f) viewport=%dx%d", center.X, center.Y, center.W, center.H)

	if err := chromedp.Do(ctx,
		chromedp.WaitVisible("span[tabindex=\"0\"].group"),
		chromedp.MouseEvent(chromedp.MouseMoved, center.X, center.Y),
	); err != nil {
		t.Fatalf("hover: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	if err := chromedp.Do(ctx, evalInto(`(() => {
		const chain = [...document.querySelectorAll(':hover')].map((el) => el.tagName + "." + (el.className || "").toString().slice(0, 60));
		const hovered = document.querySelector('span[tabindex="0"].group:hover') !== null;
		return JSON.stringify({ chain, hovered });
	})()`, &report)); err != nil {
		t.Fatalf("hover chain: %v", err)
	}
	t.Logf("hover chain after mouse move: %s", report)

	if err := chromedp.Do(ctx, evalInto(realPageProbeJS, &report)); err != nil {
		t.Fatalf("probe tooltips: %v", err)
	}
	var states []tooltipState
	if err := json.Unmarshal([]byte(report), &states); err != nil {
		t.Fatalf("parse states: %v", err)
	}

	visible := 0
	for _, st := range states {
		t.Logf("tooltip %q display=%s", st.Text, st.Display)
		if st.Display != "none" {
			visible++
		}
	}
	t.Logf("visible tooltips after hover: %d of %d", visible, len(states))
	if len(states) > 1 && visible != 1 {
		t.Errorf("REPRODUCED on real page: %d tooltips visible after hovering one badge (want 1)", visible)
	}
}
