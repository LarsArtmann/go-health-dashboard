#!/usr/bin/env bash
# CHANGELOG structural lint.
#
# The sandwiched-[Unreleased] bug (a ~100-line unreleased block stranded
# between [0.6.1] and [0.6.0] for days of doc audits) survived every human
# review; this check makes that class of drift a build failure. Encodes the
# conventions documented at the top of CHANGELOG.md:
#
#   1. exactly one `## [Unreleased]` section
#   2. `[Unreleased]` is the FIRST `## ` section (never below a release)
#   3. version sections appear in strictly descending semver order
#
# Run: bash scripts/check-changelog.sh   (wired into CI Build + pre-push-checks.sh)
set -euo pipefail

changelog="${1:-CHANGELOG.md}"
fail=0

# 1. Exactly one [Unreleased] section heading.
unreleased_count=$(grep -cE '^## \[Unreleased\]' "$changelog" || true)
if [ "$unreleased_count" -eq 1 ]; then
	echo "OK   exactly one [Unreleased] section"
else
	echo "::error::$changelog must contain exactly one '## [Unreleased]' section; found ${unreleased_count}. Merge extras into the top section — an [Unreleased] block may never sit sandwiched between released sections."
	fail=1
fi

# 2. [Unreleased] is the first ## section.
first_section=$(grep -nE '^## ' "$changelog" | head -1 || true)
if echo "$first_section" | grep -q '\[Unreleased\]'; then
	echo "OK   [Unreleased] is the first section"
else
	echo "::error::$changelog's first '## ' section must be [Unreleased]; found: ${first_section#*:}. Re-head or merge stranded unreleased content at tag time, never by forgetting."
	fail=1
fi

# 3. Version sections strictly descending semver order.
# Versions may carry pre-release/build suffixes (sort -V handles those);
# duplicates and out-of-order entries both fail.
version_order=$(grep -oE '^## \[[0-9]+\.[0-9]+\.[0-9]+[^]]*\]' "$changelog" | sed -E 's/^## \[(.*)\]$/\1/' || true)
if [ -z "$version_order" ]; then
	echo "::error::$changelog contains no version sections — is this the right file?"
	fail=1
else
	sorted=$(echo "$version_order" | sort -rV)
	if [ "$version_order" = "$sorted" ]; then
		echo "OK   $(echo "$version_order" | wc -l | tr -d ' ') version sections in descending order ($(echo "$version_order" | head -1) … $(echo "$version_order" | tail -1))"
	else
		echo "::error::$changelog version sections are out of order. File order:" >&2
		echo "$version_order" | sed 's/^/       /' >&2
		echo "     Expected (descending):" >&2
		echo "$sorted" | sed 's/^/       /' >&2
		fail=1
	fi
	if echo "$version_order" | sort | uniq -d | grep -q .; then
		echo "::error::$changelog has duplicate version sections: $(echo "$version_order" | sort | uniq -d | tr '\n' ' ')"
		fail=1
	else
		echo "OK   no duplicate version sections"
	fi
fi

if [ "$fail" -ne 0 ]; then
	echo "changelog lint: FAILED"
	exit 1
fi
echo "changelog lint: all green"
