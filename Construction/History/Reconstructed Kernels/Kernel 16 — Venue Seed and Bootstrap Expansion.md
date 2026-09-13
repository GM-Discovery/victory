# Kernel 16 — Venue Seed and Bootstrap Expansion

**Provenance:** RECONSTRUCTED — HIGH CONFIDENCE
**Original date/window:** Unrecoverable precisely; falls in the pre-Kernel-21 era (before 2026-05-05) based on surrounding numbering, no direct commit timestamp attributable
**Implementation status:** IMPLEMENTED — status not recorded (no surviving spec or reportback; PASS/FAIL was never documented)
**Evidence sources:** `backend/internal/access/kernel16_venue_bootstrap.go` (the surviving implementation); at least 7 later migrations/reportbacks reference it directly as the fresh-install venue seed baseline: `backend/migrations/056_kernel72a_index_cards_parity.sql:7`, `058_kernel73_participant_interactions_capability.sql:4`, `064_kernel73a_scene_stage_composition.sql:87`, `067_kernel74_local_projection_capability.sql:6`, `085_kernel78_writers_room_venue_seed.sql:9`, `099_kernel87_cartograph_capability.sql:4`; code comments in `backend/internal/network/session_control.go:184` and `backend/internal/actions/authority.go:57`; reportback prose in `Construction/OperatorLogs/kernel-72A-reportback.md`, `kernel-73-reportback.md`, `kernel-79-goal-ce-reportback.md`, `kernel-87-reportback.md`; and `Construction/eWrite/ewrite-domain-model.md:56`.

## Reconstructed purpose

Establish the canonical fresh-install venue seed: a Go-code bootstrap function (`EnsureKernel16VenueSurface`) that inserts the baseline set of venues (including `library`, `first-theater`) with their initial capability configuration, run on fresh installs after migrations complete. This became the reference "did the fresh-install seed keep up with new venue capability flags" baseline every later kernel that added a new venue capability flag had to check against.

## What evidence proves was built

- `EnsureKernel16VenueSurface` in `backend/internal/access/kernel16_venue_bootstrap.go` — inserts venue seed rows (`library`, `first-theater`, and others per the function body) with per-venue `config` JSON, `isPublic`/`isWorkshop` flags.
- The `library` element/venue slot this seed created sat unused for a long time — `Construction/OperatorLogs/operator-log.md:1740` (Kernel 78) explicitly notes "its half-built seeded slot from Kernel 16 finally has pages," confirming the seed predates real Library content by many kernels.
- At least seven later migrations/kernels (72A, 73, 73A, 74, 78, 87, and the 79-goal-CE reportback) had to specifically account for "does the Kernel 16 fresh-install seed also carry this new flag," each treating it as the authoritative fresh-install source of truth separate from migration-driven updates to *existing* rows.

## Files/systems affected

`backend/internal/access/kernel16_venue_bootstrap.go` (created here, amended by many later kernels to add capability flags to its seed configs); no migration file of its own — this is Go-code seed logic, not a SQL migration, which is itself notable (see Confidence section).

## Known deviations / later corrections

Kernel 72A (`kernel-72A-reportback.md:45`) found and fixed a real ordering-gap bug: because Go-seeded venues are created *after* migrations run on a fresh install, several capability flags added via migration for *existing* installations were silently missing from the Kernel 16 seed's own INSERT configs for *fresh* installations — Kernel 72A fixed this for `scene_rehearsal_enabled` and made a habit of checking it going forward.

## What this kernel handed to the next kernel

A durable, still-live fresh-install seeding pattern that every subsequent venue-capability-flag kernel has had to remember to also update — this is the concrete mechanism behind "does this feature exist on a fresh install," which is directly relevant to Kernel 99's own Part II fresh-install proof.

## Confidence / unresolved gaps

HIGH CONFIDENCE on *what* was built (the function and its role are unambiguous from seven independent citations across a 70+-kernel span). UNKNOWN: the original kernel's own stated purpose/spec text, its PASS/FAIL determination, and its exact date — no spec file or reportback survives, and no commit could be attributed specifically to its introduction (the file's own git blame was not traced further within this pass's timebox). Also notable: unlike most kernels of its era, Kernel 16 shipped as pure Go seed logic with no accompanying SQL migration, which is itself worth flagging as a historical anomaly if a future kernel ever needs to understand why fresh-install venue seeding and migration-driven venue updates are two separate code paths.
