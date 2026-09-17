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
run "coverage (CI enforces the 78% floor)" nix run .#coverage
run "bench smoke" nix develop -c go test -run xxx -bench BenchmarkHandler_HTMLRendering -benchtime 1x .
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
