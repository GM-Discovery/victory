#!/usr/bin/env bash
set -euo pipefail

# Creates (if missing) and migrates the dedicated Kernel 64 test database
# named by TEST_DATABASE_URL. Every migration file under database/migrations
# is idempotent (IF NOT EXISTS / ON CONFLICT DO UPDATE), and the Go-side
# bootstrap this also runs (see lib-migrate-and-bootstrap.sh) is idempotent
# too, so running this repeatedly never destroys existing rows - it only
# fills in anything missing. For a full wipe-and-rebuild, use
# reset-test-database.sh instead.
#
# Usage:
#   TEST_DATABASE_URL=postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable \
#     scripts/test/setup-test-database.sh

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-victory-postgres}"
POSTGRES_USER="${POSTGRES_USER:-victory}"

# shellcheck source=./require-isolated-database.sh
source "$ROOT/scripts/test/require-isolated-database.sh"
require_isolated_database

db_name="${TEST_DATABASE_URL##*/}"
db_name="${db_name%%\?*}"

if docker exec -i "$POSTGRES_CONTAINER" psql -U "$POSTGRES_USER" -d postgres -tAc \
    "SELECT 1 FROM pg_database WHERE datname = '$db_name'" | grep -q 1; then
  echo "Test database already exists: $db_name"
else
  echo "Creating test database: $db_name"
  docker exec -i "$POSTGRES_CONTAINER" createdb -U "$POSTGRES_USER" "$db_name"
fi

# shellcheck source=./lib-migrate-and-bootstrap.sh
source "$ROOT/scripts/test/lib-migrate-and-bootstrap.sh"
migrate_and_bootstrap_test_database

echo "PASS: $db_name is set up, migrated, and Go-side bootstrapped."
