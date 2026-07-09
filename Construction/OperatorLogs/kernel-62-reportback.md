# Kernel Report Back — Kernel 62: Player Relationship Matrix, Private Notes, and Relationship Journals

**Kernel spec:** `Construction/Kernels/kernel-62-player-relationship-matrix-private-notes-v0.1.md`
**Commit(s):** explicitly uncommitted (operator decision: leave the working tree for personal review and commit)
**Date:** 2026-07-09

## 1. Status

**PASS**

All privacy, two-user, archive, journal, clean-install, and browser proofs completed. Two pre-existing Discord test failures (unrelated to this kernel, present on the clean tree) are recorded honestly in §4.

## 2. Acceptance-criterion ledger

| Criterion | Status | Evidence |
|---|---|---|
| Kernel 61A contracts reused, not duplicated | PASS | Subjects resolved via `player_profile_workbooks.id` (opaque profile ID) using the K61 `resolveUserIDForWorkbookID` pattern; auth via `access.CurrentUserIDFromRequest`; stage names read from the K61 ledger; no new identity tables |
| Relationship records are directional | PASS | `player_relationships(observer_user_id, subject_user_id)`; browser proof "relationship is directional: A's own list is unaffected" |
| Subject cannot see relationship exists | PASS | Browser proof §16.6: subject gets 404 on detail URL, API read/write, and journal; DOM scan of subject's Face/Trailer contains no note text or nickname |
| Cannot create self-relationship | PASS | DB CHECK constraint + `cannot_relate_to_self` guard; fresh-install "PASS self-relationship is rejected" (HTTP 400) |
| Unique relationship per observer/subject | PASS | `UNIQUE(observer_user_id, subject_user_id)` + `ON CONFLICT DO NOTHING` upsert in `EnsureRelationship` |
| Add to My People from Trailer works | PASS | Browser proof §16.1 + screenshot `01-trailer-add-button-desktop.png` |
| Open My Notes from Trailer works | PASS | Browser proof: "A's Trailer now shows Open My Notes for B" |
| My People list works | PASS | Screenshot `06-my-people-desktop.png` (stage name, nickname, categories, qualitative summary, journal date) |
| Search/filter/sort works | PASS | Browser proof: search by nickname matched, garbage query showed no-match state; category/state filters and 3 sorts implemented client-side |
| Private nickname works | PASS | Persisted across reload in browser proof; shown beside current stage name |
| Multiple categories work | PASS | friend + collaborator both persisted (screenshot `03-person-detail-filled-desktop.png`) |
| Custom category works | PASS | `custom` requires non-empty label (unit test `TestValidateCategories`); UI shows label input when checked |
| Qualitative dropdowns work | PASS | Trust/Closeness/Reliability/Communication persisted across reload; labels shown, keys stored |
| Invalid qualitative values rejected | PASS | Unit tests `TestQualitativeVocabulariesRejectInvalidKeys`; server rejects unknown keys with 400 `invalid_*` codes |
| Shared context uses only reliable data | PASS | `ProjectSharedContext` reads only active `memberships`+`productions` overlap; unit test `TestSharedContextFoldOnlyVerifiedOverlap`; empty state shown honestly (screenshot 03) |
| No inferred trust/closeness | PASS | Shared context carries production names/roles/dates only; nothing derived |
| Relationship Workbook pages save | PASS | 4 pages (Connection / Understanding Them / Our Relationship / Shared Work and Play); browser + fresh-install page-commit `changed:true` |
| Relationship facts recompute | PASS | Full-replace `RecomputeRelationshipFacts` after every commit/delete (K61 pattern); deletion-reveals-prior unit test |
| Private journal create/edit/delete works | PASS | Browser proof §16.3 full cycle incl. tag filter; screenshot `04-journal-desktop.png` |
| Journal private from subject | PASS | Subject journal GET → 404 (browser proof); fresh-install subject read → 404 |
| Follow-ups store/open/done/dismissed | PASS | Browser proof §16.4; screenshot `05-followups-desktop.png` |
| No reminders/notifications created | PASS | No scheduler/notification/email/websocket code exists anywhere in `internal/playerrelationships`; grep-clean |
| Archive hides from default list | PASS | Browser proof + fresh-install: archived ID absent from `state=active`, present in `state=archived` |
| Archived filter works | PASS | Screenshot `11-archived-filter-desktop.png` (subdued row styling, badge) |
| Unarchive restores | PASS | Browser proof §16.5 round trip |
| No destructive relationship delete added | PASS | No DELETE route or function for the relationship row exists; only archive/unarchive |
| Subject stage-name change preserves relationship | PASS | Browser proof §16.7: rename → B's list shows new name after refresh; screenshot `10-stage-name-updated-desktop.png` |
| Private nickname unaffected by stage-name change | PASS | Asserted in same proof step |
| Cross-account reads rejected | PASS | Subject 404 + third-user 404 (browser proof; both API and page) |
| Cross-account writes rejected | PASS | Subject PATCH nickname → 404; third-user archive → 404 |
| Spoofed observer ID rejected/ignored | PASS | Observer is only ever taken from the session cookie; no request field for it exists, so any client-supplied value is structurally ignored |
| Social Face does not expose notes | PASS | Face projection untouched by this kernel; DOM scan of A's Face contains no relationship data |
| Clean-install smoke updated and passes | PASS | 9 new assertions; full run PASS (see §4) incl. migration 037 from empty |
| Existing accounts preserved | PASS | Migration 037 is purely additive (`CREATE TABLE IF NOT EXISTS`); live `/health` OK after rebuild; no `users` rows touched |
| Desktop browser proof attached | PASS | `evidence/kernel-62/*.png` (8 desktop shots) |
| Mobile browser proof attached | PASS | `07-my-people-mobile.png`, `08-person-detail-mobile.png` (390×844) |
| Two-user privacy proof attached | PASS | `09-privacy-not-found-subject.png` + scripted 404/DOM assertions |
| Required automated checks recorded | PASS | §4 below, including two pre-existing unrelated failures |
| Operator and roadmap docs updated | PASS | §11 checklist |

