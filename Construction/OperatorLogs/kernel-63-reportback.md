# Kernel Report Back — Kernel 63: Discord Test Fixture-Leak Cleanup and Back-to-Map Navigation Consistency

**Kernel spec:** operator-issued brief (this kernel has no separate pre-written spec document; scope, acceptance criteria, and both operator decisions are recorded verbatim in this reportback)
**Commit(s):** explicitly uncommitted (operator will review and commit; matches the precedent set for Kernel 62)
**Date:** 2026-07-09 / 2026-07-10

## 1. Status

**PASS**

Both parts of the kernel are complete and browser/test-proven. One pre-existing, unrelated `go test ./...` failure remains in `internal/assets` — confirmed via `git stash` to fail identically with none of this kernel's changes present, and named explicitly below rather than silently ignored.

## 2. Prerequisites confirmed with operator (before starting)

1. **Commit Kernel 62 first, as its own commit.** On starting this kernel I found the working tree already clean — the Kernel 62 work (all 44 files) was already committed as `3f907e9` prior to this session. No action was needed; verified via `git show --stat 3f907e9`.
2. **Live-DB cleanup authorized as scoped**: after the test fixes were verified to stop the leak, delete the confirmed leaked `discord_session_threads` row and `bridge_operator_testmirror...` fixture users + FK-linked rows. Completed — see §4.
3. Next unused kernel number verified via `ls Construction/Kernels/ Construction/OperatorLogs/*-reportback.md`: 62 was the highest in use, so this is **Kernel 63**.

## 3. What was built

### Part 1 — Discord test fixture-leak cleanup

Five test-fixture fixes plus one genuine production bug fix, all confirmed against live source before and after, not just agent-reported:

