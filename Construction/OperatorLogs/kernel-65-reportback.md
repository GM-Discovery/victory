# Kernel Report Back — Kernel 65: Third Place Headshot Commons MVP

**Kernel spec:** `Construction/Kernels/kernel-65-third-place-headshot-commons-mvp-v0.1.md`
**Commit(s):** uncommitted (operator will review and commit; matches the precedent set for Kernels 62/63/64)
**Date:** 2026-07-11

## 1. Status

**PASS**

Every required behavior — Headshot lifecycle, live Trailer Face projection, My People integration, privacy boundaries, dedicated-test-DB proof, browser proof (desktop + mobile), and fresh-install proof — is built and evidenced below. Deployed live (migration applied, backend rebuilt, `/health` OK).

## 2. Acceptance-criterion ledger

| Criterion | Status | Evidence |
|---|---|---|
| Third Place venue exists | PASS | Migration 038 seeds `slug='third-place'` under `amurray-family`/`main-lot`; confirmed on both `victory_test` and live `victory` |
| Third Place appears with Trailers-level visibility | PASS | `access.ResolveVisibleVenues`'s `performer_surface` branch changed from `v.slug = 'trailers'` to `v.slug IN ('trailers', 'third-place')` — identical rule, not a new one |
| Third Place requires authentication | PASS | `requireAuthenticatedUser` gate on all 3 routes; anonymous → 401, proven in Go tests, curl (fresh-install), and browser proof |
| Back to Map visible and works | PASS | `<script src="/lib/back-to-map.js">` in `.launchbar`-mount mode, same as `people.html`; visible in every screenshot |
| Headshot terminology used consistently | PASS | "Headshot"/"Third Place" throughout backend, frontend, docs; no "Faceprint" anywhere except as an explicitly-labeled historical reference in the roadmap note |
| One active Headshot per account enforced | PASS | Partial unique index `uq_third_place_headshots_active_user`; `LeaveHeadshot`'s `ON CONFLICT` targets it directly — enforced at the DB level, not just app logic |
| Leave Headshot works | PASS | `TestLeaveHeadshotIsIdempotentAndReleaveAfterRemovalCreatesNewRow`, browser proof step "A left a Headshot" |
| Leave Headshot is idempotent while active | PASS | Same test (`created=false` on repeat); HTTP test `TestHandleMeFullLifecycle`; browser proof "repeated Leave Headshot did not create a duplicate card" |
| Remove My Headshot works | PASS | `RemoveHeadshot`; browser proof "A removed their Headshot" |
| Re-leave after removal creates new active record | PASS | Same Go test: third `LeaveHeadshot` call after remove returns a new, different ID; 2 history rows |
| Historic placement/removal records preserved | PASS | `ListMyHeadshotHistory`; browser proof "A's Headshot history shows the placement/removal record" |
| History does not snapshot old Face content | PASS | `third_place_headshots` has no Face-content columns at all — structurally impossible, not just policy; `HistoryEntry` JSON has no such fields, asserted in `TestHandleMeFullLifecycle` |
| Active Headshot renders live Trailer Face | PASS | `ProjectHeadshot` calls `playerprofile.ProjectTrailerFace` fresh on every read; `TestProjectHeadshotReflectsLiveTrailerFace` |
| Stage-name/Face changes reflect in Headshot after refresh | PASS | Same test (stage name + portrait change reflected on re-projection, no reload needed at the Go level); browser proof: B sees A's new stage name via the existing `/ws/player-profile` invalidation (with reload as an automatic fallback if the socket update hadn't landed yet) |
| Headshot list/search/sort works | PASS | `ListActiveHeadshots` (server-side, most-recent-first); client-side search/sort-by-name in `third-place/index.html`, mirroring `people.html`'s established convention |
| Headshot card opens Trailer | PASS | `trailer_url` in projection; browser proof "B opened A's Trailer from the Headshot card" |
| Headshot card can Add to My People | PASS | Reuses `POST /api/player-relationships` directly from the frontend (no duplicate backend route, per spec §7's explicit preference); browser proof |
| Existing relationship shows Open My Notes | PASS | `TestProjectHeadshotRelationshipStateForViewer`; browser proof "B now sees Open My Notes instead of Add to My People for A" |
| Adding to My People is private/directional | PASS | Reuses Kernel 62's `EnsureRelationship` unchanged; third-viewer isolation proven in both Go test and browser proof |
| Owner is not notified when added | PASS | No notification code exists anywhere in the path; `TestProjectHeadshotRelationshipStateForViewer`'s "owner's own view leaked relationship state" check; browser proof "A's own Headshot shows no relationship affordance and no notification" |
| Removed Headshots absent from active list | PASS | `TestListActiveHeadshotsExcludesRemoved`; browser proof "removed Headshot no longer appears to other viewers" |
| Anonymous list/API rejected | PASS | `TestHandleCollectionRequiresAuthentication`, `TestHandleMeRequiresAuthentication`, `TestHandleMyHistoryRequiresAuthentication`; curl 401 in fresh-install; browser proof |
| Other user cannot remove owner Headshot | PASS | No route/parameter exists for a target user other than the session user — true by construction, not by a runtime check; `TestHandleMeIgnoresClientSuppliedUserID` proves a spoofed body `user_id` is ignored |
| Payload excludes email/handle/raw UUID | PASS | `HeadshotProjection` JSON struct has no such fields; browser proof scans the raw JSON for `email`/`handle`/etc. and fails the run if found |
| Payload excludes relationship notes/journal/follow-ups | PASS | Same struct-level guarantee; browser proof scans for `private_nickname`/`journal`/`followup`/`stage_name_history` |
| No impressions/autographs/tags/feeds built | PASS | Not present anywhere in the diff — verified by re-reading the full file list below |
| Desktop browser proof attached | PASS | `Construction/OperatorLogs/evidence/kernel-65/01-a-left-headshot.png`, `02-b-sees-commons-desktop.png`, `04-a-headshot-history.png` |
| Mobile browser proof attached | PASS | `Construction/OperatorLogs/evidence/kernel-65/03-commons-mobile.png` (390×844) |
| Dedicated test DB workflow used | PASS | All `internal/thirdplace` tests use `dbtest.OpenTestPool(t)`; migration 038 applied to `victory_test` via `setup-test-database.sh` before any test ran |
| go test ./... recorded with TEST_DATABASE_URL | PASS | Full `go test -count=1 ./...` green, see §4 |
| fresh-install smoke passes | PASS | Extended with 7 new Kernel 65 assertions; full run PASS, see §4 |
| Operator and roadmap docs updated | PASS | See §6 and §10 |

## 3. What was built

### Backend

- **Migration `038_kernel65_third_place.sql`**: seeds the `third-place` venue (identical idempotent pattern to `007_kernel9_profiles_greenroom_trailers.sql`'s `trailers`/`greenroom` seed — no Go-side bootstrap needed for the venue itself, avoiding the "works live, fails fresh" trap Kernel 64 documented) and creates `third_place_headshots` with a partial unique index enforcing "one active Headshot per account" at the database level.
- **`backend/internal/thirdplace/`** (new package): `types.go` (Headshot, HeadshotProjection, HeadlineFact, HistoryEntry), `headshots.go` (LeaveHeadshot/RemoveHeadshot/GetMyHeadshot/ListActiveHeadshots/ListMyHeadshotHistory/ProjectHeadshot), `http.go` (HandleCollection/HandleMe/HandleMyHistory, mirroring the response/writeOK/writeError/requireAuthenticatedUser pattern every sibling package duplicates on purpose to avoid import cycles).
  - `LeaveHeadshot` upserts against the partial unique index directly (`ON CONFLICT (user_id) WHERE removed_at IS NULL AND status = 'active'`) and uses the `(xmax = 0)` Postgres idiom to report `created` without a second round trip — idempotent and race-safe, not just idempotent by convention.
  - `ProjectHeadshot` never stores Face content; it calls `playerprofile.EnsureWorkbook` + `playerprofile.ProjectTrailerFace` fresh on every read, and `playerrelationships.GetRelationshipBySubjectProfile` fresh per viewer for the Add-to-My-People/Open-My-Notes state.
- **Three routes wired into `main.go`**: `/api/third-place/headshots`, `/api/third-place/headshots/me`, `/api/third-place/headshots/me/history` — no dynamic `{id}` segment needed anywhere, so no tail-parsing was required (simpler than `playerrelationships`' `HandleByID`).
- **`internal/access/visibility.go`**: one-line change, `v.slug = 'trailers'` → `v.slug IN ('trailers', 'third-place')`, in the exact same `performer_surface` UNION branch — literally the same rule, per the kernel's explicit instruction to follow the canonical Trailers visibility path rather than invent a new one.

### Frontend

- **`frontend/venues/third-place/index.html`**: modeled directly on `trailers/people.html`'s structure (forbidden-screen auth gate, `.launchbar`/Back-to-Map/account-badge mount, client-side search/sort over a server-fetched list). Adds: My Headshot status panel (Leave/Remove/History toggle), Headshot Commons grid (portrait, stage name, placed date, up to 3 headline facts, Open Trailer, Add to My People/Open My Notes, "You" badge), and owner-only history panel.
- **Live update**: reuses the existing `watchPlayerProfile()` client from `frontend/lib/player-profile-ws.js` (Kernel 61A's `/ws/player-profile`) unchanged — one watcher per visible Headshot's `profile_id`, refetching the whole list on any invalidation. No backend changes were needed for this.
- **`frontend/app.js`**: added `third-place` to `venueHref()`, the icon map (falls back to `/assets/default.png`), and the map-position fallback table (`{x:60,y:8}`, near but not overlapping `trailers`).

## 4. Evidence

### Automated checks

```
go build ./...   → OK
go vet ./...     → OK
gofmt -l backend/internal/thirdplace/*.go cmd/victory/main.go internal/access/visibility.go → clean
node --check (inline script extracted from third-place/index.html) → OK
node --check frontend/app.js → OK
node --check scripts/smoke/kernel65-third-place-browser.js → OK
git diff --check → clean
```

### Database/domain proof (dedicated test DB, Kernel 64 workflow)

```
$ TEST_DATABASE_URL=".../victory_test" scripts/test/setup-test-database.sh   # applies migration 038, PASS
$ unset DATABASE_URL
$ GOCACHE=/tmp/victory-gocache TEST_DATABASE_URL=".../victory_test" go test -count=1 ./...
ok  	victory/backend/cmd/victory
ok  	victory/backend/internal/access
ok  	victory/backend/internal/assets
ok  	victory/backend/internal/identity
ok  	victory/backend/internal/network
ok  	victory/backend/internal/playerprofile
ok  	victory/backend/internal/playerrelationships
ok  	victory/backend/internal/thirdplace     ← new, 10 tests, all pass
ok  	victory/backend/internal/venues
(all other packages ok, zero failures)
```

10 `internal/thirdplace` tests: 4 domain (idempotent leave/re-leave, active-list exclusion, live-Face reflection, relationship-state-per-viewer including a 3rd unrelated viewer) + 6 HTTP (auth-required ×3, full lifecycle, client-supplied-user-id rejection, method-not-allowed).

Proof that DB tests still hard-fail without `TEST_DATABASE_URL` (not silently skip):
```
$ unset TEST_DATABASE_URL
$ go test ./internal/thirdplace/... -run TestLeaveHeadshotIsIdempotentAndReleaveAfterRemovalCreatesNewRow -v
    headshots_test.go:68: dbtest: TEST_DATABASE_URL is required for database-touching tests ...
--- FAIL
```

### Browser proof

`NODE_PATH=/tmp/node_modules node scripts/smoke/kernel65-third-place-browser.js` against the live rebuilt stack, three fresh disposable accounts (A owner, B viewer, C third-party) — **17/17 assertion groups PASS**:

```
PASS three fresh browser accounts signed up
PASS anonymous Third Place API access rejected
PASS A left a Headshot and sees the active-Headshot status line
PASS A's own Headshot card is clearly marked You
PASS repeated Leave Headshot did not create a duplicate card
PASS B sees A's Headshot in the commons, desktop viewport
PASS B opened A's Trailer from the Headshot card
PASS B added A to My People from the Headshot card
PASS B now sees Open My Notes instead of Add to My People for A
PASS A's own Headshot shows no relationship affordance and no notification of being added
PASS third viewer C does not see B's private relationship with A
PASS Third Place commons payload excludes email/handle/UUID/relationship-note fields
PASS B's Third Place view reflects A's live Trailer Face change (websocket or refresh)
PASS mobile viewport renders the Headshot Commons
PASS A removed their Headshot
PASS A's Headshot history shows the placement/removal record
PASS removed Headshot no longer appears to other viewers

Kernel 65 browser acceptance: ALL CHECKS PASSED
```

4 screenshots in `Construction/OperatorLogs/evidence/kernel-65/`: `01-a-left-headshot.png` (desktop, self card with You badge), `02-b-sees-commons-desktop.png` (desktop grid), `03-commons-mobile.png` (390×844), `04-a-headshot-history.png` (history panel showing a removed row).

### Privacy proof

Covered inline above by the browser script (raw-JSON scan for `email`/`handle`/`account_uuid`/`private_nickname`/`journal`/`followup`/`stage_name_history`) and by `TestHandleMeFullLifecycle`'s equivalent check on the history payload. Third-viewer isolation (C never sees B's relationship with A) proven in both the Go test and the browser script.

### Relationship integration proof

`TestProjectHeadshotRelationshipStateForViewer` (Go) and browser proof steps 7–11 above, covering: no relationship → Add to My People; after `EnsureRelationship` → Open My Notes with the correct `relationship_id`; owner's own view unaffected either way; a third, unrelated viewer still sees "none"/"Add to My People", never the first viewer's relationship.

### Fresh-install proof

```
$ scripts/smoke/fresh-install.sh --local
...
Applying 038_kernel65_third_place.sql
...
PASS migrations from empty DB
...
PASS My People page files exist
PASS Third Place list rejects anonymous requests
PASS A left a Headshot (f203baa7-7565-4bcf-8b4c-531f99f2106b)
PASS repeated Leave Headshot did not create a duplicate
PASS B sees A's Headshot with no private account fields
PASS removed Headshot no longer appears in the active commons list
PASS A's Headshot history preserves the placement/removal record
PASS Third Place page file exists
PASS clean-install smoke complete
```
No orphaned `victory_fresh_*` database or stray process afterward (checked directly).

### Production rebuild/health

```
$ docker exec -i victory-postgres psql -U victory -d victory < database/migrations/038_kernel65_third_place.sql   # applied, additive-only
$ docker compose up -d --build backend
 Container victory-backend Started
$ docker exec victory-backend wget -qO- http://127.0.0.1:8081/health
{"ok":true,"service":"victory-backend", ...}
$ curl -s https://victory.amurray.family/venues/third-place/   → 200
$ curl -s https://victory.amurray.family/api/third-place/headshots   → {"ok":false,"data":{"error":"not_authenticated"}}
```

## 5. How to run

```bash
cd /opt/victory
TEST_DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable" \
  scripts/test/setup-test-database.sh

cd backend
GOCACHE=/tmp/victory-gocache \
  TEST_DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable" \
  go test ./...

cd /opt/victory
NODE_PATH=/tmp/node_modules node scripts/smoke/kernel65-third-place-browser.js
```

## 6. Operator notes

See `operator-notes.md`'s new "Kernel 65" section: the venue-visibility reuse pattern, why the Headshot table has no Face-content columns at all, the `(xmax = 0)` upsert idiom, and the disposable-account residue left from the browser proof.

## 7. Blockers and workarounds

- **Browser script bugs found and fixed during this pass** (not blockers in the shipped product, just script authoring mistakes): `/api/player-profile/pages/{key}/commit` needs the `/commit` suffix (missed on the first attempt); an anonymous-context `page.evaluate` fetch needs `page.goto()` first so there's a document base URL for the relative path to resolve against; the "Open Trailer" link navigates in the same tab (no `target="_blank")`, so a `waitForEvent("page")` pattern copied from habit hung forever — replaced with a plain same-tab `waitForURL`.
- No product-level blockers.

## 8. Deviations from kernel

- **No separate `POST /headshots/{id}/add-to-my-people` endpoint** — built exactly per the kernel's own stated preference (§7): the frontend calls the existing `POST /api/player-relationships` directly with the Headshot's `profile_id`, identical to how `trailers/view.html` already does it. Zero new relationship-mutation code paths.
- **`relationship_id` added to the projection JSON**, beyond the kernel's "recommended" (not exhaustive) shape — lets the frontend build the `Open My Notes` link (`/venues/trailers/person.html?id=...`) directly from the list response instead of a second network round trip per card. This is the *viewer's own* relationship ID, already visible to them elsewhere (My People list, Trailer view) — not new exposure.
- **Live-update via the existing `/ws/player-profile` invalidation was implemented** (not deferred to refresh-on-reload, which the spec explicitly would have allowed) — the frontend opens one watcher per visible card using the unmodified `frontend/lib/player-profile-ws.js`, no backend changes required.

## 9. Known issues

- None new. The Kernel 64 known-issue notes (dual Go/shell safety-gate implementations, deferred stale-test-user sweep) are unaffected by this kernel.
- The commons list re-fetches the *entire* list on any watched profile's invalidation event (not a per-card patch) — fine at current scale, worth revisiting if Third Place ever has enough simultaneous active Headshots for this to matter.

## 10. Files changed or created

Created:
- `database/migrations/038_kernel65_third_place.sql`
- `backend/internal/thirdplace/types.go`, `headshots.go`, `http.go`, `headshots_test.go`, `http_test.go`
- `frontend/venues/third-place/index.html`
- `scripts/smoke/kernel65-third-place-browser.js`
- `Construction/OperatorLogs/evidence/kernel-65/` (4 screenshots)
- `Construction/Kernels/kernel-65-third-place-headshot-commons-mvp-v0.1.md`
- `Construction/OperatorLogs/kernel-65-reportback.md`

Modified:
- `backend/cmd/victory/main.go` (import + 3 route registrations)
- `backend/internal/access/visibility.go` (one-line venue-visibility rule change)
- `scripts/smoke/fresh-install.sh` (migration 038 added to the array; 7 new Kernel 65 assertions)
- `frontend/app.js` (map href/icon/position for `third-place`)
- `Construction/Canon/roadmaps/victory-master-actual-implementation-guide-v1.md` (numbering-collision note, §4.3)
- `Construction/Canon/roadmaps/victory-track-roadmaps-v1.md` (new "Kernel 65 continuation" under V8)
- `Construction/OperatorLogs/operator-log.md`, `operator-notes.md`

## 11. Project-memory updates completed

- [x] Kernel 65 reportback saved
- [x] operator-log updated
- [x] operator-notes updated
- [x] field guide updated (new "Kernel 65 Notes" section — domain reuse patterns worth knowing, even though the test/browser *workflow* itself didn't change)
- [x] dev workflow — explicitly not needed (no new commands; existing Kernel 64 `TEST_DATABASE_URL` workflow covers this package too)
- [x] Master Actual Implementation Guide updated (numbering-collision note)
- [x] Parallel Track Roadmaps updated (V8 continuation)
- [x] fresh-install smoke updated (migration + 7 assertions)
- [x] next kernel recommendation recorded below

## 12. Next recommended step

Three independent candidates, per the kernel's own "Expected next consumer" list:
1. **Show Run primitive / Run Roster MVP** — the largest, most natural next step; Third Place's Headshot Commons and My People are both explicitly named as primitives it would build on.
2. **Stale test-user sweep** — the ~25 older fixture users on the live DB, named and deferred in Kernels 62/63/64, still untouched.
3. **Third Place impressions/autographs** — the smallest incremental Third Place expansion, explicitly deferred in this kernel's own scope.

Show Run is likely the highest-value next step given how much of Kernel 65 (and 61A/62 before it) was purpose-built as its foundation.
