# Kernel 69 — Scene Library and Show Staging Model

**Revision:** 0.1
**Status:** PASS
**Primary track:** V — Victory Core (reusable Scene primitive)
**Complementary tracks:** S (Socio's first real Scene), C (a future concierge Production's own Scene library)
**Depends on:** Kernel 66 (Show Run), Kernel 67 (Show), Kernel 68 (Stage Management authority, Production onboarding)
**Supersedes/continues:** Kernel 66/67/68's own "next recommended kernel" pointer (Scene Configuration Model / Capture Scene)
**Expected next consumer:** A later Fly Scene / Capture / Session-integration kernel; a future Script/hyperlink kernel

---

## 0. Purpose

Victory now has the identity/social spine and the show/run spine:

```text
Trailer → Trailer Face → Third Place Headshots → My People
Production → Show Run → Show → Session link
Stage Management → Production / Show Run / Show setup
```

Kernel 68 tightened venue visibility and backstage access, renamed the Show Runs surface to **Stage Management**, added minimal Production creation, and gated Third Place behind Trailer Face readiness. Kernel 69 adds the first real reusable play object: **Scenes**.

A Scene is not merely a child row of one Show. Scenes can be reused on stages, in individual Shows, in repertory/company contexts, and across Shows. Kernel 69 avoids trapping Scene content inside a single Show.

## 1. Resolved operator decisions

- **User-facing term: Scene.** Not Still, Beat, Frame, Moment, or Setup.
- **Reusability**: a Scene may be staged in multiple Shows. A Show may contain zero, one, or many Scenes.
- **Two-layer model**: Scene (reusable) + Show Scene Placement (its use inside one Show). UI hides the word "placement" and says "Scenes in this Show" / "Add Scene to Show" / "Remove Scene from Show."
- **Scope**: Production-scoped (`scenes.production_id NOT NULL REFERENCES productions(id)`), not a global/company library yet.
- **Venue targeting**: a Scene may have a default target Venue; the Show placement may override it without mutating the Scene.
- **Script/hyperlink caution**: future work. `source_ref`/`config_json` reserve room; no script editor, hyperlink graph, or line anchors built now.
- **Statuses**: draft, ready, retired, archived. No `live` status this kernel.
- **Audience/backstage fields**: curated audience fields (audience_title, audience_summary) plus backstage-only fields (player_brief, director_notes, operator_notes), with a small placement-level override set.
- **First proof example**: `Socio- : Character Making — Opening`.

## 2. Scope

### Included
- `scenes` and `show_scene_placements` tables (migration 042).
- `backend/internal/scenes` package: CRUD for Scenes, CRUD for Show Scene Placements, Production-scoped authority reusing `showruns.CanManageShowRun`/`CanViewBackstage`.
- Routes: `GET/POST /api/scenes`, `GET/PATCH /api/scenes/{id}`, `POST /api/scenes/{id}/archive`, `GET/POST /api/shows/{show_id}/scenes`, `PATCH /api/shows/{show_id}/scenes/{placement_id}`, `POST /api/shows/{show_id}/scenes/{placement_id}/archive`, `GET /api/shows/{show_id}/scenes/program`.
- Frontend: `frontend/venues/show-runs/scenes.html` (Production Scene Library), a "Scenes in this Show" section on `show.html`, a "Scene Library" link on `run.html`, and a curated Scenes card on `show-program.html`.
- Dictionary, operator log, operator notes, fresh-install smoke updates.

### Explicitly excluded
Scene capture, still/snapshot capture, Fly Scene/make-active, active scene pointer, scene transitions, cue groups, session auto-linking, script editor, hyperlink graph, full repertory/company library, tag/circle/social scarcity system, comments/feeds/impressions/autographs, destructive hard delete, full permissions matrix for crew.

## 3. Canonical model

- `scenes`: Production-scoped, `UNIQUE(production_id, slug)`, `status IN (draft, ready, retired, archived)`, `archived_at` matches status via CHECK.
- `show_scene_placements`: `UNIQUE(show_id, scene_id)` (a Scene may be staged at most once per Show, but independently in any number of other Shows), `scene_id ON DELETE RESTRICT` (a Scene can never be silently orphaned out from under Show history), same status enum as Scene.
- Archiving a placement never mutates the Scene it points at; archiving a Scene never mutates existing placements, only blocks new ones (`scene_archived` error).
- Curated Audience visibility requires `status = 'ready' AND archived_at IS NULL` on the placement; the curated shape (`AudienceScenePlacement`) has no backstage fields at the Go type level.

## 4. Authority

- `CanManageScenesForProduction`/`CanViewScenesBackstage` resolve `production_id → location_id` (same lookup `CreateShowRun` already performs) and delegate to `showruns.CanManageShowRun`/`CanViewBackstage` — no duplicated authority logic (Kernel 67's precedent for security-relevant code).
- Show Scene Placement authority uses the placement's Show's parent Show Run location directly (same two-step `LoadShowByID` → `LoadShowRunByID` lookup `shows.CreateShow` already performs).
- `CreatePlacement` validates the target Scene's `production_id` matches the Show's resolved `production_id` (`scene_production_mismatch` if not) — a client cannot stage a Scene from a different Production into a Show.
- Client-supplied `production_id`/`show_id`/`scene_id` are never trusted for authority, only for "which row" — authority is always resolved server-side from the loaded row.

## 5. Validation commands

```bash
cd /opt/victory/backend
GOCACHE=/tmp/victory-gocache go build ./...
GOCACHE=/tmp/victory-gocache go vet ./...
gofmt -l ./internal/scenes ./cmd/victory
TEST_DATABASE_URL="postgres://victory:REDACTED@127.0.0.1:5432/victory_test?sslmode=disable" go test -count=1 ./...
```

```bash
cd /opt/victory
scripts/smoke/fresh-install.sh --local
```

See `Construction/OperatorLogs/kernel-69-reportback.md` for full results.

## 6. PASS standard

1. Scene is the user-facing term. ✓
2. Scenes are reusable and not trapped under one Show. ✓
3. A Show can stage/order one or more reusable Scenes. ✓
4. Scene placement can target/override a Venue without mutating the base Scene. ✓
5. Stage Management authority protects all backstage Scene routes. ✓
6. Audience-facing Scene data is curated if exposed. ✓ (implemented, not deferred)
7. No capture/fly/transition/script-hyperlink system was built. ✓
8. Kernel 64 dedicated-test-DB workflow is followed. ✓
9. Fresh-install proof passes. ✓
10. Dictionary is updated. ✓
