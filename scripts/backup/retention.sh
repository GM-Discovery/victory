#!/usr/bin/env bash
set -euo pipefail

# Kernel 77 §9.4: retention deletion is a separate, explicit command from
# backup.sh, with a dry-run default, a path guard, and minimum-age rules.
# backup.sh never deletes remote files; only this script does, and only
# when run with --apply.
#
# Usage:
#   scripts/backup/retention.sh [--apply]
#
# Defaults (override via env):
#   RETAIN_QUICK_HOURS=48    frequent database restore points: 48 hours
#   RETAIN_FULL_DAYS=30      daily complete backups: 30 days
#   RCLONE_REMOTE=victoryvtt-crypt:VictoryBackups

APPLY=0
if [[ "${1:-}" == "--apply" ]]; then
  APPLY=1
fi

RCLONE_REMOTE="${RCLONE_REMOTE:-victoryvtt-crypt:VictoryBackups}"
RETAIN_QUICK_HOURS="${RETAIN_QUICK_HOURS:-48}"
RETAIN_FULL_DAYS="${RETAIN_FULL_DAYS:-30}"

# Path guard: refuse to operate on anything that doesn't look like our own
# dedicated backup remote, so a mistyped RCLONE_REMOTE can't turn this into
# a delete-everything command.
if [[ "$RCLONE_REMOTE" != *"VictoryBackups"* ]]; then
  echo "refusing to run retention against unexpected remote: $RCLONE_REMOTE" >&2
  exit 2
fi

run_class() {
  local subdir="$1" min_age="$2"
  local remote="$RCLONE_REMOTE/$subdir/"

  echo "--- $subdir (older than $min_age) ---"
  local candidates
  candidates="$(rclone lsf "$remote" --min-age "$min_age" 2>/dev/null || true)"

  if [[ -z "$candidates" ]]; then
    echo "nothing older than $min_age in $subdir"
    return
  fi

  echo "$candidates"

  if [[ "$APPLY" -eq 1 ]]; then
    while IFS= read -r name; do
      [[ -z "$name" ]] && continue
      echo "deleting $remote$name"
      rclone deletefile "$remote$name"
    done <<< "$candidates"
  else
    echo "(dry run -- pass --apply to actually delete the above)"
  fi
}

run_class "quick" "${RETAIN_QUICK_HOURS}h"
run_class "full" "${RETAIN_FULL_DAYS}d"

echo "retention sweep complete (apply=$APPLY)"
