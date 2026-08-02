# Kernel 77A — Canonical Venue Reconciliation

**Date:** 2026-08-02.
**Method:** cross-referenced the live `venues` table (16 rows before this kernel, 17 after),
every `backend/migrations/*.sql` file, `backend/internal/access/kernel16_venue_bootstrap.go`
(the application-level idempotent seed path, invoked every boot from `main.go`), and every
`frontend/venues/*/` directory.

Two independent, equally valid canonical-seeding mechanisms exist in this codebase: SQL
migrations, and `EnsureKernel16VenueSurface` (idempotent, `WHERE NOT EXISTS`-guarded, called on
every backend startup — not a one-time terminal command, and therefore just as reliable as a
migration for this purpose). A venue reproducible through *either* counts as canonical and
migration-truth-backed. Only a venue reproducible through *neither* is a gap.

---

## Reconciliation table

| Venue | In a migration | In `EnsureKernel16VenueSurface` | Referenced by current app | Canonical | Disposition |
|---|---|---|---|---|---|
| audition-hall | **No (was missing)** | No | Yes — `visibility.go`, `shows.go`, `app.js`, full frontend dir | Yes | **Fixed — migration 083** |
| catharsis | Yes (055) | Yes (redundant, harmless) | Yes | Yes | None |
| directors-chair | Yes (011) | No | Yes | Yes | None |
| first-theater | Yes (044) | Yes (redundant, harmless) | Yes | Yes | None |
| grants-cabin | No | **Yes (045, 46-51)** | Yes | Yes | None — already reproducible via bootstrap |
| greenroom | Yes (007) | No | Yes | Yes | None |
| info-booth | Yes (005) | No | Yes | Yes | None |
| library | Yes (001) | Yes (redundant, harmless) | Yes | Yes | None |
| middle-school-stage | Yes (055) | Yes (redundant, harmless) | Yes | Yes | None |
| producers-office | Yes (011) | No | Yes | Yes | None |
| show-runs | Yes (039) | No | Yes | Yes | None |
| soil-experts | No | **Yes (68-75)** | Yes | Yes | None — already reproducible via bootstrap |
| the-cave | Yes (006) | No | Yes | Yes | None |
| third-place | Yes (038) | No | Yes | Yes | None |
| trailers | Yes (007) | No | Yes | Yes | None |
| warehouse | No | **Yes (60-67)** | Yes — `internal/assets/warehouse.go` | Yes | None — already reproducible via bootstrap |
| workshop | Yes (005) | No | Yes | Yes | None |

## Non-venue frontend directories, deliberately not seeded

| Directory | What it actually is | Why it's not a venue row |
|---|---|---|
| `frontend/venues/shared/` | One shared JS component (`venue-shell.js`) | Not a venue at all |
| `frontend/venues/construction/`, `construction-site/` | Both titled "Construction Site"; `construction/` also contains `delayed_features.md` | Generic under-construction placeholder, matching the kernel's explicit warning against seeding one-off/temporary content |
| `frontend/venues/stage-template/` | Titled "Stage Template" | A reusable scaffold for building stage-kind venues, not an installed venue itself |
| `frontend/venues/victory-theater/` | Titled "Victory Theater" | Landing content for the *`victory-theater` Location* (the neutral-install tenant seeded by migration 025), not a venue row under it — that location is deliberately kept venue-less as a blank slate for a new install to build into |

---

## Answers to Kernel 77A §3 Goal B's required questions

- **Are any other required built-in venues missing from migrations?** No — three
  (`grants-cabin`, `soil-experts`, `warehouse`) are absent from migrations specifically but are
  fully covered by `EnsureKernel16VenueSurface`, which is invoked automatically on every backend
  boot (`main.go:91`) and is idempotent. This is not the same failure mode Audition Hall had:
  Audition Hall was reproducible through *neither* mechanism.
- **Are any seeded venues obsolete?** None found. Every live venue row is referenced by current
  frontend or backend code.
- **Are map/navigation references pointing to nonexistent venue rows?** Only the one already
  fixed: `visibility.go`'s `v.slug IN ('audition-hall', 'trailers')` clause, and
  `frontend/app.js`'s hardcoded audition-hall pin/icon/click-target, were both already-correct
  code silently missing their intended row.
- **Are any built-in venue rows dependent on historical manual SQL?** Only Audition Hall was
  found to be, and it no longer is.

## Conclusion

Audition Hall was the only genuine gap. No other canonical venue repair is needed.
