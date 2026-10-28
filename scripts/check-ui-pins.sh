#!/usr/bin/env bash
# UI-dependency pin guard.
#
# Fails unless the pinned UI dependency versions are exactly as audited.
#
# History: templ-components v1.12.0 re-introduced the LiveRegion busy-script
# `nonce=""` regression (upstream templ-components#7), and go-datastar
# v0.5.0 swapped the audited SDK bundle — three CSP-invariant tests failed
# and the breakage landed silently twice on 2026-09-04.
#
# Re-audited 2026-09-09 on templ-components v1.13.x + go-datastar v0.5.0
# (the #7 nonce guard present in v1.13.2; the v0.5.0 SDK bundle passed the
# full browser suite under strict CSP).
#
# Re-audited 2026-09-10 on templ-components v1.16.0 + go-datastar v0.5.0:
#   - the full browser suite (CSP-clean runtime, live SSE patch, a11y,
#     keyboard, metrics, aggregate, collapse, filter, pill, mobile) is
#     green on the bumped set,
#   - upstream templ-components#6 (StatCard <dl> markup) is FIXED in
#     v1.16.0 (dt+dd grouped in one wrapper div), so the axe
#     definition-list/dlitem tolerance in browser_test.go was retired,
#   - templ-components/utils is now a direct module and is pinned too.
# The v1.16.0 bump itself arrived as another unguarded sweep (fifth
# occurrence, 2026-09-10) — the guard caught it in CI, as designed.
#
# Re-audited 2026-09-16 on templ-components v1.17.0 + go-datastar v0.5.0:
#   - the v1.17.0 bump (a11y hardening + trust batch) arrived as another
#     unguarded daemon sweep (sixth occurrence, 2026-09-16); adopted as a
#     dedicated change after the full browser suite ran green on the
#     bumped set under strict CSP.
#
# Re-audited 2026-09-18 on templ-components v1.18.0 + go-datastar v0.5.0:
#   - the v1.18.0 bump arrived as an unguarded deps commit (seventh
#     occurrence, 2026-09-18); the full browser suite (incl. the dark-mode
#     axe re-audit) ran green on the bumped set under strict CSP before
#     these pins were updated in the same change.
#
# Re-audited 2026-10-09 on templ-components v1.21.0 + go-datastar v0.6.2:
#   - the v1.19.2 → v1.21.0 climb rode the 2026-10-08 fleet sweep (daemon
#     commit 54cdc59, eighth unguarded occurrence); the unit suite was
#     already green and the full browser suite (CSP, live SSE patch, axe
#     a11y both themes, keyboard, pill, collapse, filter, metrics, mobile)
#     ran green on the bumped set under strict CSP before these pins were
#     updated.
#   - templ-components/icons joined as a DIRECT dependency (timeline trend
#     glyphs, stat-card icons) at the same v1.21.0 — pinned from here on.
#
# Re-audited 2026-10-25 on templ-components v1.23.0 + go-datastar v0.6.2:
#   - the v1.21.0 → v1.23.0 climb rode the 2026-10-15 daemon churn (commit
#     93d13fd, ninth unguarded occurrence — the same commit series that
#     broke the chromedp test build); discovered red-guarded during the
#     2026-10-25 UI batch closeout.
#   - the full browser suite (19 tests: CSP, live SSE patch, axe both
#     themes, keyboard incl. links + new controls, pill, collapse, filter,
#     chips, evidence disclosure, metrics, mobile, off-row, retry) ran
#     green under strict CSP on the bumped set before these pins were
#     updated in the same change.
#
# UNAUDITED DRIFT 2026-10-28, templ-components v1.23.0 → v1.24.0 (tenth
# unguarded sweep; this guard is currently RED on purpose). The markup delta
# was measured against the module cache sources, not guessed:
#   - display.Table: the horizontal-scroll wrapper LOSES its
#     role="region"/role="img" + aria-label, and the <table> element GAINS the
#     aria-label instead. Every rendered table changes.
#   - display.CopyButton: the label span LOSES role="status" aria-live="polite"
#     and a sibling <span data-tc-copy-status role="status" class="sr-only">
#     carries the announcement now; the inline script gains tcAnnounceCopy.
#   - errorpage.NotFound404: the search field's leading icon (ps-10) is
#     replaced by a trailing submit BUTTON carrying the same icon (ps-4/pe-14),
#     and two new props (GoBackText, SearchSubmitLabel) gained defaults.
# Blast radius measured: 7 golden files drift here (severity, source,
# zeroproven, publicmode, sourceproven, trendtimeline, latencywarn) and
# DiscordSync's internal/web snapshot suite drops from 19 to 15 failures on the
# identical markup — one upstream release, two repos, same cause. (The other 4
# DiscordSync failures were a seasonal CEST/CET time bomb in its golden
# normalizer, fixed separately.) Regenerating the goldens is a DECISION (adopt
# v1.24.0) and a visual audit — the 404 change in particular is a UI redesign —
# not a chore: do not run `-golden-update` to make this guard quiet.
#
# CEREMONY RULE: if you bump any of these dependencies, update the pins
# in this file IN THE SAME CHANGE and re-run the browser suite. A bump
# without its guard update leaves CI red for every subsequent commit.
#
# REMOVAL CONDITION: the guard stays (decision 2026-09-09, user sign-off
# pending) — five unguarded sweeps say version movement needs a dedicated
# change that updates these pins and re-runs the browser suite.
set -euo pipefail