1. **Missing cleanup** (`internal/identity/discord_server_link_test.go`, `TestDiscordBootstrapReconcileRestoresMappingsAndMicCommand`): added the missing `DELETE FROM auth.discord_session_threads WHERE location_id=$1 AND venue_slug='first-theater'` to both the pre-delete block and `t.Cleanup`, mirroring the existing correct convention in `discord_mic_test.go`. This was the direct cause of `TestMirrorVictoryChatToDiscordPostsMessageAndPersistsBridgeRow`'s duplicate-key collision.
2. **Fixture completion** (same file): `channelsJSON` was missing the `catharsis` category/chat channel, all 3 required voice channels (`venueAudioChannelSpecs()`), and `parent_id` on the 4 core text channels — each gap independently caused an unexpected channel-creation POST once the previous gap was fixed and reconcile got further. Completed the fixture to represent a fully-provisioned guild (the realistic case), so reconcile finds everything by name rather than attempting creation.
3. **Stale literal** (`internal/identity/discord_channel_mapping_test.go`, `TestDiscordChannelMappingRepairCreatesAndReusesSkeleton`): the repeat-repair mapping-count assertion said `19`, never updated when `venueAudioChannelSpecs()` grew to 3 entries; the first-repair assertion two lines earlier already correctly said `21`. Fixed to `21`.
4. **Guild-ID-hardcoded mock** (same file): `discordMappingTransport.RoundTrip` only matched `/guilds/guild-1/channels`, but `TestDiscordAudioStatusIncludesVoiceParticipantsAndFeatureNotes` intentionally links a different guild (`guild-voice-1`) — generalized both the GET and POST match to guild-id-agnostic path-shape matching, safe for all 4 existing callers.
5. **Cleanup registered too late** (`internal/network/discord_chat_bridge_test.go`): `t.Cleanup` was registered by the *caller* after `setupDiscordChatBridgeFixture` returned, so a `t.Fatalf` partway through the helper (exactly what bug #1 caused, repeatedly, historically) skipped cleanup entirely — the direct source of the 12 leaked `bridge_operator_testmirror...` fixture users found live. Moved the cleanup into the top of the helper itself, closing over its named return values.

**Genuine production bug found and fixed** (not test drift — explicitly the exception the operator authorized: "avoid touching production Discord behavior unless the test exposes a real production bug"): `saveDiscordChannelMapping` (`internal/identity/discord_channel_mapping.go`) cast `created_by_user_id` directly to `::uuid` with no `NULLIF` guard (unlike the adjacent `parent_discord_channel_id` param, which has one). `ReconcileDiscordBootstrap` — the automatic startup reconcile goroutine wired into `main.go`, which runs on every real server boot — always calls this with an empty `createdBy` string, so **every single channel-mapping save on automatic startup reconcile has always failed silently** (captured into `summary.Failed`, never surfaced as an error) since this code shipped. Only the manual, authenticated `/api/discord/channel-mapping/repair` HTTP path (which passes a real user ID) ever worked. Fixed with a one-line `NULLIF($11, '')::uuid` addition, mirroring the existing pattern one parameter over.

Two more genuine test-only bugs surfaced once the above fixes let tests run further than they ever had before:
- **Test ordering assumption**: `TestDiscordAudioStatusIncludesVoiceParticipantsAndFeatureNotes` assumed `Participants[0]` would be the linked user by insertion order, but `DiscordAudioPresenceStore.Snapshot()` deliberately sorts participants alphabetically by display name for stable UI ordering — "Buddy" legitimately sorts before "Grant". Swapped the two index assertions to match the real, correct sort behavior (production code was not changed).
- **FK-violating literal**: `TestMirrorVictoryChatToDiscordPostsMessageAndPersistsBridgeRow` used a non-UUID placeholder `"action-mirror-1"` as an action ID, but `discord_chat_message_bridges.action_id` has a real foreign key to `actions(id)` — the fixture never inserted a matching `actions` row, so every upsert silently failed on the FK constraint. Added the missing `actions` row (cascade-deleted automatically via the existing session cleanup) and a real UUID literal.

### One-time live-DB cleanup

Confirmed via direct read: `auth.discord_server_links`/`auth.discord_channel_mappings` had 0 rows before cleanup (no live Discord config exists), so zero risk to real config. Deleted:
- The 1 leaked `discord_session_threads` row (scoped by exact `id`).
- 12 confirmed `bridge_operator_testmirrorvictorychattodiscordpostsmessageandpersistsbridgerow_*` fixture users (11 pre-existing + 1 added by this kernel's own initial reproduction step, before fixes landed) and their FK-linked `sessions`/`session_participants`/`location_memberships` rows.

Explicitly **not** touched: ~25 other older stale test users (`tester1`, `kernel49test...`, etc.) unrelated to these tests — named as a separate future cleanup candidate. Straturli and all real production data verified untouched throughout.

### Part 2 — Back-to-Map navigation consistency

New shared component `frontend/lib/back-to-map.js`, modeled directly on the existing `venue-account-badge.js` mount-detection pattern: idempotency guard, injects its own `<style>` (no dependency on venue CSS), a real `<a href="/" id="back-to-map-link">← Back to Map</a>` (keyboard-accessible, works without JS). Default mode mounts into `.launchbar`/`.toolbar`/`.header-right`/`.top-actions` if found, else floats top-left (`position:fixed`, avoiding collision with the account badge's top-right float). A `data-back-to-map="floating"` attribute on the script tag forces floating regardless of host detection, for pages whose obvious mount point is hover-hidden by default.

**Correction found during implementation** (not assumed from the initial survey): direct CSS reads confirmed First Theater and Catharsis's existing icon-only back-pill is **hover-hidden** by `.top-bar[data-open="false"] .header-right { opacity:0; visibility:hidden; display:none; }` — failing "visible without hover hunting" just like The Cave, Director's Chair, and Producer's Office. This kernel therefore touches **5** forced-floating venues, not the 3 implied by the original brief.

File groups:
- **Group A (untouched, already sufficient)**: `greenroom`, `warehouse`, `mailbox`, `login`, `signup`, `legal/privacy`, `legal/terms`, `audition-hall` — 8 pages with existing, always-visible, in-flow links.
- **Group B (default-mount script added)**: `account`, `workshop`, `grants-cabin`, `construction`, `victory-theater`, `trailers/{face,workbook,people,person,view}`, `middle-school-stage`, `stage-template` — 12 pages.
- **Group C (forced-floating script added)**: `the-cave`, `directors-chair`, `producers-office`, `first-theater`, `catharsis` — 5 pages, existing in-header links left untouched.

## 4. Evidence

### Automated checks

```
go build ./...   → OK
go vet ./...     → OK
gofmt -l <touched .go files>  → clean
node --check frontend/lib/back-to-map.js  → OK
node --check scripts/smoke/kernel63-back-to-map-browser.js  → OK
git diff --check → clean
```

### Discord fixture-leak proof (validation order, exactly as required)

1. `git stash` on the clean tree: all 4 originally-reported failures reproduced exactly as diagnosed; `git stash pop`.
2. Applied all fixes.
3. `go test ./internal/identity/... -run TestDiscordBootstrapReconcileRestoresMappingsAndMicCommand -v` → PASS (isolated).
4. `go test ./internal/identity/... -v` (full, unfiltered) → **ok** — this is exactly the run that originally hid two cascading failures behind filtered `-run` output; now clean.
5. `go test ./internal/network/... -v` (full, unfiltered) → **ok**.
6. `go test ./...` (whole repo) → only the pre-existing, unrelated `internal/assets` failure.
7. Re-ran steps 4–6 a second consecutive time back-to-back → identical results, proving repeatability (not a one-time pass).
8. Read-only `SELECT` after all runs confirmed no new leaked rows accumulated from the fixed test code.
9. One-time live-DB cleanup performed (§3), then `go test ./...` re-run once more → still clean.

**Pre-existing, unrelated failure, proven via `git stash`:**
```
--- FAIL: TestLoadWarehouseStorageStatsKeepsDatabaseUsageWhenFilesystemLookupFails (internal/assets)
    warehouse_test.go:35: load warehouse storage stats: ERROR: invalid input syntax for type uuid: "/path/that/does/not/exist"
```
Reproduces identically on the clean tree with none of this kernel's changes present. Not touched — outside this kernel's Discord/navigation scope. Named here per the acceptance criteria's explicit allowance.

### Back-to-Map browser proof

`NODE_PATH=/tmp/node_modules node scripts/smoke/kernel63-back-to-map-browser.js` against the live deployed stack: **10/10 assertion groups PASS**.

- Group A (8 pages) verified visible + functional on desktop.
- Group B (10 pages) verified visible + functional on **both** desktop and mobile viewport, via genuine `getComputedStyle`/`getBoundingClientRect` checks (not just DOM presence) — this is the exact check that would have caught the First Theater/Catharsis false-positive if it had been applied there first.
- 2 of the 12 Group B pages (`middle-school-stage`, `stage-template`) are **operator-only surfaces** — `access.ResolveVisibleVenues` never lists them for any non-operator role, confirmed by reading the SQL directly. Granting a disposable browser-proof account real operator status would require mutating the shared `OPERATOR_HANDLE` environment variable server-wide, which was correctly avoided as out of scope. Verified statically instead: served HTML confirmed to include the `back-to-map.js` script tag, combined with the direct CSS audit already performed (no hover-hiding rule on either page's `.header-right`).
- Group C (5 pages) verified visible + functional on both viewports, including the 2 newly-identified hover-hidden pages (First Theater, Catharsis).
- Click-through: one representative page per group correctly navigates to `/`.
- Keyboard accessibility: confirmed the link is a real, focusable `<a href="/">`.
- Regression: `#forbidden-screen` confirmed still hidden on normal loads; existing headers/identity chips unaffected.

Screenshots in `Construction/OperatorLogs/evidence/kernel-63/` (control page, Group B desktop+mobile, 5 Group C pages showing the floating pill with no collision against native headers).

### Two disposable throwaway accounts, both cleaned up

- Discord test-fixture accounts: 12 `bridge_operator_testmirror...*` rows deleted (§3).
- Browser-proof account(s): 4 `k63_navcheck_*` accounts created across debugging runs, each granted producer authority via `victory-bootstrap` to reach producer-gated pages — all 4 deleted after the proof passed (unlike Kernel 62's plain accounts, these held elevated privileges, so they were cleaned up rather than left as documented residue).

## 5. How to run

- Discord tests: `cd backend && GOCACHE=/tmp/victory-gocache go test ./...`
- Back-to-Map browser proof: `NODE_PATH=/tmp/node_modules node scripts/smoke/kernel63-back-to-map-browser.js` (creates and deletes its own disposable account; the account is auto-granted producer authority via `victory-bootstrap` for the two producer-gated Group B pages, and cleaned up on your next manual pass if you re-run and want to inspect it before it's removed).

## 6. Operator notes

See `operator-notes.md`'s new "Kernel 63" section: the `created_by_user_id` NULLIF production fix, the corrected First Theater/Catharsis hover-hidden finding, the `back-to-map.js` mount convention, and the operator-only-venue constraint on browser-proving Middle School Stage/Stage Template.

## 7. Blockers and workarounds

- Playwright reused from `/tmp/node_modules` (Kernel 62's install) via `NODE_PATH`.
- Middle School Stage / Stage Template required a workaround (static HTML check instead of full interactive render) since they're operator-only and mutating `OPERATOR_HANDLE` was correctly out of scope.

## 8. Deviations from kernel

- **5 Group C venues, not 3**: First Theater and Catharsis's native back-pill was found to be hover-hidden during implementation (not assumed from the initial survey) — flagged explicitly rather than silently expanded.
- **One production code change**: the `created_by_user_id` NULLIF fix in `discord_channel_mapping.go`. This is production code, not test code, but falls squarely within the operator's own explicit exception ("unless the test exposes a real production bug") — the test genuinely exposed a real, previously-silent startup-reconcile failure.
- **2 pages statically-verified only** (Middle School Stage, Stage Template) rather than full interactive browser proof, due to the operator-only-venue constraint discovered during implementation.
- Audition Hall's existing back-to-map link keeps its inconsistent inline styling (cosmetic-only, not required for PASS, noted as an optional future polish item).

## 9. Known issues

- Pre-existing, unrelated `internal/assets` test failure (§4) — not part of this kernel's scope.
- ~25 other older stale test-fixture users remain in the live DB, unrelated to the 4 tests this kernel fixed — named as a separate future cleanup candidate, not touched here.
- The deeper systemic fix for DB-test isolation (wiring `TEST_DATABASE_URL` through the already-existing-but-unused `scripts/test/require-isolated-database.sh`, or refactoring Discord handlers to accept a `pgx.Tx`/querier interface so tests can transaction-wrap) remains deferred, per the operator's explicit preference for targeted fixture cleanup over broad production-code rewrites this pass.

## 10. Files changed or created

Created:
- `frontend/lib/back-to-map.js`
- `scripts/smoke/kernel63-back-to-map-browser.js`
- `Construction/OperatorLogs/evidence/kernel-63/` (5 screenshots)
- `Construction/OperatorLogs/kernel-63-reportback.md`

Modified:
- `backend/internal/identity/discord_server_link_test.go`, `discord_channel_mapping_test.go`, `discord_audio_test.go`
- `backend/internal/identity/discord_channel_mapping.go` (one-line production fix)
- `backend/internal/network/discord_chat_bridge_test.go`
- 17 frontend HTML pages (`account/index.html` + 16 venue pages) — one or two `<script src="/lib/back-to-map.js">` lines each
- `Construction/OperatorLogs/operator-log.md`, `operator-notes.md`

## 11. Project-memory updates completed

- [x] Kernel 63 reportback saved
- [x] operator-log updated
- [x] operator-notes updated
- [x] field guide / dev workflow — not touched (testing workflow itself did not change this pass)
- [x] next kernel recommendations recorded below

## 12. Next recommended step

Two independent, small candidates:
1. **DB-test isolation systemic fix** — wire `TEST_DATABASE_URL` through the existing `scripts/test/require-isolated-database.sh` (or transaction-wrap DB-touching tests), so future kernels don't need to hand-audit fixture cleanup one test at a time.
2. **Stale test-user sweep** — the ~25 older stale fixture users identified but explicitly not touched in this kernel's cleanup.

Either is small, contained, and independent of the other — pick whichever is more useful to unblock next.
