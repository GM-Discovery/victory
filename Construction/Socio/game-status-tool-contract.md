# Socio Game Status Tool Contract (Kernel 85)

## Repository audit finding this kernel had to close

Before this kernel, `character_cards` carried zero attribute or HP fields — no Might, no Health, nothing. This was not a "narrow missing operation" as the kernel spec initially anticipated (§2.2); the entire mechanical layer for the eight HP pools and any status/condition registry was absent, confirmed by grepping the whole `backend/internal` tree and cross-checking `Construction/Kernels/kernel-60-socio-skills-progression-and-sheet-DRAFT.md`, which already lists HP pools and statuses as rules-defined-but-not-app-enforced.

**Operator decision (2026-08-09):** rather than splitting this into a prerequisite sub-kernel (as kernel-85 §23's guardrail allows for a "substantial subsystem"), it was scoped deliberately narrow and built inline: paired current/maximum integers with fixed labels, no character-sheet UI wiring, no derived fields. `backend/internal/socio` is the result.

## Schema (migration 097)

```text
character_socio_state (character_card_id PK, {health,psyche,motion,will,essence,focus,perception,heart}_{current,max}, updated_at)
socio_statuses (key PK, label, description)          -- seed data, not a Go const list
character_socio_status_effects (id, character_card_id, status_key, intensity, applied_by/at, cleared_by/at)
```

`socio_statuses` is a registry table, seeded with the Socio source's named statuses (Raw, Winded, Wounded 1-3, Burning, ...) but intentionally not treated as exhaustive — adding a status later is a seed-row `INSERT`, not a code change (kernel-85 §2.3). A `character_socio_state` row is created lazily on first read or write (`ensureStateRow`), not backfilled for every existing Character — there is no meaningful default HP for a Character nobody has configured yet.

`character_socio_status_effects` has a **partial unique index** on `(character_card_id, status_key) WHERE cleared_at IS NULL` — applying an already-active status again updates its intensity via `ON CONFLICT ... WHERE cleared_at IS NULL DO UPDATE` rather than creating a second row. Clearing sets `cleared_at`/`cleared_by_user_id` rather than deleting, so `applied_at`/history for a status is never lost, and clearing an already-cleared status is an idempotent no-op (matching this codebase's established idempotent-mutation convention, e.g. `tutorial.RecordMilestone`).

## The one canonical write path

`socio.SetPool` is the only function that writes an HP pool value. It clamps `current` into `[0, max]` and rejects a negative `max` — a caller cannot request more current HP than max, or a negative pool, ever. Every mutation (`SetPool`, `ApplyStatus`, `ClearStatus`) re-checks, server-side, that the actor has Director+ authority over the given Show *and* that the target Character is actually on that Show's roster (`requireShowCharacterAuthority`) — a Director of Show A cannot mutate a Character who only appears on Show B's roster, even if they somehow know its id.

No second status store exists anywhere else in the codebase; Game Status reads and writes exclusively through this package.

## Game Status assembly is cohort-scoped, not venue-wide

`socio.BuildGameStatusForCohort` takes a `cohortID` that may be the literal string `"ungrouped"` (kernel-85 §1.11 explicitly allows Ungrouped as a selectable view) and reuses `cohorts.ListRosterForShow` to resolve exactly which Participants are in scope — it does not independently re-derive membership. A Participant with no `character_card_id` selected yet is skipped entirely rather than shown with invented zeroed state, since there is no real Character to attach HP/statuses to.

## What was deliberately not built

- **Social Stance**: confirmed not to exist as persisted state anywhere (only as manuscript prose inside eWrite fixtures) — kernel-85 §2.5 says not to invent a Stance subsystem merely to decorate this panel, so Game Status does not display or claim to support one.
- **Automatic collapse/death/recovery consequences**: the panel may show a depleted-HP warning in the frontend, but nothing server-side enforces a consequence when current hits 0 — that stays a Director judgment call (kernel-85 §2.4).
- **Character-sheet integration**: per the operator's inline-build decision, `character_cards` itself was not touched, and no character-sheet page reads or writes `character_socio_state`. Game Status is the only consumer.
