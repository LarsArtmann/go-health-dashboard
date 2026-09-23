#!/usr/bin/env bash
# verify-dep-bump.sh — the full gate chain for a dependency bump.
#
# The go-health v0.2.0 bump ran build+test+race+lint ad-hoc and skipped
# vulncheck/coverage/bench; this script is the fixed checklist so a bump
# cannot skip gates by forgetting them. Placement note: a fleet-level
# shared version in BuildFlow is an open question — this is the per-repo
# answer, mirroring scripts/verify-release.sh (see TODO_LIST/ROADMAP).
#
# Usage: run from the repo root, inside OR outside the devShell — every
# direct `go` invocation goes through `nix develop -c`, so the script
# self-provisions the correct toolchain (the ambient PATH go is the wrong
# version and dies on go.mod's floor):
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
# CI is the single source for the coverage floor: parse it out of the
# ci.yml coverage step instead of hardcoding a second, drift-ready copy.
floor=$(grep -oE 't >= [0-9]+' .github/workflows/ci.yml | head -1 | grep -oE '[0-9]+')
if [ -z "$floor" ]; then
	echo "!! FAILED: could not parse the coverage floor from .github/workflows/ci.yml (expected: awk -v t=... 'BEGIN { exit !(t >= <N>) }')"
	fail=1
	floor=80
fi
run "coverage (CI enforces the ${floor}% floor)" nix run .#coverage
total=$(nix develop -c go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | tr -d '%')
echo "   coverage total: ${total}% (floor ${floor}%)"
if ! awk -v t="$total" -v f="$floor" 'BEGIN { exit !(t >= f) }'; then
	echo "!! FAILED: coverage floor (${total}% < ${floor}%) — a bump that drops coverage needs tests, not a lower bar"
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
# The auto-daemon can commit the fmt output between nix fmt and this
# check, making a clean tree look dirty (AGENTS.md "Gates race the
# auto-daemon"). Retry once before trusting a failure.
fmt_dirty=0
git diff --exit-code --quiet || fmt_dirty=1
if [ "$fmt_dirty" -ne 0 ]; then
	sleep 3
	if git diff --exit-code --quiet; then
		echo "   dirty-tree check was a daemon race; clean on retry"
		fmt_dirty=0
	fi
fi
if [ "$fmt_dirty" -ne 0 ]; then
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
