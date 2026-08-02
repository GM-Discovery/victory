#!/usr/bin/env bash
set -euo pipefail

# Kernel 77 Goal C: encrypted off-host backup.
#
# Usage: scripts/backup/backup.sh [quick|full]
#   quick — database only. Meets the 15-minute recovery-point-objective
#           target on its own timer.
#   full  — database + uploaded assets + manifest + checksums. Runs once
#           daily.
#
# Never uses `rclone sync` (which can delete remote files); always `rclone
# copy`, so a broken run cannot destroy prior good backups. Retention
# deletion is a separate, explicit script (retention.sh) with a dry-run
# mode — this script never deletes anything remotely.

MODE="${1:-full}"
if [[ "$MODE" != "quick" && "$MODE" != "full" ]]; then
  echo "usage: $0 [quick|full]" >&2
  exit 2
fi

ROOT="/opt/victory"
STAGING_ROOT="${BACKUP_STAGING_ROOT:-$ROOT/backups/scheduled}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-victory-postgres}"
POSTGRES_USER="${POSTGRES_USER:-victory}"
POSTGRES_DB="${POSTGRES_DB:-victory}"
STORAGE_ROOT="${STORAGE_ROOT:-$ROOT/storage}"
RCLONE_REMOTE="${RCLONE_REMOTE:-victoryvtt-crypt:VictoryBackups}"
LOCK_FILE="$ROOT/backups/.backup.lock"
STATUS_FILE="$ROOT/backups/backup-status.json"
LOG_DIR="$ROOT/backups/logs"
LOCAL_RETENTION_COUNT="${LOCAL_RETENTION_COUNT:-5}"

mkdir -p "$STAGING_ROOT" "$LOG_DIR" "$(dirname "$STATUS_FILE")"

exec 9>"$LOCK_FILE"
if ! flock -n 9; then
  echo "backup already running (lock held), exiting" >&2
  exit 1
fi

TS="$(date -u +%Y%m%d-%H%M%S)"
RUN_DIR="$STAGING_ROOT/$MODE-$TS"
LOG_FILE="$LOG_DIR/$MODE-$TS.log"

# Every following line's stdout/stderr is both shown and logged.
exec > >(tee -a "$LOG_FILE") 2>&1

write_status() {
  local status="$1" detail="${2:-}"
  python3 - "$MODE" "$status" "$detail" "$STATUS_FILE" <<'PYEOF'
import json, sys, datetime
mode, status, detail, path = sys.argv[1:5]
with open(path, "w") as f:
    json.dump({
        "mode": mode,
        "status": status,
        "timestamp": datetime.datetime.now(datetime.UTC).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "detail": detail,
    }, f)
PYEOF
}

fail() {
  echo "BACKUP FAILED ($MODE): $1"
  write_status "failed" "$1"
  # Keep the staged directory for diagnosis rather than deleting it
  # immediately -- see Kernel 77 §9.7 step 11 and §9.10.
  exit 1
}

echo "=== Victory backup ($MODE) starting at $(date -u +%Y-%m-%dT%H:%M:%SZ) ==="
mkdir -p "$RUN_DIR"

# 1. Database dump. pg_dump lives in the container, not the host (see
# Construction/OperatorLogs/kernel-76-reportback.md §5) -- run it there and
# stream the output back.
DUMP_FILE="$RUN_DIR/database.dump"
if ! docker exec "$POSTGRES_CONTAINER" pg_dump --format=custom -U "$POSTGRES_USER" -d "$POSTGRES_DB" > "$DUMP_FILE"; then
  fail "pg_dump failed"
fi
if [ ! -s "$DUMP_FILE" ]; then
  fail "pg_dump produced an empty file"
fi

# 2. Assets (full mode only). -a without -L never follows/dereferences
# symlinks (copies the link itself); --safe-links additionally drops any
# symlink that points outside the tree being copied.
if [ "$MODE" = "full" ]; then
  ASSETS_DIR="$RUN_DIR/assets"
  mkdir -p "$ASSETS_DIR"
  if [ -d "$STORAGE_ROOT" ]; then
    if ! rsync -a --safe-links --exclude='exports/' "$STORAGE_ROOT/" "$ASSETS_DIR/"; then
      fail "asset staging failed"
    fi
  fi
fi

# 3. Manifest + safe metadata only -- never raw .env (§9.2).
COMMIT_HASH="$(cd "$ROOT" && git rev-parse HEAD 2>/dev/null || echo unknown)"
MIGRATION_COUNT="$(docker exec "$POSTGRES_CONTAINER" psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -tAc 'SELECT COUNT(*) FROM schema_migrations' 2>/dev/null | tr -d '[:space:]' || echo unknown)"

python3 - "$RUN_DIR/manifest.json" "$MODE" "$COMMIT_HASH" "$MIGRATION_COUNT" "$(hostname)" <<'PYEOF'
import json, sys, datetime
path, mode, commit, migrations, host = sys.argv[1:6]
with open(path, "w") as f:
    json.dump({
        "backup_format_version": 1,
        "mode": mode,
        "created_at": datetime.datetime.now(datetime.UTC).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "commit_hash": commit,
        "migration_count": migrations,
        "source_host": host,
        "database_dump_command": "pg_dump --format=custom -U victory -d victory",
        "database_restore_command": "pg_restore --format=custom --dbname=<target> database.dump",
    }, f, indent=2)
PYEOF

# 4. Checksums of everything staged.
(cd "$RUN_DIR" && find . -type f ! -name checksums.sha256 -exec sha256sum {} \; > checksums.sha256) || fail "checksum generation failed"

# 5. Package as one immutable, timestamped archive.
ARCHIVE_NAME="$MODE-$TS.tar.gz"
ARCHIVE="$STAGING_ROOT/$ARCHIVE_NAME"
if ! tar -czf "$ARCHIVE" -C "$STAGING_ROOT" "$MODE-$TS"; then
  fail "packaging failed"
fi
rm -rf "$RUN_DIR"

# 6. Upload. `copy`, never `sync` -- see the file header.
if ! rclone copy "$ARCHIVE" "$RCLONE_REMOTE/$MODE/" --checksum; then
  fail "rclone upload failed"
fi

# 7. Independent integrity check: compare the local archive's checksum
# against the encrypted remote's decrypted view (Kernel 77 §9.8 -- "an
# upload status of zero errors is not enough by itself").
if ! rclone check "$STAGING_ROOT" "$RCLONE_REMOTE/$MODE/" --include "$ARCHIVE_NAME" --one-way; then
  fail "post-upload integrity check failed"
fi

# 8. Local retention: keep the last N staged archives of this mode. This is
# a bounded local cache, not the backup's real retention policy -- that's
# retention.sh operating on the remote, per §9.4.
ls -1t "$STAGING_ROOT/$MODE"-*.tar.gz 2>/dev/null | tail -n "+$((LOCAL_RETENTION_COUNT + 1))" | xargs -r rm -f

write_status "success" "uploaded $ARCHIVE_NAME"
echo "=== Victory backup ($MODE) completed at $(date -u +%Y-%m-%dT%H:%M:%SZ) ==="
