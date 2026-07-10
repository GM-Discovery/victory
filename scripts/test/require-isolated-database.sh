#!/usr/bin/env bash
set -euo pipefail

# Kernel 64 safety gate. This is the one place that decides whether a
# database is safe to write test data into or run a destructive
# reset/truncate/drop against. scripts/test/setup-test-database.sh and
# scripts/test/reset-test-database.sh both source this file rather than
# re-implementing the rules - do not add a second copy of this logic.
#
# The equivalent Go-side gate (used by `go test` directly, since a shell
# wrapper can't gate an individual test binary) is
# backend/internal/dbtest.ValidateTestDatabaseURL. Keep the two in sync if
# either changes.
#
# The discriminator is the DATABASE NAME, not the host: the dedicated test
# database lives on the same Postgres server as the live app database
# (there is only one Postgres instance in this deployment), just under a
# different, clearly-named database. An earlier version of this script
# rejected "localhost"/"127.0.0.1"/"victory-postgres" hosts outright, which
# would have made a same-host dedicated test database impossible to use -
# that check has been removed in favor of the name-based rule below.

require_isolated_database() {
  local url="${TEST_DATABASE_URL:-}"

  if [[ -z "$url" ]]; then
    echo "TEST_DATABASE_URL is required for database-touching tests/tooling." >&2
    return 1
  fi

  if [[ -n "${DATABASE_URL:-}" && "$url" == "${DATABASE_URL}" ]]; then
    echo "TEST_DATABASE_URL must not match DATABASE_URL (the live app database)." >&2
    return 1
  fi

  # Database name: the path segment after the last '/', before any
  # '?query' suffix.
  local db_name="${url##*/}"
  db_name="${db_name%%\?*}"
  local lower_name
  lower_name="$(printf '%s' "$db_name" | tr '[:upper:]' '[:lower:]')"

  if [[ -z "$lower_name" || "$lower_name" == "victory" || "$lower_name" == "postgres" ]]; then
    echo "TEST_DATABASE_URL must not point at the live/shared database (got database name '$db_name')." >&2
    return 1
  fi

  if [[ "$lower_name" == *"prod"* ]]; then
    echo "TEST_DATABASE_URL must not point at a production-looking database (got '$db_name')." >&2
    return 1
  fi

  if [[ "$lower_name" != *"test"* ]]; then
    echo "TEST_DATABASE_URL database name '$db_name' must contain 'test' to be recognized as a dedicated test database (e.g. victory_test)." >&2
    return 1
  fi

  echo "TEST_DATABASE_URL looks like a dedicated test database: $db_name" >&2
  return 0
}

# Allow this file to be both sourced (for the function, used by the other
# test-db scripts) and executed directly (as a standalone check).
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  require_isolated_database
fi