## 3. What was built

- **Migration** `database/migrations/037_kernel62_player_relationships.sql`: six tables — `player_relationships` (directional, unique pair, CHECK not-self), `player_relationship_categories`, `player_relationship_facts`, `player_relationship_events`, `player_relationship_journal_entries` (soft delete), `player_relationship_followups`.
- **Backend package** `backend/internal/playerrelationships/` mirroring the Kernel 61 `playerprofile` slice: embedded versioned catalogue (`player-relationship-v1.0.0.json`, 4 pages, freeform text fields), fixed qualitative vocabularies (§6 of the spec) and category set, validation, pure events→facts fold with full-replace recompute, relationship CRUD (ensure/get/list/archive/unarchive/update), journal CRUD (soft delete, optional authority-checked production context), follow-up store (open/done/dismissed + reopen), shared-context projection over verified production-membership overlap, and an HTTP layer where **every non-owner access returns 404** (never 403) and raw account UUIDs are struct-tagged out of all JSON.
- **Routes** in `cmd/victory/main.go`: `/api/player-relationships/catalogue`, `/api/player-relationships`, `/api/player-relationships/` (single prefix handler for detail/patch/archive/unarchive/pages/events/journal/followups). Startup-fatal catalogue validation added beside the K61 bootstrap.
- **Frontend**: `frontend/venues/trailers/people.html` (My People list: search, category filter, Active/Archived/All, 3 sorts, empty states, privacy copy) and `person.html` (subject Face header with live `/ws/player-profile` refresh, private-notes banner, nickname/categories/dropdowns, honest "Victory can currently verify" panel, 4-page workbook with history + deletion impact preview, journal with tag/category filters, follow-ups, archive/unarchive). Entry points: `Add to My People` / `Open My Notes` on `view.html`, `My People` chips on `face.html` and `/account/`.
- **Verification**: 9 new two-user fresh-install assertions; new `scripts/smoke/kernel62-browser.js` Playwright proof (3 fresh accounts, 14 assertion groups, 11 screenshots, desktop + mobile).

## 4. Evidence

### Automated checks

```
GOCACHE=/tmp/victory-gocache go build ./...   → OK
GOCACHE=/tmp/victory-gocache go vet ./...     → OK
go test ./internal/playerrelationships/...    → ok (10 tests)
go test ./internal/playerprofile/... ./internal/access/... → ok
go test ./internal/identity/... → FAIL (pre-existing: TestDiscordBootstrapReconcileRestoresMappingsAndMicCommand)
go test ./internal/network/...  → FAIL (pre-existing: TestMirrorVictoryChatToDiscordPostsMessageAndPersistsBridgeRow, fixture-leak duplicate key)
git diff --check → clean
node --check → OK for people.html, person.html, view.html, face.html, account/index.html inline scripts + kernel62-browser.js
```

Both Discord failures were re-confirmed on a clean tree (`git stash` → same failures → `git stash pop`); they match the known "Discord fixture-leak cleanup" debt already named in the kernel spec's next-candidates list. No file in `identity/` or `network/` was touched by Kernel 62.

### Database/domain proof

Pure unit tests in `internal/playerrelationships`: vocab accept/reject, archived-not-settable, category spec-match + custom-label requirement, later-event-wins fold, deletion-reveals-prior, no-op commit detection, page-answer validation (unknown field / non-string rejected), UUID non-leak serialization test, shared-context verified-overlap-only fold.

### Browser proof

`node scripts/smoke/kernel62-browser.js` against the deployed stack: **14/14 groups PASS**. Evidence in `Construction/OperatorLogs/evidence/kernel-62/` (11 PNGs, desktop + mobile).

### Two-user privacy proof

Subject (A): detail page shows safe not-found; API GET/PATCH/journal → 404. Third user (C): GET/archive → 404. A's Face and Trailer-view DOM scanned for the private marker text and nickname — absent. A's own My People list empty (directional).

### Stage-name update proof

A renamed mid-proof; B's list showed the new stage name on refresh with nickname intact (screenshot 10).

### Archive proof

