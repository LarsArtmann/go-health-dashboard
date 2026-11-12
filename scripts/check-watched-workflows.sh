#!/usr/bin/env bash
# scripts/check-watched-workflows.sh — drift guard for the watchdog's
# watched list.
#
# .github/watched-workflows.txt is THE single source of what
# scripts/check-master-green.sh watches. This guard keeps it complete
# both ways:
#   1. Every workflow that triggers on a schedule or on master pushes
#      must appear in the watched list — a new scheduled workflow that
#      isn't watched would red silently, the exact 11-day failure mode
#      the watchdog exists to prevent.
#   2. Every watched entry must name a real workflow — a renamed or
#      deleted workflow must not leave a stale watch (gh run list against
#      a dead name returns nothing, which reads as "green" forever).
#
# Run locally (same drift guard CI Hygiene runs) and from
# scripts/pre-push-checks.sh gate 12.

set -euo pipefail

cd "$(dirname "$0")/.."

watched_file=".github/watched-workflows.txt"
if [ ! -f "$watched_file" ]; then
	echo "::error::missing $watched_file — the watchdog's single source is gone"
	exit 1
fi

# shellcheck disable=SC2016 # sed program is literal
mapfile -t watched < <(sed -e 's/#.*//' -e '/^[[:space:]]*$/d' "$watched_file" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
if [ "${#watched[@]}" -eq 0 ]; then
	echo "::error::$watched_file is empty — nothing is watched"
	exit 1
fi

# Collect name + trigger kind per workflow file. YAML is flat enough at the
# top of these files for a grep-level parse; a full YAML parser is not
# available in the CI runner.
declare -A should_watch=()
declare -A real_names=()
name_seen_at=()
for f in .github/workflows/*.yml; do
	name="$(sed -n 's/^name:[[:space:]]*//p' "$f" | head -1)"
	if [ -z "$name" ]; then
		echo "::error::$f has no name: field"
		exit 1
	fi
	if [ -n "${real_names[$name]:-}" ]; then
		echo "::error::two workflow files declare the name \"${name}\" (${real_names[$name]} and $f) — the watchdog would watch one entry for two run streams and neither guard view could tell them apart"
		exit_code=1
	fi
	real_names["$name"]="$f"
	name_seen_at+=("$name")

	trigger=""
	if grep -q '^on:' "$f" && awk '/^on:/{on=1;next} on&&/^[^ \t#]/{on=0} on' "$f" | grep -qE 'schedule:|branches: \[master\]|branches: \[master,|branches: \[.*, master\]'; then
		trigger="master-or-scheduled"
	fi
	if [ "$trigger" = "master-or-scheduled" ]; then
		should_watch["$name"]="$f"
	fi
done

exit_code=0

# Duplicate watched entries: the list is a set; a duplicate means someone
# appended without reading — the drift guard treats it as drift too.
for w in "${watched[@]}"; do
	count=0
	for other in "${watched[@]}"; do
		if [ "$other" = "$w" ]; then
			count=$((count + 1))
		fi
	done
	if [ "$count" -gt 1 ]; then
		echo "::error::$watched_file lists \"${w}\" ${count} times — the watched list is a set"
		exit_code=1
	fi
done

for name in "${!should_watch[@]}"; do
	found=""
	for w in "${watched[@]}"; do
		if [ "$w" = "$name" ]; then
			found="$w"
			break
		fi
	done
	if [ -z "$found" ]; then
		echo "::error::workflow \"${name}\" (${should_watch[$name]}) triggers on master/schedule but is NOT in $watched_file — it would red silently"
		exit_code=1
	fi
done

for w in "${watched[@]}"; do
	if [ -z "${real_names[$w]:-}" ]; then
		echo "::error::$watched_file watches \"${w}\" but no workflow file declares that name — stale watch"
		exit_code=1
	fi
done

if [ "$exit_code" -eq 0 ]; then
	echo "watched-workflows guard: ${#watched[@]} watched, ${#should_watch[@]} master/scheduled — consistent"
fi

exit "$exit_code"
