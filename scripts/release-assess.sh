#!/usr/bin/env bash
# Phase-0 release triage — answers "is a release warranted?" mechanically
# instead of hand-rolling the assessment each release session (three
# sessions did exactly that before this script existed; 2026-11-05 plan M04).
#
# Usage: bash scripts/release-assess.sh
#
# Reads, in order:
#   1. latest tag (v:refname) + its date
#   2. the delta range <tag>..HEAD, classified by conventional prefix:
#      feat/fix/perf are consumer-facing; chore/docs/test/ci are not
#   3. whether templ-GENERATED files carry drift with no source behind it
#      (the "templ files net-zero vs tag" trap from the 2026-11-05 verdict)
#   4. the [Unreleased] CHANGELOG state (real entries vs "Nothing yet.")
#   5. last master CI conclusion + release/draft state (gh; skipped offline)
#
# Verdict (printed as a table, exit 0 = cut warranted, exit 1 = not):
#   - any feat/fix/perf in the delta  -> CUT (suggested version per bump rule)
#   - chore/docs-only delta           -> NO CUT (standing default: chores
#     batch into the next consumer-facing cut; revisit via plan g1/M26)
#   - empty delta                     -> NO CUT
#
# Read-only: no tree mutation, no pushes, no network requirement (steps
# that need gh degrade to a warning when it is unavailable).
#
# Companion: scripts/release-cut.sh performs the cut this script proposes.
set -euo pipefail

cd "$(dirname "$0")/.."

ok() { echo "   OK  $1"; }
warn() { echo "   WARN $1"; }
info() { echo "   ..  $1"; }

echo "== Release assessment (phase 0 triage)"

# 1. Latest tag + date.
tag=$(git tag --sort=-v:refname | head -1)
if [ -z "$tag" ]; then
	echo "   FAIL no tags found — cannot triage"
	exit 2
fi
tag_date=$(git log -1 --format=%cs "$tag")
info "latest tag: $tag ($tag_date)"

# 2. Delta + classification.
range="$tag..HEAD"
delta=$(git log --format=%s "$range")
if [ -z "$delta" ]; then
	echo "   OK  no commits since $tag"
	echo
	echo "VERDICT: NO CUT — zero delta"
	exit 1
fi

commit_count=$(git rev-list --count "$range")
info "delta: $commit_count commits in $range"

facing=""
internal=""
while IFS= read -r subject; do
	case "$subject" in
	feat* | fix* | perf*) facing="${facing}${subject}\n" ;;
	*) internal="${internal}${subject}\n" ;;
	esac
done < <(git log --format=%s "$range")

facing_n=$(printf '%b' "$facing" | grep -c . || true)
internal_n=$(printf '%b' "$internal" | grep -c . || true)

echo
echo "   consumer-facing ($facing_n):"
[ -n "$facing" ] && printf '%b' "$facing" | sed 's/^/     /'
echo "   internal ($internal_n):"
[ -n "$internal" ] && printf '%b' "$internal" | sed 's/^/     /'

# 3. Generated-file drift with no committed source behind it: if only
# *_templ.go files changed in the range, the "delta" is generator shape,
# not product change (2026-11-05 verdict evidence).
changed=$(git diff --name-only "$tag" HEAD || true)
src_changed=$(printf '%s\n' "$changed" | grep -v '_templ\.go$' | grep -c . || true)
templ_changed=$(printf '%s\n' "$changed" | grep -c '_templ\.go$' || true)
if [ "$templ_changed" -gt 0 ] && [ "$src_changed" -eq 0 ]; then
	warn "$templ_changed templ-generated files changed with NO source change — generator-shape drift only"
else
	ok "source changes present: $src_changed files ($templ_changed generated-only)"
fi

# 4. [Unreleased] state: entries beyond the three "Nothing yet." placeholders.
unreleased=$(awk '/^## \[Unreleased\]/{f=1;next} /^## /{f=0} f' CHANGELOG.md)
unreleased_items=$(printf '%s\n' "$unreleased" | grep -cE '^- ' || true)
unreleased_nothing=$(printf '%s\n' "$unreleased" | grep -c 'Nothing yet\.' || true)
if [ "$unreleased_items" -gt "$unreleased_nothing" ]; then
	ok "[Unreleased] carries $((unreleased_items - unreleased_nothing)) real entries"
else
	warn "[Unreleased] holds only placeholders"
fi

# 5. Remote state (best-effort — offline triage stays useful).
if command -v gh > /dev/null 2>&1; then
	last_run=$(gh run list --branch master --limit 1 --json conclusion,displayTitle --jq '.[0] | "\(.conclusion): \(.displayTitle)"' 2> /dev/null || true)
	if [ -n "$last_run" ]; then
		info "last master run: $last_run"
		case "$last_run" in
		success*) ok "master CI green" ;;
		*) warn "master CI NOT green — resolve before cutting (11-day red window precedent)" ;;
		esac
	else
		warn "gh run list returned nothing (auth or offline?)"
	fi

	latest_release=$(gh release list --limit 5 --json tagName,isDraft,isLatest --jq '.[] | select(.isLatest) | "\(.tagName) draft=\(.isDraft)"' 2> /dev/null || true)
	info "latest GitHub release: ${latest_release:-none visible}"
else
	warn "gh not available — remote state unchecked"
fi

# Verdict + suggested version (0.x convention: feat/perf -> minor, fix -> patch).
echo
if [ "$facing_n" -gt 0 ]; then
	if printf '%s' "$facing" | grep -qE '^(feat|perf)(\(|:|!)'; then
		suggested=$(printf '%s' "$tag" | awk -F. '{printf "v%d.%d.0", $1, $2+1}')
		bump="minor (feat/perf present)"
	else
		suggested=$(printf '%s' "$tag" | awk -F. '{printf "v%d.%d.%d", $1, $2, $3+1}')
		bump="patch (fix-only)"
	fi
	echo "VERDICT: CUT WARRANTED — $bump; suggested $suggested (confirm the level yourself)"
	exit 0
fi

echo "VERDICT: NO CUT — chore/docs-only delta ($internal_n commits); batch into the next consumer-facing cut (standing default, plan g1/M26 pending)"
exit 1
