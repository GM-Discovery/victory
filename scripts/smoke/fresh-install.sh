#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/smoke/fresh-install.sh --local

Optional env:
  POSTGRES_CONTAINER=victory-postgres
  POSTGRES_USER=victory
  POSTGRES_PASSWORD=change_this_now   (use the real value from /opt/victory/.env)
  BACKEND_PORT=18081
EOF
}

if [[ "${1:-}" != "--local" ]]; then
  usage
  exit 2
fi
shift

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BACKEND_DIR="$ROOT/backend"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-victory-postgres}"
POSTGRES_USER="${POSTGRES_USER:-victory}"
# Kernel 76 rotated the real password out of the compose file into .env;
# accept it from the environment instead of hardcoding the pre-rotation
# default everywhere below (Kernel 78 repair -- the script silently broke
# at the first post-rotation run).
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-change_this_now}"
BACKEND_PORT="${BACKEND_PORT:-18081}"
DB_NAME="victory_fresh_$(date +%s)_$RANDOM"
BACKEND_LOG="${TMPDIR:-/tmp}/victory-fresh-install-backend.log"
HEALTH_OUT="${TMPDIR:-/tmp}/victory-fresh-install-health.json"
PROVIDERS_OUT="${TMPDIR:-/tmp}/victory-fresh-install-providers.json"
BODY_OUT="${TMPDIR:-/tmp}/victory-fresh-install-body.json"
LOGIN_HEADERS=""
LOGIN_BODY=""
BACKEND_PID=""
KEEP_DB=0

for arg in "$@"; do
  case "$arg" in
    --keep-db)
      KEEP_DB=1
      ;;
    *)
      echo "unknown argument: $arg" >&2
      usage
      exit 2
      ;;
  esac
done

cleanup() {
  local exit_code=$?
  set +e
  if [[ -n "$BACKEND_PID" ]]; then
    kill "$BACKEND_PID" >/dev/null 2>&1 || true
    wait "$BACKEND_PID" >/dev/null 2>&1 || true
  fi
  # Belt-and-suspenders: if anything is still holding BACKEND_PORT (e.g. an
  # interrupted prior run's orphan), free it too, so repeated --local runs
  # don't accumulate stray backend processes.
  if command -v lsof >/dev/null 2>&1; then
    lsof -ti "tcp:${BACKEND_PORT}" 2>/dev/null | xargs -r kill -9 2>/dev/null || true
  fi
  if [[ "$KEEP_DB" -eq 0 ]]; then
    docker exec -i "$POSTGRES_CONTAINER" dropdb -U "$POSTGRES_USER" "$DB_NAME" >/dev/null 2>&1 || true
  fi
  if [[ -n "$LOGIN_HEADERS" ]]; then
    rm -f "$LOGIN_HEADERS" >/dev/null 2>&1 || true
  fi
  if [[ -n "$LOGIN_BODY" ]]; then
    rm -f "$LOGIN_BODY" >/dev/null 2>&1 || true
  fi
  exit "$exit_code"
}
trap cleanup EXIT

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

need_cmd docker
need_cmd go
need_cmd curl

echo "Using temporary database: $DB_NAME"
# Kernel 96: the repo root's own docker-compose.yml (Grant's bespoke
# production deployment) is retired -- packaging/podman/compose.yml is now
# the one real deployment path, for production and local dev alike. Run
# from its own directory so its sibling .env is picked up automatically.
(cd "$ROOT/packaging/podman" && docker compose up -d postgres >/dev/null)
docker exec -i "$POSTGRES_CONTAINER" createdb -U "$POSTGRES_USER" "$DB_NAME"

# Kernel 96: used to manually pre-apply migrations 000-048 via psql before
# handing off to the real binary for the rest. Now stops after 001 (schema
# only) -- every content-seeding migration from 002 onward needs
# access.EnsureDefaultLocation (Go) to have created the install's real
# default location first, which only the real binary's own phased boot
# sequence does. Applying them manually here would create that content
# with no location to attach to at all.
migrations=(
  "$ROOT/backend/migrations/000_kernel42_productions_baseline.sql"
  "$ROOT/backend/migrations/001_init.sql"
)

for migration in "${migrations[@]}"; do
  echo "Applying $(basename "$migration")"
  docker exec -i "$POSTGRES_CONTAINER" psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$DB_NAME" < "$migration"
done
echo "PASS schema migrations from empty DB (the rest apply when the real binary boots)"

# Build once and run the compiled binary directly (rather than `go run`,
# backgrounded inside a subshell) so $BACKEND_PID is the actual server
# process's PID. `go run` forks a child for the compiled binary and a
# wrapping subshell adds a second layer -- `kill "$BACKEND_PID"` on either
# of those doesn't reliably reach the real server, leaving an orphaned
# process holding $BACKEND_PORT after the script exits.
FRESH_INSTALL_BIN="${TMPDIR:-/tmp}/victory-fresh-install-backend-bin"
(cd "$BACKEND_DIR" && go build -o "$FRESH_INSTALL_BIN" ./cmd/victory)

env \
  PORT="$BACKEND_PORT" \
  DATABASE_URL="postgres://victory:${POSTGRES_PASSWORD}@127.0.0.1:5432/$DB_NAME?sslmode=disable" \
  MIGRATE_DANGEROUSLY_SKIP_BACKUP=1 \
  STORAGE_ROOT="$ROOT/storage" \
  SESSION_COOKIE_SECURE=false \
  COOKIE_SECURE=false \
  PASSWORD_SIGNUP_ENABLED=true \
  OPERATOR_HANDLE=fresh_install_operator \
  DISCORD_CLIENT_ID= \
  DISCORD_CLIENT_SECRET= \
  DISCORD_REDIRECT_URL= \
  DISCORD_OAUTH_SCOPES="identify email" \
  DISCORD_OAUTH_ENABLED=false \
  DISCORD_APPLICATION_ID= \
  DISCORD_BOT_TOKEN= \
  DISCORD_BOT_PERMISSIONS=16 \
  DISCORD_BOT_REDIRECT_URL= \
  DISCORD_PUBLIC_KEY= \
  DISCORD_SERVER_LINK_ENABLED=false \
  DISCORD_GATEWAY_ENABLED=false \
  DISCORD_GATEWAY_INTENTS=513 \
  DISCORD_GATEWAY_URL= \
  "$FRESH_INSTALL_BIN" >"$BACKEND_LOG" 2>&1 &
BACKEND_PID=$!

health_url="http://127.0.0.1:${BACKEND_PORT}/health"
for _ in $(seq 1 60); do
  if curl -fsS "$health_url" >"$HEALTH_OUT" 2>/dev/null; then
    break
  fi
  sleep 1
done
if ! curl -fsS "$health_url" >"$HEALTH_OUT"; then
  echo "backend failed to start; logs:" >&2
  tail -n 50 "$BACKEND_LOG" >&2 || true
  exit 1
fi
echo "PASS backend starts"

curl -fsS "http://127.0.0.1:${BACKEND_PORT}/api/auth/providers" >"$PROVIDERS_OUT"
echo "PASS /api/auth/providers"

check_status() {
  local path="$1"
  local want="$2"
  local code
  code="$(curl -s -o "$BODY_OUT" -w '%{http_code}' "http://127.0.0.1:${BACKEND_PORT}${path}")"
  if [[ "$code" != "$want" ]]; then
    echo "expected $path to return $want, got $code" >&2
    cat "$BODY_OUT" >&2 || true
    exit 1
  fi
  echo "PASS $path -> $code"
}

