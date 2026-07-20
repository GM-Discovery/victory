#!/usr/bin/env bash
# Shared by setup-test-database.sh and reset-test-database.sh: applies every
# SQL migration, then boots the real backend once against the (already
# validated) TEST_DATABASE_URL purely so its Go-side Ensure*Surface startup
# bootstrap runs - see backend/cmd/victory/main.go. Several venues and
# tables (e.g. the "first-theater"/"catharsis"/"middle-school-stage" venues
# from internal/access.EnsureKernel16VenueSurface) are only ever created by
# that Go-side bootstrap, not by any SQL migration file, so a database that
# only has the SQL migrations applied is missing them. Every Ensure*Surface
# call is idempotent, so re-running this is always safe.
#
# Callers must have already run require_isolated_database and set db_name.

migrate_and_bootstrap_test_database() {
  echo "Applying migrations to $db_name..."
  for migration in "$ROOT"/backend/migrations/*.sql; do
    echo "  $(basename "$migration")"
    docker exec -i "$POSTGRES_CONTAINER" psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$db_name" < "$migration"
  done

  local bootstrap_port="${TEST_DB_BOOTSTRAP_PORT:-18082}"
  local bootstrap_bin="${TMPDIR:-/tmp}/victory-test-db-bootstrap-bin"
  local bootstrap_log="${TMPDIR:-/tmp}/victory-test-db-bootstrap.log"
  local bootstrap_storage_root
  bootstrap_storage_root="$(mktemp -d)"

  echo "Booting the real backend once against $db_name to run Go-side Ensure*Surface bootstrap..."
  (cd "$ROOT/backend" && GOCACHE=/tmp/victory-gocache go build -o "$bootstrap_bin" ./cmd/victory)

  env \
    PORT="$bootstrap_port" \
    DATABASE_URL="$TEST_DATABASE_URL" \
    MIGRATE_DANGEROUSLY_SKIP_BACKUP=1 \
    STORAGE_ROOT="$bootstrap_storage_root" \
    SESSION_COOKIE_SECURE=false \
    COOKIE_SECURE=false \
    OPERATOR_HANDLE=victory_test_db_bootstrap \
    DISCORD_OAUTH_ENABLED=false \
    DISCORD_SERVER_LINK_ENABLED=false \
    DISCORD_GATEWAY_ENABLED=false \
    "$bootstrap_bin" >"$bootstrap_log" 2>&1 &
  local bootstrap_pid=$!

  local health_url="http://127.0.0.1:${bootstrap_port}/health"
  local bootstrap_ok=0
  for _ in $(seq 1 60); do
    if curl -fsS "$health_url" >/dev/null 2>&1; then
      bootstrap_ok=1
      break
    fi
    sleep 1
  done

  kill "$bootstrap_pid" >/dev/null 2>&1 || true
  wait "$bootstrap_pid" >/dev/null 2>&1 || true
  if command -v lsof >/dev/null 2>&1; then
    lsof -ti "tcp:${bootstrap_port}" 2>/dev/null | xargs -r kill -9 2>/dev/null || true
  fi
  rm -rf "$bootstrap_storage_root"

  if [[ "$bootstrap_ok" != "1" ]]; then
    echo "Go-side bootstrap failed to reach /health; logs:" >&2
    tail -n 50 "$bootstrap_log" >&2 || true
    return 1
  fi

  return 0
}
