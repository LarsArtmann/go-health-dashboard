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
# CEREMONY RULE: if you bump any of these dependencies, update the pins
# in this file IN THE SAME CHANGE and re-run the browser suite. A bump
# without its guard update leaves CI red for every subsequent commit.
#
# REMOVAL CONDITION: the guard stays (decision 2026-09-09, user sign-off
# pending) — five unguarded sweeps say version movement needs a dedicated
# change that updates these pins and re-runs the browser suite.
set -euo pipefail

expected_templ_components="v1.16.0"
expected_templ_components_datastar="v1.16.0"
expected_templ_components_utils="v1.16.0"
expected_go_datastar="v0.5.0"

fail=0
check_pin() {
	local module="$1" expected="$2"
	actual=$(go list -m "$module" | awk '{print $2}')
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
check_pin github.com/larsartmann/go-datastar "$expected_go_datastar"
check_pin github.com/larsartmann/go-datastar/static "$expected_go_datastar"

exit "$fail"
