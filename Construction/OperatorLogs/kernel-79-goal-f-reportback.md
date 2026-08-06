# Kernel Report Back — Kernel 79 Goal F: Ruleset-Wide Library Navigation

## 1. Status

**PASS, DEPLOYED LIVE 2026-08-06.**

Grant's chosen next slice after Kernel 79A landed and was committed
(`03d4137`): Goal F, "improve rules navigation" (kernel spec §9) — Ruleset/
Series/Module landing pages, breadcrumbs, previous/next (already existed),
Ruleset-scoped search, and directory matches surfaced in search. No schema
change — this is entirely new read-side API + frontend on data that
already existed.

Still deferred, per Kernel 79A's own reportback: hierarchical export
(Goal E — 9.1/9.3's "export where authorized" bullets were built without
export, since export itself doesn't exist yet), health-system/tutorial/
Cue/index-card links (Goal C remainder), Brevo (checked separately this
session, see [[project_brevo_recovery_email_status]] — still blocked).

Baseline: clean tree at `03d4137`, all containers up before this work
started. Everything from this pass is uncommitted for Grant's review.

---

## 2. What Was Built

### Collection landing pages (spec §9.1-9.3)

- `GET /api/library/collections/{collection_id}` (new,
  `library_http.go`), generic across Ruleset/Series/Module — spec asks for
  three separate landing-page shapes but they're the same fields
  (description, children, publications, directories, breadcrumb) at three
  different tree depths, so one endpoint serves all three rather than
  three near-duplicate handlers. Returns: the collection itself,
  root-first breadcrumb ancestors (recursive CTE), direct child
  collections (Series under a Ruleset, Modules under a Series), readable
  publications directly under it (same `visiblePublicationsClause` every
  other reader route uses), and directories scoped to it
  (`ListDirectoriesForCollection`, from Kernel 79A).
- Frontend: `frontend/venues/library/collection.html` (new) — breadcrumb,
  title/summary, directories/children/publications sections, and a
  search box scoped to that collection's subtree.
- `frontend/venues/library/index.html`'s tree: collection titles are now
  links into their landing page (previously inert text spans).

### Publication reader breadcrumb (spec §9.4)

- `HandleLibraryPublication` now returns `ancestors` (same recursive-CTE
  shape as the collection endpoint) alongside the existing
  previous/next-among-siblings, TOC, and export link — all of which
  already existed from Kernel 78 and needed no changes.
- `read.html` renders it above the title.

### Ruleset-scoped search + directory matches (spec §9.5)

- `SearchPublications` gained an optional `collectionID` parameter: empty
  string is the exact pre-existing unscoped behavior (verified byte-for-
  byte via the pre-existing search test, updated call sites only); a real
  ID scopes the FTS query to that collection's subtree via a shared
  `collectionSubtreeCTE` recursive CTE (`WITH RECURSIVE sub AS (...)`,
  same idiom `HandleLibraryTree`'s ancestor lookup already used, just
  walking down instead of up).
- New `SearchDirectoryEntries` — surfaces matching Skill Directory entries
  (and any future directory type) in search results. Deliberately routes
  through `ListDirectoryEntries`'s existing visibility-safe resolution
  rather than a fresh query, so a search box can't become a second way to
  leak a hidden target that the browse view already protects.
- `HandleLibrarySearch` response gained a sibling `directory_results`
  array (`results` unchanged) and an optional `collection_id` query
  param. All three Library UIs (index, collection, read-adjacent search)
  render both arrays.

---

## 3. Evidence (MANDATORY)

### Unit/integration tests

```
$ cd backend && GOCACHE=/tmp/victory-gocache go build ./... && go vet ./...
(clean, no output)

$ TEST_DATABASE_URL=postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable \
  CONFIRM_TEST_DB_RESET=1 scripts/test/reset-test-database.sh
PASS: victory_test reset, migrated, and Go-side bootstrapped from empty.

$ TEST_DATABASE_URL=postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable \
  go test -count=1 ./...
(full repo, zero failures, same 30+ packages as Kernel 79A's evidence)
```

New test: `backend/internal/ewrite/library_navigation_dbtest_test.go` — 2
tests:
- `TestSearchPublicationsScopedToCollection`: unscoped search finds two
  publications in two unrelated Rulesets; scoped to Ruleset A finds only
  A's; **scoped to a Series one level below the Ruleset root still reaches
  a Publication two levels further down inside a Module** (proves the
  recursive CTE, not just a one-hop join); scoped to Ruleset B correctly
  excludes A's publication. Used a nonsense search term
  (`"quorlathorn"`) specifically to avoid the false-positive trap found
  during manual live verification below.
- `TestSearchDirectoryEntriesScopedToCollection`: same shape for directory
  search, two directories under two Rulesets.

Existing `store_dbtest_test.go`'s search test updated for the new
`SearchPublications` signature (added a `""` collectionID argument to its
3 existing calls) — no behavior change, confirmed by the test still
passing unmodified otherwise.

**No unit-level coverage added for `HandleLibraryCollection` or the
`ancestors` field on `HandleLibraryPublication`** — consistent with this
package's existing pattern: `HandleLibraryTree`/`HandleLibraryPublication`
have never had `httptest` coverage in this repo, only live-HTTP proof in
each kernel's reportback (see below). Followed the established pattern
rather than introducing a new testing style for one endpoint.

**A real ambiguity was caught during manual verification, not shipped as
a bug**: scoping a search for "courtyard" to the Core Rulebook Series
returned 1 result even though "courtyard" is a Quickstart term. Traced it
before assuming a scoping bug — the Core Rulebook's own text genuinely
cross-references "Quickstart: The Locked Courtyard" and separately uses
"courtyard" in a combat-range example, so the match is correct, not a
leak. This is exactly why the automated test above uses a nonsense term
instead of real manuscript vocabulary.

### Live deploy

```
$ docker exec -i victory-postgres pg_dump -U victory victory > /tmp/victory-pre-kernel79-goalf-deploy-backup.sql
$ docker compose build backend && docker compose up -d backend
 Container victory-backend  Recreated
 Container victory-backend  Started
$ docker logs victory-backend --since 30s
2026/08/06 02:57:31 migrate: schema current (88 migrations recorded)
2026/08/06 02:57:31 victory backend listening on :8081
(no pending migrations -- this pass added no schema)
```

Live database, before and after — zero drift:
```
before: users=4  location_memberships=1  ewrite_publications=3
after:  users=4  location_memberships=1  ewrite_publications=3
```

### Full end-to-end live proof (real HTTP, real auth)

Same disposable-fixture technique as Kernel 79A (real user + session
inserted directly, driven over real HTTP, fully deleted after):

1. `GET /api/library/collections/<socio-ruleset-id>` → `ancestors: []`,
   `children`: the 3 real Series (Core Rulebook, Quickstart, Niava).
2. `GET /api/library/collections/<core-rulebook-series-id>` →
   `ancestors: [{Socio: Stories of Us}]`, `publications: [Core Rulebook]`.
3. `GET /api/library/publications/<core-rulebook-id>` → `ancestors:
   [{Socio: Stories of Us}, {Core Rulebook}]` (two-level breadcrumb,
   correct depth).
4. `GET /api/library/search?q=courtyard&collection_id=<quickstart-series-id>`
   → the real Quickstart publication, exactly (see the ambiguity note
   above for why the Core-Rulebook-scoped variant also legitimately
   returns 1).
5. `GET /api/library/search?q=alertness` → `directory_results` correctly
   resolves the live Skill Directory's Alertness entry with its real
   target anchor.
6. Unauthenticated `GET /api/library/collections/<id>` → `401`.
7. `GET /venues/library/collection.html?id=<id>` and
   `/venues/library/index.html` → `200`.
8. Cleanup verified: `users` count identical before/after.

### Static checks

```
$ git diff --check
(clean, no output)

$ node --check <extracted inline script from library/index.html>
$ node --check <extracted inline script from library/collection.html>
$ node --check <extracted inline script from library/read.html>
(all clean)
```

---

## 4. How to Run (Operator Steps)

Already deployed live — nothing to run. Click any Series/Module/Ruleset
title in Library (previously inert) to see its landing page; the reader's
breadcrumb and any collection's search box are both live.

---

## 5. Operator Notes

- Ruleset-scoped export ("export where authorized" in spec §9.1/9.3) was
  **not** built — Goal E (hierarchical export) doesn't exist yet at all,
  so there's nothing to link to. The landing pages have no export
  affordance rather than a broken one.
- Directory search results that are `unlinked` or `hidden` still appear in
  results (so a reader can discover the skill exists), but link to the
  Skill Directory page itself rather than a specific rule when there's no
  resolved target — same non-leaking behavior `ListDirectoryEntries`
  already enforces, just surfaced through a different entry point.

---

## 6. Blockers & Workarounds

**BLOCKER:** No browser automation tooling in this environment (unchanged,
same as every prior kernel).

**WORKAROUND:** Real disposable account + session driven through the
actual deployed HTTP API (§3 above), plus 2 new dbtest tests for the
scoping logic itself.

**OPERATOR ACTION REQUIRED:** One visual pass when convenient — open
Library, click into a Series, confirm the breadcrumb and search box read
correctly, then open a Publication and confirm its breadcrumb matches.

---

## 7. Deviations from Kernel

- **One landing-page endpoint, not three.** Spec §9.1-9.3 describes
  Ruleset/Series/Module landing pages as if they're different surfaces;
  the actual field list is identical at every depth (title, description,
  children, publications, directories, source metadata via breadcrumb),
  so `HandleLibraryCollection` is generic across `kind`. No functional
  gap versus the spec — same information, one code path instead of three
  near-duplicates.
- **No export affordance** — see Operator Notes; Goal E is still fully
  deferred.
- **No unit-level test for `HandleLibraryCollection` itself** — matches
  this package's established pattern (see §3); would have been a new
  testing style introduced for one endpoint rather than following
  precedent.

---

## 8. Known Issues

None new. Pre-existing `TestEnsureCanonicalSocioManuscriptSeedsAndIsIdempotent`
flake (documented since Kernel 79 Phase 1) not re-triggered this pass —
ran against a freshly reset test database throughout.

---

## 9. Next Recommended Step

Grant's choice: Goal E (hierarchical export — Module/Series/Ruleset
export packages, which the new landing pages are now ready to link to
once it exists), the remaining Goal C rule-link surfaces (tutorial
actions, index cards, Cues, Scene Elements — all reuse the proven
`ewrite_object_links`/directory pattern), or something outside Kernel 79
entirely.

---

## 10. Files Changed / Created

- `backend/internal/ewrite/library_navigation_dbtest_test.go` (new)
- `frontend/venues/library/collection.html` (new)
- `backend/internal/ewrite/library_http.go` (modified — `HandleLibraryCollection`,
  `CollectionAncestor`, breadcrumb on `HandleLibraryPublication`)
- `backend/internal/ewrite/search.go` (modified — collection scoping,
  `SearchDirectoryEntries`, `directory_results` in the search response)
- `backend/internal/ewrite/store_dbtest_test.go` (modified — 3 call sites
  updated for `SearchPublications`'s new parameter)
- `backend/internal/ewrite/seed_skill_directory_dbtest_test.go` (modified
  — gofmt only, no behavior change)
- `backend/cmd/victory/main.go` (modified — new route registration)
- `frontend/venues/library/index.html` (modified — collection titles are
  now links; directory results rendered in search)
- `frontend/venues/library/read.html` (modified — breadcrumb)
