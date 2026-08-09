# Show Cohort / Scene Progression Contract (Kernel 85)

## The problem

Before Kernel 85, a Show had exactly one current Scene (`shows.current_show_scene_placement_id`), broadcast identically to everyone connected. Kernel 74 already proved a *narrower* per-viewer divergence works (an individual Player's tutorial-handoff projection, `participant_local_projections`), but nothing let a Director split a Show into multiple independently-progressing groups for sustained play. This document is the schema + resolution-order decision for that.

## Schema (migration 096)

```text
show_cohort_serial_counters (show_id PK, next_serial)
show_cohorts (id, show_id, serial_number, slug, name, current_show_scene_placement_id, ...)
show_cohort_assignments (show_id, user_id PK, cohort_id, assigned_by_user_id, assigned_at)
```

`show_cohort_assignments` has a **primary key on `(show_id, user_id)`**, not a plain foreign-key membership table with a history of rows. This is deliberate: it makes "at most one active cohort per participant" true by construction rather than an application-level check that could drift — moving a participant is an `UPDATE`, returning them to Ungrouped is a `DELETE`. Ungrouped is never a stored row: it is computed as every active Player-role `show_run_roster_members` row for the Show minus whoever has an assignment row (`cohorts.ListRosterForShow`).

Serial allocation (`cohorts.CreateCohort`) locks a per-show counter row (`SELECT ... FOR UPDATE`) inside the same transaction that inserts the cohort, so concurrent Director requests can never collide, and archiving a cohort never decrements the counter — a deleted Cohort 2's number is never reissued. See `backend/internal/cohorts/cohorts_test.go`'s `TestConcurrentCohortCreationIsSerialSafe` for the concurrency proof.

A cohort's current Scene is `current_show_scene_placement_id`, the exact same column shape as the Show's own pointer (`shows.current_show_scene_placement_id`) — reusing `show_scene_placements`, not inventing a parallel Scene-pointer concept.

## Resolution order: reusing Kernel 74's per-viewer pattern, not inventing a new one

`world.LoadVenueSnapshot` already resolved a *different* composition per viewer before this kernel: role-based visibility filtering, then (if present) Kernel 74's `participant_local_projections` override. Kernel 85 adds one more link in that same chain, in `backend/internal/world/kernel85_cohort_projection.go`'s `resolveCohortPlacementForViewer`:

```text
1. localProjection (Kernel 74 individual tutorial-handoff) -- always wins if present
2. cohort's current_show_scene_placement_id (Kernel 85) -- checked only if no local projection
3. Show's global current_show_scene_placement_id (pre-Kernel-70A/85 default) -- unchanged fallback
```

`snap.Session.CurrentShowScenePlacementID` in the wire payload always reports the Show's own unchanged global pointer regardless of which branch actually loaded — the same "what moved is only which Scene's elements load" guarantee `LocalProjection`'s doc comment already establishes. Ungrouped participants have no `show_cohort_assignments` row, so step 2 always returns `""` for them and they fall straight through to step 3 — this is the entire mechanism behind "Ungrouped cannot cross the sustained-play boundary" (kernel-85 §5.6). No separate boundary-enforcement check exists or is needed: there is simply no assignment row for a cohort-scoped Scene to resolve.

**Why not import `backend/internal/cohorts` into `world`:** `cohorts/http.go` imports `backend/internal/network` for its broadcast call, and `network` already imports `world` (`director_console.go`) — importing `cohorts` from `world` would close that cycle (`world → cohorts → network → world`). `kernel85_cohort_projection.go` duplicates the narrow read query instead, the same avoidance already used for `shows.current_show_scene_placement_id` two lines above it in `snapshot.go`.

## Live sync

Activating a Scene for a cohort (`cohorts.HandleCohortCurrentScene`) broadcasts the exact same `network.BroadcastShowStageInvalidation` event every other current-Scene change already uses — a uniform "go refetch" signal to every session linked to the Show. There is no per-cohort-targeted push: the fan-out stays cheap and untouched, and correctness comes from each viewer's own refetch resolving through the chain above. A cohort whose Scene didn't change simply refetches and gets the same data back.

## Authority

Every cohort mutation (`cohorts.requireManage`) re-checks `showruns.CanManageShowRun` against the Show's parent Show Run location — Director+/Producer/Operator only, matching kernel-85 §10. `AssignParticipant` additionally verifies the target is an active Player-role roster member of *this* Show (`isActivePlayerOnShow`) before allowing assignment, and `ActivateSceneForCohort` verifies the placement belongs to the same Show as the cohort before accepting it.
