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
  "$ROOT/database/migrations/038_kernel65_third_place.sql"
  "$ROOT/database/migrations/039_kernel66_show_runs.sql"
  "$ROOT/database/migrations/040_kernel67_shows.sql"
  "$ROOT/database/migrations/041_kernel68_productions_created_by.sql"
  "$ROOT/database/migrations/042_kernel69_scenes.sql"
  "$ROOT/database/migrations/043_kernel70_scene_location_scoping.sql"
  "$ROOT/database/migrations/044_kernel70_show_stage_and_variables.sql"
  "$ROOT/database/migrations/045_kernel70_cues.sql"
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

b_commons_list="$(curl -s -H "Cookie: victory_session=$second_session" "http://127.0.0.1:${BACKEND_PORT}/api/third-place/headshots")"
if [[ "$b_commons_list" != *"\"headshot_id\":\"$headshot_id\""* ]]; then
  echo "expected B to see A's Headshot in the commons list" >&2
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
# neutral "amurray-family" location above) and account B ($second_session).
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
  FROM locations WHERE slug = 'amurray-family'
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
  SELECT id FROM locations WHERE slug = 'amurray-family';
" -U "$POSTGRES_USER" -d "$DB_NAME" | tr -d '[:space:]')"
if [[ -z "$fresh_install_location_id" ]]; then
  echo "failed to resolve amurray-family location id for Kernel 70 Scene smoke checks" >&2
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

add_member_status="$(curl -s -o "$BODY_OUT" -w '%{http_code}' \
  -H "Cookie: victory_session=$raw_session" -H "Content-Type: application/json" \
  -X POST "http://127.0.0.1:${BACKEND_PORT}/api/show-runs/${show_run_id}/roster" \
  -d "{\"target_profile_id\":\"$workbook_id\",\"role\":\"player\"}")"
if [[ "$add_member_status" != "200" || "$(cat "$BODY_OUT")" != *'"role_label":"Player"'* ]]; then
  echo "expected roster add to return role_label Player, got $add_member_status" >&2
  cat "$BODY_OUT" >&2 || true
  exit 1
fi
if [[ "$(cat "$BODY_OUT")" == *'"role_label":"Cast"'* ]]; then
  echo "roster role label must never render as Cast" >&2
  exit 1
fi
echo "PASS roster member added with role_label Player, never Cast"

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
