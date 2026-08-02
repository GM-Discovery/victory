# Victory Restore Runbook

**Purpose.** Restore Victory from an encrypted off-host backup. This is a trusted
server/operator operation — Victory exposes no restore control through the application itself
(Kernel 77 §1.4, §14.4).

**Always restore into an isolated environment first**, never directly over the live database,
unless the live database is already gone and there is nothing left to protect.

---

## 1. Locate a backup

```bash
rclone lsf victoryvtt-crypt:VictoryBackups/full/ | sort   # daily complete backups
rclone lsf victoryvtt-crypt:VictoryBackups/quick/ | sort  # 15-minute database-only points
```

Filenames are plaintext-decrypted by the crypt remote automatically; pick the most recent
`full-*` for a complete restore, or a `quick-*` if you specifically need the freshest possible
database state and can live without the asset files in that same archive.

---

## 2. Download and verify

```bash
RESTORE_DIR=/tmp/victory-restore-$(date +%s)
mkdir -p "$RESTORE_DIR/download" "$RESTORE_DIR/extracted"

LATEST=$(rclone lsf victoryvtt-crypt:VictoryBackups/full/ | sort | tail -1)
rclone copy "victoryvtt-crypt:VictoryBackups/full/$LATEST" "$RESTORE_DIR/download/" --checksum
tar -xzf "$RESTORE_DIR/download/$LATEST" -C "$RESTORE_DIR/extracted/"

cd "$RESTORE_DIR/extracted"/*/
sha256sum -c checksums.sha256
cat manifest.json
```

Do not proceed if `sha256sum -c` reports anything but `OK` for every file.

---

## 3. Create an isolated PostgreSQL instance

Never restore onto the live `victory-postgres` container's `victory` database directly. Use a
throwaway container on a different port:

```bash
docker run -d --name victory-restore-test \
  -e POSTGRES_PASSWORD=<any local test password> \
  -e POSTGRES_USER=victory -e POSTGRES_DB=postgres \
  -p 127.0.0.1:15432:5432 postgres:16-alpine

# wait for it:
until docker exec victory-restore-test pg_isready -U victory >/dev/null 2>&1; do sleep 1; done

# the database name must contain "test" or "fresh" -- see step 5's note on
# MIGRATE_DANGEROUSLY_SKIP_BACKUP, which refuses anything that doesn't.
docker exec victory-restore-test psql -U victory -d postgres -c "CREATE DATABASE victory_restore_test;"
```

---

## 4. Restore the database and assets

```bash
docker cp "$RESTORE_DIR/extracted"/*/database.dump victory-restore-test:/tmp/database.dump
docker exec victory-restore-test pg_restore --format=custom -U victory \
  -d victory_restore_test --no-owner --role=victory /tmp/database.dump

docker exec victory-restore-test psql -U victory -d victory_restore_test -tAc "SELECT COUNT(*) FROM users;"

mkdir -p "$RESTORE_DIR/storage"
cp -r "$RESTORE_DIR/extracted"/*/assets/. "$RESTORE_DIR/storage/" 2>/dev/null || true
```

`pg_dump`/`pg_restore` live inside the postgres image, not on the bare host — this is why the
commands above run through `docker exec`, matching how `backup.sh` produces the dump in the
first place.

---

## 5. Launch Victory against the restored state

```bash
cd /opt/victory/backend
DATABASE_URL="postgres://victory:<password>@127.0.0.1:15432/victory_restore_test?sslmode=disable" \
PORT=18099 \
STORAGE_ROOT="$RESTORE_DIR/storage" \
EXPORTS_ROOT="$RESTORE_DIR/exports" \
BACKUP_DIR="$RESTORE_DIR/pg-backups" \
OPERATOR_HANDLE=straturli \
SESSION_COOKIE_SECURE=false \
MIGRATE_DANGEROUSLY_SKIP_BACKUP=true \
go run ./cmd/victory
```

`MIGRATE_DANGEROUSLY_SKIP_BACKUP=true` is required because the host has no `pg_dump` on
`PATH` for the pre-migration safety backup, and is only honored when the database name looks
disposable (`test`/`fresh`) — the same guard `internal/dbtest` uses. This is exactly why step 3
named the database `victory_restore_test`.

If the backup predates a migration that has since shipped, you'll see it apply here — that's
expected and correct; it proves the restored state can catch up to current schema.

---

## 6. Verify

```bash
curl -s http://127.0.0.1:18099/health

cd /opt/victory/backend
DATABASE_URL="postgres://victory:<password>@127.0.0.1:15432/victory_restore_test?sslmode=disable" \
OPERATOR_HANDLE=straturli \
go run ./cmd/victory-recover whoami --handle straturli
```

`whoami` against the restored database is the "authenticate a test account" proof required by
Kernel 77 §9.11 — it exercises real account resolution (operator flag, Discord link,
memberships) without needing a browser or a real Discord OAuth round-trip.

Spot-check a few more tables against known live counts if you want extra confidence:

```bash
docker exec victory-restore-test psql -U victory -d victory_restore_test -tAc "
  SELECT 'locations', COUNT(*) FROM locations
  UNION ALL SELECT 'venues', COUNT(*) FROM venues
  UNION ALL SELECT 'equipment_items', COUNT(*) FROM equipment_items;
"
```

---

## 7. Tear down the isolated environment

```bash
kill <the go run PID>
docker rm -f victory-restore-test
rm -rf "$RESTORE_DIR"
```

Confirm the live containers are untouched: `docker ps` should still show only
`victory-backend`, `victory-postgres`, and whatever else was already running.

---

## 8. Restoring over the live database (last resort)

Only if the live database is actually gone or corrupted beyond repair — not for routine
verification, which is what sections 1–7 are for.

1. Stop the backend: `docker compose stop backend`.
2. Take a fresh `pg_dump` of whatever remains, even if it's broken — you may need it.
3. Drop and recreate the live database, or point `DATABASE_URL` at a new one.
4. Run steps 2 and 4 above against the **live** `victory-postgres` container and the real
   `victory` database name (the `MIGRATE_DANGEROUSLY_SKIP_BACKUP` disposable-name guard will
   correctly refuse to skip the backup step here — let it take one).
5. `docker compose up -d backend` and confirm `/health` and a real login.

This path has not been separately rehearsed in Kernel 77 beyond the isolated proof in sections
1–7; treat it as documented but not drilled, and prefer the isolated path whenever the live
database is merely being double-checked rather than actually lost.
