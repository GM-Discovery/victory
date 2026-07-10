#!/usr/bin/env bash
set -euo pipefail

# DESTRUCTIVE: drops and recreates the dedicated Kernel 64 test database
# named by TEST_DATABASE_URL, then re-applies every migration from empty.
# This is the only script in the repo allowed to DROP a database. It must
# never be able to reach the live/shared victory database - it goes through
# require_isolated_database, plus one more explicit inline name check right
# before the drop, plus an explicit confirmation env var.
#
# Usage:
#   TEST_DATABASE_URL=postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable \
#     CONFIRM_TEST_DB_RESET=1 \
#     scripts/test/reset-test-database.sh

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-victory-postgres}"
POSTGRES_USER="${POSTGRES_USER:-victory}"

if [[ "${CONFIRM_TEST_DB_RESET:-}" != "1" ]]; then
  echo "Refusing to reset a database without CONFIRM_TEST_DB_RESET=1 set explicitly." >&2
  exit 1
fi

# shellcheck source=./require-isolated-database.sh
source "$ROOT/scripts/test/require-isolated-database.sh"
require_isolated_database

db_name="${TEST_DATABASE_URL##*/}"
db_name="${db_name%%\?*}"

# Belt-and-suspenders: require_isolated_database already rejects
# "victory"/"postgres"/production-looking names and anything not containing
# "test", but a DROP DATABASE gets one more explicit check immediately
# before it runs, independent of the sourced function.
lower_name="$(printf '%s' "$db_name" | tr '[:upper:]' '[:lower:]')"
if [[ -z "$lower_name" || "$lower_name" == "victory" || "$lower_name" == "postgres" || "$lower_name" == *"prod"* || "$lower_name" != *"test"* ]]; then
  echo "Refusing to drop database '$db_name' - does not look like a dedicated test database." >&2
  exit 1
fi

echo "Dropping test database (if it exists): $db_name"
docker exec -i "$POSTGRES_CONTAINER" dropdb -U "$POSTGRES_USER" --if-exists "$db_name"

echo "Recreating test database: $db_name"
docker exec -i "$POSTGRES_CONTAINER" createdb -U "$POSTGRES_USER" "$db_name"

# shellcheck source=./lib-migrate-and-bootstrap.sh
source "$ROOT/scripts/test/lib-migrate-and-bootstrap.sh"
migrate_and_bootstrap_test_database

echo "PASS: $db_name reset, migrated, and Go-side bootstrapped from empty."
