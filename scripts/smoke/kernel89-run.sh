#!/usr/bin/env bash
# Kernel 89 proof runner.
#
# Boots a real compiled backend against the DISPOSABLE test database, builds
# the fixture Show, and runs the acceptance proof. Never touches production:
# the only database it will accept is one whose URL is obviously a test
# database, enforced twice (here and in backend/cmd/k89fixture).
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

if [[ -z "${TEST_DATABASE_URL:-}" ]]; then
  # .env carries prose comments without # in places, so read only the one
  # variable this script needs rather than sourcing the whole file.
  POSTGRES_PASSWORD="$(grep -E '^POSTGRES_PASSWORD=' .env | head -1 | cut -d= -f2-)"
  export TEST_DATABASE_URL="postgres://victory:${POSTGRES_PASSWORD}@127.0.0.1:5432/victory_test?sslmode=disable"
fi
case "$TEST_DATABASE_URL" in
  *victory_test*) ;;
  *) echo "refusing: TEST_DATABASE_URL is not a test database" >&2; exit 1 ;;
esac

PORT="${K89_PORT:-8093}"
BIN=/tmp/k89-backend

echo "== building backend =="
(cd backend && go build -o "$BIN" ./cmd/victory)

echo "== booting backend on :$PORT against victory_test =="
PORT="$PORT" DATABASE_URL="$TEST_DATABASE_URL" \
  STORAGE_ROOT=/tmp/k89-storage EXPORTS_ROOT=/tmp/k89-exports BACKUP_DIR=/tmp/k89-backups \
  "$BIN" >/tmp/k89-backend.log 2>&1 &
BACKEND_PID=$!
trap 'kill "$BACKEND_PID" 2>/dev/null || true' EXIT

for _ in $(seq 1 40); do
  if curl -sf "http://127.0.0.1:$PORT/health" >/dev/null 2>&1; then break; fi
  sleep 0.5
done

# --first-theater runs only the announcement half, on the other venue that
# Kernel 89 §10 names. First Theater carries none of the Socio/merchant/
# Aftercare capability flags, so nothing else in this kernel applies there.
VENUE_SLUG="catharsis"
PROOF_SCRIPT="scripts/smoke/kernel89-director-prepared-play.js"
if [[ "${1:-}" == "--first-theater" ]]; then
  VENUE_SLUG="first-theater"
  PROOF_SCRIPT="scripts/smoke/kernel89-first-theater-announcement.js"
fi

echo "== building fixture Show on $VENUE_SLUG =="
FIXTURE="$(cd backend && K89_VENUE_SLUG="$VENUE_SLUG" go run ./cmd/k89fixture)"
echo "$FIXTURE" | head -3

echo "== running the proof =="
set +e
K89_FIXTURE="$FIXTURE" K89_BASE="http://127.0.0.1:$PORT" node "$PROOF_SCRIPT"
PROOF_STATUS=$?
set -e

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