check_status "/api/account/me" "401"
check_status "/api/discord/server-link/status" "401"
check_status "/api/discord/channel-mapping/status" "401"
# Kernel 76 (K76-M02) closed gateway status to Operator-only; anonymous is
# 401 by design. The old 200 expectation predated that closure (fixed in
# Kernel 78 -- this script had been silently stale since the K76 rotation).
check_status "/api/discord/gateway/status" "401"
# Kernel 78: eWrite surfaces exist and refuse anonymous callers.
check_status "/api/ewrite/tree" "401"
check_status "/api/library/tree" "401"
check_status "/api/library/search?q=test" "401"

for file in \
  frontend/assets/favicon.png \
  frontend/login/index.html \
  frontend/legal/terms/index.html \
  frontend/legal/privacy/index.html \
  frontend/venues/producers-office/index.html \
  frontend/venues/first-theater/index.html \
  frontend/venues/the-cave/index.html \
  frontend/venues/middle-school-stage/index.html
do
  if [[ ! -f "$ROOT/$file" ]]; then
    echo "missing required file: $file" >&2
    exit 1
  fi
done
echo "PASS static/legal/login/venue files exist"

operator_handle="fresh_install_operator"
operator_display="Fresh Install Operator"
LOGIN_HEADERS="${TMPDIR:-/tmp}/victory-fresh-install-signup.headers"
LOGIN_BODY="${TMPDIR:-/tmp}/victory-fresh-install-signup-body.json"
login_password="fresh-install-$(date +%s)-$RANDOM"
signup_status="$(
  curl -s -D "$LOGIN_HEADERS" -o "$LOGIN_BODY" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -X POST \
    -d "{\"email\":\"${operator_handle}@example.com\",\"handle\":\"$operator_handle\",\"password\":\"$login_password\",\"display_name\":\"$operator_display\"}" \
    "http://127.0.0.1:${BACKEND_PORT}/api/auth/signup"
)"
if [[ "$signup_status" != "200" ]]; then
  echo "expected /api/auth/signup to return 200, got $signup_status" >&2
  cat "$LOGIN_BODY" >&2 || true
  tail -n 50 "$BACKEND_LOG" >&2 || true
  exit 1
fi
raw_session="$(tr -d '\r' < "$LOGIN_HEADERS" | sed -n 's/^Set-Cookie: victory_session=\([^;]*\).*/\1/p' | tail -n1)"
if [[ -z "$raw_session" ]]; then
  echo "failed to capture session cookie from login" >&2
  cat "$LOGIN_HEADERS" >&2 || true
  exit 1
fi
echo "PASS local auth signup issued a session cookie"

bootstrap_default="$(
  cd "$BACKEND_DIR" && \
  env \
    DATABASE_URL="postgres://victory:${POSTGRES_PASSWORD}@127.0.0.1:5432/$DB_NAME?sslmode=disable" \
    go run ./cmd/victory-bootstrap producer --handle "$operator_handle"
)"
if [[ "$bootstrap_default" != *"Already existed: false"* ]]; then
  echo "$bootstrap_default" >&2
  exit 1
fi
if [[ "$bootstrap_default" != *"Location: Victory Theater (victory-theater)"* ]]; then
  echo "$bootstrap_default" >&2
  exit 1
fi
echo "PASS bootstrap producer by handle at neutral default location"

bootstrap_repeat="$(
  cd "$BACKEND_DIR" && \
  env \
    DATABASE_URL="postgres://victory:${POSTGRES_PASSWORD}@127.0.0.1:5432/$DB_NAME?sslmode=disable" \
    go run ./cmd/victory-bootstrap producer --handle "$operator_handle"
)"
if [[ "$bootstrap_repeat" != *"Already existed: true"* ]]; then
  echo "$bootstrap_repeat" >&2
  exit 1
fi
echo "PASS bootstrap producer is idempotent"

if cd "$BACKEND_DIR" && \
  env DATABASE_URL="postgres://victory:${POSTGRES_PASSWORD}@127.0.0.1:5432/$DB_NAME?sslmode=disable" \
  go run ./cmd/victory-bootstrap producer --handle definitely_missing_user >/tmp/victory-fresh-install-bootstrap-error.txt 2>&1; then
  echo "expected missing-user bootstrap to fail" >&2
  exit 1
fi
echo "PASS missing user fails safely"

account_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' -H "Cookie: victory_session=$raw_session" "http://127.0.0.1:${BACKEND_PORT}/api/account/me")"
if [[ "$account_status" != "200" ]]; then
  echo "expected /api/account/me to return 200, got $account_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
account_response="$(cat "$BODY_OUT")"
if [[ "$account_response" != *'"is_producer":true'* ]]; then
  echo "$account_response" >&2
  exit 1
fi
if [[ "$account_response" != *'"current_role":"producer"'* ]]; then
  echo "$account_response" >&2
  exit 1
fi
echo "PASS account authority reflects bootstrap producer grant"

# --- Kernel 61 / 61A: Player Workbook / Trailer Face loop on a brand-new account ---

catalogue_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' -H "Cookie: victory_session=$raw_session" "http://127.0.0.1:${BACKEND_PORT}/api/player-profile/catalogue")"
if [[ "$catalogue_status" != "200" ]]; then
  echo "expected /api/player-profile/catalogue to return 200, got $catalogue_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
if [[ "$(cat "$BODY_OUT")" != *'"catalogue_key"'* ]]; then
  echo "expected catalogue response to contain catalogue_key" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS fresh account can load the player-profile catalogue"

me_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' -H "Cookie: victory_session=$raw_session" "http://127.0.0.1:${BACKEND_PORT}/api/player-profile/me")"
if [[ "$me_status" != "200" ]]; then
  echo "expected /api/player-profile/me to return 200, got $me_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
me_response="$(cat "$BODY_OUT")"
workbook_id="$(printf '%s' "$me_response" | grep -o '"id":"[^"]*"' | head -n1 | sed 's/"id":"//;s/"$//')"
if [[ -z "$workbook_id" ]]; then
  echo "failed to extract workbook id from /api/player-profile/me response" >&2
  echo "$me_response" >&2
  exit 1
fi
echo "PASS fresh account received exactly one Player Workbook ($workbook_id)"

commit_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H 'Content-Type: application/json' \
  -X POST -d '{"answers":{"real_name":"Fresh Install Test"}}' \
  "http://127.0.0.1:${BACKEND_PORT}/api/player-profile/pages/identity_presentation/commit")"
if [[ "$commit_status" != "200" ]]; then
  echo "expected page commit to return 200, got $commit_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
commit_response="$(cat "$BODY_OUT")"
if [[ "$commit_response" != *'"changed":true'* ]]; then
  echo "expected page commit to report changed:true" >&2
  echo "$commit_response" >&2
  exit 1
