# Kernel Report Back — Kernel 79 (Phase 1): eWrite Image-Visibility Fix + Socio Hierarchy

## 1. Status

**PASS, DEPLOYED LIVE 2026-08-05.**

This is Phase 1 of the full Kernel 79 spec, scoped down after discussion
with Grant: the Skill Directory + skill rule-links become a separate
Kernel 79A; health-system links, tutorial/Cue/index-card links,
hierarchical export, and the full ruleset/series/module navigation rework
are deferred further out (none of the underlying data existed yet to link
to — see Deviations below).

What shipped: (1) a real security gap in the Kernel 78 asset-serving
endpoint — eWrite-embedded images never consulted the referencing
publication's own visibility — is closed; (2) the Socio Ruleset is
reorganized into the spec's intended `Ruleset -> {Core Rulebook,
Quickstart, Niava} Series` shape, with Grant's actual Quickstart and Niava
manuscripts imported. A real non-idempotency bug in the reparenting logic
was found and fixed by actually booting the compiled binary twice against
a test database, not just running unit tests.

Baseline: clean tree at `4227dea08b01f3cf2efd86c35bf567e6519dadcb`, all
containers up before this work started. Everything from this kernel is
uncommitted for Grant's review, per house practice — two unrelated files
(`frontend/venues/library/index.html`, `read.html`) are also modified in
the working tree from earlier in this session (a Library color/layout
request, not part of Kernel 79) and are called out separately below.

---

## 2. What Was Built

### Image-visibility fix (Goal D)

- New table `ewrite_publication_assets` (migration 086) recording which
  asset(s) a publication's current source actually references, in the
  same "real FK per side, CASCADE with the owner" idiom as
  `ewrite_object_links`.
- `backend/internal/ewrite/asset_refs.go`: `reconcilePublicationAssetRefs`
  (wired into `SavePublicationSource`'s existing transaction — covers both
  manual saves and imports through one hook) and
  `BackfillPublicationAssetRefs` (covers publications that predate the
  table, e.g. the live Core Rulebook; runs once at boot).
- `backend/internal/assets/read.go`: new `userCanReadAssetConsideringEwrite`
  — the entry point every asset read (content and metadata routes) now
  goes through. An eWrite-bound asset is gated by the referencing
  publication's own `ewrite.CanReadPublication` instead of the asset's
  incidental location membership; non-eWrite-bound assets are completely
  unaffected. **The content-serving route previously had zero access
  check for any non-map asset** — this is now closed, not just refined.
- Added `Cache-Control: private, max-age=0, must-revalidate` to the
  metadata route (previously had no cache header at all).

### Socio Ruleset hierarchy (Goal A, partial)

- `EnsureSocioSeriesHierarchy`, `EnsureQuickstartManuscript`,
  `EnsureNiavaManuscript` in `backend/internal/ewrite/seed.go` — same
  raw-SQL-in-a-transaction idiom as the existing
  `EnsureCanonicalSocioManuscript` (boot-time seeds can't use the
  authenticated store.go API).
- Core Rulebook / Quickstart / Niava Series created under the existing
  Ruleset; the pre-existing Core Rulebook Publication is reparented into
  its Series via a guarded one-time `UPDATE` — its ID, slug, revisions,
  sections/anchors, and any `ewrite_object_links` are completely
  untouched.
- Grant's actual manuscripts imported: "Socio-: The Locked Courtyard &
  Beyond" (Quickstart, 10,158 words) and "Niava Setting Supplement"
  (Niava, 13,266 words), both cleaned of Google-Docs export artifacts
  (see `Construction/Domains/eWrite/socio-ruleset-organization.md` for exact
  cleanup detail).

### A real bug found and fixed

Reparenting the Core Rulebook publication broke
`EnsureCanonicalSocioManuscript`'s own idempotency check (it joined
through a collection with the *ruleset's* slug, which stopped matching
once the publication moved to the Series). Found by actually booting the
compiled binary twice against the test database (not just running `go
test` once) — the second boot tried to recreate a duplicate publication,
then collided with the reparent's own `UPDATE`. Fixed by keying the
existence check off `ewrite_publications.location_id` + `slug` directly.
Full detail and the regression test in
`Construction/Domains/eWrite/socio-ruleset-organization.md`.

---

## 3. Evidence (MANDATORY)

### Unit/integration tests

```
$ cd backend && GOCACHE=/tmp/victory-gocache go build ./... && go vet ./...
(clean, no output)

$ TEST_DATABASE_URL=postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable \
  go test -count=1 ./...
