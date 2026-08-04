# Kernel Report Back — Kernel 78: eWrite Foundation

## 1. Status

**PASS** (with the operator-approved anonymous-read deferral recorded below), **DEPLOYED LIVE 2026-08-04**.

The complete vertical works end to end: create/import Markdown → hierarchy
recognized → safe draft saves → append-forward revisions → publish with
visibility → Library browse/read → direct section links → an equipment item
opens an exact rule section → Markdown exports back out. The scale proof ran
against the real Socio v1.1 manuscript, not a fixture.

Baseline: clean tree at `d57b488`, all containers up. Everything from this
kernel is uncommitted for Grant's review, per house practice.

---

## 2. Operator amendments (locked before implementation)

1. **Two venues** — Library adopts the half-built seeded slot (row, icon,
   map pin all pre-existed; the tile 404'd); Writer's Room is new, Crew+
   map-visible only. `frontend/assets/writers-room.png` is a copy of
   `default.png` until Grant paints the real icon; provisional pin at
   `{x:58, y:80}` near the Library.
2. **Anonymous public reading deferred** — the `public/authenticated/production`
   enum ships in schema and API, but `public` is served to authenticated
   readers only. The first unauthenticated content API in Victory deserves
   its own kernel. This is the one deviation from spec §21's browser-proof
   list ("anonymous public read") and §22 — operator-approved, not a PARTIAL.
3. **Sociov1_1.md is the real manuscript** for §18.2 — already in the repo;
   no PASS-WITH-FOLLOW-UP clause needed.
4. **Editor/reader polish all in scope** — formatting strip, Ctrl+S, word
   count, localStorage autosave/restore, scrollspy TOC, hover-¶ links, FTS
   snippets, print CSS.

---

## 3. What was built

### Schema (migrations 084, 085 — next was 084, verified against directory and ledger)

`084_kernel78_ewrite_schema.sql`: `ewrite_collections` (typed tree,
ruleset/series/module, CHECK-constrained), `ewrite_publications` (source
truth + sanitized `rendered_html` cache + `search_text` + generated
`search_tsv` GIN column — Victory's first FTS), `ewrite_revisions`
(append-forward, UNIQUE(publication, number)), `ewrite_sections`
(stable-identity heading rows), `ewrite_anchor_aliases`, `ewrite_editors`
(edit/publish/manage_editors grants), `ewrite_object_links` (typed-FK
binding per the migration-064 idiom). FK behaviors chosen and documented:
containers RESTRICT, satellites CASCADE, user refs SET NULL as backstop
under tombstone reassignment, `object_links.section_id` SET NULL (links
degrade, never dangle). `085` seeds `writers-room` (083's exact pattern)
and makes `library`'s seed migration-canonical (`ON CONFLICT DO NOTHING`).

### Backend — new package `backend/internal/ewrite/` (~2,900 lines incl. tests)

- `markdown.go`/`markdown_policy.go` — the repo's first Markdown pipeline:
  goldmark v1.8.5 + bluemonday v1.0.27 (first new backend deps beyond the
  original four; pinned). Server-side render+sanitize only; the client
  never converts Markdown to DOM. Explicit `{#anchor}` extraction is a
  fence-aware regex pre-pass — a recorded spike proved goldmark's
  `WithHeadingAttribute` mangles the manuscript's `(`/`?` ids. Policy built
  from empty `NewPolicy()` (UGCPolicy allows external images; additive
  policies can't be narrowed). Full design in
  `Construction/eWrite/ewrite-markdown-security.md`.
- `store.go`/`revisions.go`/`sections.go` — hierarchy rules, slug dedupe,
  and the single source-write path: FOR-UPDATE conflict check
  (`base_revision_id` mismatch → 409 with current-revision payload; the
  submitted text is never touched), no-change saves defined as no-ops,
  render + revision insert + section reconcile in one transaction (cache
  staleness structurally impossible). Section reconcile matches by anchor
  first (row UUIDs survive → object links survive), then (title, level)
  with alias creation; live anchors beat aliases.
- `authority.go` — Operator short-circuit, `CurrentLocationRoleForLocation`
  scoping, named grants. Crew edit their own/granted work; Producers/
  Directors edit all in-location; creators demoted below Crew keep nothing.
  Draft read = edit authority (recorded contrast with storysofar's
  owner-only precedent). Reader denials return `publication_not_found`,
  never confirming hidden titles.
- `importer.go` — paste + `.md` upload through the same save path; strict
  UTF-8, CRLF/BOM normalization; ImportReport per spec 7.4.
- `search.go` — `websearch_to_tsquery` + `ts_rank` + `ts_headline`
  snippets, visibility predicate in SQL; drafts structurally excluded.
- `export.go` — per-publication zip (byte-preserved source + metadata.json
  with hierarchy path/section map/link manifest), built in memory, nothing
  under `storage/exports/`.
- `links.go` — equipment→section bindings; resolution returns published
  targets only, so draft titles cannot leak into Player payloads.
- 23 routes registered in `main.go`, all method-prefixed (CSRF posture test
  stays green automatically), writes behind the shared `actionLimiter`.

### Frontend

- **Library** (`frontend/venues/library/`): browse tree grouped by
  Ruleset/Series/Module, FTS search with `<mark>` snippets (escaped-then-
  re-marked, so a document that *discusses* `<script>` stays text), and
  `read.html` — server-rendered content injection, scrollspy TOC
  (IntersectionObserver), hover-¶ copy-links, alias-aware `#anchor` deep
  links with a visible "section no longer exists" notice, prev/next among
  readable siblings, export button, print CSS.
- **Writer's Room** (`frontend/venues/writers-room/`): dashboard (hierarchy
  browser, recent drafts, create Ruleset/Series/Module/eWriting, file or
  paste import with the full ImportReport rendered) and `edit.html` — the
  editor: docked live outline rail (click → jump to line), top bar with
  status/revision/visibility/save/publish/conflict state, formatting strip
  (bold/italic/H2/H3/link wrapping the selection), Ctrl+S, live word count,
  localStorage autosave keyed per (publication, base revision) with a
  restore banner, server-rendered preview toggle, revision history with
  load-into-editor restore (restore = new revision, history never
  rewritten), named-editor grants panel, and the Linked-objects panel
  (equipment picker → section dropdown).
- Pure logic extracted to `frontend/lib/ewrite/outline.js` and
  `editor-core.js` (UMD pattern), covered by 16 `node --test` tests in
  `tests/ewrite/`; alpha-gate step 4 glob extended (dice-exception
  accounting untouched).
- `app.js`: `writers-room` added to all five venue structures;
  `ewrite_author_surface` group label; library wiring already existed.

### Existing-object link proof (Goal H)

`rule_links` maps (equipment_item_id → publication/section) added to Equip
Mode's open payload (`merchant.OpenEquipModeContext`) and the character
inventory response; "View rule →" deep links render in the Equip Mode stock
list (`participant-interactions.js`) and the Greenroom inventory page. The
binding model is reusable — a second object type adds one nullable FK
column and a CHECK arm.

### Account lifecycle (spec 12.3/12.4)

Account export gained an `ewrite/` section (source Markdown + metadata +
revision history for publications the user created; `format_version` 1→2).
Deletion: sole-owned drafts hard-delete with their author; published/
archived/shared work reassigns to the tombstone (authorship anonymized, no
ownerless rows); plan preview counts `ewritings`.

---

## 4. Scale proof (§18.2 — real manuscript, measured)

`import_scale_dbtest_test.go` runs against the real
`frontend/assets/rulesets/Sociov1_1.md` on every suite run:

| Metric | Value |
|---|---|
| Source | 357,153 bytes / 43,207 words |
| Headings | 763 outline (137 explicit anchors preserved **verbatim**, incl. `why-this-game?` and the parenthesized ids) + 154 spacer headings skipped |
| Render (parse+sanitize) | ≈ 136 ms |
| Full save (render + revision + 763 section rows, one tx) | ≈ 0.9 s |
| Re-save (reconcile over existing rows — the everyday edit) | ≈ 1.5 s |
| Warnings | 12, all legitimate duplicate-heading disambiguations (repeated "Actions"/"Risks" subsections) |
| Rendered HTML | 521 KB; reader page remains responsive |

The manuscript's known 4d12/5d12 math error is imported as-is per operator
decision — now fixable in eWrite itself.

---

## 5. Evidence

- **Full Go suite**: `go test -count=1 ./...` with isolated
  `TEST_DATABASE_URL` — PASS (exit 0), including ~40 new eWrite tests
  (markdown safety vector-by-vector, hierarchy, authority incl.
  cross-production denial and cast-with-stolen-id save rejection,
  409 conflict semantics, section/alias stability, visibility + search
  exclusion, export round-trip, object-link vertical, lifecycle).
- **Alpha gate**: all automated steps PASS (node suites carry only the
  tracked dice exception; step 7 remains the manual browser checklist).
- **Fresh-install smoke**: PASS end to end — and this kernel repaired the
  script itself, which had been silently unrunnable since Kernel 76
  (see §6).
- **Static checks**: `git diff --check` clean; `node --check` clean on both
  new UMD modules and every new inline script.
- **Deploy (2026-08-04)**: `docker compose build backend && up -d` —
  pre-apply backup written
  (`victory_pre_migrate_20260804_063746_2pending.dump`, 366 KB), 084 applied
  in 276 ms, 085 in 9 ms, schema current, `/health` OK. Live checks:
  `/api/ewrite/tree`, `/api/library/tree`, `/api/library/search` all 401
  anonymous; `/venues/writers-room/` and `/venues/library/read.html` served.

### Browser proof (manual — Grant)

Remaining §21 walkthrough for the operator, minus the deferred
anonymous-read item: Crew author opens Writer's Room · Cast cannot author ·
import + structure preview · draft save · revision creation · conflict
warning (two tabs) · publish · Library browse · authenticated read ·
Production-only denial · exact section deep link · equipment "View rule" ·
export · unsafe Markdown inert. No browser tooling exists in the build
environment (standing gap since Kernel 65), so these are hand-walked.

---

## 6. Repairs to pre-existing infrastructure (found, not caused)

`scripts/smoke/fresh-install.sh` had been broken since Kernel 76 and stale
since Kernel 68 — it could not have fully passed on any post-K76 run:

1. Hardcoded the pre-rotation DB password (K76 moved it to `.env`) — now
   `POSTGRES_PASSWORD` env with the old default.
2. Expected `/api/discord/gateway/status` → 200; K76-M02 deliberately
   closed it to Operator-only — expectation corrected to 401.
3. Expected password signup → 200; K76 closed signup — the script's booted
   backend now sets `PASSWORD_SIGNUP_ENABLED=true` (the documented local-dev
   flag).
4. Expected a Face-unready account to read the Third Place commons list;
   K68's readiness gate forbids exactly that — the check now proves the
   gate refuses (new PASS line) and verifies the listing with the
   Face-ready account, preserving the later map-fog fixture that needs B
   unready.

Plus three new eWrite 401 probes. `filepaths.md` was also stale (claimed 69
migrations and a mandatory Go-bootstrap step for new venues) — corrected.

---

## 7. Known follow-ups

1. **Anonymous public reading** — the deferral. One route + one authority
   arm when Grant wants it; schema is ready.
2. **Writer's Room map icon** — replace `frontend/assets/writers-room.png`
   (currently a copy of default.png); adjust the provisional pin if desired.
3. **Sociov1_1.md import as live content** — the scale test imports into
   the test DB; Grant performs the real import through the browser (which
   doubles as browser proof).
4. **Embedded-image gating** — asset reads gate only `map`-type assets, so
   images embedded in eWritings are link-knowable; acceptable now, recorded
   in the security note.
5. **Collection-level export** (spec 12.2's bounded form) — per-publication
   export + account export ship; a whole-ruleset zip is deferred, schema
   traps nothing.

---

## 8. Operator outcome (spec §26, answered)

Upload/paste a large rulebook: **yes, proven at 357 KB / 43 k words**.
Sections/subsections recognized: **763, with your Google-Docs anchors kept**.
Crew authors without Cast access: **yes, enforced server-side and tested**.
Safe drafts: **yes — revisions, 409 conflicts, localStorage snapshots**.
Two editors can't silently overwrite: **yes, tested**. Public vs
Production visibility: **yes (public = signed-in for now, your call)**.
Player opens exact rule from an item: **yes — Equip Mode and inventory**.
Library browsing: **yes**. Unsafe Markdown: **cannot execute — two-layer
defense, vector-by-vector tests**. Export: **byte-exact source**. Account
export/deletion: **integrated and tested**. Security/recovery gains from
76–77A: **intact — full suite, CSRF posture test, and a fresh-install smoke
that finally runs again all say so**.
