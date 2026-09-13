# Kernel 69 Reportback — Scene Library and Show Staging Model

**Status: PASS** (2026-07-13). Deployed live (migration 042 applied, backend rebuilt, `/health` OK). Changes left uncommitted for operator review, same as Kernels 62–68.

## 1. Repo audit summary

Before writing any code, searched for `scenes`, `scene`, `stage_actions`, `showings`, `shows`, `sessions`, `venues`, `productions`, and any script/document/link primitives:

- **No prior Scene domain concept exists in the real backend.** `grep -ril scene` outside this kernel's own new files turns up only `frontend/venues/{first-theater,catharsis}/runtime/scene-nodes.js` — a PIXI.js canvas scene-*graph* helper (rendering "scene," as in a graphics library concept) with zero relationship to a playable/viewable Scene domain object. Flagged in the dictionary explicitly so future readers don't confuse the two.
- The roadmap documents (`victory-master-actual-implementation-guide-v1.md`, `victory-track-roadmaps-v1.md`) describe an aspirational, **unbuilt** Socio-track "Scene Configuration Model" under document-internal Kernel 64/65/66/67 headers — per the roadmap's own collision note, those numbers were superseded by real Kernels 64–68 built to different, real operator briefs. Kernel 69 is genuinely additive, not a refactor of any existing stub.
- `productions` already exists with `location_id` (Kernel 15/42) and `created_by_user_id` (Kernel 68) — confirmed via `identity/permissions.go` and `showruns.CreateShowRun`'s own `productions.location_id` resolution pattern, which Kernel 69's `resolveProductionLocationID` copies verbatim.
- `shows`/`showruns` packages and their HTTP layers (`shows/http.go`, `showruns/http.go`) were read in full to match existing conventions: column-list `const` + `scanX(pgx.Row)` scan helpers, `CreateX`/`UpdateXPatch`/`ArchiveX` shape, per-package duplicated `response`/`writeJSON`/`writeError`/`requireAuthenticatedUser` helpers (small-helper duplication is this codebase's stated convention; only authority-critical functions get exported and reused across packages — Kernel 67's precedent).

**Conclusion: this kernel is purely additive.** No duplicate concept created, no existing table/package refactored.

## 2. Final table/model names