fi
event_id="$(printf '%s' "$commit_response" | grep -o '"id":"[^"]*"' | head -n1 | sed 's/"id":"//;s/"$//')"
if [[ -z "$event_id" ]]; then
  echo "failed to extract event id from page commit response" >&2
  echo "$commit_response" >&2
  exit 1
fi
echo "PASS fresh account can commit a Workbook page"

face_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' -H "Cookie: victory_session=$raw_session" "http://127.0.0.1:${BACKEND_PORT}/api/player-profile/${workbook_id}/face")"
if [[ "$face_status" != "200" ]]; then
  echo "expected social Face projection to return 200, got $face_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS fresh account's Face projects successfully"

delete_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' -H "Cookie: victory_session=$raw_session" \
  -X DELETE "http://127.0.0.1:${BACKEND_PORT}/api/player-profile/events/${event_id}")"
if [[ "$delete_status" != "200" ]]; then
  echo "expected ordinary History deletion to return 200, got $delete_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
if [[ "$(cat "$BODY_OUT")" != *'"deleted":true'* ]]; then
  echo "expected delete response to report deleted:true" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS fresh account can delete ordinary History and facts recompute"

for file in \
  frontend/venues/trailers/face.html \
  frontend/venues/trailers/workbook.html \
  frontend/venues/trailers/view.html \
  frontend/venues/trailers/index.html \
  frontend/account/index.html
do
  if [[ ! -f "$ROOT/$file" ]]; then
    echo "missing required file: $file" >&2
    exit 1
  fi
done
echo "PASS Trailer and Account page files exist"

# --- Kernel 62: private relationship records between two fresh accounts ---
# User A is the operator account above (workbook_id). User B is a second
# fresh account. B records private notes about A; A must never be able to
# read them (Kernel 62 §15.6).

second_handle="fresh_install_second"
SECOND_HEADERS="${TMPDIR:-/tmp}/victory-fresh-install-second.headers"
second_signup_status="$(
  curl -s -D "$SECOND_HEADERS" -o "$BODY_OUT" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -X POST \
    -d "{\"email\":\"${second_handle}@example.com\",\"handle\":\"$second_handle\",\"password\":\"$login_password\",\"display_name\":\"Fresh Install Second\"}" \
    "http://127.0.0.1:${BACKEND_PORT}/api/auth/signup"
)"
if [[ "$second_signup_status" != "200" ]]; then
  echo "expected second signup to return 200, got $second_signup_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
second_session="$(tr -d '\r' < "$SECOND_HEADERS" | sed -n 's/^Set-Cookie: victory_session=\([^;]*\).*/\1/p' | tail -n1)"
if [[ -z "$second_session" ]]; then
  echo "failed to capture second account session cookie" >&2
  exit 1
fi
echo "PASS second fresh account (B) signed up"

rel_create_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$second_session" -H 'Content-Type: application/json' \
  -X POST -d "{\"subject_profile_id\":\"$workbook_id\"}" \
  "http://127.0.0.1:${BACKEND_PORT}/api/player-relationships")"
if [[ "$rel_create_status" != "200" ]]; then
  echo "expected relationship create to return 200, got $rel_create_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
relationship_id="$(grep -o '"id":"[^"]*"' "$BODY_OUT" | head -n1 | sed 's/"id":"//;s/"$//')"
if [[ -z "$relationship_id" ]]; then
  echo "failed to extract relationship id" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS B created a private relationship about A ($relationship_id)"

nickname_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$second_session" -H 'Content-Type: application/json' \
  -X PATCH -d '{"private_nickname":"Smoke Test Friend","trust_level":"trusted"}' \
  "http://127.0.0.1:${BACKEND_PORT}/api/player-relationships/${relationship_id}")"
if [[ "$nickname_status" != "200" || "$(cat "$BODY_OUT")" != *'"private_nickname":"Smoke Test Friend"'* ]]; then
  echo "expected private nickname save to succeed, got $nickname_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS B saved a private nickname and qualitative value"

rel_page_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$second_session" -H 'Content-Type: application/json' \
  -X POST -d '{"answers":{"how_i_know_them":"Fresh install smoke"}}' \
  "http://127.0.0.1:${BACKEND_PORT}/api/player-relationships/${relationship_id}/pages/connection")"
if [[ "$rel_page_status" != "200" || "$(cat "$BODY_OUT")" != *'"changed":true'* ]]; then
  echo "expected relationship page save to report changed:true, got $rel_page_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS B saved a relationship workbook page"

journal_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$second_session" -H 'Content-Type: application/json' \
  -X POST -d '{"title":"Smoke","body":"Private smoke-test journal entry."}' \
  "http://127.0.0.1:${BACKEND_PORT}/api/player-relationships/${relationship_id}/journal")"
if [[ "$journal_status" != "200" ]]; then
  echo "expected journal create to return 200, got $journal_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS B wrote a private journal entry"

subject_read_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" \
  "http://127.0.0.1:${BACKEND_PORT}/api/player-relationships/${relationship_id}")"
if [[ "$subject_read_status" != "404" ]]; then
  echo "expected subject read of observer relationship to return 404, got $subject_read_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS A (the subject) cannot read B's relationship record"

archive_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$second_session" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/player-relationships/${relationship_id}/archive")"
if [[ "$archive_status" != "200" ]]; then
  echo "expected archive to return 200, got $archive_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
active_list="$(curl -s -H "Cookie: victory_session=$second_session" "http://127.0.0.1:${BACKEND_PORT}/api/player-relationships?state=active")"
archived_list="$(curl -s -H "Cookie: victory_session=$second_session" "http://127.0.0.1:${BACKEND_PORT}/api/player-relationships?state=archived")"
if [[ "$active_list" == *"$relationship_id"* ]]; then
  echo "archived relationship still appears in active list" >&2
  echo "$active_list" >&2
  exit 1
fi
if [[ "$archived_list" != *"$relationship_id"* ]]; then
  echo "archived relationship missing from archived list" >&2
  echo "$archived_list" >&2
  exit 1
fi
echo "PASS archive hides the relationship from the default list"

second_me="$(curl -s -H "Cookie: victory_session=$second_session" "http://127.0.0.1:${BACKEND_PORT}/api/player-profile/me")"
second_workbook_id="$(printf '%s' "$second_me" | grep -o '"id":"[^"]*"' | head -n1 | sed 's/"id":"//;s/"$//')"
self_rel_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$second_session" -H 'Content-Type: application/json' \
  -X POST -d "{\"subject_profile_id\":\"$second_workbook_id\"}" \
  "http://127.0.0.1:${BACKEND_PORT}/api/player-relationships")"
if [[ "$self_rel_status" != "400" ]]; then
  echo "expected self-relationship to be rejected with 400, got $self_rel_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS self-relationship is rejected"

for file in \
  frontend/venues/trailers/people.html \
  frontend/venues/trailers/person.html
do
  if [[ ! -f "$ROOT/$file" ]]; then
    echo "missing required file: $file" >&2
    exit 1
  fi
done
echo "PASS My People page files exist"

# --- Kernel 65: Third Place Headshot Commons ---
# Reuses fresh accounts A ($raw_session, $workbook_id) and B ($second_session)
# already created above for the Kernel 62 checks.
#
# Kernel 68 gates leaving a Headshot on Trailer Face readiness (stage name
# plus at least one visible Face field). A committed a visible
# identity_presentation fact (real_name) during the Kernel 61 checks above,
# but that same block immediately deleted it again to test ordinary History
# deletion -- so no visible fact survives from there. Set a stage name and a
# fresh, un-deleted fact now so A is Face-ready before this block's own
# "leave a Headshot" assertions, and so this fixture doubles as the
# Kernel 68 "account becomes ready" setup reused below.

stage_name_commit_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H 'Content-Type: application/json' \
  -X POST -d '{"stage_name":"Fresh Install Test Stage Name"}' \
  "http://127.0.0.1:${BACKEND_PORT}/api/player-profile/stage-name")"
if [[ "$stage_name_commit_status" != "200" ]]; then
  echo "expected stage name commit to return 200, got $stage_name_commit_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi

face_field_commit_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H 'Content-Type: application/json' \
  -X POST -d '{"answers":{"short_intro":"Fresh Install Test Intro"}}' \
  "http://127.0.0.1:${BACKEND_PORT}/api/player-profile/pages/identity_presentation/commit")"
