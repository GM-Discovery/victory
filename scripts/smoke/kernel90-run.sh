#!/usr/bin/env bash
# Kernel 90 proof runner.
#
# Boots a real compiled backend against the DISPOSABLE test database, builds
# the fixture Show, and runs the acceptance proof. Never touches production:
# the only database it will accept is one whose URL is obviously a test
# database, enforced twice (here and in backend/cmd/k90fixture).
#
# Pass --ui to additionally run the Playwright browser proof against the same
# booted backend and fixture, which is what §40/§42 mean by capturing
# screenshots of the real Director controls.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

if [[ -z "${TEST_DATABASE_URL:-}" ]]; then
  # .env carries prose lines that are not shell-safe (a bare `email:` line
  # parses as a command), so read only the one variable this script needs
  # rather than sourcing the whole file. Do not add `source .env` here.
  POSTGRES_PASSWORD="$(grep -E '^POSTGRES_PASSWORD=' .env | head -1 | cut -d= -f2-)"
  export TEST_DATABASE_URL="postgres://victory:${POSTGRES_PASSWORD}@127.0.0.1:5432/victory_test?sslmode=disable"
fi
case "$TEST_DATABASE_URL" in
  *victory_test*) ;;
  *) echo "refusing: TEST_DATABASE_URL is not a test database" >&2; exit 1 ;;
esac

PORT="${K90_PORT:-8094}"
BIN=/tmp/k90-backend

echo "== building backend =="
(cd backend && go build -o "$BIN" ./cmd/victory)

echo "== booting backend on :$PORT against victory_test =="
PORT="$PORT" DATABASE_URL="$TEST_DATABASE_URL" \
  STORAGE_ROOT=/tmp/k90-storage EXPORTS_ROOT=/tmp/k90-exports BACKUP_DIR=/tmp/k90-backups \
  "$BIN" >/tmp/k90-backend.log 2>&1 &
BACKEND_PID=$!
trap 'kill "$BACKEND_PID" 2>/dev/null || true' EXIT

for _ in $(seq 1 40); do
  if curl -sf "http://127.0.0.1:$PORT/health" >/dev/null 2>&1; then break; fi
  sleep 0.5
done

echo "== building fixture Show =="
FIXTURE="$(cd backend && go run ./cmd/k90fixture)"
echo "$FIXTURE" | head -4

echo "== running the acceptance proof =="
set +e
K90_FIXTURE="$FIXTURE" K90_BASE="http://127.0.0.1:$PORT" \
  node scripts/smoke/kernel90-stage-object-visibility.js
PROOF_STATUS=$?
set -e

if [[ "${1:-}" == "--ui" ]]; then
  echo "== running the browser proof =="
  set +e
  K90_FIXTURE="$FIXTURE" K90_BASE="http://127.0.0.1:$PORT" \
    OUT_DIR="${OUT_DIR:-Construction/OperatorLogs/evidence/kernel-90}" \
    node scripts/smoke/kernel90-visibility-browser.js
  UI_STATUS=$?
  set -e
  if [[ "$PROOF_STATUS" -eq 0 ]]; then PROOF_STATUS="$UI_STATUS"; fi
fi

# Close the Session this run opened. Catharsis is a singleton-session venue,
# and a leftover open session there is precisely what broke internal/shows
# and internal/showtime for the whole shared test database during Kernel 88.
SESSION_ID="$(printf '%s' "$FIXTURE" | sed -n 's/.*"session_id": "\([^"]*\)".*/\1/p')"
if [[ -n "$SESSION_ID" ]]; then
  docker exec victory-postgres psql -U victory -d victory_test -q -c \
    "UPDATE sessions SET status='closed', ended_at=NOW() WHERE id='$SESSION_ID';" >/dev/null
  echo "== closed fixture session $SESSION_ID =="
fi

exit "$PROOF_STATUS"
