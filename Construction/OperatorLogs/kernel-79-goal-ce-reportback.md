# Kernel Report Back — Kernel 79 Goals C & E: Object Rule Links + Hierarchical Export

## 1. Status

**PASS, DEPLOYED LIVE 2026-08-06.**

Two goals in one pass, at Grant's explicit request after asking "you can't
do both in one pass?" for the two remaining pieces of Kernel 79:

- **Goal C remainder** (spec §7, §8): extends the reusable
  `ewrite_object_links` model (spec §8.4) to four more Victory objects —
  Cues, index cards, Scene stage elements, and guided-dialogue Topics
  (the one concrete "tutorial choice" object that exists today).
- **Goal E** (spec §11): hierarchical export — a Module, Series, or
  Ruleset can now be exported as one zip preserving the full hierarchy,
  not just a single Publication (which Kernel 78 already had).

Only Kernel 79's Goal H (Brevo) and the health-system-link half of Goal C
remain open now — the latter is still blocked on the same thing it always
was: Character sheets have no health-system data model to link to yet.

Baseline: clean tree at Kernel 79 Goal F's uncommitted state (this
session's prior pass), all containers up before this work started.
Everything from this pass is uncommitted for Grant's review.

---

## 2. What Was Built

### Goal C: four new object-link types (migration 089)

`ewrite_object_links` gained four nullable FK columns (`cue_id`,
`index_card_element_id`, `scene_element_id`, `dialogue_topic_id`), a
broadened `object_type` CHECK, a matching fk-shape CHECK (exactly one
typed column set per row, same idiom as `equipment_item_id`), and one
partial unique index per new column (`WHERE object_type = 'x'`) for the
`ON CONFLICT` upsert each `SetXRuleLink` function uses.

- **Cue** (spec §8.2): `SetCueRuleLink`/`RuleLinksForCues`. Authority
  resolves a Cue's location through its only path there —
  `show_scene_placement -> show -> show_run.location_id` — since Cues
  carry no location_id of their own.
- **Index card** (spec §8.1): `SetIndexCardRuleLink`/
  `RuleLinksForIndexCards`. Index cards are `elements` rows
  (`element_type='index_card'`, Kernel 16-era model, no dedicated table);
  location resolves via `elements.library_id -> libraries.location_id`,
  the same scope `resolveIndexCardLibrary` already uses to create one.
- **Scene element** (spec §8.3): `SetSceneElementRuleLink`/
  `RuleLinksForSceneElements`. Resolves via `scene_stage_elements.scene_id
  -> scenes.location_id` (Kernel 70's ownership-scope column).
- **Dialogue Topic** (spec §7, narrowed): `SetDialogueTopicRuleLink`/
  `RuleLinksForDialogueTopics`. Spec §7 names "stance choices, actions,
  reactions, health consequences, equipment interactions, social
  mechanics" broadly — none of those exist as stable per-object rows
  except Kernel 74's guided-dialogue Topics (Ra's conversation choices),
  so that's what's wired. Resolves via `dialogue_topics.packet_id ->
  dialogue_packets.location_id`.

`HandleObjectLinks`/`HandleObjectLinkItem` (`GET`/`POST`
`/api/ewrite/object-links`, `DELETE .../object-links/{id}`) now dispatch
across all five object types instead of hardcoding `equipment_item`;
`RemoveObjectLink` and `ListObjectLinksForPublication` were already
type-agnostic (only the SELECT column list needed broadening).

