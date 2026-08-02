# Kernel Report Back — Kernel 77A: Canonical Venue Seed Repair

## 1. Status

**PASS.**

Audition Hall was the only venue missing canonical migration/bootstrap truth. It is now
represented by an additive, idempotent migration, deployed live, and proven both on a
from-scratch database rebuild and on the live host.

---

## 2. What was found

- `audition-hall` had zero rows in the live `venues` table and zero coverage in any migration
  or in `EnsureKernel16VenueSurface` (the application-level bootstrap that seeds several other
  venues on every backend start) — the one true gap.
- `internal/access/visibility.go`'s `ResolveVisibleVenues` already hardcoded
  `v.slug IN ('audition-hall', 'trailers')` for its `authenticated_surface` visibility class,
  and `frontend/app.js` already had a complete, unconditional pin position, icon, and click
  target for it. Both were correct code silently returning/rendering one fewer venue than
  written, since Kernel 76's clean rebuild.
- Full reconciliation of all 16 existing venues plus every `frontend/venues/*/` directory found
  no other gap. Three venues (`grants-cabin`, `soil-experts`, `warehouse`) are also absent from
  migrations, but are fully covered by `EnsureKernel16VenueSurface`, invoked automatically on
  every backend boot (`main.go:91`) — not the same failure mode, since that mechanism is just as
  reliable as a migration, not a one-time terminal command. Five other `frontend/venues/`
  directories (`shared`, `construction`, `construction-site`, `stage-template`,
  `victory-theater`) were confirmed to be non-venue content (a shared component, placeholder
  pages, a template, and the *other* Location's landing page) and were correctly not seeded.
  Full detail in `Construction/Operations/kernel-77a-canonical-venue-reconciliation.md`.

---

## 3. What was built

`backend/migrations/083_kernel77a_audition_hall_seed_repair.sql` — additive,
`ON CONFLICT (lot_id, slug) DO UPDATE`, seeds Audition Hall under the same
`amurray-family`/`main-lot` every other canonical venue already lives in. No map-placement or
capability rows were needed: map pin/icon data is frontend-static (confirmed already present
and unconditional in `app.js`), and `venue_grid_configs` is empty for every venue, not just
this one.

Two new tests in `backend/internal/access/kernel77a_audition_hall_test.go`:
- `TestResolveVisibleVenuesAuditionHallOpenToAnyAuthenticatedUser` — a plain audience-role
  account sees Audition Hall and Trailers, and does *not* see the restricted `show-runs` venue.
- `TestAuditionHallExistsExactlyOnceAndUnderCanonicalLot` — exactly one row, correct
  Location/Lot.

---

## 4. Evidence

```bash
$ git rev-parse HEAD && git branch --show-current
c3a7a3a58c6f3a9e19216a7e0793c3d11821b7f1
main
```

**Idempotency**, run twice against the same database:
```
$ psql < 083_kernel77a_audition_hall_seed_repair.sql   # first run
INSERT 0 1
$ psql < 083_kernel77a_audition_hall_seed_repair.sql   # second run
INSERT 0 1
$ SELECT COUNT(*) FROM venues WHERE slug='audition-hall';
1
```

**Fresh, from-scratch database rebuild** (`reset-test-database.sh`, `CONFIRM_TEST_DB_RESET=1`):
```
PASS: victory_test reset, migrated, and Go-side bootstrapped from empty.
$ SELECT slug, name FROM venues WHERE slug='audition-hall';
audition-hall | Audition Hall
$ SELECT COUNT(*) FROM venues;
17
```

**Test suite**, fresh run against the rebuilt database:
```
$ go test -count=1 ./...
ok all packages, 0 failures
$ git diff --check
(clean)
```

**Live deployment:**
```
$ docker compose up -d --build backend
migrate: 1 pending migration(s): 083_kernel77a_audition_hall_seed_repair.sql
migrate: pre-apply backup written to /opt/victory/backups/victory_pre_migrate_20260802_180812_1pending.dump
migrate: applied 083_kernel77a_audition_hall_seed_repair.sql in 15ms
victory backend listening on :8081

$ curl https://victory.amurray.family/health
{"ok":true,"service":"victory-backend","time":"..."}

$ SELECT slug, name, kind FROM venues WHERE slug='audition-hall';
audition-hall | Audition Hall | commons
$ SELECT COUNT(*) FROM venues;
17
$ SELECT COUNT(*) FROM venues WHERE slug='audition-hall';
1

$ victory-recover whoami --handle straturli
is_operator: true, memberships: amurray-family producer (active), live_sessions: 1
```

An off-host encrypted backup (`scripts/backup/backup.sh full`) was taken immediately before the
live migration, in addition to `migrate`'s own automatic pre-apply dump.

---

## 5. Required Reportback Additions (per Kernel 77A §10)

- **Was only the Audition Hall missing?** Yes.
- **Were any other canonical venues repaired?** No — three others (`grants-cabin`,
  `soil-experts`, `warehouse`) were found absent from migrations but already reliably
  reproducible via `EnsureKernel16VenueSurface`; no repair needed.
- **Were any obsolete seeded venues discovered?** No.
- **Exact fresh-install venue count:** 17.
- **Exact visibility result:** a plain audience-role account sees `audition-hall` (reason:
  `authenticated_surface`) and `trailers`; does not see `show-runs` (Producer/Director/crew
  only).
- **Was historical manual data merged?** No merge was needed — the historical manually-inserted
  row no longer existed (removed by Kernel 76's clean rebuild); the migration created it fresh.

No blockers. No deviations from the kernel spec.
