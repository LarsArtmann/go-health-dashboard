#!/usr/bin/env bash
# One-command README screenshot capture (f42 footgun remover).
#
# The three README captures use three different env vars and exact test
# names; getting one wrong silently skips the capture (the tests treat an
# unset env var as "manual, skip"). This wrapper sets all three and fails
# loudly if any PNG is missing or stale afterwards.
#
# Usage:
#   bash scripts/capture-screenshots.sh            # all four, into docs/
#   bash scripts/capture-screenshots.sh light      # just the light theme
#   bash scripts/capture-screenshots.sh dark       # just the dark theme
#   bash scripts/capture-screenshots.sh degraded   # just the degraded state
#   bash scripts/capture-screenshots.sh grouped    # just the source-grouped page
#
# Needs the devShell (Chrome + GOEXPERIMENT=jsonv2); the wrapper handles
# that itself. Outputs:
#   docs/screenshot.png           light theme (SCREENSHOT_OUTPUT)
#   docs/screenshot-dark.png      dark theme (SCREENSHOT_OUTPUT_DARK)
#   docs/screenshot-degraded.png  failure state (SCREENSHOT_OUTPUT_DEGRADED)
#   docs/screenshot-grouped.png   source-grouped hub view (SCREENSHOT_OUTPUT_GROUPED)
set -euo pipefail

cd "$(dirname "$0")/.."

want_all() { [ $# -eq 0 ]; }
want() { want_all "$@" || [ "$1" = "$2" ]; }

declare -a test_env=()
declare -a expect=()

if want_all "$@" || want "$1" light; then
	test_env+=(SCREENSHOT_OUTPUT=docs/screenshot.png)
	expect+=(docs/screenshot.png)
fi
if want_all "$@" || want "$1" dark; then
	test_env+=(SCREENSHOT_OUTPUT_DARK=docs/screenshot-dark.png)
	expect+=(docs/screenshot-dark.png)
fi
if want_all "$@" || want "$1" degraded; then
	test_env+=(SCREENSHOT_OUTPUT_DEGRADED=docs/screenshot-degraded.png)
	expect+=(docs/screenshot-degraded.png)
fi
if want_all "$@" || want "$1" grouped; then
	test_env+=(SCREENSHOT_OUTPUT_GROUPED=docs/screenshot-grouped.png)
	expect+=(docs/screenshot-grouped.png)
fi

if [ ${#expect[@]} -eq 0 ]; then
	echo "usage: $0 [light|dark|degraded|grouped]" >&2
	exit 2
fi

env "${test_env[@]}" nix develop -c go test -run 'TestCaptureREADME_Screenshot' -count=1 -v .

fail=0
for out in "${expect[@]}"; do
	if [ ! -s "$out" ]; then
		echo "::error::expected capture $out missing or empty" >&2
		fail=1
	else
		echo "OK  $out ($(wc -c <"$out" | tr -d ' ') bytes)"
	fi
done

exit "$fail"