ok  	victory/backend/cmd/victory	0.015s
ok  	victory/backend/internal/access	1.747s
ok  	victory/backend/internal/actions	0.183s
ok  	victory/backend/internal/aftercare	0.350s
ok  	victory/backend/internal/assets	1.677s
ok  	victory/backend/internal/characters	0.013s
ok  	victory/backend/internal/commands	0.012s
ok  	victory/backend/internal/cues	8.088s
ok  	victory/backend/internal/dice	0.006s
ok  	victory/backend/internal/ewrite	8.718s
ok  	victory/backend/internal/identity	10.116s
ok  	victory/backend/internal/merchant	5.150s
ok  	victory/backend/internal/messages	0.008s
ok  	victory/backend/internal/migrate	0.116s
ok  	victory/backend/internal/network	3.937s
ok  	victory/backend/internal/participation	0.540s
ok  	victory/backend/internal/playerprofile	0.958s
ok  	victory/backend/internal/playerrelationships	0.013s
ok  	victory/backend/internal/profiles	0.006s
ok  	victory/backend/internal/ratelimit	0.019s
ok  	victory/backend/internal/scenes	6.860s
ok  	victory/backend/internal/showings	0.007s
ok  	victory/backend/internal/showruns	3.189s
ok  	victory/backend/internal/shows	3.903s
ok  	victory/backend/internal/showtime	1.294s
ok  	victory/backend/internal/storysofar	0.021s
ok  	victory/backend/internal/thirdplace	2.957s
ok  	victory/backend/internal/tickets	2.584s
ok  	victory/backend/internal/venues	0.010s
ok  	victory/backend/internal/world	2.254s
(full repo, zero failures — run against a freshly reset victory_test to
avoid a stale-revision-number test artifact, see Known Issues)
```

New tests added:
- `backend/internal/ewrite/asset_refs_dbtest_test.go` — reconcile tracks
  additions/removals across saves; backfill covers pre-existing
  publications and is idempotent.
- `backend/internal/ewrite/seed_hierarchy_dbtest_test.go` — series
  created with correct parentage; Core Rulebook reparented with stable ID;
  Quickstart/Niava seeded with real word counts; **running the full
  boot-seed sequence twice produces zero duplication** (the regression
  test for the bug above).
- `backend/internal/assets/ewrite_visibility_dbtest_test.go` — 5 tests,
  each authoring a real publication through the actual API (not
  hand-inserted rows): owner always allowed; non-eWrite-bound asset
  unchanged (regression); public/published image readable by an
  authenticated non-member (the false-negative half of the bug); draft
  image denied to non-editor, allowed to editor; production-visibility
  image denied to a user with unrelated membership at the *asset's own*
  location, allowed to a member of the *publication's* location (the
  false-positive / "unrelated asset access" half — the core scenario the
  spec called out).

### Kernel 64 DB isolation proof (required — this kernel touches
`backend/internal/assets`)

Tests hard-fail, not skip, without `TEST_DATABASE_URL`:

```
$ unset TEST_DATABASE_URL
$ go test -count=1 ./internal/assets/... ./internal/ewrite/...
--- FAIL: TestUserCanReadAssetConsideringEwrite_OwnerAlwaysAllowed (0.00s)
    ewrite_visibility_dbtest_test.go:134: dbtest: TEST_DATABASE_URL is
    required for database-touching tests (see Construction/kernel-maker-
    field-guide.md, Backend Test Commands)
[... 16 more DB-touching tests in these two packages, same hard failure ...]
FAIL
```

Live database (`victory`, not `victory_test`) row counts, before this
kernel's deploy and right now — zero drift, proving no test run ever
touched the live database:

```
pre-deploy backup (users/location_memberships/locations/assets):
  users: 4   location_memberships: 1   locations: 2   assets: 0

live `victory` right now:
  users: 4   location_memberships: 1   locations: 2   assets: 0
```

### Live deploy

```
$ docker exec -i victory-postgres pg_dump -U victory victory > \
    /tmp/victory-pre-kernel79-deploy-backup.sql   # manual backup, extra to
                                                    # the automatic one below

$ docker compose build backend   # clean build
$ docker compose up -d backend
 Container victory-backend  Recreated
 Container victory-backend  Started

$ docker logs victory-backend --since 30s
2026/08/05 08:36:09 migrate: 1 pending migration(s): 086_kernel79_ewrite_publication_assets.sql
2026/08/05 08:36:10 migrate: pre-apply backup written to /opt/victory/backups/victory_pre_migrate_20260805_083609_1pending.dump (887278 bytes)
2026/08/05 08:36:10 migrate: applied 086_kernel79_ewrite_publication_assets.sql in 42ms
2026/08/05 08:36:10 migrate: 1 migration(s) applied, schema current
2026/08/05 08:36:11 victory backend listening on :8081
```

Live database hierarchy, verified directly against `victory` (not the test
DB) after deploy:

```
$ psql -d victory -c "SELECT c.kind, c.slug, c.title FROM ewrite_collections c ORDER BY c.kind, c.slug;"
  kind   |        slug         |        title
