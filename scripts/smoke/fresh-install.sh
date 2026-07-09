#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/smoke/fresh-install.sh --local

Optional env:
  POSTGRES_CONTAINER=victory-postgres
  POSTGRES_USER=victory
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
docker compose up -d postgres >/dev/null
docker exec -i "$POSTGRES_CONTAINER" createdb -U "$POSTGRES_USER" "$DB_NAME"

migrations=(
  "$ROOT/database/migrations/000_kernel42_productions_baseline.sql"
  "$ROOT/database/migrations/001_init.sql"
  "$ROOT/database/migrations/002_seed_world.sql"
  "$ROOT/database/migrations/003_users_handle.sql"
  "$ROOT/database/migrations/004_seed_session.sql"
  "$ROOT/database/migrations/005_kernel2_identity_invites_assets.sql"
  "$ROOT/database/migrations/006_kernel6_action_authority.sql"
  "$ROOT/database/migrations/007_kernel9_profiles_greenroom_trailers.sql"
  "$ROOT/database/migrations/008_kernel10_info_booth_mailbox.sql"
  "$ROOT/database/migrations/009_kernel11_note_cards.sql"
  "$ROOT/database/migrations/010_kernel13_workshop_placement.sql"
  "$ROOT/database/migrations/011_kernel15_producer_director_offices.sql"
  "$ROOT/database/migrations/012_kernel15_production_label.sql"
  "$ROOT/database/migrations/013_element_context_class.sql"
  "$ROOT/database/migrations/014_kernel21_venue_chat.sql"
  "$ROOT/database/migrations/015_kernel22_showings.sql"
  "$ROOT/database/migrations/016_kernel23_character_cards.sql"
  "$ROOT/database/migrations/017_kernel24_character_sheet_links.sql"
  "$ROOT/database/migrations/018_kernel32_discord_oauth.sql"
  "$ROOT/database/migrations/019_kernel35_discord_server_link.sql"
  "$ROOT/database/migrations/020_kernel35_discord_server_bootstrap.sql"
  "$ROOT/database/migrations/021_kernel36_discord_channel_mappings.sql"
  "$ROOT/database/migrations/022_kernel37_discord_mic_threads.sql"
  "$ROOT/database/migrations/023_kernel38_discord_chat_bridges.sql"
  "$ROOT/database/migrations/024_kernel39_discord_gateway_intake.sql"
  "$ROOT/database/migrations/025_kernel42_neutral_install_location.sql"
  "$ROOT/database/migrations/026_kernel46_first_theater_map.sql"
  "$ROOT/database/migrations/027_kernel47_map_display_mode.sql"
  "$ROOT/database/migrations/028_kernel47_grid_config.sql"
  "$ROOT/database/migrations/029_kernel49_warehouse_storage.sql"
  "$ROOT/database/migrations/030_kernel51_capacity_guardrails.sql"
  "$ROOT/database/migrations/031_kernel53_character_workbook_foundation.sql"
  "$ROOT/database/migrations/032_kernel59_command_registry.sql"
  "$ROOT/database/migrations/033_kernel60_character_skills.sql"
  "$ROOT/database/migrations/034_kernel59a_character_face_overrides.sql"
  "$ROOT/database/migrations/035_kernel59a_director_value_overrides.sql"
  "$ROOT/database/migrations/036_kernel61_player_workbook_foundation.sql"
  "$ROOT/database/migrations/037_kernel62_player_relationships.sql"
)

for migration in "${migrations[@]}"; do
  echo "Applying $(basename "$migration")"
  docker exec -i "$POSTGRES_CONTAINER" psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$DB_NAME" < "$migration"
done
echo "PASS migrations from empty DB"

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
  DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/$DB_NAME?sslmode=disable" \
  STORAGE_ROOT="$ROOT/storage" \
  SESSION_COOKIE_SECURE=false \
  COOKIE_SECURE=false \
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
check_status "/api/discord/gateway/status" "200"

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
    DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/$DB_NAME?sslmode=disable" \
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
    DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/$DB_NAME?sslmode=disable" \
    go run ./cmd/victory-bootstrap producer --handle "$operator_handle"
)"
if [[ "$bootstrap_repeat" != *"Already existed: true"* ]]; then
  echo "$bootstrap_repeat" >&2
  exit 1
fi
echo "PASS bootstrap producer is idempotent"

bootstrap_legacy="$(
  cd "$BACKEND_DIR" && \
  env \
    DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/$DB_NAME?sslmode=disable" \
    go run ./cmd/victory-bootstrap producer --handle "$operator_handle" --location amurray-family
)"
if [[ "$bootstrap_legacy" != *"Location: amurray.family (amurray-family)"* && "$bootstrap_legacy" != *"Location: amurray-family"* ]]; then
  echo "$bootstrap_legacy" >&2
  exit 1
fi
echo "PASS bootstrap --location works with legacy compatibility location"

if cd "$BACKEND_DIR" && \
  env DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/$DB_NAME?sslmode=disable" \
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

echo "PASS clean-install smoke complete"
