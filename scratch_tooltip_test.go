package dashboard_test

// SCRATCH reproduction for the "hover one tooltip, all tooltips light up"
// report. Serves a static fixture mirroring the golden badge-disclosure
// markup, loads the Tailwind Play CDN (the page's no-CSSPath path), hovers
// the first badge, and reports every tooltip's computed visibility plus the
// matching stylesheet rules. Delete when diagnosed.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/remote"
)

const tooltipFixturePage = `<!doctype html>
<html>
<head>
<meta charset="utf-8">
<script src="https://cdn.tailwindcss.com"></script>
</head>
<body>
<div style="margin:200px 40px;display:flex;flex-direction:column;gap:200px;width:400px">
  <span id="badge-1" tabindex="0" class="group relative inline-flex cursor-help rounded-md focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600 dark:focus-visible:outline-blue-500"><span class="sr-only">tip one</span><span aria-hidden="true" class="contents"><span class="bg-green-100 font-medium inline-flex items-center px-2.5 py-0.5 rounded-md text-green-800 text-xs">pass</span><span role="tooltip" class="pointer-events-none absolute bottom-full left-1/2 z-30 mb-1.5 hidden w-max max-w-[16rem] -translate-x-1/2 rounded-md bg-gray-900 px-2.5 py-1.5 text-left text-xs leading-snug text-gray-50 shadow-lg group-hover:block group-focus-within:block dark:bg-gray-700 dark:text-gray-100">tip one</span></span></span>
  <span id="badge-2" tabindex="0" class="group relative inline-flex cursor-help rounded-md focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600 dark:focus-visible:outline-blue-500"><span class="sr-only">tip two</span><span aria-hidden="true" class="contents"><span class="bg-green-100 font-medium inline-flex items-center px-2.5 py-0.5 rounded-md text-green-800 text-xs">pass</span><span role="tooltip" class="pointer-events-none absolute bottom-full left-1/2 z-30 mb-1.5 hidden w-max max-w-[16rem] -translate-x-1/2 rounded-md bg-gray-900 px-2.5 py-1.5 text-left text-xs leading-snug text-gray-50 shadow-lg group-hover:block group-focus-within:block dark:bg-gray-700 dark:text-gray-100">tip two</span></span></span>
  <span id="badge-3" tabindex="0" class="group relative inline-flex cursor-help rounded-md focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600 dark:focus-visible:outline-blue-500"><span class="sr-only">tip three</span><span aria-hidden="true" class="contents"><span class="bg-green-100 font-medium inline-flex items-center px-2.5 py-0.5 rounded-md text-green-800 text-xs">pass</span><span role="tooltip" class="pointer-events-none absolute bottom-full left-1/2 z-30 mb-1.5 hidden w-max max-w-[16rem] -translate-x-1/2 rounded-md bg-gray-900 px-2.5 py-1.5 text-left text-xs leading-snug text-gray-50 shadow-lg group-hover:block group-focus-within:block dark:bg-gray-700 dark:text-gray-100">tip three</span></span></span>
</div>
</body>
</html>`

type tooltipState struct {
	Text    string `json:"text"`
	Display string `json:"display"`
}

type tooltipReport struct {
	State []tooltipState `json:"state"`
	Rules []string       `json:"rules"`
}

const tooltipVisibilityJS = `(() => {
	const tips = [...document.querySelectorAll('[role="tooltip"]')];
	const state = tips.map((tip) => ({
		text: tip.textContent.trim(),
		display: getComputedStyle(tip).display,
	}));
	const styles = [...document.styleSheets].flatMap((sheet) => {
		try { return [...sheet.cssRules]; } catch { return []; }
	}).map((rule) => rule.cssText).filter((css) => css.indexOf("group-hover") !== -1);
	return JSON.stringify({ state, rules: styles });
})()`

func TestScratch_TooltipHoverIsolation(t *testing.T) {
	chromePath := usableChrome(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(tooltipFixturePage))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	devtools, stopChrome := startHeadlessChrome(t, chromePath)
	defer stopChrome()

	runCtx, runCancel := context.WithTimeout(context.Background(), browserRunTimeout)
	defer runCancel()

	allocCtx, allocCancel := remote.NewAllocator(runCtx, devtools)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	if err := chromedp.Do(ctx, chromedp.Navigate(server.URL)); err != nil {
		t.Fatalf("navigate: %v", err)
	}

	// Wait for the Play CDN stylesheet to land: before hover, tooltip 1 must
	// already compute display:none (the compiled `hidden` class exists).
	var report string
	deadline := time.Now().Add(45 * time.Second)
	for {
		if err := chromedp.Do(ctx, evalInto(`getComputedStyle(document.querySelector('[role="tooltip"]')).display`, &report)); err != nil {
			t.Fatalf("probe display: %v", err)
		}
		if report == "none" || time.Now().After(deadline) {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Logf("pre-hover tooltip display = %q (stylesheet compiled: %v)", report, report == "none")

	var centerJSON string
	if err := chromedp.Do(ctx, evalInto(`(() => { const r = document.querySelector('#badge-1').getBoundingClientRect(); return JSON.stringify({x: r.x + r.width/2, y: r.y + r.height/2}); })()`, &centerJSON)); err != nil {
		t.Fatalf("badge center: %v", err)
	}
	var center struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	}
	if err := json.Unmarshal([]byte(centerJSON), &center); err != nil {
		t.Fatalf("parse badge center %q: %v", centerJSON, err)
	}

	if err := chromedp.Do(ctx,
		chromedp.WaitVisible("#badge-1"),
		chromedp.MouseEvent(chromedp.MouseMoved, center.X, center.Y),
	); err != nil {
		t.Fatalf("hover: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	if err := chromedp.Do(ctx, evalInto(tooltipVisibilityJS, &report)); err != nil {
		t.Fatalf("visibility report: %v", err)
	}
	var parsed tooltipReport
	if err := json.Unmarshal([]byte(report), &parsed); err != nil {
		t.Fatalf("parse report: %v", err)
	}

	visible := 0
	for _, st := range parsed.State {
		t.Logf("tooltip %q display=%s", st.Text, st.Display)
		if st.Display != "none" {
			visible++
		}
	}
	t.Logf("visible tooltips after hovering badge-1: %d of %d", visible, len(parsed.State))
	for _, rule := range parsed.Rules {
		t.Logf("group-hover rule: %.300s", rule)
	}

	if visible != 1 {
		t.Errorf("REPRODUCED: %d tooltips visible after hovering one badge (want 1)", visible)
	}
}