- `scenes` (migration 042) — `production_id NOT NULL REFERENCES productions(id) ON DELETE RESTRICT`, `UNIQUE(production_id, slug)`, `status CHECK IN (draft, ready, retired, archived)`, `archived_at` CHECK-paired with status (same pattern as `shows`/`show_runs`).
- `show_scene_placements` (migration 042) — `show_id NOT NULL REFERENCES shows(id) ON DELETE CASCADE`, `scene_id NOT NULL REFERENCES scenes(id) ON DELETE RESTRICT` (a Scene can never be silently orphaned out from under a Show's history), `UNIQUE(show_id, scene_id)` (at most one placement of a given Scene per Show, but any number of *other* Shows may each have their own independent placement of the same Scene), same status enum as Scene.
- `backend/internal/scenes` — new Go package: `types.go`, `authority.go`, `scenes.go` (Scene CRUD), `placements.go` (Show Scene Placement CRUD + curated Audience projection), `http.go` (7 routes).

## 3. Reusable Scene vs. Show placement distinction

Exactly the two-layer model the spec required:

- **Scene** (`scenes` table): reusable, Production-scoped, has its own audience/backstage fields and default venue.
- **Show Scene Placement** (`show_scene_placements` table): the use of one Scene inside one specific Show — ordering (`sort_order`), an optional venue override, and a small override set (`audience_title_override`, `audience_summary_override`, `director_notes_override`). Archiving/removing a placement is a status change (never a delete, never touches the Scene); archiving a Scene blocks *new* placements (`CreatePlacement` rejects a non-active Scene with `scene_archived`) but leaves every existing placement completely untouched.

Proven live (§6 below): the same Scene was staged in two different Shows under one Production, edited/archived independently at the placement level with zero effect on the other placement or the base Scene, and archiving the base Scene afterward still left both historical placements exactly as they were.

## 4. Authority rule

`CanManageScenesForProduction`/`CanViewScenesBackstage` (`backend/internal/scenes/authority.go`) resolve `production_id → location_id` (the same one-line lookup `showruns.CreateShowRun` already performs — `SELECT location_id::text FROM productions WHERE id = $1`) and delegate straight to `showruns.CanManageShowRun`/`CanViewBackstage`. No new authority logic was written — this reuses Kernel 66/68's existing Operator/Producer/Director-at-location (+ crew visibility-only exception) rule verbatim, following Kernel 67's explicit precedent that authority-critical code should be reused, not duplicated per package.

Show Scene Placement routes authority-check against the placement's Show's parent Show Run location directly (`showRunForShow`, the same two-step `shows.LoadShowByID` → `showruns.LoadShowRunByID` lookup `shows.CreateShow`/`UpdateShow` already perform for their own checks).

`CreatePlacement` additionally validates that the target Scene's `production_id` matches the Show's own resolved `production_id` (`scene_production_mismatch` otherwise) — a Producer at Location A cannot stage a Scene belonging to a different Production into one of their Shows, even if they otherwise pass the location-authority check (this matters once a location has more than one Production). Client-supplied `production_id`/`show_id`/`scene_id`/`venue_id` are never trusted for authority, only as row lookups — authority is always resolved server-side from the loaded row, matching every prior kernel's stated rule.

## 5. Audience data exposure rule

`AudienceScenePlacement` (`types.go`) is a distinct Go type with only three fields: `title` (placement override → Scene's own `audience_title` → Scene `title`, in that fallback order), `audience_summary` (placement override → Scene's `audience_summary`), and `sort_order`. It structurally cannot carry `director_notes`, `operator_notes`, `source_ref`, `config_json`, `created_by_user_id`, or any internal id — there is no code path that could leak them, not just a filtering step that might be forgotten. `ListAudiencePlacementsForShow` only returns rows where `status = 'ready' AND archived_at IS NULL`; a draft-staged Scene never appears in the curated program regardless of the base Scene's own status.

Verified live and in tests that a `director_notes` value containing a literal tripwire string never appears anywhere in the curated response (§6, §7).

## 6. Was Audience Program scene integration implemented or deferred?

**Implemented**, not deferred. `GET /api/shows/{show_id}/scenes/program` is a **separate route** from the existing `GET /api/shows/{show_id}/program` (Kernel 66/67's roster-based Audience Program), deliberately not merged into that response — merging would require `backend/internal/shows` to import `backend/internal/scenes`, but `scenes` already imports `shows` (to resolve a Show's parent Show Run/Production), which would create an import cycle. The Show Program frontend (`show-program.html`) calls both routes and renders them together as two sections on one page — no user-visible difference from a merged response, at the cost of one extra fetch.

## 7. Tests and evidence

### Commands run

```bash
cd /opt/victory/backend
GOCACHE=/tmp/victory-gocache go build ./...
GOCACHE=/tmp/victory-gocache go vet ./...
gofmt -l ./internal/scenes ./cmd/victory
git diff --check

TEST_DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable" \
  scripts/test/setup-test-database.sh

GOCACHE=/tmp/victory-gocache TEST_DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable" \
  go test -count=1 ./...

GOCACHE=/tmp/victory-gocache go test -count=1 ./internal/scenes/...   # no TEST_DATABASE_URL, hard-fail check

cd /opt/victory
scripts/smoke/fresh-install.sh --local
```

### Results

- `go build`/`go vet` clean, `gofmt -l` empty after one fix, `git diff --check` clean.
- New `backend/internal/scenes/scenes_test.go`: 12 test functions covering: manage-authority-at-production-location (correct producer / cross-location producer rejected / outsider rejected), slug uniqueness scoped to Production (same slug rejected within one Production, accepted across two), a Scene staged in two different Shows under the same Production with independent placement rows, cross-Production placement rejected (`scene_production_mismatch`), placement venue override leaves the Scene's own default venue untouched, archiving a placement does not archive the Scene, archiving a Scene blocks new placements but preserves the existing one, curated Audience program only returns `ready`+non-archived placements with curated fields (draft → empty, ready → visible, archived → empty again; explicit assertion that a `director_notes` tripwire string never appears in the audience entry), placement creation authority (outsider rejected), optional-field-clearing PATCH semantics, and the DB `CHECK` constraint rejecting a `'live'` status (Kernel 69 explicitly excludes it). **All 12 pass.**
- Full `go test -count=1 ./...` — every existing package (`access`, `actions`, `assets`, `characters`, `commands`, `dice`, `identity`, `messages`, `network`, `playerprofile`, `playerrelationships`, `profiles`, `showings`, `showruns`, `shows`, `thirdplace`, `venues`, plus the new `scenes`) green, zero regressions.
- Confirmed `internal/scenes` tests hard-fail cleanly without `TEST_DATABASE_URL` (Kernel 64's gate: `dbtest: TEST_DATABASE_URL is required for database-touching tests`), not silently skip or touch the live DB.
- One real bug caught during test-writing, not by inspection: the joined `ListPlacementsForShow` query reused the single-table `placementColumns` constant unqualified against a two-table JOIN, producing `ERROR: column reference "id" is ambiguous`. Fixed by writing an explicit `p.`-qualified column list for the join query; the single-table constant is unchanged and still used by `scanPlacement`/`LoadPlacementByID`/`CreatePlacement`/`UpdatePlacement`.

### Live authenticated proof against the real deployed server

No `chromium-cli`/headless browser tooling available in this environment (same gap Kernels 66–68 flagged) — substituted the same real-authenticated-HTTP-session proof method those kernels used, run inside a disposable `curlimages/curl` container attached to the live `victory_victory_internal` Docker network (not `fresh-install.sh`'s disposable stack — this hit the real deployed `victory-backend` container and the real `victory` database directly).

Three disposable accounts via real `/api/auth/signup`: `k69_producer_1783981115` (granted `producer` at `amurray-family` via direct SQL, mirroring the test-fixture convention), `k69_audience_1783981115` (granted `audience` at `amurray-family`), `k69_outsider_1783981115` (left with only its default signup membership, no relevant location role — the negative-authority account).

Full path executed and confirmed correct:
1. Producer created a real Production, a Show Run, and **two** Shows under it.
2. Producer created the named example Scene `Socio- : Character Making — Opening` with a `director_notes`/`operator_notes` tripwire value.
3. The **same** Scene was staged into **both** Shows (two independent placement rows, confirmed by distinct placement ids).
4. Backstage listing for each Show correctly showed its own placement of the shared Scene.
5. Placement A marked `ready`; curated `GET /api/shows/{id}/scenes/program` for Show A returned exactly `{"title":"Character Making","audience_summary":"Come make a character with us.","sort_order":1}` — no `director_notes`/`operator_notes`/ids of any kind. Show B's program (still draft) returned `"scenes":null`.
6. **Negative security proof, all confirmed with exact status codes**: audience-only account → `403 not_authorized` on backstage scene list and on Scene creation; anonymous (no cookie) → `401 not_authenticated` on Scene list; outsider (no relevant location role) → `403 not_authorized` on placement creation.
7. Archived placement A (`POST .../scenes/{placement_id}/archive`) → Scene itself remained `"status":"draft"` (not archived); Show B's independent placement of the same Scene was completely untouched.
8. Archived the reusable Scene itself → Show B's existing placement remained present and unchanged; a **new** placement attempt of the archived Scene was rejected `400 scene_archived`.

Live DB row counts after cleanup: `scenes=1`, `show_scene_placements=2`, `productions=3`, `show_runs=1`, `shows=2` — exactly the additive residue of the kept disposable-account run, nothing else touched. Per Kernel 65's precedent, only the final successful run's three disposable accounts were kept (an earlier attempt that failed on an unrelated cookie-`Secure`-flag curl issue, before any real data was created, was deleted).

### Fresh-install proof

`scripts/smoke/fresh-install.sh --local` extended with migration 042 added to the hardcoded migration array (applying Kernel 66/67's own documented "it's an array, not a glob" lesson on the first attempt) and a new Kernel 69 section with 9 new `PASS` assertions: anonymous Scenes list rejected, two fresh Shows created for staging, Producer created the named example Scene, Audience-only account rejected creating a Scene, the same Scene staged in two different Shows, curated Scene Program shows only audience-safe fields (with an explicit `director_notes`-must-not-leak check), backstage Scene list rejects a non-backstage user, archiving one Show's placement neither archives the Scene nor affects the other Show's placement, and archiving a Scene blocks new placements while preserving existing ones. Full script run: **`PASS clean-install smoke complete`**, no assertions skipped.

One bug caught only by actually running the script (not by review): the initial `-d` payload embedded `\xe2\x80\x94` as a literal 4-character escape sequence inside a bash double-quoted string (which bash does not interpret as raw bytes, unlike `printf`/`echo -e`), producing invalid JSON and a `400 invalid_request_body`. Fixed by writing the em dash as a literal UTF-8 character in the script file instead, matching how every other kernel's fixture strings in this file are written.

## 8. Blockers and workarounds

**BLOCKER:** Live-proof curl requests against the real deployed backend initially failed with `401 not_authenticated` despite a signup response containing `Set-Cookie: victory_session=...; Secure; SameSite=None`.

**CAUSE:** `COOKIE_SECURE` defaults to `true` in `docker-compose.yml`; curl (correctly, per the `Secure` cookie attribute spec) will not re-send a `Secure` cookie over a plain-HTTP connection, and the live-proof container talked to `victory-backend:8081` over HTTP inside the Docker network (no TLS terminator in that path — Caddy terminates TLS only on the external-facing route).

**WORKAROUND:** Extracted the raw `victory_session=...` token from the `Set-Cookie` response header and sent it back explicitly via a manual `Cookie:` request header on every subsequent call, bypassing curl's cookie-jar `Secure`-flag enforcement. Permanent workaround for any future same-network live-proof script in this environment, not a code change — the `Secure` cookie behavior itself is correct and was not touched.

**OPERATOR ACTION REQUIRED:** None.

## 9. Deviations from the kernel

- **Scene default-venue and placement-venue-override UI fields are plain text ID inputs**, not a venue picker dropdown — matching `show.html`'s own pre-existing "paste a session id" pattern for the Session Link section, since no general "list all venues" API exists in this codebase yet (`GET /api/venues` only returns *currently-visible-to-this-user* map tiles, not a full venue catalogue, and building one was out of this kernel's modest-UI scope). Reason: keep the kernel additive and avoid inventing a new venue-listing endpoint. Consequence: an operator must know/copy a venue id by hand. No approval needed — the kernel spec explicitly allowed "Recommended" field shapes and said not to redesign Stage Management.
- **Scene Library's "create a new Scene and add it immediately" from the Show page** was implemented as a link out to the Scene Library page (which then requires navigating back to the Show), not an inline embedded create-and-add form on `show.html` itself. Reason: the spec marked this "optionally" and asked to keep the Show page's Scenes section modest; an inline duplicate of the full Scene-create form would have doubled the surface area maintained in two places. Existing Scenes can still be added to a Show in one step from `show.html` directly.

## 10. Known issues

- No screenshot-based browser evidence — `chromium-cli` still not available in this environment; substituted real authenticated-HTTP-session proof (§7), consistent with Kernels 66–68's own documented gap. A manual checklist is included below for whenever browser access is available.
- The Scene Library page's inline "Edit" action uses a plain `prompt()` for the title field only (not a full edit form for every Scene field) — sufficient to prove PATCH round-trips correctly, but a future kernel could build a fuller in-page editor if Scene editing becomes a frequent operator workflow.

## 11. Manual visual checklist

1. Sign in as a Producer/Director at a location with at least one Production. Visit `/venues/show-runs/run.html?id=<a Show Run id>` → a "Scene Library" chip appears in the header launchbar (Producer/Director only).
2. Click it → lands on `/venues/show-runs/scenes.html?production_id=<id>` → "Create a Scene" card is visible; create one titled `Socio- : Character Making — Opening`.
3. Navigate to a Show under that Show Run (`show.html?id=<show id>`) → a "Scenes in this Show" card appears with an "Add existing Scene" dropdown containing the new Scene.
4. Add it → it appears in the list with its status and order; change its status dropdown to `ready`.
5. Visit that Show's Audience Program (`show-program.html?id=<show id>`) → a "Scenes" card appears above the roster grid showing only the Scene's title and audience summary.
6. As a plain Audience-only account, confirm Stage Management (and therefore the Scene Library and Show Scenes section) remains hidden/rejected, while the Audience Program page (including its Scenes card) still loads.

## 12. Files changed or created

**Backend (new):**
- `backend/internal/scenes/types.go`, `authority.go`, `scenes.go`, `placements.go`, `http.go`, `scenes_test.go`

**Backend (modified):**
- `backend/cmd/victory/main.go` (import + 12 new route registrations)
- `backend/internal/shows/http.go` (added `production_id` to the Show detail GET response)

**Database:**
- `database/migrations/042_kernel69_scenes.sql` (new: `scenes`, `show_scene_placements`)

**Frontend (new):**
- `frontend/venues/show-runs/scenes.html`

**Frontend (modified):**
- `frontend/venues/show-runs/run.html` (Scene Library launchbar link)
- `frontend/venues/show-runs/show.html` ("Scenes in this Show" section)
- `frontend/venues/show-runs/show-program.html` (curated Scenes card)

**Scripts/tests:**
- `scripts/smoke/fresh-install.sh` (migration 042 added to array; 9 new Kernel 69 assertions)

**Construction/docs:**
- `Construction/Kernels/kernel-69-scene-library-show-staging-model-v0.1.md` (new)
- `Construction/OperatorLogs/kernel-69-reportback.md` (this file)
- `Construction/OperatorLogs/operator-log.md` (appended)
- `Construction/Canon/Dictionary.txt` (new "Scene" and "Show Scene Placement" entries; updated "Show (Show Instance)" hierarchy line)

## 13. Required project-memory updates completed

- [x] Kernel spec status/commit/report path updated
- [x] Reportback saved in repository
- [x] `operator-log.md` appended
- [x] `operator-notes.md` — not needed; no new durable authority rule or runtime trap beyond what's already documented in the dictionary and this reportback's §4/§8 (both are Scene-specific facts, appropriately dictionary-scoped rather than cross-cutting operator knowledge)
- [x] `kernel-maker-field-guide.md` — not needed; no change to repo layout, test commands, or runtime modes
- [x] `dev-workflow.md` — not needed; no change to startup, ports, services, or validation steps beyond the new migration (already covered by the existing "apply new migrations" step)
- [x] Master Actual Implementation Guide — collision note already covers this; no further edit needed since the real kernel sequence tracking lives in `operator-log.md`, not that document's own stale Socio-track headers
- [x] Fresh-install/bootstrap migration list updated
- [x] Help/command documentation — not applicable, no new user-facing command

## 14. Next recommended step

A **Fly Scene / Capture / Session-integration kernel**: now that a reusable Scene primitive with Production scoping and Show staging exists, the natural next step is letting a Director make one staged Scene "live" for an active Session — the explicitly-deferred `live` status, active-scene pointer, and projection/broadcast work this kernel intentionally left out. A smaller, independent candidate: a fuller in-page Scene editor on the Scene Library page (currently a single-field `prompt()`-based edit).
