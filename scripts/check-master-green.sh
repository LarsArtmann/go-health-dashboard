#!/usr/bin/env bash
# scripts/check-master-green.sh — turn a red master into an open issue.
#
# Runs from .github/workflows/master-red-alert.yml on every completed CI
# run of master (fast path) and as a nightly sweep (backstop). The
# 2026-10-14 → 2026-10-25 lesson: master CI sat red for 11 days with
# nobody noticing because nothing alerts on a failing default branch.
#
# Two detectors, one tracking issue per kind:
#   - RED: the latest completed master run of a watched workflow failed.
#   - STALE (schedule/dispatch only): no successful master CI run within
#     GREEN_RECENCY_DAYS — CI may be silently dead (billing, runner,
#     broken trigger), which the red detector alone cannot see.
#
# Secretless: authenticates with the workflow's GITHUB_TOKEN (issues:
# write); paging happens via GitHub notifications on the issue.
#
# Fleet precedent: templ-components scripts/check-master-green.sh (the
# v1.11.0 nine-day lesson) + go-cqrs-lite sentinel.yml (green recency).
#
# Requires: gh (authenticated via GH_TOKEN), jq.

set -euo pipefail

REPO="${GITHUB_REPOSITORY:-}"
if [ -z "$REPO" ]; then
	REPO="$(gh repo view --json nameWithOwner --jq .nameWithOwner)"
fi

# The watched list is single-sourced in .github/watched-workflows.txt (the
# drift guard scripts/check-watched-workflows.sh keeps it complete); do not
# add workflow names here directly.
WORKFLOWS=()
while IFS= read -r line; do
	WORKFLOWS+=("$line")
done < <(sed -e 's/#.*//' -e '/^[[:space:]]*$/d' "${GITHUB_WATCHED_WORKFLOWS:-.github/watched-workflows.txt}")
GREEN_RECENCY_DAYS=3

declare -a RED_WORKFLOW=()
declare -a RED_STALE=()

for wf in "${WORKFLOWS[@]}"; do
	json="$(gh run list --repo "$REPO" --workflow "$wf" --branch master \
		--status completed --limit 1 --json name,conclusion,createdAt,url 2>/dev/null || echo '[]')"
	conclusion="$(echo "$json" | jq -r '.[0].conclusion // ""')"
	if [ "$conclusion" = "failure" ]; then
		RED_WORKFLOW+=("$(echo "$json" | jq -r '.[0] | "- \(.name): latest master run failed at \(.createdAt) — \(.url)"')")
	fi
done

# Stale-green detector: only meaningful on schedule/dispatch, where a
# missing or dead trigger is exactly the failure mode to catch. After a
# push-triggered invocation the green run is fresh by construction.
EVENT="${GITHUB_EVENT_NAME:-}"
if [ "$EVENT" = "schedule" ] || [ "$EVENT" = "workflow_dispatch" ]; then
	latest_green="$(gh api \
		"repos/$REPO/actions/runs?branch=master&event=push&status=success&per_page=1" \
		--jq '.workflow_runs[0].created_at' 2>/dev/null || echo '')"
	if [ -z "$latest_green" ] || [ "$latest_green" = "null" ]; then
		RED_STALE+=("- no successful CI run found on master at all (CI may be dead)")
	else
		age_days=$((($(date +%s) - $(date -d "$latest_green" +%s)) / 86400))
		if [ "$age_days" -ge "$GREEN_RECENCY_DAYS" ]; then
			RED_STALE+=("- last successful master CI run is $age_days day(s) old ($latest_green) — CI may be silently dead")
		fi
	fi
fi

if [ "${#RED_WORKFLOW[@]}" -eq 0 ] && [ "${#RED_STALE[@]}" -eq 0 ]; then
	echo "master-green: all watched workflows are green and recent. Nothing to do."
	exit 0
fi

if [ "${#RED_WORKFLOW[@]}" -gt 0 ]; then
	TITLE="master CI is red"
	declare -a FINDINGS=("${RED_WORKFLOW[@]}" "${RED_STALE[@]}")
else
	TITLE="master CI has not been green recently"
	declare -a FINDINGS=("${RED_STALE[@]}")
fi

BODY="Automated master health check $(date -u '+%Y-%m-%dT%H:%M:%SZ'):

$(printf '%s\n' "${FINDINGS[@]}")

Opened by the master-red-alert workflow. Close once master is green again."

existing="$(gh issue list --repo "$REPO" --state open --json number,title \
	--jq "map(select(.title == \"$TITLE\")) | .[0].number // empty")"

if [ -n "$existing" ]; then
	echo "master-red: updating existing tracking issue #$existing"
	gh issue comment "$existing" --repo "$REPO" --body "$BODY"
else
	echo "master-red: opening new tracking issue"
	gh issue create --repo "$REPO" --title "$TITLE" --body "$BODY"
fi