---------+---------------------+----------------------
 ruleset | socio-stories-of-us | Socio: Stories of Us
 series  | core-rulebook       | Core Rulebook
 series  | niava               | Niava
 series  | quickstart          | Quickstart

$ psql -d victory -c "SELECT p.slug, c.slug AS collection_slug, p.word_count, p.status FROM ewrite_publications p JOIN ewrite_collections c ON c.id=p.collection_id ORDER BY p.slug;"
           slug            | collection_slug | word_count |  status
---------------------------+-----------------+------------+-----------
 core-rulebook             | core-rulebook   |      43207 | published
 niava-setting-supplement  | niava           |      13266 | published
 the-locked-courtyard      | quickstart      |      10158 | published
```

Live endpoint checks:

```
$ curl -s https://victory.amurray.family/health
{"ok":true,"service":"victory-backend","time":"2026-08-05T08:36:42Z"}

$ curl -o /dev/null -w "%{http_code}\n" https://victory.amurray.family/venues/writers-room/
200
$ curl -o /dev/null -w "%{http_code}\n" https://victory.amurray.family/venues/library/
200
$ curl -o /dev/null -w "%{http_code}\n" https://victory.amurray.family/api/ewrite/tree
401
$ curl -o /dev/null -w "%{http_code}\n" https://victory.amurray.family/api/library/tree
401
```

### Static checks

```
$ git diff --check
(clean, no output)
```

No JS files changed this kernel (Go + SQL + Markdown only), so `node
--check` doesn't apply here.

---

## 4. How to Run (Operator Steps)

Already deployed live — nothing to run. For a fresh install, the new
Series/manuscripts seed automatically at boot (same as the existing Core
Rulebook), no operator action needed.

To reproduce the test evidence from clean state:

```
git pull
cd /opt/victory
TEST_DATABASE_URL=postgres://victory:<POSTGRES_PASSWORD>@127.0.0.1:5432/victory_test?sslmode=disable \
  CONFIRM_TEST_DB_RESET=1 scripts/test/reset-test-database.sh
cd backend
TEST_DATABASE_URL=postgres://victory:<POSTGRES_PASSWORD>@127.0.0.1:5432/victory_test?sslmode=disable \
  go test -count=1 ./...
