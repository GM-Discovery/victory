# Kernel Report Back — Kernel 64: DB Test Isolation and Live-DB Safety Gate

**Kernel spec:** operator-issued brief (this kernel has no separate pre-written spec document; scope, acceptance criteria, and the five resolved operator decisions are recorded verbatim in the operator's own kernel brief for this session)
**Commit(s):** uncommitted (operator will review and commit; matches the precedent set for Kernels 62/63)
**Date:** 2026-07-10

## 1. Status

**PASS**

`go test ./...` now requires a dedicated `TEST_DATABASE_URL` for every DB-touching package and hard-fails clearly without one. Destructive reset logic is gated so it can never reach the live/shared `victory` database. All three previously-hardcoded-to-live-DB test factories now go through the gate. The one known pre-existing baseline failure (`internal/assets`) is fixed as a contained test-fixture bug. Full `go test ./...` is green with `TEST_DATABASE_URL` set, with zero unrelated/unexplained failures.

## 2. What was built

### The safety gate (Go + shell, kept in sync by convention)

- `backend/internal/dbtest/dbtest.go` (new package): `ValidateTestDatabaseURL(testDatabaseURL, liveDatabaseURL string) error` is the single Go-side rule set. Rejects: empty, equal to `DATABASE_URL`, database name empty/`victory`/`postgres`, database name containing `prod`, and database name not containing `test`. `OpenTestPool(t *testing.T) *pgxpool.Pool` is the standard entry point for DB-touching tests — it reads `TEST_DATABASE_URL`, validates it, and `t.Fatalf`s (never `t.Skip`s) on any failure, then opens and pings the pool and registers `t.Cleanup(pool.Close)`.
- `scripts/test/require-isolated-database.sh` (existing file, rewritten): the equivalent shell-side rule set as a sourceable `require_isolated_database` function. **Fixed a real logic bug in the previous version**: it rejected any `TEST_DATABASE_URL` whose host was `localhost`/`127.0.0.1`/`victory-postgres` outright — but the dedicated test database necessarily lives on the same (only) Postgres server as the live app database, just under a different name. The discriminator is now the database *name*, not the host.
- `scripts/test/setup-test-database.sh` (new): idempotent, non-destructive. Creates `victory_test` if missing, applies every SQL migration, then builds and briefly boots the real `victory` binary against it so the Go-side `Ensure*Surface` bootstrap functions (`cmd/victory/main.go`) run too — several venues (`first-theater`, `catharsis`, `middle-school-stage`, `warehouse`, `workshop`, etc.) are only ever created by that Go bootstrap, not by any SQL migration file. Discovered this the hard way: a database with only the SQL migrations applied was missing them, and several DB-touching tests genuinely need them.
- `scripts/test/reset-test-database.sh` (new): the one destructive operation in this kernel — drops and recreates the dedicated test database from empty, then re-runs the same migrate+bootstrap sequence. Requires `TEST_DATABASE_URL` to pass `require_isolated_database`, requires `CONFIRM_TEST_DB_RESET=1` explicitly, and does one more inline name check immediately before the `dropdb` call, independent of the sourced function (defense in depth on the only command in the repo allowed to drop a database).
- `scripts/test/lib-migrate-and-bootstrap.sh` (new): the shared migrate+bootstrap logic, sourced by both scripts above so it exists in exactly one place.

### Wiring the three DB-touching test factories

Found via full audit (`grep` for `pgxpool`/`db.NewPool`/`sql.Open` across every `_test.go`): exactly three factory functions, each previously hardcoding `postgres://victory:REDACTED@127.0.0.1:5432/victory?sslmode=disable` — the live app database — directly in test source, with no env var read at all, falling back to `t.Skip` (not fail) if unreachable.

- `internal/identity/discord_oauth_test.go`: `openDiscordTestPool` (also used by 7 more files in the same package: `account_test.go`, `bootstrap_test.go`, `discord_server_link_test.go`, `discord_audio_test.go`, `discord_channel_mapping_test.go`, `discord_gateway_test.go`, `discord_mic_test.go`; `account_email_test.go` in the same package is pure/unit)
- `internal/network/discord_gateway_test.go`: `openDiscordGatewayTestPool` (also used by `discord_chat_bridge_test.go`; `dice_roll_test.go`/`kernel7_integration_test.go`/`session_control_test.go`/`presence_test.go`/`director_console_test.go`/`commands_http_test.go`/`char_skills_http_test.go` in the same package are pure/unit)
- `internal/assets/warehouse_test.go`: `openWarehouseTestPool`

All three now delegate to `dbtest.OpenTestPool(t)`. No other package opens a database connection in tests — every other `_test.go` in the repo is pure/unit (confirmed by grep, spot-checked several files that merely *reference* seeded slugs like `amurray-family` as string literals in pure-function tests with a `nil` pool).

### Fixed the known pre-existing baseline failure (contained test-fixture bug, as predicted)

`internal/assets/warehouse_test.go`'s `TestLoadWarehouseStorageStatsKeepsDatabaseUsageWhenFilesystemLookupFails` called `loadWarehouseStorageStats(ctx, pool, "/path/that/does/not/exist")` — but that function's second argument is a location UUID (`WHERE a.location_id = $1::uuid`), not a filesystem path; `loadWarehouseFilesystemStats` (a separate, non-DB function) is what takes a storage-root path, and the two are never combined inside `loadWarehouseStorageStats` itself. The test always failed the `::uuid` cast, on any database, live or isolated — it also implicitly depended on the live DB already having real asset rows for whatever location it *would* have resolved, since it asserted `TotalStoredBytes > 0`/`ActiveAssets > 0` with no fixture of its own.

Fixed by creating a real, self-contained fixture: a fresh location + user + one `assets` row with a known `stored_bytes` value, cleaned up via `t.Cleanup`. The test now asserts the exact known value rather than merely "> 0", which is a stronger assertion than the original.

### Two more fixture bugs surfaced only once tests ran against a genuinely fresh database

These didn't fail against the live DB (which has extra state never captured in any migration), so they were invisible until `victory_test` was built from empty migrations:

1. `internal/network/discord_chat_bridge_test.go`'s `setupDiscordChatBridgeFixture` assumed a venue with `slug = 'first-theater'` under `amurray-family` — genuinely real on the live DB, but created there by `internal/access.EnsureKernel16VenueSurface` (Go bootstrap), which I hadn't yet wired into the test-DB setup at that point in the session. Rather than depend on ordering between the SQL migrations and the Go bootstrap step, switched the fixture to `the-cave` (seeded directly by `database/migrations/002_seed_world.sql`, and equally eligible for chat-bridge mirroring per `discordChatBridgeEligibleVenues`).
2. The same fixture also assumed a `productions` row already existed for `amurray-family` (`showings.EnsureForSession` requires one, returns `production_required` otherwise) — real on live DB but created out-of-band, not by any migration or bootstrap code I could find. Added an idempotent `INSERT ... ON CONFLICT (location_id, slug) DO NOTHING` to the fixture itself.

Both are genuinely contained test-fixture fixes (mirroring the existing self-seeding pattern already used by `ensureDiscordServerTestSchema` for `producers-office`/`directors-chair`), not scope creep — they were required to make `go test ./...` "meaningful again" against an isolated database, which is this kernel's whole point.

## 3. Evidence (MANDATORY)

### Automated checks
```
go build ./...   → OK
go vet ./...     → OK
gofmt -l <touched .go files>  → clean
git diff --check → clean
```

### Full go test ./... with TEST_DATABASE_URL set (DATABASE_URL unset)
```
$ unset DATABASE_URL
$ GOCACHE=/tmp/victory-gocache TEST_DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable" go test -count=1 ./...
ok  	victory/backend/cmd/victory
?   	victory/backend/cmd/victory-bootstrap	[no test files]
ok  	victory/backend/internal/access
ok  	victory/backend/internal/actions
ok  	victory/backend/internal/assets      ← previously the one known baseline failure, now green
ok  	victory/backend/internal/characters
ok  	victory/backend/internal/commands
?   	victory/backend/internal/db	[no test files]
?   	victory/backend/internal/dbtest	[no test files]
ok  	victory/backend/internal/dice
ok  	victory/backend/internal/identity
ok  	victory/backend/internal/messages
ok  	victory/backend/internal/network
ok  	victory/backend/internal/playerprofile
ok  	victory/backend/internal/playerrelationships
ok  	victory/backend/internal/profiles
?   	victory/backend/internal/sessions	[no test files]
ok  	victory/backend/internal/showings
ok  	victory/backend/internal/venues
?   	victory/backend/internal/world	[no test files]
```
Zero failures, zero known-unrelated exceptions to disclaim. Re-ran a second consecutive time (`-count=1` to defeat Go's test cache) — identical result, proving repeatability rather than a one-time pass.

### Proof: missing TEST_DATABASE_URL hard-fails (not skips)
```
$ unset TEST_DATABASE_URL
$ go test ./internal/assets/... -run TestLoadWarehouseStorageStatsKeepsDatabaseUsageWhenFilesystemLookupFails -v
=== RUN   TestLoadWarehouseStorageStatsKeepsDatabaseUsageWhenFilesystemLookupFails
    warehouse_test.go:42: dbtest: TEST_DATABASE_URL is required for database-touching tests (see Construction/kernel-maker-field-guide.md, Backend Test Commands)
--- FAIL: TestLoadWarehouseStorageStatsKeepsDatabaseUsageWhenFilesystemLookupFails (0.00s)
FAIL
```
`go test ./...` with no `TEST_DATABASE_URL` at all: every DB-touching test in `internal/identity`/`internal/network`/`internal/assets` fails with this same clear message; every pure/unit package still passes.

### Proof: unsafe TEST_DATABASE_URL is rejected
```
TEST_DATABASE_URL=...victory  (the live DB name)         → "must not point at the live/shared database \"victory\""
TEST_DATABASE_URL=...victory_production                  → "must not point at a production-looking database \"victory_production\""
TEST_DATABASE_URL == DATABASE_URL (both victory_test)     → "must not be the same as DATABASE_URL (the live app database)"
```
Same three cases proven at the shell-script layer (`scripts/test/require-isolated-database.sh`), plus: `reset-test-database.sh` refuses to run without `CONFIRM_TEST_DB_RESET=1`, and refuses even *with* that flag set if `TEST_DATABASE_URL` points at the live database (both checks independently reject it).

### Proof: the dedicated test database does not touch live/shared rows
Live DB row counts, before and after a full `go test -count=1 ./internal/identity/... ./internal/network/... ./internal/assets/...` run against `victory_test` (`DATABASE_URL` unset throughout):
```
            tbl             | count
-----------------------------+-------
 auth.discord_server_links  |     0
 location_memberships       |    33
 player_profile_workbooks   |    26
 player_relationships       |     1
 sessions                   |    12
 users                      |    27
```
Identical before and after (`diff` of the two snapshots: no output). Straturli's `users` row independently re-confirmed present and resolvable (`SELECT handle FROM users WHERE handle = 'straturli'` → 1 row) after all test runs in this session.

**One incidental live-DB touch, fully self-reverted**: early in the audit (before any code changes), I ran `TestDiscordAudioStatusTracksMappedVoiceChannels` once against the live DB via the *original* hardcoded pool, specifically to observe real pre-change behavior (this test's `DELETE FROM auth.discord_server_link_settings/discord_server_links/discord_channel_mappings WHERE location_id=$1` pattern is exactly the one that caused the Discord-config incident named in project memory). Verified before running that these three tables were already at 0 rows live (per Kernel 63's same-day finding), so the deletes were no-ops; the test's own `t.Cleanup` then deleted the `audio_operator` user and all rows it inserted. Confirmed post-hoc: all three tables still at 0 rows, no `audio_operator` user residue. No further test runs in this session used the live DB — every run after the first used the isolated `victory_test` database via the new gate.

### fresh-install.sh --local
```
$ scripts/smoke/fresh-install.sh --local
...
PASS migrations from empty DB
PASS backend starts
...
PASS clean-install smoke complete
```
Full pass, all 28 assertions including the Kernel 61/61A/62 ones. Confirmed no orphaned `victory_fresh_*` database and no stray process left on its port afterward. This script is untouched by this kernel — it manages its own disposable database and never reads `TEST_DATABASE_URL`.

## 4. How to run

```bash
cd /opt/victory
TEST_DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable" \
  scripts/test/setup-test-database.sh    # one-time (or after a schema change), non-destructive

cd backend
GOCACHE=/tmp/victory-gocache \
  TEST_DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable" \
  go test ./...
```

## 5. Operator notes

See `operator-notes.md`'s new "Kernel 64" section: the host-vs-name safety-gate fix, why the Go-side bootstrap boot step is necessary, and the two fixture dependency gaps found only by testing against a truly empty database.

## 6. Blockers and workarounds

None. The two additional fixture gaps (venue seeding, production seeding) were found and fixed within scope — see §2.

## 7. Deviations from kernel

- **Fixed two more test-fixture bugs beyond the one named in the brief** (`internal/network/discord_chat_bridge_test.go`'s `first-theater`-venue and missing-production assumptions). Both are the same class of bug as the one the brief explicitly asked me to investigate (a test that only worked by accident against the live database's un-tracked organic state), and both were necessary for `go test ./...` to actually pass against an isolated database — leaving them broken would have meant claiming PASS on a kernel whose entire point is "make DB tests trustworthy" while two of them still silently depended on the live DB's history. Flagged here rather than silently expanded.
- **Added a Go-side "boot the real binary once" bootstrap step** to the test-DB setup scripts, which the original scope didn't explicitly anticipate (it described "apply migrations or reset schema safely"). Necessary because several venues genuinely don't exist without it — discovered this empirically when `TestDiscordAudioStatusTracksMappedVoiceChannels` failed with `venue_not_found` against a migrations-only database.
- **`scripts/test/require-isolated-database.sh` existed but was unused** (confirmed via repo-wide grep for callers) and had a logic bug (host-based rejection instead of name-based) that would have made it actively unusable for this kernel's own dedicated-test-database approach. Fixed it rather than replacing it, since decision #2 pointed at reusing this exact file.

## 8. Known issues

- None new. The stale-test-user sweep (~25 older fixture users on the live DB, named in Kernel 63's reportback) remains explicitly out of scope per this kernel's decision #3, untouched.
- The Go-side (`backend/internal/dbtest`) and shell-side (`scripts/test/require-isolated-database.sh`) safety-gate rule sets are two independent implementations of the same rules, not a single shared source of truth (Go tests can't easily shell out per-test, and the shell scripts run before any Go process exists). Both are commented pointing at each other; if the rules ever need to change, both must be updated together.

## 9. Next recommended step

Two independent candidates, either useful to unblock next:
1. **Stale test-user sweep** — the ~25 older fixture users on the live DB (`tester1`, `kernel49test...`, etc.), explicitly out of scope for Kernels 63 and 64.
2. **CI wiring** — there's no CI pipeline in this repo today; if one is ever added, it should provision `victory_test` (via `scripts/test/setup-test-database.sh`) as a setup step so `go test ./...` is meaningful in CI the same way it now is locally.

## 10. Files changed or created

Created:
- `backend/internal/dbtest/dbtest.go`
- `scripts/test/setup-test-database.sh`
- `scripts/test/reset-test-database.sh`
- `scripts/test/lib-migrate-and-bootstrap.sh`
- `Construction/OperatorLogs/kernel-64-reportback.md`

Modified:
- `scripts/test/require-isolated-database.sh` (host-based check replaced with name-based check)
- `backend/internal/identity/discord_oauth_test.go`, `backend/internal/network/discord_gateway_test.go`, `backend/internal/assets/warehouse_test.go` (pool factories now use `dbtest.OpenTestPool`)
- `backend/internal/assets/warehouse_test.go` (fixture fix for the known baseline failure)
- `backend/internal/network/discord_chat_bridge_test.go` (two fixture dependency fixes)
- `Construction/kernel-maker-field-guide.md` (Backend Test Commands, new Kernel 64 Notes section, checklist item 10)
- `Construction/workflow/dev-workflow.md` (Common Checks, new Database Changes subsection)
- `Construction/reportbacktemplate.txt` (DB isolation proof now mandatory for kernels touching DB tests)
- `Construction/OperatorLogs/operator-log.md`, `operator-notes.md`

## 11. Project-memory updates completed

- [x] Kernel 64 reportback saved
- [x] operator-log updated
- [x] operator-notes updated
- [x] field guide updated (testing workflow changed this pass)
- [x] dev-workflow updated (developer commands changed this pass)
- [x] reportback template updated (DB isolation proof now mandatory going forward)
- [x] next kernel recommendations recorded above