expected_templ_components="v1.23.0"
expected_templ_components_datastar="v1.23.0"
expected_templ_components_utils="v1.23.0"
expected_templ_components_icons="v1.23.0"
expected_go_datastar="v0.6.2"
expected_go_datastar_static="v0.6.1"

# The exact bytes of the pinned SDK bundle (go-datastar/static datastar.js).
# Upstream pins the same sha in its own checksum_test.go; this is the
# consumer-side belt-and-suspenders: even a same-version module-cache
# anomaly cannot swap the bundle silently. Size is the coarse canary.
expected_sdk_sha256="5d6b7794a50a83d82da962aec5e382f5ae83ac7afbc751f903f7a9c6bd433c65"
expected_sdk_bytes="33538"

cd "$(dirname "$0")/.."

# Self-select the right go: the ambient PATH go may predate go.mod's
# floor, and with GOTOOLCHAIN=local it dies on a misleading "go: updates
# to go.mod needed" (AGENTS.md "Repo scripts that exec go need the
# devShell"). Probe with a cheap module load; fall back to the devShell.
go_list() {
	if go list -m github.com/larsartmann/go-health >/dev/null 2>&1; then
		go list -m "$1"
	else
		nix develop -c go list -m "$1" | grep -E "^${1} "
	fi
}

fail=0
check_pin() {
	local module="$1" expected="$2"
	actual=$(go_list "$module" | awk '{print $2}')
	if [[ "$actual" == "$expected" ]]; then
		echo "OK  $module $actual"
	else
		echo "::error::$module is $actual, expected pinned $expected — UI dependency movement requires a dedicated change with a green browser suite; see scripts/check-ui-pins.sh"
		fail=1
	fi
}

check_pin github.com/larsartmann/templ-components "$expected_templ_components"
check_pin github.com/larsartmann/templ-components/datastar "$expected_templ_components_datastar"
check_pin github.com/larsartmann/templ-components/utils "$expected_templ_components_utils"
check_pin github.com/larsartmann/templ-components/icons "$expected_templ_components_icons"
check_pin github.com/larsartmann/go-datastar "$expected_go_datastar"
check_pin github.com/larsartmann/go-datastar/static "$expected_go_datastar_static"

# --- SDK bundle content drift (2026-10-25) ------------------------------
# Version pins alone cannot see a content swap inside a module-cache
# anomaly; the embedded datastar.js bytes are pinned here too.

go_run() {
	if go list -m github.com/larsartmann/go-health >/dev/null 2>&1; then
		go "$@"
	else
		nix develop -c go "$@"
	fi
}

modcache=$(go_run env GOMODCACHE)

bundle="$modcache/github.com/larsartmann/go-datastar/static@$expected_go_datastar_static/datastar.js"

if [ ! -f "$bundle" ]; then
	# Cold module cache (fresh CI runner, clean checkout): fetch the pinned
	# module instead of failing — the guard must be order-independent where
	# it runs (before any build step populates the cache).
	go_run mod download github.com/larsartmann/go-datastar/static
fi

if [ ! -f "$bundle" ]; then
	echo "::error::SDK bundle not found at $bundle even after 'go mod download github.com/larsartmann/go-datastar/static'"
	fail=1
else
	actual_sha=$(sha256sum "$bundle" | cut -d' ' -f1)
	if [ "$actual_sha" != "$expected_sdk_sha256" ]; then
		echo "::error::SDK bundle sha256 $actual_sha != pinned $expected_sdk_sha256 — the embedded Datastar SDK changed bytes; a dedicated change must re-run the browser suite and update this pin"
		fail=1
	else
		echo "OK  SDK bundle sha256 $actual_sha"
	fi

	actual_bytes=$(wc -c <"$bundle" | tr -d ' ')
	if [ "$actual_bytes" != "$expected_sdk_bytes" ]; then
		echo "::error::SDK bundle size $actual_bytes bytes, pinned $expected_sdk_bytes — bundle-size drift (see sha error above if present)"
		fail=1
	else
		echo "OK  SDK bundle size $actual_bytes bytes"
	fi
fi

# --- RecommendedCSP <-> SDK sync (2026-10-25) ---------------------------
# csp.go's policy must keep shipping 'unsafe-eval' while the pinned SDK
# compiles data-* expressions with the Function constructor. If a future
# SDK drops the requirement, removing the directive is a dedicated change
# that updates this guard together with the pin comment above.

if ! grep -q "'unsafe-eval'" csp.go; then
	echo "::error::csp.go no longer ships 'unsafe-eval' — the pinned Datastar SDK ($expected_go_datastar) requires it (Function-constructor expression compiler); CSP and SDK pin have drifted"
	fail=1
else
	echo "OK  csp.go 'unsafe-eval' present (required by pinned SDK $expected_go_datastar)"
fi

if ! grep -q "Datastar SDK" csp.go; then
	echo "::error::csp.go lost the comment explaining WHY 'unsafe-eval' is required — restore the rationale; the CSP↔SDK sync check depends on it"
	fail=1
else
	echo "OK  csp.go unsafe-eval rationale comment present"
fi

exit "$fail"