```

---

## 5. Operator Notes (CRITICAL)

- The new `ewrite_publication_assets` table has zero rows on this
  install right now — none of the three seeded manuscripts (Core
  Rulebook, Quickstart, Niava) embed any images. The fix is real and
  tested (see the 5 dedicated tests), but there's nothing to see visually
  in the Library until an eWriting with an embedded image exists. Worth a
  manual click-through once one does.
- Two stale host-level `go run ./cmd/victory` processes (PIDs from
  2026-07-31, pointed at `victory_test` via a leftover `DATABASE_URL`)
  were found blocking the test-database reset and were stopped — they
  were unrelated leftover dev processes, not the production service
  (which runs containerized). Worth a `ps aux | grep victory` check next
  session if `reset-test-database.sh` ever refuses with "database is
  being accessed by other users" again.
- `TEST_DATABASE_URL` is not set in `.env` — export it manually per the
  commands above whenever running DB-touching tests.

---

## 6. Blockers & Workarounds

**BLOCKER:**
No browser session available to click through Writer's Room / Library as
an authenticated user (Producer/Crew) on the live site.

**CAUSE:**
I don't have Grant's login credentials, and the app has no operator
break-glass path for "browse as" without them.

**WORKAROUND:**
Verified everything reachable without credentials instead: live database
state directly (hierarchy, word counts, stable publication ID), the
compiled binary's boot log, static page load (200s), and that the API
correctly 401s unauthenticated requests. The actual authorization logic
(the part that matters for the security fix) is proven by 5 dedicated
integration tests against a real database with real users/roles/
publications — stronger evidence than a single manual click-through would
have been for that specific claim.

**OPERATOR ACTION REQUIRED:**
Grant should do one visual pass when convenient: open Writer's Room and
confirm the tree shows Ruleset -> 3 Series -> respective Publications as
expected, and open the two new Library entries (Quickstart, Niava) to
confirm they read cleanly (no empty-titled sections from the stripped
Google-Docs spacer headings).

---

## 7. Deviations from Kernel

This is **Phase 1 of 2** — explicitly agreed with Grant before
implementation, not a silent scope cut:

- **Kernel 79A** (separate, follow-up): Skill Directory + Character skill
  rule-links. Deferred because it's a self-contained, sizeable feature on
  its own and Grant wanted it split into its own reportback/commit.
- **Deferred further out, no data exists yet to link to**:
  - Health-system links (spec 6.2) — the "eight health systems" have no
    data model on Character sheets at all yet, only manuscript prose and
    a roadmap line item. Building the mechanic itself would have been a
    large scope expansion beyond "add a rule link."
  - Tutorial/Cue/index-card rule links (Goal C remainder, spec 7-8) —
    the reusable pattern is proven (equipment already does this,
    `ewrite_object_links`) but no other object type was wired this pass.
  - Hierarchical export (Goal E, spec 11) and the full ruleset/series/
    module navigation rework (Goal F, spec 9) — both large, separable
    features; existing Writer's Room/Library tree rendering already
    handles the new Series generically with zero frontend changes, so
    there was no functional blocker forcing this into Phase 1.
  - Brevo bounded check (Goal H, spec 13) — recovery email is already
    live and configured from Kernel 77; not re-verified this pass since
    it's orthogonal to eWrite and Phase 1 was scoped to the two
    self-contained pieces above.
- **Image-visibility approach**: the spec (10.3) offered either
  "publication-bound asset references" or "permission-aware reference
  lookup" as acceptable models. Grant chose the former explicitly; that's
  what shipped.
- **Quickstart/Niava manuscripts**: no fabricated content — Grant
  provided the real manuscript files this session, placed on disk at
  `Construction/Domains/eWrite/manuscript/`, imported after cleanup (see
  `Construction/Domains/eWrite/socio-ruleset-organization.md`).

---

## 8. Known Issues

- Running the ewrite package's `go test` twice in a row against the same
  persistent `victory_test` database (rather than against a freshly reset
  one) makes `TestEnsureCanonicalSocioManuscriptSeedsAndIsIdempotent`
  flaky — it hardcodes "the edit lands as revision 2," which only holds
  the first time the test runs against a given database, since the
  canonical singleton publication it edits is never cleaned up between
  runs. Pre-existing test design (not introduced this kernel); only
  surfaced because of the extra manual verification runs this session did
  against a live-ish test database. Reset the test database before a
  clean full-suite run, as shown in the evidence above.
- `ewrite_publication_assets` is currently empty on the live install
  (see Operator Notes) — real but untested-in-production-content until an
  eWriting with an image exists.

---

## 9. Next Recommended Step

Kernel 79A: Skill Directory + Character skill rule-links, reusing the
`ewrite_object_links` pattern exactly as equipment already proves it
(one new nullable FK column + one CHECK arm on the existing table, plus a
new `ewrite_skill_directory`/`ewrite_skill_directory_entries` pair
following the spec's conceptual model in section 5.1).

---

## 10. Files Changed / Created

Part of this kernel:

- `backend/migrations/086_kernel79_ewrite_publication_assets.sql` (new)
- `backend/internal/ewrite/asset_refs.go` (new)
- `backend/internal/ewrite/asset_refs_dbtest_test.go` (new)
- `backend/internal/ewrite/seed_hierarchy_dbtest_test.go` (new)
- `backend/internal/ewrite/seed/quickstart-v1.md` (new)
- `backend/internal/ewrite/seed/niava-v1.md` (new)
- `backend/internal/assets/ewrite_visibility_dbtest_test.go` (new)
- `backend/internal/ewrite/seed.go` (modified — hierarchy + manuscript
  seed functions, existence-check fix)
- `backend/internal/ewrite/seed_dbtest_test.go` (modified — updated to
  match the existence-check fix)
- `backend/internal/ewrite/revisions.go` (modified — wired in the asset
  reconcile call)
- `backend/internal/assets/read.go` (modified — the security fix)
- `backend/cmd/victory/main.go` (modified — new boot sequence calls)
- `Construction/Kernels/Kernel 79 — eWrite Integration and Rules
  Navigation.md` (new — the kernel spec, saved to the repo; see note
  below)
- `Construction/Domains/eWrite/socio-ruleset-organization.md` (new)
- `Construction/Domains/eWrite/ewrite-image-visibility.md` (new)
- `Construction/Domains/eWrite/manuscript/` (new — Grant's original, uncleaned
  manuscript files, untouched)

**Not part of this kernel** (already modified in the working tree from
earlier in this session, unrelated request):

- `frontend/venues/library/index.html`
- `frontend/venues/library/read.html`

**Note on the saved kernel spec doc**: it was handed to me as pasted chat
text, and the transport had already lost the specific identity of every
smart-quote/em-dash/en-dash character (collapsed to a single stray byte,
irrecoverable exactly) before I ever saw it. I reconstructed it by
contextual inference (apostrophes, em-dashes, number-range en-dashes,
ASCII-art tree diagrams re-rendered plainly) rather than leaving the
mojibake in a permanent repo file — flagging this so the saved copy isn't
mistaken for a byte-exact original. The two actual manuscripts (Quickstart,
Niava) did NOT have this problem — Grant placed those directly on disk as
real files, which I verified byte-exact before use (see
`socio-ruleset-organization.md`).
