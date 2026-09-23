#!/usr/bin/env bash
# Local session-closing gate — encodes the exact drift-guard formulas CI
# runs, so a red job is discovered at the desk instead of after a push:
#
#   1. FEATURES.md test-suite counts vs the actual *_test.go files
#   2. Version const vs the latest git tag (version-guard job)
#   3. UI dependency pins (scripts/check-ui-pins.sh)
#   4. CHANGELOG structural lint (scripts/check-changelog.sh)
#   5. Fuzz-target registry: fuzz.yml steps vs `func Fuzz*` definitions
#   6. templ generator pin: no bare `templ generate` / `pkgs.templ`
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

# 5. Fuzz-target registry (AGENTS.md: "New fuzz target ⇒ fuzz.yml + this
# file updated in the SAME change"). Every `func FuzzX` needs exactly one
# `-fuzz` step in .github/workflows/fuzz.yml, and the step's package
# argument must point at the file that defines the target.
code_root=$(cat ./*_test.go | grep -hoE '^func Fuzz[A-Za-z0-9_]+' | sed -E 's/^func (Fuzz[A-Za-z0-9_]+).*/\1/' | sed '/^$/d' | sort)
code_hub=$(cat cmd/health-hub/*_test.go 2>/dev/null | grep -hoE '^func Fuzz[A-Za-z0-9_]+' | sed -E 's/^func (Fuzz[A-Za-z0-9_]+).*/\1/' | sed '/^$/d' | sort)
code_all=$({ printf '%s\n' "$code_root" "$code_hub"; } | sed '/^$/d' | sort)
wf_all=$(grep -E -- '-fuzz Fuzz' .github/workflows/fuzz.yml | sed -E 's/.*-fuzz[[:space:]]+(Fuzz[A-Za-z0-9_]+).*/\1/' | sed '/^$/d' | sort)

echo "fuzz registry: $(printf '%s\n' "$code_all" | wc -l) code targets, $(printf '%s\n' "$wf_all" | wc -l) workflow steps"
if ! diff <(printf '%s\n' "$code_all") <(printf '%s\n' "$wf_all") >/dev/null; then
	echo "::error::fuzz.yml steps drifted from the Fuzz targets defined in the test files — every target needs exactly one -fuzz step (diff: <code> vs <fuzz.yml>)"
	diff <(printf '%s\n' "$code_all") <(printf '%s\n' "$wf_all") || true
	fail=1
fi
while IFS= read -r line; do
	[ -z "$line" ] && continue
	target=$(printf '%s\n' "$line" | sed -E 's/.*-fuzz[[:space:]]+(Fuzz[A-Za-z0-9_]+).*/\1/')
	pkg=$(printf '%s\n' "$line" | awk '{print $NF}')
	case "$pkg" in
	./cmd/health-hub)
		if ! printf '%s\n' "$code_hub" | grep -qx "$target"; then
			echo "::error::fuzz step for ${target} runs in ./cmd/health-hub but the target is not defined in cmd/health-hub/fuzz_test.go"
			fail=1
		fi
		;;
	.)
		if ! printf '%s\n' "$code_root" | grep -qx "$target"; then
			echo "::error::fuzz step for ${target} runs in the root package but the target is not defined in a root *_test.go"
			fail=1
		fi
		;;
	*)
		echo "::error::fuzz step for ${target} has an unrecognized package argument '${pkg}' (expected '.' or './cmd/health-hub')"
		fail=1
		;;
	esac
done < <(grep -E -- '-fuzz Fuzz' .github/workflows/fuzz.yml)

# 6. templ generator pin: go.mod's `tool` directive is the only sanctioned
# generator path, so its version structurally cannot diverge from the
# module's requirement (AGENTS.md "The templ generator is pinned..."). A
# bare `templ generate` (whatever binary is on PATH) or `pkgs.templ`
# (nixpkgs' unpinned attr) reintroduces the split-brain.
templ_violations=$(
	grep -rnE '(^|[[:space:]]|[;&|`]|\$\(| && | \|\| )templ generate' flake.nix .github/workflows scripts 2>/dev/null |
		grep -v 'go tool templ generate' |
		grep -vE ':[0-9]+:[[:space:]]*(#|- )' || true
)
pkgs_templ=$(grep -rnE 'pkgs\.templ([^[:alnum:]_-]|$)' flake.nix .github/workflows scripts 2>/dev/null | grep -vE ':[0-9]+:[[:space:]]*#' || true)
if [ -n "$templ_violations" ]; then
	echo "::error::bare 'templ generate' invocation outside the go.mod tool pin — use 'go tool templ generate':"
	echo "$templ_violations"
	fail=1
fi
if [ -n "$pkgs_templ" ]; then
	echo "::error::unpinned nixpkgs templ attr found — the generator must come from go.mod's tool directive (go tool templ generate):"
	echo "$pkgs_templ"
	fail=1
fi

if [ "$fail" -ne 0 ]; then
	echo "pre-push checks: FAILED"
	exit 1
fi

echo "pre-push checks: all green"