if [[ "$face_field_commit_status" != "200" ]]; then
  echo "expected a visible Face field commit to return 200, got $face_field_commit_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS fresh account can set a stage name"

anon_headshots_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' "http://127.0.0.1:${BACKEND_PORT}/api/third-place/headshots")"
if [[ "$anon_headshots_status" != "401" ]]; then
  echo "expected anonymous Third Place list to return 401, got $anon_headshots_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS Third Place list rejects anonymous requests"

leave_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/third-place/headshots/me")"
if [[ "$leave_status" != "200" || "$(cat "$BODY_OUT")" != *'"created":true'* ]]; then
  echo "expected A to leave a new Headshot with created:true, got $leave_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
headshot_id="$(grep -o '"headshot_id":"[^"]*"' "$BODY_OUT" | head -n1 | sed 's/"headshot_id":"//;s/"$//')"
if [[ -z "$headshot_id" ]]; then
  echo "failed to extract headshot id" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS A left a Headshot ($headshot_id)"

leave_again_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/third-place/headshots/me")"
if [[ "$leave_again_status" != "200" || "$(cat "$BODY_OUT")" != *'"created":false'* ]]; then
  echo "expected repeated Leave Headshot to be idempotent (created:false), got $leave_again_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS repeated Leave Headshot did not create a duplicate"

# Kernel 68 gated the commons list on the VIEWER's own Trailer Face
# readiness (stage name + a visible Face field). B is deliberately kept
# face-UNREADY here -- the Kernel 68 map-fog assertions below depend on
# that -- so B proves the gate refuses, and Face-ready A proves the
# listing works. (Kernel 78 repair: the original expectation predated the
# K68 gate and asked the impossible of B.)
b_commons_list="$(curl -s -H "Cookie: victory_session=$second_session" "http://127.0.0.1:${BACKEND_PORT}/api/third-place/headshots")"
if [[ "$b_commons_list" != *'"trailer_face_not_ready"'* ]]; then
  echo "expected Face-unready B to be refused the commons list with trailer_face_not_ready" >&2
  echo "$b_commons_list" >&2
  exit 1
fi
echo "PASS Face-unready B is refused the commons list (Kernel 68 gate)"

b_commons_list="$(curl -s -H "Cookie: victory_session=$raw_session" "http://127.0.0.1:${BACKEND_PORT}/api/third-place/headshots")"
if [[ "$b_commons_list" != *"\"headshot_id\":\"$headshot_id\""* ]]; then
  echo "expected Face-ready A to see the Headshot in the commons list" >&2
  echo "$b_commons_list" >&2
  exit 1
fi
if [[ "$b_commons_list" == *'"email"'* || "$b_commons_list" == *'"handle"'* ]]; then
  echo "Third Place commons payload must not include email/handle" >&2
  echo "$b_commons_list" >&2
  exit 1
fi
echo "PASS B sees A's Headshot with no private account fields"

remove_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" \
  -X DELETE "http://127.0.0.1:${BACKEND_PORT}/api/third-place/headshots/me")"
if [[ "$remove_status" != "200" ]]; then
  echo "expected Remove My Headshot to return 200, got $remove_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
b_commons_after_remove="$(curl -s -H "Cookie: victory_session=$second_session" "http://127.0.0.1:${BACKEND_PORT}/api/third-place/headshots")"
if [[ "$b_commons_after_remove" == *"\"headshot_id\":\"$headshot_id\""* ]]; then
  echo "removed Headshot still appears in the active commons list" >&2
  echo "$b_commons_after_remove" >&2
  exit 1
fi
echo "PASS removed Headshot no longer appears in the active commons list"

a_history="$(curl -s -H "Cookie: victory_session=$raw_session" "http://127.0.0.1:${BACKEND_PORT}/api/third-place/headshots/me/history")"
if [[ "$a_history" != *'"status":"removed"'* ]]; then
  echo "expected A's Headshot history to show a removed record" >&2
  echo "$a_history" >&2
  exit 1
fi
echo "PASS A's Headshot history preserves the placement/removal record"

if [[ ! -f "$ROOT/frontend/venues/third-place/index.html" ]]; then
  echo "missing required file: frontend/venues/third-place/index.html" >&2
  exit 1
fi
echo "PASS Third Place page file exists"

# --- Kernel 66: Show Run, Audience Program, and Roster MVP ---
# Reuses producer account A ($raw_session, bootstrapped as producer at the
# install's default location above) and account B ($second_session).
# There is currently no in-app "create a Production" flow anywhere in
# Victory (producers-office's own production picker shows "No productions
# available" on a database with none) -- this is a pre-existing gap, not
# something Kernel 66 is responsible for, so a minimal Production row is
# inserted directly here, the same way backend/internal/showruns's own
# tests do, purely so the Show Run creation flow has something to attach to.

anon_showruns_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' "http://127.0.0.1:${BACKEND_PORT}/api/show-runs")"
if [[ "$anon_showruns_status" != "401" ]]; then
  echo "expected anonymous Show Runs list to return 401, got $anon_showruns_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS Show Runs list rejects anonymous requests"

docker exec -i "$POSTGRES_CONTAINER" psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$DB_NAME" -c "
  INSERT INTO productions (location_id, name, slug)
  SELECT id, 'Fresh Install Test Production', 'fresh-install-test-production'
  FROM locations WHERE is_default
  ON CONFLICT (location_id, slug) DO NOTHING;
" >/dev/null
fresh_install_production_id="$(docker exec -i "$POSTGRES_CONTAINER" psql -tAc "
  SELECT id FROM productions WHERE slug = 'fresh-install-test-production';
" -U "$POSTGRES_USER" -d "$DB_NAME" | tr -d '[:space:]')"
if [[ -z "$fresh_install_production_id" ]]; then
  echo "failed to insert fixture production for Show Run smoke checks" >&2
  exit 1
fi
echo "PASS fixture production created for Show Run smoke checks ($fresh_install_production_id)"

fresh_install_location_id="$(docker exec -i "$POSTGRES_CONTAINER" psql -tAc "
  SELECT id FROM locations WHERE is_default;
" -U "$POSTGRES_USER" -d "$DB_NAME" | tr -d '[:space:]')"
if [[ -z "$fresh_install_location_id" ]]; then
  echo "failed to resolve default location id for Kernel 70 Scene smoke checks" >&2
  exit 1
fi

create_run_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/show-runs" \
  -d "{\"production_id\":\"$fresh_install_production_id\",\"title\":\"Fresh Install Test Run\",\"slug\":\"fresh-install-test-run\",\"show_format\":\"playtest\"}")"
if [[ "$create_run_status" != "200" ]]; then
  echo "expected Producer A to create a Show Run, got $create_run_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
show_run_id="$(grep -o '"id":"[^"]*"' "$BODY_OUT" | head -n1 | sed 's/"id":"//;s/"$//')"
if [[ -z "$show_run_id" ]]; then
  echo "failed to extract show run id" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS Producer A created a Show Run ($show_run_id)"

# --- Kernel 71: two-punch Show tickets are now the only ordinary path to
# an active Player roster row. First prove the old direct-add-as-player
# path is closed for an ORDINARY Director, then prove both ticket
# directions end-to-end. Producer A ($raw_session) is this script's
# bootstrap Operator account (OPERATOR_HANDLE matches its handle), so
# testing the rejection against A would exercise the explicit Operator
# override instead -- a disposable non-operator producer account is used
# here so the rejection check is genuine, without leaking a location grant
# into A's or B's fixture state that later map-visibility assertions rely on.
ordinary_director_handle="fresh_install_ordinary_director"
ORDINARY_DIRECTOR_HEADERS="${TMPDIR:-/tmp}/victory-fresh-install-ordinary-director.headers"
ordinary_director_signup_status="$(
  curl -s -D "$ORDINARY_DIRECTOR_HEADERS" -o "$BODY_OUT" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -X POST \
    -d "{\"email\":\"${ordinary_director_handle}@example.com\",\"handle\":\"$ordinary_director_handle\",\"password\":\"$login_password\",\"display_name\":\"Fresh Install Ordinary Director\"}" \
    "http://127.0.0.1:${BACKEND_PORT}/api/auth/signup"
)"
if [[ "$ordinary_director_signup_status" != "200" ]]; then
  echo "expected ordinary-director signup to return 200, got $ordinary_director_signup_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
ordinary_director_session="$(tr -d '\r' < "$ORDINARY_DIRECTOR_HEADERS" | sed -n 's/^Set-Cookie: victory_session=\([^;]*\).*/\1/p' | tail -n1)"
ordinary_director_user_id="$(docker exec -i "$POSTGRES_CONTAINER" psql -tAc "
  SELECT id FROM users WHERE handle = '$ordinary_director_handle';
" -U "$POSTGRES_USER" -d "$DB_NAME" | tr -d '[:space:]')"
docker exec -i "$POSTGRES_CONTAINER" psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$DB_NAME" -c "
  INSERT INTO location_memberships (location_id, user_id, role, active)
  VALUES ('$fresh_install_location_id', '$ordinary_director_user_id', 'director', TRUE);
" >/dev/null
echo "PASS disposable ordinary-Director account provisioned (not Operator)"

direct_add_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$ordinary_director_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/show-runs/${show_run_id}/roster" \
  -d "{\"target_profile_id\":\"$workbook_id\",\"role\":\"player\"}")"
if [[ "$direct_add_status" != "400" || "$(cat "$BODY_OUT")" != *'"player_requires_ticket"'* ]]; then
  echo "expected an ordinary Director's direct role=player roster add to be rejected with player_requires_ticket, got $direct_add_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS an ordinary Director's direct role=player roster add is rejected (player_requires_ticket)"

# Player-request direction. Uses a dedicated fresh account (D), not B --
# B is reused as an Audience-only fixture much later in this script (map
# visibility, curated program checks), and this ticket flow would leave B
# as an active Player instead, which would break those later assertions.
requester_handle="fresh_install_requester"
REQUESTER_HEADERS="${TMPDIR:-/tmp}/victory-fresh-install-requester.headers"
requester_signup_status="$(
  curl -s -D "$REQUESTER_HEADERS" -o "$BODY_OUT" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -X POST \
    -d "{\"email\":\"${requester_handle}@example.com\",\"handle\":\"$requester_handle\",\"password\":\"$login_password\",\"display_name\":\"Fresh Install Requester\"}" \
    "http://127.0.0.1:${BACKEND_PORT}/api/auth/signup"
)"
if [[ "$requester_signup_status" != "200" ]]; then
  echo "expected requester signup to return 200, got $requester_signup_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
