# Dice Roll Audience Resolution Contract (Kernel 86)

## The problem

Before Kernel 86, `actions.StoreDiceRoll`'s visibility was effectively public-only: `normalizeDiceVisibilityMode` accepted only `""`/`"public"`, and every roll's stored `visibility` JSONB was hardcoded to `{"toRoles":["audience","cast","crew","director","producer"],"privateTo":[]}` regardless of what a caller sent. Worse, `toRoles`/`privateTo` were write-only — nothing ever read them back. Live delivery used `hub.Broadcast`, a genuinely global fan-out to *every* connected socket on the server, not just this session's. This document is the audience-resolution model that replaces both halves.

## Package: `backend/internal/rollaudience`

A small leaf package (only `pgx` as a dependency) so it can be imported by both `backend/internal/actions` (roll storage, inside a transaction) and `backend/internal/world` (snapshot filtering, against the pool) without an import cycle — the same avoidance reason `world/kernel85_cohort_projection.go` gives for duplicating cohort-assignment SQL instead of importing `backend/internal/cohorts` (which imports `network`, and `network` already imports both `actions` and `world`).

Four modes: `show`, `cohort`, `director`, `private`. `"Director+"` means exactly what it means everywhere else in this codebase's authority checks (`actions/authority.go`'s `canActDiceRoll`, `cohorts.requireManage`): role `producer` or `director` — deliberately narrower than `world.isBackstageRole` (which also admits operator/crew for stage-Elements visibility).

```text
NormalizeMode(raw) -> "" | "cohort"    (both mean: apply the Kernel 86 default)
                    -> "public"/"show" -> "show"  (legacy backward-compat alias)
                    -> "director"      -> "director"
                    -> "private"       -> "private"
                    -> anything else   -> "" (rejected upstream as unsupported_visibility_mode)
```

`Resolve(ctx, q, sessionID, actorID, mode)` computes the actual audience server-side:

1. resolve the session's `show_id` (`sessions.show_id`, nullable — a session with no linked Show has no cohort concept);
2. if mode is `cohort` and a Show exists, resolve the actor's own `show_cohort_assignments` row for that Show;
3. if the actor has no cohort row (Ungrouped) — or there's no Show at all — fall back to `show` mode (spec §1.10's safe fallback).

The client can request a mode; it can never provide a cohort ID or recipient list. `Resolve` always derives the cohort from the *roller's own current* `show_cohort_assignments` row, the same idiom `world/kernel85_cohort_projection.go`'s `resolveCohortPlacementForViewer` already established for cohort Scene resolution — this file's `resolveCohortIDForActor` is its sibling, returning `cohort_id` instead of `current_show_scene_placement_id`.

## Storage: activating the existing `visibility` JSONB, not inventing a parallel field

`actions.StoreDiceRoll` (`backend/internal/actions/dice.go`) now writes:

```json
{
  "toRoles": [...],          // legacy/informational only, kept populated for display parity with every other action writer
  "privateTo": [],           // legacy; deliberately left empty -- see "why recipients aren't frozen" below
  "audienceMode": "cohort",  // the enforcement field
  "cohortId": "<uuid-or-empty>",
  "showId": "<uuid-or-empty>"
}
```

`payload.visibility_mode` is also set to the *resolved* mode (post-fallback), not the raw client request, so a roll stored while Ungrouped correctly reads `"show"`, not `"cohort"`.

**Why recipients aren't frozen at roll time:** both live delivery (`rollaudience.LiveRecipients`) and later snapshot-read filtering (`rollaudience.VisibleToViewer`) recompute membership fresh from `audienceMode`/`cohortId` against the viewer's *current* cohort assignment, actor identity, and role — never from a stored recipient list. This means leaving a cohort after a roll was made retroactively loses access to that old roll, and joining one retroactively gains access to it — the same "computed fresh each read" idiom the pre-existing Elements role-filter in `world.LoadVenueSnapshot` already uses, not a new pattern.

## Viewer omniscience rules

- The roller always sees their own roll, in every mode.
- `show` mode: everyone who can load this session/show's snapshot at all (unchanged default reach).
- `cohort` mode: cohort members (current assignment matching the roll's stored `cohortId`) **plus** Director+.
- `director` mode: Director+ only (besides the roller).
- `private` mode: **the roller only — no exception, not even for Director+.** This is the one deliberately strict case; `VisibleToViewer`'s `ModePrivate` branch returns `false` unconditionally for anyone but the actor. Proven by `rollaudience_dbtest_test.go`'s `"private: director does NOT see another user's private roll"` case and the live browser proof's Private-roll block.

## Read-side enforcement: the history-leak gap

`world.LoadVenueSnapshot`'s actions query previously returned every session/show-scoped `actions` row unfiltered — `a.Visibility` was decoded onto the `Action` struct but never checked against `viewerRole`/`viewerUserID`. Kernel 86 adds a narrow filter, scoped to `a.Type == "roll/dice"` only (every other action type's visibility handling is unchanged and out of scope — indexcard.go's `{"toRoles":["director","producer"]}` precedent remains exactly as unenforced as it was before this kernel, a pre-existing gap this kernel did not fix): before appending a `roll/dice` action to `snap.Actions`, `rollaudience.VisibleToViewer` is checked using the action's own stored `audienceMode`/`cohortId` and the snapshot's already-resolved `showID`/`viewerUserID`/`viewerRole`. A pre-Kernel-86 roll (no `audienceMode` key at all) or a malformed value defaults to `show` (`resolveStoredRollAudienceMode`, deliberately distinct from `NormalizeMode` — see the same-named helper's doc comment in `network/ws.go`) — fail-open to the old unrestricted behavior for legacy rows, never silently reinterpreted as the new Cohort default.

## What Kernel 86 explicitly did not touch

`canActDiceRoll` (Kernel 52) still gates *submitting* a roll to Director/Producer/Operator session role only — a pre-existing, deliberate authority boundary ("future controlled-character roll authority," per `Construction/OperatorLogs/kernel-52-reportback.md`), unrelated to and unchanged by this kernel's audience-*resolution* work. In practice this means a Cohort-scoped roll is normally submitted by a Director/Producer *for* their own cohort context, not by an ordinary Cast participant rolling for themselves — see the reportback's operator-note flagging this as worth a future kernel's attention if Victory ever widens roll authority to players directly.
