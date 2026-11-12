#!/usr/bin/env bash
# Markdown lint, two passes (M21, 2026-11-12).
#
# markdownlint-cli resolves config from the INVOCATION directory only, not
# per-file, so the archived-report relaxation (docs/status/archived and
# docs/planning/archived carry their own .markdownlint.json — those reports
# are annotate-only under the docs-health doctrine) must run as a separate
# pass with -c. Live docs take the root .markdownlint.jsonc conventions:
# dprint owns wrapping (MD013 off), Keep-a-Changelog repeats section names
# (MD024 siblings_only), historical numbering is load-bearing (MD029 off).
set -euo pipefail

cd "$(dirname "$0")/.."

if ! command -v markdownlint >/dev/null 2>&1; then
	echo "markdownlint not on PATH — run inside 'nix develop' (devShell ships markdownlint-cli)"
	exit 2
fi

rc=0

live_files=$(git ls-files '*.md' | grep -vE '^docs/(status|planning)/archived/' || true)
if [ -n "$live_files" ]; then
	# shellcheck disable=SC2086 # word splitting is the point: one file per argv
	if ! markdownlint -c .markdownlint.jsonc $live_files; then
		rc=1
	fi
fi

if ! markdownlint -c docs/status/archived/.markdownlint.json \
	docs/status/archived docs/planning/archived; then
	rc=1
fi

if [ "$rc" -eq 0 ]; then
	echo "markdown lint: all green"
fi

exit "$rc"
