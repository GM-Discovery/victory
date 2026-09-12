#!/usr/bin/env bash
# Shared by setup-test-database.sh and reset-test-database.sh: boots the
# real backend once against the (already validated) TEST_DATABASE_URL,
# which applies every SQL migration itself (backend/internal/migrate) AND
# runs its Go-side Ensure*Surface startup bootstrap - see
# backend/cmd/victory/main.go. Several venues and tables (e.g. the
# "first-theater"/"catharsis"/"middle-school-stage" venues from
# internal/access.EnsureKernel16VenueSurface) are only ever created by that
# Go-side bootstrap, not by any SQL migration file. Every migration and
# every Ensure*Surface call is idempotent, so re-running this is always
# safe.
#
# Kernel 96: this used to also manually pre-apply every migration file via
# a raw psql loop, run BEFORE this boot step. That's no longer just
# redundant, it's actively wrong: migrations from 002 onward now attach
# their content to whichever location access.EnsureDefaultLocation (Go)
# marks is_default, and that only happens partway through THIS boot's own
# phased migration sequence (schema first, then EnsureDefaultLocation, then
# the rest) - pre-applying them separately would run them before any
# location is marked default, silently seeding nothing. The real backend
# boot is now the only thing that touches migrations at all.
#
# DEFAULT_LOCATION_SLUG/DEFAULT_LOCATION_NAME are set to the historical
# "amurray-family"/"amurray.family" fixture name below deliberately -
# dozens of existing DB-touching tests already hardcode looking for
# content under that specific location (Kernel 96 §4: "Tests may retain
# fixtures where clearly isolated"). This is that isolation boundary: an
# arbitrary, isolated choice of name for a disposable test database, not a
# leaked default for a real install - no different from naming it
# "test-lot", except every existing test already expects this one.
#
# Callers must have already run require_isolated_database and set db_name.

migrate_and_bootstrap_test_database() {
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
    DEFAULT_LOCATION_SLUG=amurray-family \
    DEFAULT_LOCATION_NAME=amurray.family \
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