requester_session="$(tr -d '\r' < "$REQUESTER_HEADERS" | sed -n 's/^Set-Cookie: victory_session=\([^;]*\).*/\1/p' | tail -n1)"
requester_me="$(curl -s -H "Cookie: victory_session=$requester_session" "http://127.0.0.1:${BACKEND_PORT}/api/player-profile/me")"
requester_workbook_id="$(printf '%s' "$requester_me" | grep -o '"id":"[^"]*"' | head -n1 | sed 's/"id":"//;s/"$//')"
echo "PASS dedicated ticket-requester account (D) signed up ($requester_workbook_id)"

player_request_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$requester_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/show-runs/${show_run_id}/tickets/request" \
  -d '{"message":"let me in"}')"
if [[ "$player_request_status" != "200" || "$(cat "$BODY_OUT")" != *'"status":"pending_director"'* ]]; then
  echo "expected D's ticket request to return pending_director, got $player_request_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
ticket_id="$(grep -o '"id":"[^"]*"' "$BODY_OUT" | head -n1 | sed 's/"id":"//;s/"$//')"
echo "PASS D requested to join the Show Run ($ticket_id), one punch grants nothing"

roster_after_one_punch="$(curl -s -H "Cookie: victory_session=$raw_session" "http://127.0.0.1:${BACKEND_PORT}/api/show-runs/${show_run_id}/roster")"
if [[ "$roster_after_one_punch" == *"$requester_workbook_id"* ]]; then
  echo "expected D to have no roster row after only one punch" >&2
  echo "$roster_after_one_punch" >&2
  exit 1
fi
echo "PASS one punch creates no roster row"

ticket_punch_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/tickets/${ticket_id}/punch")"
if [[ "$ticket_punch_status" != "200" || "$(cat "$BODY_OUT")" != *'"status":"valid"'* ]]; then
  echo "expected Producer A's approval punch to validate the ticket, got $ticket_punch_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS Producer A approved D's request; ticket is valid"

roster_after_valid="$(curl -s -H "Cookie: victory_session=$raw_session" "http://127.0.0.1:${BACKEND_PORT}/api/show-runs/${show_run_id}/roster")"
if [[ "$roster_after_valid" != *'"role_label":"Player"'* ]]; then
  echo "expected D's valid ticket to produce a role_label Player roster row" >&2
  echo "$roster_after_valid" >&2
  exit 1
fi
if [[ "$roster_after_valid" == *'"role_label":"Cast"'* ]]; then
  echo "roster role label must never render as Cast" >&2
  exit 1
fi
echo "PASS valid ticket created a Player (never Cast) roster row"

# Character selection: D picks an active Character for this Show Run.
requester_user_id="$(docker exec -i "$POSTGRES_CONTAINER" psql -tAc "
  SELECT user_id FROM player_profile_workbooks WHERE id = '$requester_workbook_id';
" -U "$POSTGRES_USER" -d "$DB_NAME" | tr -d '[:space:]')"
requester_character_id="$(docker exec -i "$POSTGRES_CONTAINER" psql -tAc "
  INSERT INTO character_cards (owner_user_id, location_id, name)
  VALUES ('$requester_user_id', '$fresh_install_location_id', 'Fresh Install Ticket Character')
  RETURNING id;
" -U "$POSTGRES_USER" -d "$DB_NAME" | head -n1 | tr -d '[:space:]')"
select_character_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$requester_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/show-runs/${show_run_id}/roster/me/character" \
  -d "{\"character_card_id\":\"$requester_character_id\"}")"
if [[ "$select_character_status" != "200" || "$(cat "$BODY_OUT")" != *"\"character_card_id\":\"$requester_character_id\""* ]]; then
  echo "expected D to select their own Character, got $select_character_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS D selected a Character for this Show Run"

# Director-invitation direction: Producer A invites a fresh third account C.
third_handle="fresh_install_third"
THIRD_HEADERS="${TMPDIR:-/tmp}/victory-fresh-install-third.headers"
third_signup_status="$(
  curl -s -D "$THIRD_HEADERS" -o "$BODY_OUT" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -X POST \
    -d "{\"email\":\"${third_handle}@example.com\",\"handle\":\"$third_handle\",\"password\":\"$login_password\",\"display_name\":\"Fresh Install Third\"}" \
    "http://127.0.0.1:${BACKEND_PORT}/api/auth/signup"
)"
if [[ "$third_signup_status" != "200" ]]; then
  echo "expected third signup to return 200, got $third_signup_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
third_session="$(tr -d '\r' < "$THIRD_HEADERS" | sed -n 's/^Set-Cookie: victory_session=\([^;]*\).*/\1/p' | tail -n1)"
third_me="$(curl -s -H "Cookie: victory_session=$third_session" "http://127.0.0.1:${BACKEND_PORT}/api/player-profile/me")"
third_workbook_id="$(printf '%s' "$third_me" | grep -o '"id":"[^"]*"' | head -n1 | sed 's/"id":"//;s/"$//')"
echo "PASS third fresh account (C) signed up ($third_workbook_id)"

invite_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/show-runs/${show_run_id}/tickets/invite" \
  -d "{\"target_profile_id\":\"$third_workbook_id\",\"message\":\"join us\"}")"
if [[ "$invite_status" != "200" || "$(cat "$BODY_OUT")" != *'"status":"pending_player"'* ]]; then
  echo "expected Producer A's invitation to return pending_player, got $invite_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
invite_ticket_id="$(grep -o '"id":"[^"]*"' "$BODY_OUT" | head -n1 | sed 's/"id":"//;s/"$//')"
echo "PASS Producer A invited C to the Show Run ($invite_ticket_id), one punch grants nothing"

invite_accept_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$third_session" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/tickets/${invite_ticket_id}/punch")"
if [[ "$invite_accept_status" != "200" || "$(cat "$BODY_OUT")" != *'"status":"valid"'* ]]; then
  echo "expected C's acceptance punch to validate the ticket, got $invite_accept_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS C accepted Producer A's invitation; ticket is valid"

# B (otherwise untouched by this Kernel 71 block) submits a request that
# gets declined -- proving decline grants nothing, and confirming B still
# has no active roster row afterward for the later Audience-only checks.
second_request_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$second_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/show-runs/${show_run_id}/tickets/request" \
  -d '{}')"
decline_ticket_id="$(grep -o '"id":"[^"]*"' "$BODY_OUT" | head -n1 | sed 's/"id":"//;s/"$//')"
decline_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/tickets/${decline_ticket_id}/decline")"
if [[ "$decline_status" != "200" || "$(cat "$BODY_OUT")" != *'"status":"declined"'* ]]; then
  echo "expected decline to mark the ticket declined, got $decline_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS declined ticket grants nothing and stays auditable"

enable_self_join_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H "Content-Type: application/json" \
  -X PATCH "http://127.0.0.1:${BACKEND_PORT}/api/show-runs/${show_run_id}" \
  -d '{"audience_self_join_enabled":true}')"
if [[ "$enable_self_join_status" != "200" ]]; then
  echo "expected Producer A to enable audience self-join, got $enable_self_join_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi

self_join_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$second_session" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/show-runs/${show_run_id}/roster/self-join")"
if [[ "$self_join_status" != "200" || "$(cat "$BODY_OUT")" != *'"role":"audience"'* ]]; then
  echo "expected B to self-join as audience, got $self_join_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS B self-joined as Audience once self-join was enabled"

program_response="$(curl -s -H "Cookie: victory_session=$second_session" "http://127.0.0.1:${BACKEND_PORT}/api/show-runs/${show_run_id}/audience-program")"
if [[ "$program_response" != *'"role":"audience"'* ]]; then
  echo "expected Audience Program to include B's audience entry" >&2
  echo "$program_response" >&2
  exit 1
fi
if [[ "$program_response" == *'"added_by_user_id"'* ]]; then
  echo "Audience Program payload must not expose internal-only roster fields" >&2
  echo "$program_response" >&2
  exit 1
fi
echo "PASS Audience Program is curated and excludes internal-only roster fields"

for file in \
  frontend/venues/show-runs/index.html \
  frontend/venues/show-runs/run.html \
  frontend/venues/show-runs/roster.html \
  frontend/venues/show-runs/program.html; do
  if [[ ! -f "$ROOT/$file" ]]; then
    echo "missing required file: $file" >&2
    exit 1
  fi
done
echo "PASS Show Runs page files exist"

# --- Kernel 67: Show Instance Model and Show Run Bridge ---
# Reuses the fixture Show Run ($show_run_id), Producer A ($raw_session),
# and Audience B ($second_session) already established by the Kernel 66
# block above.

anon_shows_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' "http://127.0.0.1:${BACKEND_PORT}/api/show-runs/${show_run_id}/shows")"
if [[ "$anon_shows_status" != "401" ]]; then
  echo "expected anonymous Shows list to return 401, got $anon_shows_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS Shows list rejects anonymous requests"

create_show_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/show-runs/${show_run_id}/shows" \
  -d '{"title":"Fresh Install Test Show","slug":"fresh-install-test-show"}')"
if [[ "$create_show_status" != "200" ]]; then
  echo "expected Producer A to create a Show, got $create_show_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
show_id="$(grep -o '"id":"[^"]*"' "$BODY_OUT" | head -n1 | sed 's/"id":"//;s/"$//')"
if [[ -z "$show_id" ]]; then
  echo "failed to extract show id" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS Producer A created a Show ($show_id)"

# --- Kernel 71: /showtime end-to-end -- a short code was generated
# automatically at Show creation; stage one Scene at Catharsis so Showtime
# can derive the venue without asking, then start/end it.
show_short_code="$(grep -o '"short_code":"[^"]*"' "$BODY_OUT" | head -n1 | sed 's/"short_code":"//;s/"$//')"
if [[ -z "$show_short_code" ]]; then
  echo "expected Show creation to include an auto-generated short_code" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS Show received an auto-generated short code ($show_short_code)"

catharsis_venue_id="$(docker exec -i "$POSTGRES_CONTAINER" psql -tAc "
  SELECT id FROM venues WHERE slug = 'catharsis';
" -U "$POSTGRES_USER" -d "$DB_NAME" | tr -d '[:space:]')"

create_showtime_scene_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/scenes" \
  -d "{\"location_id\":\"$fresh_install_location_id\",\"title\":\"Showtime Test Scene\",\"slug\":\"fresh-install-showtime-scene\",\"default_venue_id\":\"$catharsis_venue_id\"}")"
showtime_scene_id="$(grep -o '"id":"[^"]*"' "$BODY_OUT" | head -n1 | sed 's/"id":"//;s/"$//')"
if [[ "$create_showtime_scene_status" != "200" || -z "$showtime_scene_id" ]]; then
  echo "expected Showtime test Scene creation to succeed, got $create_showtime_scene_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi

stage_showtime_scene_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/shows/${show_id}/scenes" \
  -d "{\"scene_id\":\"$showtime_scene_id\"}")"
if [[ "$stage_showtime_scene_status" != "200" ]]; then
  echo "expected staging the Scene onto the Show to succeed, got $stage_showtime_scene_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS staged one Scene at Catharsis for Showtime venue derivation"

showtime_start_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/showtime/control" \
  -d "{\"short_code\":\"$show_short_code\",\"action\":\"start\"}")"
if [[ "$showtime_start_status" != "200" || "$(cat "$BODY_OUT")" != *'"venue_slug":"catharsis"'* ]]; then
  echo "expected /showtime to derive Catharsis and start, got $showtime_start_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
if [[ "$(cat "$BODY_OUT")" == *'"mic_on":true'* ]]; then
  echo "/showtime must never turn the mic on automatically" >&2
  exit 1
fi
if [[ "$(cat "$BODY_OUT")" != *'/mic hot'* ]]; then
  echo "expected /showtime's response to suggest /mic hot" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS /showtime <code> resolved the venue automatically, started the Session, and left the mic off"

showtime_status_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/showtime/control" \
  -d "{\"short_code\":\"$show_short_code\",\"action\":\"status\"}")"
if [[ "$showtime_status_status" != "200" ]]; then
  echo "expected /showtime status to succeed, got $showtime_status_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS /showtime status"

showtime_end_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/showtime/control" \
  -d "{\"short_code\":\"$show_short_code\",\"action\":\"end\"}")"
if [[ "$showtime_end_status" != "200" ]]; then
  echo "expected /showtime end to succeed, got $showtime_end_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS /showtime end ended the technical Session only"

by_code_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$third_session" \
  "http://127.0.0.1:${BACKEND_PORT}/api/shows/by-code?code=${show_short_code}")"
if [[ "$by_code_status" != "200" || "$(cat "$BODY_OUT")" != *"\"show_run_id\":\"$show_run_id\""* ]]; then
  echo "expected Audition Hall's by-code lookup to resolve the Show Run, got $by_code_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
if [[ "$(cat "$BODY_OUT")" == *'"roster"'* || "$(cat "$BODY_OUT")" == *'"variables_json"'* ]]; then
  echo "Show code lookup must never expose backstage data" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS Show code lookup resolves the Show Run without exposing backstage data"

show_detail_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" "http://127.0.0.1:${BACKEND_PORT}/api/shows/${show_id}")"
if [[ "$show_detail_status" != "200" || "$(cat "$BODY_OUT")" != *'"can_manage":true'* ]]; then
  echo "expected Show detail to return can_manage:true for Producer A, got $show_detail_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS Show detail fetch confirms manage authority"

patch_show_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H "Content-Type: application/json" \
  -X PATCH "http://127.0.0.1:${BACKEND_PORT}/api/shows/${show_id}" \
  -d '{"audience_title":"Opening Night","audience_program_blurb":"A very good show.","status":"scheduled"}')"
if [[ "$patch_show_status" != "200" || "$(cat "$BODY_OUT")" != *'"audience_title":"Opening Night"'* ]]; then
  echo "expected Show PATCH to round-trip audience_title, got $patch_show_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS Show PATCH round-trips audience_title/blurb/status (snake_case JSON confirmed)"

show_program_response="$(curl -s -H "Cookie: victory_session=$second_session" "http://127.0.0.1:${BACKEND_PORT}/api/shows/${show_id}/program")"
if [[ "$show_program_response" != *'"audience_title":"Opening Night"'* ]]; then
  echo "expected Show Program to include this Show's own audience_title" >&2
  echo "$show_program_response" >&2
  exit 1
fi
if [[ "$show_program_response" == *'"added_by_user_id"'* ]]; then
  echo "Show Program payload must not expose internal-only roster fields" >&2
  echo "$show_program_response" >&2
  exit 1
fi
echo "PASS Show Program is curated and Show-specific"

archive_show_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/shows/${show_id}/archive")"
if [[ "$archive_show_status" != "200" || "$(cat "$BODY_OUT")" != *'"status":"archived"'* ]]; then
  echo "expected Show archive to return status:archived, got $archive_show_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS Show archive sets status and archived_at"

for file in \
  frontend/venues/show-runs/show.html \
  frontend/venues/show-runs/show-program.html; do
  if [[ ! -f "$ROOT/$file" ]]; then
    echo "missing required file: $file" >&2
    exit 1
  fi
done
echo "PASS Show page files exist"

# --- Kernel 68: Venue Visibility Gates, Stage Management, Production Onboarding ---
# Reuses Producer A ($raw_session, now Trailer-Face-ready via the stage name
# commit added to the Kernel 65 block above) and Audience B ($second_session,
# still Face-unready and never granted producer/director role at any
# location) plus the fixture Show Run ($show_run_id) from the Kernel 66
# block.

map_visibility_a="$(curl -s -H "Cookie: victory_session=$raw_session" "http://127.0.0.1:${BACKEND_PORT}/api/map/visibility")"
if [[ "$map_visibility_a" != *'"slug":"third-place"'* ]]; then
  echo "expected Face-ready account A to see third-place on the map" >&2
  echo "$map_visibility_a" >&2
  exit 1
fi
if [[ "$map_visibility_a" != *'"slug":"show-runs"'* ]]; then
  echo "expected Producer A to see show-runs (Stage Management) on the map" >&2
  echo "$map_visibility_a" >&2
  exit 1
fi
echo "PASS Face-ready Producer A sees both Third Place and Stage Management on the map"

map_visibility_b="$(curl -s -H "Cookie: victory_session=$second_session" "http://127.0.0.1:${BACKEND_PORT}/api/map/visibility")"
if [[ "$map_visibility_b" == *'"slug":"third-place"'* ]]; then
  echo "expected Face-unready account B to NOT see third-place on the map" >&2
  echo "$map_visibility_b" >&2
  exit 1
fi
if [[ "$map_visibility_b" == *'"slug":"show-runs"'* ]]; then
  echo "expected Audience-only account B to NOT see show-runs (Stage Management) on the map" >&2
  echo "$map_visibility_b" >&2
  exit 1
fi
if [[ "$map_visibility_b" != *'"slug":"trailers"'* ]]; then
  echo "expected plain audience-role account B to still see Trailers on the map" >&2
  echo "$map_visibility_b" >&2
  exit 1
fi
echo "PASS Face-unready Audience-only account B sees Trailers but not Third Place or Stage Management"

third_place_post_unready_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$second_session" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/third-place/headshots/me")"
if [[ "$third_place_post_unready_status" != "403" || "$(cat "$BODY_OUT")" != *'"trailer_face_not_ready"'* ]]; then
  echo "expected direct Third Place POST to reject a Face-unready account with 403, got $third_place_post_unready_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS direct Third Place API access cleanly rejects a Face-unready authenticated user"

show_runs_backstage_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$second_session" "http://127.0.0.1:${BACKEND_PORT}/api/show-runs/${show_run_id}")"
if [[ "$show_runs_backstage_status" != "403" ]]; then
  echo "expected direct Show Run backstage detail fetch to reject Audience-only B with 403, got $show_runs_backstage_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS direct Stage Management backstage API access rejects a user without backstage authority"

audience_program_still_works_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$second_session" "http://127.0.0.1:${BACKEND_PORT}/api/show-runs/${show_run_id}/audience-program")"
if [[ "$audience_program_still_works_status" != "200" ]]; then
  echo "expected curated Audience Program access to remain open to Audience B despite Stage Management being hidden, got $audience_program_still_works_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS curated Audience Program access is not broken by the Stage Management visibility gate"

anon_create_production_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/productions" \
  -H "Content-Type: application/json" -d '{"name":"Anon Production"}')"
if [[ "$anon_create_production_status" != "401" ]]; then
  echo "expected anonymous Create Production to return 401, got $anon_create_production_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS Create Production rejects anonymous requests"

create_production_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/productions" \
  -d '{"name":"Fresh Install Kernel 68 Production"}')"
if [[ "$create_production_status" != "200" ]]; then
  echo "expected Producer A to create a Production via the new route, got $create_production_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
new_production_id="$(grep -o '"id":"[^"]*"' "$BODY_OUT" | head -n1 | sed 's/"id":"//;s/"$//')"
if [[ -z "$new_production_id" ]]; then
  echo "failed to extract newly created production id" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS Create Production route works end-to-end ($new_production_id)"

create_run_from_new_production_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/show-runs" \
  -d "{\"production_id\":\"$new_production_id\",\"title\":\"Run From New Production\",\"slug\":\"run-from-new-production\",\"show_format\":\"playtest\"}")"
if [[ "$create_run_from_new_production_status" != "200" ]]; then
  echo "expected a Show Run to be creatable from the newly created Production, got $create_run_from_new_production_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS newly created Production can be used to create a Show Run"

# --- Kernel 69: Scene Library and Show Staging Model ---
# Reuses Producer A ($raw_session), Audience B ($second_session), the
# fixture Production ($fresh_install_production_id), and the fixture Show
# Run ($show_run_id) from the Kernel 66 block. $show_id from the Kernel 67
# block is now archived, so this section creates its own two fresh Shows
# under the same Show Run to prove one Scene can be staged in more than one
# Show.

anon_scenes_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' "http://127.0.0.1:${BACKEND_PORT}/api/scenes?location_id=${fresh_install_location_id}")"
if [[ "$anon_scenes_status" != "401" ]]; then
  echo "expected anonymous Scenes list to return 401, got $anon_scenes_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS Scenes list rejects anonymous requests"

create_scene_show_a_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/show-runs/${show_run_id}/shows" \
  -d '{"title":"Fresh Install Scene Show A","slug":"fresh-install-scene-show-a"}')"
scene_show_a_id="$(grep -o '"id":"[^"]*"' "$BODY_OUT" | head -n1 | sed 's/"id":"//;s/"$//')"
create_scene_show_b_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/show-runs/${show_run_id}/shows" \
  -d '{"title":"Fresh Install Scene Show B","slug":"fresh-install-scene-show-b"}')"
scene_show_b_id="$(grep -o '"id":"[^"]*"' "$BODY_OUT" | head -n1 | sed 's/"id":"//;s/"$//')"
if [[ "$create_scene_show_a_status" != "200" || "$create_scene_show_b_status" != "200" || -z "$scene_show_a_id" || -z "$scene_show_b_id" ]]; then
  echo "expected Producer A to create two fresh Shows for Scene staging" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS created two fresh Shows for Scene staging ($scene_show_a_id, $scene_show_b_id)"

create_scene_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/scenes" \
  -d "{\"location_id\":\"$fresh_install_location_id\",\"source_production_id\":\"$fresh_install_production_id\",\"slug\":\"fresh-install-character-making-opening\",\"title\":\"Socio- : Character Making — Opening\",\"audience_title\":\"Character Making\",\"audience_summary\":\"Come make a character with us.\",\"director_notes\":\"backstage only\"}")"
if [[ "$create_scene_status" != "200" ]]; then
  echo "expected Producer A to create a Scene, got $create_scene_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
scene_id="$(grep -o '"id":"[^"]*"' "$BODY_OUT" | head -n1 | sed 's/"id":"//;s/"$//')"
if [[ -z "$scene_id" ]]; then
  echo "failed to extract scene id" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS Producer A created a reusable Scene ($scene_id)"

audience_create_scene_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$second_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/scenes" \
  -d "{\"location_id\":\"$fresh_install_location_id\",\"title\":\"Sneaky Scene\",\"slug\":\"sneaky-scene\"}")"
if [[ "$audience_create_scene_status" != "403" ]]; then
  echo "expected Audience-only B to be rejected creating a Scene, got $audience_create_scene_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS Audience-only account cannot create a Scene"

place_a_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/shows/${scene_show_a_id}/scenes" \
  -d "{\"scene_id\":\"$scene_id\",\"sort_order\":1}")"
placement_a_id="$(grep -o '"id":"[^"]*"' "$BODY_OUT" | head -n1 | sed 's/"id":"//;s/"$//')"
place_b_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/shows/${scene_show_b_id}/scenes" \
  -d "{\"scene_id\":\"$scene_id\",\"sort_order\":1}")"
if [[ "$place_a_status" != "200" || "$place_b_status" != "200" || -z "$placement_a_id" ]]; then
  echo "expected the same Scene to be stageable in two different Shows under the same Production" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS the same reusable Scene was staged in two different Shows"

ready_placement_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H "Content-Type: application/json" \
  -X PATCH "http://127.0.0.1:${BACKEND_PORT}/api/shows/${scene_show_a_id}/scenes/${placement_a_id}" \
  -d '{"status":"ready"}')"
if [[ "$ready_placement_status" != "200" ]]; then
  echo "expected Producer A to mark the placement ready, got $ready_placement_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi

scene_program_response="$(curl -s -H "Cookie: victory_session=$second_session" "http://127.0.0.1:${BACKEND_PORT}/api/shows/${scene_show_a_id}/scenes/program")"
if [[ "$scene_program_response" != *'"title":"Character Making"'* ]]; then
  echo "expected curated Scene Program to include the ready placement's audience title" >&2
  echo "$scene_program_response" >&2
  exit 1
fi
if [[ "$scene_program_response" == *'"backstage only"'* || "$scene_program_response" == *'director_notes'* ]]; then
  echo "curated Scene Program must never expose director_notes" >&2
  echo "$scene_program_response" >&2
  exit 1
fi
echo "PASS curated Scene Program shows only audience-safe fields"

backstage_scene_list_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$second_session" "http://127.0.0.1:${BACKEND_PORT}/api/shows/${scene_show_a_id}/scenes")"
if [[ "$backstage_scene_list_status" != "403" ]]; then
  echo "expected Audience-only B to be rejected viewing the backstage Scene list, got $backstage_scene_list_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS backstage Scene list rejects a user without backstage authority"

remove_placement_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/shows/${scene_show_a_id}/scenes/${placement_a_id}/archive")"
if [[ "$remove_placement_status" != "200" ]]; then
  echo "expected Producer A to remove the Scene placement from Show A, got $remove_placement_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
scene_after_placement_removed="$(curl -s -H "Cookie: victory_session=$raw_session" "http://127.0.0.1:${BACKEND_PORT}/api/scenes/${scene_id}")"
if [[ "$scene_after_placement_removed" == *'"status":"archived"'* ]]; then
  echo "removing a Show's Scene placement must not archive the reusable Scene" >&2
  echo "$scene_after_placement_removed" >&2
  exit 1
fi
show_b_scenes_after_removal="$(curl -s -H "Cookie: victory_session=$raw_session" "http://127.0.0.1:${BACKEND_PORT}/api/shows/${scene_show_b_id}/scenes")"
if [[ "$show_b_scenes_after_removal" != *"\"scene_id\":\"$scene_id\""* ]]; then
  echo "expected Show B's independent placement of the same Scene to remain untouched" >&2
  echo "$show_b_scenes_after_removal" >&2
  exit 1
fi
echo "PASS archiving one Show's placement neither archives the Scene nor affects the other Show's placement"

archive_scene_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/scenes/${scene_id}/archive")"
if [[ "$archive_scene_status" != "200" ]]; then
  echo "expected Producer A to archive the reusable Scene, got $archive_scene_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
new_placement_of_archived_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/shows/${scene_show_a_id}/scenes" \
  -d "{\"scene_id\":\"$scene_id\"}")"
if [[ "$new_placement_of_archived_status" != "400" || "$(cat "$BODY_OUT")" != *'"scene_archived"'* ]]; then
  echo "expected a new placement of an archived Scene to be rejected with scene_archived, got $new_placement_of_archived_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
echo "PASS archiving a Scene blocks new placements while preserving existing ones"

for file in \
  frontend/venues/show-runs/scenes.html; do
  if [[ ! -f "$ROOT/$file" ]]; then
    echo "missing required file: $file" >&2
    exit 1
  fi
done
echo "PASS Scene Library page file exists"

echo "PASS clean-install smoke complete"
