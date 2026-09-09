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
# Re-audited 2026-09-09 on templ-components v1.13.x + go-datastar v0.5.0:
#   - the #7 nonce guard is present (datastar@v1.13.2 liveRegionBusyScriptAttrs
#     emits no nonce attribute when the nonce is empty),
#   - the swapped v0.5.0 SDK bundle passed the full browser suite under
#     strict CSP (CSPCleanRuntime, LiveSSEPatch, MetricsUnderStrictCSP,
#     AggregateCSPClean),
#   - the deliberate v1.13.1+ constant rename (DatastarVersion1_0_3) requires
#     templ-components/datastar v1.13.2.
# Known remaining upstream issue: StatCard figure markup nests <dd> inside
# <div> levels a <dl> may not contain (templ-components#6) — tolerated
# narrowly in TestBrowser_Accessibility until upstream fixes it.
#
# REMOVAL CONDITION: delete this guard once the sweep pattern stops
# recurring AND a future browser-suite ceremony re-validates whatever the
# current dependency set is; until then, ANY version movement must be a
# dedicated change that updates these pins and re-runs the browser suite.
set -euo pipefail

expected_templ_components="v1.13.0"
expected_templ_components_datastar="v1.13.2"
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
check_pin github.com/larsartmann/go-datastar "$expected_go_datastar"
check_pin github.com/larsartmann/go-datastar/static "$expected_go_datastar"

exit "$fail"
