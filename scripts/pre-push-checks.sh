#!/usr/bin/env bash
# Local session-closing gate — encodes the exact drift-guard formulas CI
# runs, so a red job is discovered at the desk instead of after a push:
#
#   1. FEATURES.md test-suite counts vs the actual *_test.go files
#   2. Version const vs the latest git tag (version-guard job)
#   3. UI dependency pins (scripts/check-ui-pins.sh)
#   4. CHANGELOG structural lint (scripts/check-changelog.sh)
#
# Run: bash scripts/pre-push-checks.sh
# Exits non-zero on the first failure with the same wording CI prints.
set -euo pipefail

cd "$(dirname "$0")/.."

fail=0

# 1. FEATURES test-count drift guard (same formula as ci.yml).
actual_funcs=$(cat ./*_test.go | grep -cE '^func (Test|Benchmark|Fuzz)' || true)
set -- ./*_test.go
actual_files=$#
expected_funcs=$(grep -oE '[0-9]+ top-level test/benchmark/fuzz functions' FEATURES.md | grep -oE '^[0-9]+')
expected_files=$(grep -oE 'across [0-9]+ test files' FEATURES.md | grep -oE '[0-9]+')

echo "FEATURES claims ${expected_funcs} functions across ${expected_files} files; actual ${actual_funcs} across ${actual_files}"
if [ "$actual_funcs" != "$expected_funcs" ] || [ "$actual_files" != "$expected_files" ]; then
	echo "::error::FEATURES.md test-suite counts drifted from reality — recount and update the Test suite row"
	fail=1
fi

# 2. Version const vs latest tag (version-guard job). A mismatch is only
# correct in the one commit that tags; the guard exists so that state
# never lingers.
version_const=$(grep -oE 'Version = "[^"]+"' dashboard.go | head -1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')
latest_tag=$(git tag --sort=-v:refname | head -1 | sed 's/^v//')

echo "Version const ${version_const}; latest tag v${latest_tag}"
if [ "$version_const" != "$latest_tag" ]; then
	echo "::error::Version const (${version_const}) does not match the latest tag (v${latest_tag}) — bump the const in the same commit as the tag, or tag the released const"
	fail=1
fi

# 3. UI dependency pins.
if ! bash scripts/check-ui-pins.sh; then
	fail=1
fi

# 4. CHANGELOG structural lint (same script CI runs).
if ! bash scripts/check-changelog.sh; then
	fail=1
fi

if [ "$fail" -ne 0 ]; then
	echo "pre-push checks: FAILED"
	exit 1
fi

echo "pre-push checks: all green"
