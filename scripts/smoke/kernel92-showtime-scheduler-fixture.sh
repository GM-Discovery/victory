#!/usr/bin/env bash
# Kernel 92 browser proof fixture: disposable Director account (fixture-row
# insertion, per Construction/Process/kernel-maker-field-guide.md's "Two-Browser /
# Live-Update Verification Technique" -- production password signup is
# closed) + a disposable Production/Show Run under the real amurray-family
# location, so the popup's Show Run picker has something real to select.
set -euo pipefail
cd /opt/victory

PGPASSWORD="${POSTGRES_PASSWORD:?set POSTGRES_PASSWORD}"
export PGPASSWORD
PSQL="docker exec -i victory-postgres psql -U victory -d victory -qtA"

SUFFIX="k92proof_$(date +%s)"
HANDLE="director_${SUFFIX}"
RAW_TOKEN=$(node -e "console.log(require('crypto').randomBytes(32).toString('base64url'))")
TOKEN_HASH_HEX=$(node -e "console.log(require('crypto').createHash('sha256').update(process.argv[1]).digest('hex'))" "$RAW_TOKEN")

USER_ID=$($PSQL -c "INSERT INTO users (handle, display_name) VALUES ('${HANDLE}', 'K92 Proof Director') RETURNING id;")
LOCATION_ID=$($PSQL -c "SELECT id FROM locations WHERE slug = 'amurray-family';")

$PSQL -c "INSERT INTO location_memberships (location_id, user_id, role, active) VALUES ('${LOCATION_ID}', '${USER_ID}', 'producer', TRUE);" >/dev/null

PRODUCTION_ID=$($PSQL -c "INSERT INTO productions (location_id, name, slug) VALUES ('${LOCATION_ID}', 'K92 Proof Production ${SUFFIX}', 'k92-proof-production-${SUFFIX}') RETURNING id;")
SHOW_RUN_ID=$($PSQL -c "INSERT INTO show_runs (location_id, production_id, title, slug, created_by_user_id) VALUES ('${LOCATION_ID}', '${PRODUCTION_ID}', 'K92 Proof Show Run ${SUFFIX}', 'k92-proof-show-run-${SUFFIX}', '${USER_ID}') RETURNING id;")

$PSQL -c "INSERT INTO auth.sessions (user_id, token_hash, expires_at, user_agent) VALUES ('${USER_ID}', decode('${TOKEN_HASH_HEX}','hex'), NOW() + interval '2 hours', 'kernel92-proof');" >/dev/null

cat > /tmp/k92_fixture.json <<JSON
{"token": "${RAW_TOKEN}", "showRunID": "${SHOW_RUN_ID}", "nickname": "K92 Proof Showing ${SUFFIX}", "userID": "${USER_ID}", "productionID": "${PRODUCTION_ID}", "showRunSlug": "k92-proof-show-run-${SUFFIX}"}
JSON

echo "Fixture ready: $(cat /tmp/k92_fixture.json)"
