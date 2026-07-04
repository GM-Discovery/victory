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
)

for migration in "${migrations[@]}"; do
  echo "Applying $(basename "$migration")"
  docker exec -i "$POSTGRES_CONTAINER" psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$DB_NAME" < "$migration"
done
echo "PASS migrations from empty DB"

(
  cd "$BACKEND_DIR"
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
    go run ./cmd/victory >"$BACKEND_LOG" 2>&1
) &
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

echo "PASS clean-install smoke complete"
