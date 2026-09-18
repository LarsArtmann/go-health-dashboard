#!/usr/bin/env bash
# verify-dep-bump.sh — the full gate chain for a dependency bump.
#
# The go-health v0.2.0 bump ran build+test+race+lint ad-hoc and skipped
# vulncheck/coverage/bench; this script is the fixed checklist so a bump
# cannot skip gates by forgetting them. Placement note: a fleet-level
# shared version in BuildFlow is an open question — this is the per-repo
# answer, mirroring scripts/verify-release.sh (see TODO_LIST/ROADMAP).
#
# Usage: run from the repo root, OUTSIDE the devShell (the nix apps set
# up their own environment):
#
#   bash scripts/verify-dep-bump.sh
#
# The CHANGELOG entry for the bump is authored by hand (the lint guard
# below checks structure, not content).

set -u

fail=0

run() {
	name="$1"
	shift
	echo "=== ${name} ==="
	if ! "$@"; then
		echo "!! FAILED: ${name}"
		fail=1
	fi
	echo
}

run "build (templ generate + go build)" nix run .#build
run "tests" nix run .#test
run "tests with race detector" nix run .#test-race
run "lint" nix run .#lint
run "vet" nix run .#vet
run "vulncheck" nix run .#vulncheck
run "coverage (CI enforces the 80% floor)" nix run .#coverage
total=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | tr -d '%')
echo "   coverage total: ${total}% (floor 80%)"
if ! awk -v t="$total" 'BEGIN { exit !(t >= 80) }'; then
	echo "!! FAILED: coverage floor (${total}% < 80%) — a bump that drops coverage needs tests, not a lower bar"
	fail=1
fi
trash-put coverage.out 2>/dev/null || true
if [ -e coverage.out ]; then
	echo "WARN: coverage.out left behind (trash-cli unavailable) — run: nix run .#clean"
fi
echo
run "bench smoke (all benchmarks, 1 iteration each)" nix develop -c go test -run xxx -bench . -benchtime 1x .
run "UI pin guard" bash scripts/check-ui-pins.sh
run "changelog lint" bash scripts/check-changelog.sh

echo "=== formatting (canonical order: generate already ran via build) ==="
nix fmt
if ! git diff --exit-code --quiet; then
	echo "!! nix fmt left changes — commit the formatted tree"
	fail=1
fi
echo

if [ "$fail" -eq 0 ]; then
	echo "verify-dep-bump: ALL GREEN"
else
	echo "verify-dep-bump: FAILURES (see above)"
fi
exit "$fail"
