#!/usr/bin/env bash
# UI-dependency pin guard.
#
# Fails unless the pinned UI dependency versions are exactly as audited.
# Rationale: templ-components v1.12.0 re-introduced the LiveRegion
# busy-script `nonce=""` regression (upstream templ-components#7), and
# go-datastar v0.5.0 swapped the audited SDK bundle — three CSP-invariant
# tests failed and the breakage landed silently twice on 2026-09-04.
# This guard makes the next sweep impossible to miss.
#
# REMOVAL CONDITION: delete this guard (and the go.mod pins) once
# templ-components ships the #7 nonce guard, the browser suite
# (TestBrowser_CSPCleanRuntime) validates the new templ-components and
# go-datastar bundles, and the pins are lifted in a dedicated change.
set -euo pipefail

expected_templ_components="v1.11.0"
expected_go_datastar="v0.4.0"

fail=0
check_pin() {
	local module="$1" expected="$2"
	actual=$(go list -m "$module" | awk '{print $2}')
	if [[ "$actual" == "$expected" ]]; then
		echo "OK  $module $actual"
	else
		echo "::error::$module is $actual, expected pinned $expected — see scripts/check-ui-pins.sh for the rationale and removal condition (upstream templ-components#7)"
		fail=1
	fi
}

check_pin github.com/larsartmann/templ-components "$expected_templ_components"
check_pin github.com/larsartmann/templ-components/datastar "$expected_templ_components"
check_pin github.com/larsartmann/go-datastar "$expected_go_datastar"
check_pin github.com/larsartmann/go-datastar/static "$expected_go_datastar"

exit "$fail"
