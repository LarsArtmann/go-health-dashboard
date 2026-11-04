package dashboard_test

import (
	"math"
	"testing"
)

// The WCAG verdicts for the status palette live in exactly ONE place: this
// file computes them from the Tailwind palette constants. Everything else
// points here. The four hand-maintained copies this replaces (test
// comments, status.go, ADR-0006, the dark-contrast research note) had
// already drifted — the research note claimed gray-500 on white ≈ 7.5:1
// while the render test said 4.83:1 (4.84 is correct; 2026-11-04).

// tailwindV3Color is an sRGB color from Tailwind v3's default palette: the
// 8-bit R/G/B channels the browser renders for the utility class.
type tailwindV3Color struct {
	r, g, b uint8
}

// tailwindPalette pins the Tailwind v3 default-palette hex values the page
// actually renders: the dashboard loads the v3 Play CDN
// (https://cdn.tailwindcss.com, view.templ dashboardHead), whose compiler
// emits exactly these colors. Verified byte-for-byte against
// tailwindlabs/tailwindcss@v3 src/public/colors.js (2026-11-04). If the UI
// ever moves to Tailwind v4 — whose default palette is OKLCH-defined and
// renders subtly different sRGB values — re-derive this map from the v4
// theme and re-run this file's tests before trusting any verdict below.
var tailwindPalette = map[string]tailwindV3Color{
	"white":     {0xff, 0xff, 0xff},
	"gray-400":  {0x9c, 0xa3, 0xaf},
	"gray-500":  {0x6b, 0x72, 0x80},
	"gray-800":  {0x1f, 0x29, 0x37},
	"gray-900":  {0x11, 0x18, 0x27},
	"green-400": {0x4a, 0xde, 0x80},
	"green-600": {0x16, 0xa3, 0x4a},
	"green-700": {0x15, 0x80, 0x3d},
	"amber-400": {0xfb, 0xbf, 0x24},
	"amber-600": {0xd9, 0x77, 0x06},
	"amber-700": {0xb4, 0x53, 0x09},
	"red-400":   {0xf8, 0x71, 0x71},
	"red-600":   {0xdc, 0x26, 0x26},
}

// relativeLuminance is WCAG 2.x relative luminance (WCAG 2.1 §1.4.3
// definition): gamma-expand each sRGB channel with the spec's piecewise
// curve (threshold 0.03928) and take the weighted sum.
func (c tailwindV3Color) relativeLuminance() float64 {
	channel := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.03928 {
			return s / 12.92
		}

		return math.Pow((s+0.055)/1.055, 2.4)
	}

	return 0.2126*channel(c.r) + 0.7152*channel(c.g) + 0.0722*channel(c.b)
}

// contrastRatio is the WCAG 2.x contrast ratio of two colors:
// (L_lighter + 0.05) / (L_darker + 0.05), range [1, 21].
func contrastRatio(a, b tailwindV3Color) float64 {
	la, lb := a.relativeLuminance(), b.relativeLuminance()
	lighter, darker := math.Max(la, lb), math.Min(la, lb)

	return (lighter + 0.05) / (darker + 0.05)
}

// TestWCAG_StatusPaletteContrast derives the AA verdicts for every status
// palette pair the page ships, from the constants above. It pins BOTH
// directions of the decision, so a Tailwind palette change that moves any
// ratio across the 4.5:1 floor fails the build instead of silently
// invalidating a hand-copied number:
//
//   - the chosen text tones must clear AA (>= 4.5:1) on their surface —
//     white in light mode, gray-800 (the card surface) in dark mode;
//   - the banned -600 tones must stay BELOW the floor, so the render
//     ban in TestRender_ContrastSafeStatusColors remains justified by
//     computation, not habit.
func TestWCAG_StatusPaletteContrast(t *testing.T) {
	t.Parallel()

	const aaNormalText = 4.5

	surfaceLight := tailwindPalette["white"]
	surfaceDark := tailwindPalette["gray-800"]

	// The pairs the page renders, in the same terms the render test
	// asserts them (class -> palette entry). The light half sits on
	// white; every dark:* variant sits on the gray-800 card surface.
	safePairs := []struct {
		class string
		fg    string
		dark  bool
	}{
		{"text-green-700 dark:text-green-400", "green-700", false},
		{"text-green-700 dark:text-green-400", "green-400", true},
		{"text-amber-700 dark:text-amber-400", "amber-700", false},
		{"text-amber-700 dark:text-amber-400", "amber-400", true},
		{"text-red-600 dark:text-red-400", "red-600", false},
		{"text-red-600 dark:text-red-400", "red-400", true},
		{"text-gray-500 dark:text-gray-400 (metadata line, raw keys)", "gray-500", false},
		{"text-gray-500 dark:text-gray-400 (metadata line, raw keys)", "gray-400", true},
	}

	for _, pair := range safePairs {
		surface := surfaceLight
		if pair.dark {
			surface = surfaceDark
		}

		ratio := contrastRatio(tailwindPalette[pair.fg], surface)
		if ratio < aaNormalText {
			t.Errorf(
				"%s (%s on dark surface) computes to %.2f:1 — BELOW the AA floor %v: the class pair is no longer contrast-safe",
				pair.class,
				pair.fg,
				ratio,
				aaNormalText,
			)
		}

		t.Logf("safe  %-24s on %s: %.2f:1", pair.class, pair.fg, ratio)
	}

	// The banned tones: prove the AA floor is what bans them.
	bannedPairs := []struct {
		class string
		fg    string
	}{
		{"text-green-600", "green-600"},
		{"text-amber-600", "amber-600"},
		{"text-gray-400 (raw-key pre-fix form)", "gray-400"},
	}

	for _, pair := range bannedPairs {
		ratio := contrastRatio(tailwindPalette[pair.fg], surfaceLight)
		if ratio >= aaNormalText {
			t.Errorf(
				"banned tone %s computes to %.2f:1 on white — at or above the AA floor %v: relax the TestRender_ContrastSafeStatusColors ban deliberately if this movement is intended",
				pair.class,
				ratio,
				aaNormalText,
			)
		}

		t.Logf("banned %-24s on white: %.2f:1", pair.class, ratio)
	}

	// The research-note surfaces, computed for the record: the dim
	// metadata line (gray-400) sits on BOTH dark surfaces, and the
	// decorative chevron's gray-500 needs only the 3:1 non-text floor
	// (WCAG 1.4.11).
	recordOnly := []struct {
		pair   string
		fg, bg string
		floor  float64
	}{
		{"gray-400 on gray-900 (metadata on table)", "gray-400", "gray-900", aaNormalText},
		{"gray-500 on gray-900 (decorative chevron, 1.4.11)", "gray-500", "gray-900", 3.0},
	}

	for _, pair := range recordOnly {
		ratio := contrastRatio(tailwindPalette[pair.fg], tailwindPalette[pair.bg])
		if ratio < pair.floor {
			t.Errorf("%s computes to %.2f:1 — below its %.1f:1 floor", pair.pair, ratio, pair.floor)
		}

		t.Logf("record %-42s: %.2f:1", pair.pair, ratio)
	}

	if t.Failed() {
		return
	}

	t.Log(
		"all status-palette verdicts recomputed from the Tailwind v3 constants; ADR-0006's Context figures are the 2026-09-10 audit snapshot",
	)
}
