#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${TEST_DATABASE_URL:-}" ]]; then
  echo "TEST_DATABASE_URL is required for database-writing tests." >&2
  exit 1
fi

if [[ -n "${DATABASE_URL:-}" && "${TEST_DATABASE_URL}" == "${DATABASE_URL}" ]]; then
  echo "TEST_DATABASE_URL must not match DATABASE_URL." >&2
  exit 1
fi

lower_url="$(printf '%s' "$TEST_DATABASE_URL" | tr '[:upper:]' '[:lower:]')"

if [[ "$lower_url" == *"production"* || "$lower_url" == *"prod"* ]]; then
  echo "TEST_DATABASE_URL must not point at a production-looking database." >&2
  exit 1
fi

if [[ "$lower_url" == *"localhost"* || "$lower_url" == *"127.0.0.1"* || "$lower_url" == *"victory-postgres"* ]]; then
  echo "TEST_DATABASE_URL must point at an isolated test database, not the live app database." >&2
  exit 1
fi

echo "TEST_DATABASE_URL looks isolated."
