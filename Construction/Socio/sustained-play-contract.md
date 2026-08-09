# Socio Sustained Play Contract (Kernel 85)

## What "sustained" means here, precisely

Kernel 83's Group Leader / Current Turn state is **presence-scoped**: it lives in an in-memory `venuecoordination.Registry`, keyed by a live "who's currently watching" session, and evaporates when the last watcher leaves. That was the right call for Kernel 83's problem (turn-taking during an active viewing session) but is the wrong shape for cohort membership and Scene progression, which must survive a Show being closed and reopened days later.

Kernel 85's cohort/Scene/Character state is **Show-row-scoped, persistent Postgres state**, not presence:

```text
show_cohorts, show_cohort_assignments        -- persist independent of any live session
show_cohorts.current_show_scene_placement_id -- persists the same way shows.current_show_scene_placement_id already does
character_socio_state, character_socio_status_effects -- persist independent of any live session
```

None of it is cleared on session end, disconnect, or Show pause. `shows.current_show_scene_placement_id` already had this property since Kernel 70 (its own doc comment: "Ending a Session never clears it; only an explicit SetCurrentScenePlacement/ClearCurrentScenePlacement call does") — Kernel 85's cohort pointer and Socio state simply extend the same persistence guarantee to cohort-scoped and Character-scoped data.

## Reconnect

`world.LoadVenueSnapshot` resolves a viewer's cohort Scene fresh on every call (see `Construction/Shows/cohort-scene-progression-contract.md`'s resolution-order section) — there is no separate "restore on reconnect" code path to keep in sync, because there is no cached per-connection state to restore. A participant closing their laptop and reopening the Show a week later gets exactly the same resolution their still-connected cohort-mates would have gotten the whole time: their assignment row (if any) is looked up, their cohort's current placement is loaded, and the composition is rendered. Character HP/status reads the same way — `socio.GetState`/`ListActiveStatuses` have no session or connection concept at all.

## What Kernel 85 did not build

- **No Story So Far / narrative summarizer.** If a Show already has history/audit support elsewhere in the codebase, Kernel 85 does not duplicate or replace it — it was out of scope and not touched.
- **No append-forward audit log for cohort mutations beyond what Postgres columns already record** (`assigned_by_user_id`/`assigned_at` on `show_cohort_assignments`, `created_by_user_id` on `show_cohorts`, `applied_by_user_id`/`applied_at`/`cleared_by_user_id`/`cleared_at` on status effects). A dedicated action-log entry per cohort mutation was considered and deferred as not required for the kernel's pass criteria — the mutation columns already answer "who did this and when" for every write this kernel introduces.
- **Presence itself remains ephemeral**, per kernel-85 §9 — only cohort membership, cohort Scene, and Character mechanical state are durable.
