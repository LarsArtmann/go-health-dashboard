#!/usr/bin/env bash
# The release cut — mechanical ceremony for docs/release-checklist.md steps
# 1–5 (assess -> edit -> gate -> commit+tag -> tag-first push), replacing the
# hand-ceremony that cost whole sessions (2026-11-05 plan M04; the v0.12.0
# tag went red because gate 0 ran only BEFORE the release edits — the
# changelog lint reads the EDITED file, so this script re-runs the battery
# AFTER editing).
#
# Usage: bash scripts/release-cut.sh <version>   # e.g. v0.13.2
#
# Steps (abort on first failure; nothing pushes until all pass):
#   0. preconditions: version arg, clean tree, version > latest tag,
#      [Unreleased] carries real entries (a cut ships content, not air —
#      run scripts/release-assess.sh first for the verdict)
#   1. bump the Version const in dashboard.go
#   2. re-head [Unreleased] -> [version] - today, insert a fresh
#      [Unreleased] section, add the footer compare link
#   3. re-run the FULL pre-push battery (which includes gate 0, the
#      version-guard formula, and the changelog structural lint on the
#      EDITED file) — Go tooling, self-selected per repo convention
#   4. commit exactly the two edited files (never -A), annotated tag on
#      that commit
#   5. push TAG FIRST, then master (fleet release rule 3), then print the
#      manual tail: verify-release.sh + draft publish (the release-draft
#      workflow opens the Draft; publish with `gh release edit <ver>
#      --draft=false` after CI green + verify-release — the gh-fallback is
#      retired)
#
# NOT automated here, deliberately: the CI-green wait and the draft
# publish — both are remote-state decisions a human owns (release
# discipline: never claim unpushed/green state without checking).
set -euo pipefail

cd "$(dirname "$0")/.."

if [ $# -ne 1 ]; then
	echo "usage: $0 <version>   (e.g. v0.13.2)"
	exit 2
fi

VERSION="v${1#v}"
VERSION_N="${VERSION#v}"
TODAY=$(date +%F)

die() {
	echo "::error::$1"
	exit 1
}

# 0. Preconditions.
[ -z "$(git status --porcelain)" ] || die "working tree dirty — commit or stash first (a cut is a clean-tree operation)"
git rev-parse -q --verify "refs/tags/$VERSION" >/dev/null && die "tag $VERSION already exists"

latest_tag=$(git tag --sort=-v:refname | head -1)
if ! printf '%s\n%s\n' "${latest_tag#v}" "$VERSION_N" | sort -V | tail -1 | grep -qx "$VERSION_N"; then
	die "$VERSION is not greater than latest tag $latest_tag"
fi

unreleased_items=$(awk '/^## \[Unreleased\]/{f=1;next} /^## /{f=0} f' CHANGELOG.md | grep -cE '^- ' || true)
unreleased_nothing=$(awk '/^## \[Unreleased\]/{f=1;next} /^## /{f=0} f' CHANGELOG.md | grep -c 'Nothing yet\.' || true)
[ "$unreleased_items" -gt "$unreleased_nothing" ] || die "[Unreleased] holds only placeholders — run scripts/release-assess.sh; a cut ships content"

echo "== Cutting $VERSION (from $latest_tag, $TODAY)"

# 1. Version const bump.
const_line="const Version = \"$VERSION_N\""
if ! grep -qF "const Version = " dashboard.go; then
	die "Version const not found in dashboard.go"
fi
sed -i "s|const Version = \"[^\"]*\"|$const_line|" dashboard.go
grep -qF "$const_line" dashboard.go || die "const bump failed"
echo "   ..  const bumped: $const_line"

# 2. CHANGELOG re-head + fresh Unreleased + footer link. Idempotent: a
#    re-run after a battery failure (see step 3's rollback) finds the
#    version section already present and skips the transform.
if grep -qF "## [$VERSION_N] - " CHANGELOG.md; then
	echo "   ..  CHANGELOG already re-headed to [$VERSION_N] — resuming"
else
	old_section=$(grep -n '^## \[Unreleased\]' CHANGELOG.md | head -1 | cut -d: -f1)
	[ -n "$old_section" ] || die "[Unreleased] section not found in CHANGELOG.md"

	awk -v ver="$VERSION_N" -v today="$TODAY" '
		/^## \[Unreleased\]$/ && !done {
			print "## [Unreleased]"
			print ""
			print "### Added"
			print ""
			print "- Nothing yet."
			print ""
			print "### Changed"
			print ""
			print "- Nothing yet."
			print ""
			print "### Fixed"
			print ""
			print "- Nothing yet."
			print ""
			print "## [" ver "] - " today
			done=1
			next
		}
		{ print }
	' CHANGELOG.md >CHANGELOG.md.tmp && mv CHANGELOG.md.tmp CHANGELOG.md
fi

link_line="[$VERSION_N]: https://github.com/LarsArtmann/go-health-dashboard/compare/${latest_tag}...${VERSION}"
if grep -qF "$link_line" CHANGELOG.md; then
	: # already present (re-run after a partial failure)
else
	printf '%s\n' "$link_line" >>CHANGELOG.md
fi
grep -qF "$link_line" CHANGELOG.md || die "footer link missing after edit"
echo "   ..  CHANGELOG re-headed to [$VERSION_N] - $TODAY + fresh [Unreleased] + footer link"

# 3. Commit + local tag FIRST, then the full battery. Order matters:
#    the battery's version-guard check (const == latest tag) can only
#    pass once the tag exists, and the changelog structural lint must
#    read the EDITED file (the v0.12.0 lesson — gate 0 after the release
#    edits, not before). If the battery fails, roll back both the commit
#    (soft — the edits stay staged for fixing) and the local tag, then
#    abort; nothing was pushed.
git add dashboard.go CHANGELOG.md
if ! git commit -m "chore(release): cut $VERSION" \
	-m "Version const + CHANGELOG re-head + footer link; battery green follows before any push. Tag-first push; draft publishes manually after CI green + verify-release."; then
	die "commit failed"
fi

git tag -a "$VERSION" -m "$VERSION" || die "tag creation failed"
echo "   ..  committed $(git rev-parse --short HEAD) + tagged $VERSION (local)"

echo "   ..  running pre-push-checks.sh on the tagged tree (minutes; Go tooling)"
if ! bash scripts/pre-push-checks.sh; then
	git reset --soft HEAD~1
	git tag -d "$VERSION"
	echo "::error::pre-push battery red on the tagged tree — rolled back (edits stay staged, local tag removed); fix and re-run the cut"
	exit 1
fi

# 4. Tag-first push, then master.
git push origin "$VERSION" || die "TAG PUSH FAILED — master untouched; resolve remote state before any further push"
echo "   ..  tag pushed"
git push origin HEAD || die "master push failed (tag is out) — push master manually once resolved"
echo "   ..  master pushed"

# 5. Human tail.
cat <<RELEASE_NEXT

NEXT STEPS (human-owned, remote state):
  1. wait for CI on $(git rev-parse --short HEAD) — gh run watch (or gh run list)
  2. bash scripts/verify-release.sh $VERSION   # after CI green
  3. publish the auto-draft: gh release edit $VERSION --draft=false
RELEASE_NEXT
