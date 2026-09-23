#!/usr/bin/env bash
# set-tag-protection.sh — rulesets-as-code for `protect-release-tags`.
#
# Reproduces the release-tag protection ruleset (docs/release-checklist.md
# §1) so it survives repo mishaps and replicates fleet-wide: target `tag`,
# include `refs/tags/v*`, rules `deletion` + `non_fast_forward`,
# enforcement `active`, bypass NEVER (break-glass = disabling the rule in
# the web UI, a deliberate, visible act).
#
# The three API quirks that cost three 422s when the rule was first
# created by hand (2026-09-18 session):
#   1. `target` must be `tag` — the rulesets API rejects the rule under
#      the branch target, even for valid tag patterns.
#   2. Pattern strings are plain `refs/tags/v*` — no `**` glob forms,
#      no bare `v*` (that silently matches nothing).
#   3. `exclude` must be a literal JSON array — omitting it (or sending
#      null) 422s; an empty array is required.
#
# Idempotent: resolves the ruleset id by NAME (ids are repo-local, so a
# fresh repo gets a create, an existing repo an update — same result).
#
# Usage:
#   bash scripts/set-tag-protection.sh --check   # read-only drift check
#   bash scripts/set-tag-protection.sh           # create or update (mutates!)
#
# Requires: gh CLI authenticated with repo admin (rulesets: write).
set -euo pipefail

cd "$(dirname "$0")/.."

ruleset_name="protect-release-tags"
mode="${1:-apply}"

command -v gh >/dev/null 2>&1 || {
	echo "::error::gh CLI is required for the rulesets API"
	exit 1
}

body=$(mktemp)

trap 'rm -f "$body"' EXIT

cat >"$body" <<'JSON'
{
  "name": "protect-release-tags",
  "target": "tag",
  "enforcement": "active",
  "conditions": {
    "ref_name": {
      "include": ["refs/tags/v*"],
      "exclude": []
    }
  },
  "bypass_actors": [],
  "rules": [
    { "type": "deletion" },
    { "type": "non_fast_forward" }
  ]
}
JSON

live_state() {
	gh api "repos/:owner/:repo/rulesets/$1" | jq -S '{
	  name: .name,
	  target: .target,
	  enforcement: .enforcement,
	  include: .conditions.ref_name.include,
	  exclude: .conditions.ref_name.exclude,
	  rules: [.rules[].type] | sort,
	  bypass_actors: (.bypass_actors | length)
	}'
}

# Ids are repo-local; resolve by name so the script is fleet-replicable.
ruleset_id=$(gh api repos/:owner/:repo/rulesets --jq ".[] | select(.name == \"${ruleset_name}\") | .id" | head -n1 || true)

if [ -n "$ruleset_id" ]; then
	if [ "$mode" = "--check" ]; then
		desired=$(echo '{"name":"protect-release-tags","target":"tag","enforcement":"active","include":["refs/tags/v*"],"exclude":[],"rules":["deletion","non_fast_forward"],"bypass_actors":0}' | jq -S .)
		if [ "$(live_state "$ruleset_id")" = "$desired" ]; then
			echo "OK  ruleset ${ruleset_name} (id ${ruleset_id}) matches the desired state"
			exit 0
		fi
		echo "::error::ruleset ${ruleset_name} (id ${ruleset_id}) has DRIFTED from the desired state — live:"
		live_state "$ruleset_id"
		echo "desired: $desired"
		exit 1
	fi
	gh api -X PUT "repos/:owner/:repo/rulesets/${ruleset_id}" --input "$body" --jq '{id: .id, name: .name, enforcement: .enforcement}' >/dev/null
	echo "OK  ruleset ${ruleset_name} (id ${ruleset_id}) updated to the desired state"
else
	if [ "$mode" = "--check" ]; then
		echo "::error::ruleset ${ruleset_name} is MISSING — run scripts/set-tag-protection.sh to create it"
		exit 1
	fi
	gh api -X POST repos/:owner/:repo/rulesets --input "$body" --jq '{id: .id, name: .name, enforcement: .enforcement}'
	echo "OK  ruleset ${ruleset_name} created"
fi