Archive → gone from Active, present in Archived filter (screenshot 11) → unarchive → restored. Subject uninvolved throughout.

### Fresh-install proof

`bash scripts/smoke/fresh-install.sh --local` → full PASS including migration 037 from empty DB and all 9 Kernel 62 assertions (relationship create, nickname+qualitative save, page commit, journal, subject-404, archive filters, self-rejection, page files).

### Production rebuild/health

Migration 037 applied to the live DB (additive, `IF NOT EXISTS`). `docker compose up -d --build backend` → `/health` `{"ok":true}`. Browser proof ran against this deployed stack, so the live routes are confirmed working.

## 5. How to run

- My People: `https://victory.amurray.family/venues/trailers/people.html`
- From another player's Trailer (`view.html?id=…`): `Add to My People` / `Open My Notes`
- Browser proof: `NODE_PATH=/tmp/node_modules node scripts/smoke/kernel62-browser.js` (Playwright lives in `/tmp/node_modules` from the K59A run; browsers in `/root/.cache/ms-playwright`)
- Fresh install: `bash scripts/smoke/fresh-install.sh --local`

## 6. Operator notes

See `operator-notes.md` §Kernel 62 (directional model, subject invisibility, vocab, archive semantics, no follow-up notifications, shared-context limits, operator caveat).

## 7. Blockers and workarounds

- Playwright is not in the repo; reused the K59A install via `NODE_PATH=/tmp/node_modules`. If `/tmp` is cleared, reinstall with `npm i playwright` in a scratch dir plus `npx playwright install chromium`.

## 8. Deviations from kernel

- **`relationship_state='archived'` cannot be set from the state dropdown** — only Archive/Unarchive enter/leave it, keeping `archived_at` and the state consistent. The dropdown shows the other four states (spec §6.5 lists archived as a display label; §14 makes archive an operation — this resolves the tension in favor of one mental model).
- **Journal `session_id` is stored but unused** — spec §7.6 allows session context "only if reliable session IDs exist"; no reliable player-facing session model exists yet. `production_id` IS supported and authority-checked server-side, but the v1 journal UI doesn't expose a production picker (title/date/category/tags/body only).
- **Relationship events are hard-deleted** (matching the K61 playerprofile convention); the `deleted_at` column exists for a future soft-delete switch. Spec §5.4 allows either.
- **Search/filter/sort run client-side** over the fetched private list (repo convention; lists are per-observer and small).
- Three throwaway browser-proof accounts exist on the live install (`k62_subject_1783620159501`, `k62_observer_1783620159501`, `k62_third_1783620159501`, all `@example.com`) — harmless, but deletable whenever account-deletion tooling exists.

## 9. Known issues

- Pre-existing Discord test failures in `internal/identity` and `internal/network` (fixture leak) — unchanged by this kernel, still the top hygiene debt.
- The My People list loads categories with one query per row; fine at private-list scale, worth batching if lists grow past hundreds.

## 10. Files changed or created

Created:
- `database/migrations/037_kernel62_player_relationships.sql`
- `backend/internal/playerrelationships/` — `catalogue.go`, `catalogues/player-relationship-v1.0.0.json`, `types.go`, `vocab.go`, `validation.go`, `facts.go`, `events.go`, `relationships.go`, `journal.go`, `followups.go`, `sharedcontext.go`, `http.go`, `catalogue_test.go`, `vocab_test.go`, `facts_test.go`, `privacy_pure_test.go`
- `frontend/venues/trailers/people.html`, `frontend/venues/trailers/person.html`
- `scripts/smoke/kernel62-browser.js`
- `Construction/Kernels/kernel-62-player-relationship-matrix-private-notes-v0.1.md`
- `Construction/OperatorLogs/evidence/kernel-62/` (11 screenshots)

Modified:
- `backend/cmd/victory/main.go` (import, startup catalogue validation, 3 route registrations)
- `frontend/venues/trailers/view.html` (Add to My People / Open My Notes)
- `frontend/venues/trailers/face.html`, `frontend/account/index.html` (My People chips)
- `scripts/smoke/fresh-install.sh` (migration 037 + 9 Kernel 62 assertions)
- `Construction/OperatorLogs/operator-log.md`, `operator-notes.md`, `Construction/kernel-maker-field-guide.md`, roadmap docs (see §11)

## 11. Project-memory updates completed

- [x] Kernel 62 reportback saved
- [x] operator-log updated
- [x] operator-notes updated
- [x] field guide updated (Kernel 62 baseline + two-browser privacy workflow note)
- [x] dev workflow updated (Playwright NODE_PATH note)
- [x] Master Actual Implementation Guide updated
- [x] Parallel Track Roadmaps updated
- [x] fresh-install smoke updated
- [x] next kernel recommendation recorded

## 12. Next recommended step

**Discord fixture-leak cleanup** — smallest honest next kernel. Two Discord tests now fail on every full-suite run (one leaks rows into a shared DB), which erodes the value of `go test ./...` as a gate for every future kernel. Cheap, contained, and it restores a clean baseline before the larger Third Place / Show Run roster work builds on Kernel 62's relationship primitive.