**Frontend UI added for all four, same session, on request** (Grant
pushed back on leaving this backend-only: *"go ahead and apply it all
now"*) — see §2A below.

### Goal E: hierarchical export (spec §11)

- `BuildCollectionExport(ctx, pool, userID, collectionID)` in `export.go`:
  walks the full subtree under any Ruleset/Series/Module via the same
  `collectionSubtreeCTE` Goal F's scoped search introduced, builds each
  readable publication's zip path from its own collection-slug chain
  (`collectionSlugPath`, `collectionPath`'s filesystem-safe twin), and
  writes `{path}.md` + `{path}.metadata.json` per publication, one JSON
  file per directory (`directories/{slug}.json`, entries resolved through
  the existing visibility-safe `ListDirectoryEntries` — an export can't
  leak what browsing already protects), and a `manifest.json` with the
  full collection tree, per-publication hierarchy paths, and a real
  sha256 checksum for every file.
- **A publication the requester can't read is silently omitted, never
  partially included** (spec §11.5) — recorded by ID in
  `manifest.omitted_publications`, not just dropped silently from the
  file list.
- `GET /api/library/collections/{collection_id}/export`; `collection.html`
  (from Goal F) gained an "Export this section (.zip) →" link.
- No image bytes embedded — matches `BuildPublicationExport`'s existing
  precedent (Kernel 78 never embedded image bytes either, only lists
  `ImageReferences` in metadata); not a new gap this pass introduced.

---

## 2A. Frontend for the Four Goal C Object Types (same-session follow-up)

Originally scoped out (see §5/§7 below, left in place as a record of the
reasoning) on the grounds that the authoring surfaces are large, shared,
tightly-tested pages this environment can't browser-verify. Grant
corrected the venue naming in that reasoning — Catharsis and **First
Theater** share the Pixi-backed `stage-runtime` (not "Cave/Catharsis"),
and The Cave is DOM-only with no renderer, closer to a human test surface
than a tested production runtime — then asked for it to be built anyway.

**What shipped:**

- `RuleLink` (the shared struct every rule-link resolver returns) gained
  an `ID` field, populated by all five `RuleLinksForX` resolvers (the
  four new ones plus the original equipment one, for consistency) — the
  frontend needs the underlying `ewrite_object_links` row id to call
  `DELETE /api/ewrite/object-links/{id}`, and nothing exposed it before.
- New shared widget `frontend/lib/ewrite-rule-link.js`
  (`attachEwriteRuleLink(container, {objectType, objectId})`): fetches
  the current link, renders either "Rule: {title} →" + Remove, or a
  "+ Link to rule" toggle that reveals a **paste-ID form** (Publication
  ID + optional Section ID, plus a "Browse Library ↗" link to go find
  them) — deliberately the same interaction pattern this exact console
  already uses (the Cue editor's "Target Show Scene Placement ID" field
  says "Paste the target placement id"), not a new UX idiom introduced
  for this feature alone. Styled with plain inline CSS rather than
  borrowing a page-specific class, so it drops into any page's existing
  design system without assuming one exists.
- Wired into all four real authoring surfaces, found by reading the
  actual code rather than assumed:
  - **Cues**: `frontend/venues/show-runs/show.html`'s `renderCue` (the
    Director's per-Show Cue Setup panel).
  - **Dialogue Topics**: the same file's `renderDialogueEditor` topic
    loop (Kernel 75's guided-dialogue authoring surface).
  - **Scene stage elements**: the same file's `renderCompositionElement`
    (the Scene composer's per-element row).
  - **Index cards**: `frontend/venues/the-cave/index.html`'s index-card
    editor panel (`loadIndexCardEditor`/`clearIndexCardEditor`) — shown
    only while an existing card is loaded for editing, not in "new card"
    mode (an unsaved card has no `element_id` yet to link against).

**Verification, and its real limit:** no browser exists in this
environment, so the actual rendered appearance was never seen. What *was*
verified is the exact request sequence the widget performs, end to end,
against the live site with a disposable fixture (a real Cue built through
the full production → show run → show → scene → placement → cue chain,
inserted directly since none of those packages can be imported from
`ewrite`'s own code — see §3): `GET` before linking (empty) →
`POST` create (returns the link with a real `id`) → `GET` after (renders
exactly what the widget's "linked" branch expects, including `section_
anchor`/`section_title`) → `DELETE` by that `id` → `GET` after (empty
again). Every step matched the widget's own code path exactly. Static
`node --check` passed on both edited pages (`show.html` has one inline
script block; `the-cave/index.html` has three, checked individually since
the naive single-range extraction silently concatenates all of them and
produces false syntax errors — worth remembering for this file
specifically next time). **What's still unverified: that the buttons
actually render in the right place, look right, and the form usably
overlays this page's existing layout.** Grant's own click-through remains
the only way to close that gap — see Operator Notes below, and the
Deviations/Blockers sections below are otherwise unchanged from before
this follow-up (their reasoning is why UI wasn't the safe default choice,
not that the feature can't work).

---

## 3. A Real Structural Discovery: the ewrite/characters Import Boundary Is Now Load-Bearing

Kernel 79A made `internal/characters` import `internal/ewrite` (for
`RuleLinksForCharacterSkills`). This session discovered that boundary is
more fragile than it looked: **`internal/ewrite`'s own test binary cannot
import `internal/cues`, `internal/shows`, or `internal/scenes`** — all
three transitively reach `internal/characters` (via `internal/actions`,
`internal/network`, and `shows` respectively), which reaches back into
`ewrite`, closing a cycle specifically for `ewrite`'s test files (not its
production code, which never imports any of the three).

This was **not caught by `go build`** — only `go test` compiles test
files, so `go build ./...` stayed clean through three consecutive
cycle-closing attempts before the real shape of the constraint became
clear. The new Goal C test fixtures
(`object_links_extended_dbtest_test.go`) therefore build their entire
fixture chain (production, show run, show, scene, placement, stage
element, Cue) by direct SQL against the same columns those packages' own
`Create*` functions would fill, rather than importing the packages. Full
detail and the exact package chain recorded in code comments at the top
of that file and in memory for future kernels that touch this boundary.

---

## 4. Evidence (MANDATORY)

### Unit/integration tests

```
$ cd backend && GOCACHE=/tmp/victory-gocache go build ./... && go vet ./...
(clean, no output)

$ TEST_DATABASE_URL=postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable \
  CONFIRM_TEST_DB_RESET=1 scripts/test/reset-test-database.sh
PASS: victory_test reset, migrated, and Go-side bootstrapped from empty.

$ TEST_DATABASE_URL=postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable \
  go test -count=1 ./...
(full repo, zero failures, same package set as every prior Kernel 79 pass)
```

New tests:
- `backend/internal/ewrite/object_links_extended_dbtest_test.go` — 4
  tests, one per new object type: non-Crew+ denial, draft-not-resolved,
  published-resolves-with-correct-anchor. Mirrors
  `TestEquipmentRuleLinkVertical`'s shape; doesn't re-prove the shared
  `SET NULL` section-degrade behavior (a DB FK property already covered
  there, not new code here).
- `backend/internal/ewrite/export_collection_dbtest_test.go` — 1 test
  covering: a publication two levels down (Series → Module) is reachable
  and lands at the correct hierarchy-derived zip path; a draft is silently
  omitted for an ordinary reader (**not** the draft's own creator — an
  earlier draft of this test used the creator, which trivially always
  passes since a creator always has edit authority over their own draft
  regardless of role; fixed to use a separate audience-role member before
  it would have proven nothing real); a Production-visibility publication
  is included for an active member; checksums are recomputed from the
  actual zip bytes and compared, not just asserted present; a requester
  with zero access gets a valid empty-content export, not an error.
- `library_navigation_dbtest_test.go` (Goal F, already covered by the
  prior pass's evidence, unaffected).

### Kernel 64 DB isolation proof

```
$ unset TEST_DATABASE_URL
$ go test -count=1 ./internal/ewrite/...
--- FAIL: TestCueRuleLinkVertical (0.00s)
    dbtest: TEST_DATABASE_URL is required for database-touching tests
[... remaining new and pre-existing dbtest tests, same hard failure ...]
FAIL
```

### Live deploy

```
$ docker exec -i victory-postgres pg_dump -U victory victory > /tmp/victory-pre-kernel79-goalce-deploy-backup.sql
$ docker compose build backend && docker compose up -d backend
$ docker logs victory-backend --since 30s
2026/08/06 05:01:25 migrate: 1 pending migration(s): 089_kernel79_goalc_object_link_types.sql
2026/08/06 05:01:26 migrate: pre-apply backup written to /opt/victory/backups/victory_pre_migrate_20260806_050125_1pending.dump (1288014 bytes)
2026/08/06 05:01:26 migrate: applied 089_kernel79_goalc_object_link_types.sql in 74ms
2026/08/06 05:01:26 migrate: 1 migration(s) applied, schema current
2026/08/06 05:01:27 victory backend listening on :8081
```

Live database, before and after — zero drift (including pre-existing
`cues`=0 and `scene_stage_elements`=5 rows, untouched by the additive
ALTER TABLE):
```
before: users=4  location_memberships=1  ewrite_publications=3  cues=0  scene_elements=5
after:  users=4  location_memberships=1  ewrite_publications=3  cues=0  scene_elements=5
```

### Full end-to-end live proof (real HTTP, real auth)

Same disposable-fixture technique as every prior pass this Kernel:

1. `GET /api/library/collections/<socio-ruleset-id>/export` → `200`,
   244,108-byte real zip. Inspected directly: 8 files — all 3 real
   publications each at their correct `{ruleset}/{series}/{pub}.md` +
   `.metadata.json` path, `directories/skills.json` (the live Skill
   Directory), `manifest.json` with `root_title: "Socio: Stories of Us"`,
   all 3 publications listed, `omitted_publications` correctly empty
   (nothing was denied to this reader).
2. Unauthenticated `GET .../export` → `401`.
3. `GET /venues/library/collection.html?id=<ruleset-id>` → `200` (the new
   Export link renders on the already-live Goal F landing page).
4. `GET /api/ewrite/object-links?object_type=cue&object_id=nonexistent` →
   surfaced a raw Postgres error string instead of a clean 400 — see
   Known Issues; confirmed pre-existing (same shape on the original
   `equipment_item` path), not a regression from this pass.
5. Cleanup verified: `users` count identical before/after.

### Static checks

```
$ git diff --check
(clean, no output)

$ node --check <extracted inline script from library/collection.html>
(clean)
```

---

## 5. Operator Notes

- **Frontend UI for the four new object-link types now exists (§2A) but
  is browser-unverified.** The full API sequence each control performs
  was proven live end-to-end with a disposable Cue fixture; the actual
  rendered appearance in the Cue Setup panel, dialogue Topic editor,
  Scene composer, and index-card editor was never seen, since no browser
  exists in this environment. **Please do one click-through pass**:
  open a Show in `show.html`, expand a placement's Cue Setup / dialogue /
  composer panels, confirm the "+ Link to rule" control appears where
  expected and doesn't visually break the existing layout; open the-cave,
  select an existing index card, confirm the same under the editor.
- Export currently has no operator-facing size/rate limit beyond the
  existing 60s request timeout — fine at Victory's current content scale
  (the whole Socio Ruleset export is ~240KB), worth revisiting if/when a
  much larger Ruleset gets authored.

---

## 6. Blockers & Workarounds

**BLOCKER:** No browser automation tooling in this environment (unchanged
constraint, every prior kernel).

**WORKAROUND:** Real disposable account + session over real HTTP (§4
above) plus 5 new dbtest tests (4 object-link verticals + 1 export test)
against a real database.

**OPERATOR ACTION REQUIRED:** None blocking. When convenient: download a
real export zip from the Library UI and spot-check it opens cleanly.

---

## 7. Deviations from Kernel

- **Dialogue Topics, not the full "tutorial actions and choices" surface
  named in spec §7.** Stance choices, health consequences, and equipment-
  interaction outcomes have no stable per-object row anywhere in the
  codebase to hang a link off — Topics are the one concrete object that
  exists. Same kind of narrowing Kernel 79 Phase 1 already recorded for
  health systems generally.
- **No frontend UI for any of the four Goal C object types** — see
  Operator Notes. Consistent with 79A's precedent (the venue right-tray
  skill links were left backend-only for the same reason: shared, tested
  live surfaces this environment can't browser-verify).
- **Export has no size cap or streaming** — everything is built in memory
  like the existing single-publication export. Fine at current scale;
  flagged rather than silently assumed fine forever.

---

## 8. Known Issues

- **Malformed `object_id` on `GET /api/ewrite/object-links` leaks a raw
  Postgres error string** (`invalid input syntax for type uuid: "..."`)
  instead of a clean `400 invalid_id`. Confirmed pre-existing — the
  original Kernel 78 `equipment_item` path has the identical shape, this
  pass's four new types just inherited it by using the same pattern. Not
  a security leak (no schema/data details beyond confirming Postgres
  rejected malformed input, and the route is already auth-gated), just
  unpolished. Not fixed this pass — touching `writeError`'s shared error
  mapping affects every `ewrite` HTTP handler, out of scope for a
  same-session discovery.
- Pre-existing `TestEnsureCanonicalSocioManuscriptSeedsAndIsIdempotent`
  flake (documented since Kernel 79 Phase 1) reproduced once during this
  session's second, non-reset test run — confirmed still present and
  unrelated to this pass's changes (gofmt-only diff between the clean run
  and the flaky one).

---

## 9. Next Recommended Step

Kernel 79 is now down to two open items: Goal H (Brevo — blocked on
Grant checking Brevo's own dashboard, see
[[project_brevo_recovery_email_status]]) and the frontend wiring for this
pass's four object-link types (needs a browser session to do safely).
Neither is blocking; Kernel 79 could reasonably be called complete for
now pending those two, or Grant may want to open a different kernel
entirely.

---

## 10. Files Changed / Created

- `backend/migrations/089_kernel79_goalc_object_link_types.sql` (new)
- `backend/internal/ewrite/object_links_extended_dbtest_test.go` (new)
- `backend/internal/ewrite/export_collection_dbtest_test.go` (new)
- `backend/internal/ewrite/links.go` (modified — 4 new object types'
  Set/RuleLinksFor functions, broadened HTTP dispatch, broadened
  `ListObjectLinksForPublication`)
- `backend/internal/ewrite/types.go` (modified — `ObjectLink`'s 4 new ID
  fields)
- `backend/internal/ewrite/export.go` (modified — `BuildCollectionExport`,
  `CollectionExportManifest`, `collectionSlugPath`,
  `HandleLibraryCollectionExport`)
- `backend/cmd/victory/main.go` (modified — export route registration)
- `frontend/venues/library/collection.html` (modified — Export link)
- `frontend/lib/ewrite-rule-link.js` (new — §2A, shared rule-link widget)
- `frontend/venues/show-runs/show.html` (modified — §2A, widget wired into
  Cue rows, dialogue Topic editor, Scene composer element rows)
- `frontend/venues/the-cave/index.html` (modified — §2A, widget wired into
  the index-card editor panel)
