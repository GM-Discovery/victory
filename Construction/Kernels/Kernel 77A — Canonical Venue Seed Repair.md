# Kernel 77A — Canonical Venue Seed Repair: Audition Hall and Map Visibility

**Status:** READY FOR IMPLEMENTATION
**Type:** Mini-kernel / migration repair
**Parent sequence:** After Kernel 77, before Kernel 78
**Primary tracks:** Victory Core, Operational Integrity
**Scope:** Restore canonical venue data lost by clean database rebuilds

---

## 0. Reason for this kernel

Kernel 76 rebuilt the live database from migrations and deliberate seeds. The rebuild proved that the current migration history restores only the venue rows represented in code and migrations.

The Audition Hall had previously been added manually from the terminal as a historical fill rather than through a migration. Because of that, a clean rebuild removed it.

The Audition Hall is not optional decorative content. It is a durable navigation and progression venue:

- players use it to request access to other venues;
- it participates in tutorial progression;
- it must remain visible to nearly all users;
- its existence must survive clean installs, database resets, and restores.

This kernel repairs that missing canonical data and verifies whether any other currently required venues share the same problem.

---

# 1. Product decision

The Audition Hall is part of Victory's canonical venue topology.

It must:

- exist after a clean database rebuild;
- have stable identifiers;
- appear on the intended map;
- remain visible to authenticated users by default;
- remain available before a user has admission to more restricted venues;
- support the tutorial and venue-request flow;
- not depend on a one-time terminal command or live-database patch.

The exact permissions for performing actions inside the Audition Hall remain governed by existing admission and role rules. Broad visibility does not imply unrestricted administrative authority.

---

# 2. Scope rule

Do **not** blindly convert every venue currently present in the live database into migration seed data.

The live database may contain:

- temporary venues;
- test fixtures;
- historical leftovers;
- one-off experiments;
- user-created content;
- manually repaired rows;
- obsolete venue records.

Instead, reconcile three sources:

1. current migrations and bootstrap seeds;
2. current repository references and navigation expectations;
3. the intended canonical venue list documented by current product behavior.

Only venues proven to be part of Victory's canonical installed world should be added or repaired through migration.

---

# 3. Goals

## Goal A — Restore the Audition Hall canonically

Create an additive, idempotent migration that ensures the Audition Hall and its required related records exist.

The migration must establish or repair:

- Location association;
- Lot association if the schema requires one;
- Venue row;
- stable slug or key;
- display name;
- map placement or map-registration data;
- default visibility/admission policy;
- navigation metadata;
- any capability rows required for the venue to load;
- any tutorial/progression references that depend on it.

Do not generate duplicate rows when the migration runs against a database where the Audition Hall was previously added manually.

## Goal B — Audit the canonical venue set

Compare the clean-install venue set against the intended canonical installed venue set.

Produce a small reconciliation table:

| Venue | Exists in migration/bootstrap | Referenced by current app | Canonical installed venue | Action |
|---|---|---|---|---|
| Audition Hall | No | Yes | Yes | Add migration |
| Example venue | Yes | Yes | Yes | None |
| Temporary test venue | No | No/unclear | No | Do not seed |

The audit must answer:

- Are any other required built-in venues missing from migrations?
- Are any seeded venues obsolete?
- Are map/navigation references pointing to nonexistent venue rows?
- Are any built-in venue rows dependent on historical manual SQL?

Repair additional venues only when evidence clearly shows they are canonical and missing.

## Goal C — Prove clean rebuild behavior

On a fresh scratch database:

- apply all migrations;
- verify the Audition Hall exists;
- verify all canonical venue rows exist exactly once;
- verify the map/navigation API includes the Audition Hall;
- verify expected default users can see it;
- verify an unrelated restricted venue remains restricted;
- verify tutorial/request flow can resolve the Audition Hall.

## Goal D — Preserve live data safely

The migration must be additive and safe against the current live database.

Before live migration:

- take a database backup;
- inspect whether an Audition Hall row already exists;
- inspect its identifiers and related records;
- merge or update rather than duplicate;
- preserve any intentional current placement/configuration unless it conflicts with the canonical definition.

---

# 4. Required investigation

Before writing the migration, inspect:

- venue schema;
- Location and Lot schema;
- map visibility tables or policies;
- venue capability tables;
- navigation and map APIs;
- tutorial progression code;
- venue-request/admission code;
- current venue migrations;
- bootstrap seeds;
- current live Audition Hall row, if any historical record or dump still contains it;
- references to `audition`, `audition-hall`, or equivalent names across the repository.

Do not ask Grant for table or file names that the repository can reveal.

---

# 5. Canonical venue reconciliation

Create:

```text
Construction/Domains/Operations/kernel-77a-canonical-venue-reconciliation.md
```

For every built-in venue currently expected by the application, record:

- stable key/slug;
- display name;
- Location;
- Lot if applicable;
- seeded by which migration;
- map registration source;
- default visibility;
- default admission behavior;
- whether referenced by current frontend/backend code;
- whether present on a fresh database;
- disposition.

The document is evidence, not a new permanent competing roadmap.

---

# 6. Migration requirements

Create the next valid migration number after inspecting the repository.

The migration must be:

- additive;
- idempotent;
- safe on clean install;
- safe on the current live database;
- explicit rather than dependent on application startup side effects;
- stable across future rebuilds;
- free of test-only data.

Use stable keys and conflict handling appropriate to the existing schema.

Do not key the repair solely on display name if a stable slug or canonical identifier exists.

If an existing manually created Audition Hall row uses a different identifier:

- preserve linked data where possible;
- normalize it to the canonical identifier safely;
- document the merge;
- do not leave two Audition Halls.

---

# 7. Visibility and admission behavior

The Audition Hall should be broadly visible because it is the place users request access and advance the tutorial.

Required behavior:

- visible to authenticated users by default;
- visible even when the user lacks admission to restricted venues;
- present in map/navigation responses used by ordinary users;
- does not reveal private data from other venues;
- does not grant Producer, Director, or operator authority merely through visibility;
- venue-specific actions remain server-authorized.

If current architecture distinguishes visibility, entry, and action authority, preserve that distinction:

```text
Visible on map
  ↓ admitted to every private function
  ↓ granted elevated role
```

Anonymous/public visibility should follow current product policy and must not be broadened casually. The kernel's default target is authenticated users.

---

# 8. Required tests

## Migration tests

- fresh database creates Audition Hall;
- migration rerun does not duplicate it;
- database with historical manually created Audition Hall is normalized without duplication;
- all canonical built-in venues exist exactly once;
- no temporary/test venue is added accidentally.

## Visibility tests

- ordinary authenticated user sees Audition Hall;
- user without other venue admission still sees Audition Hall;
- anonymous user behavior matches current policy;
- restricted venue remains hidden or denied as designed;
- map/navigation endpoint includes correct Audition Hall metadata;
- venue-request/tutorial flow resolves the Audition Hall.

## Regression tests

- full Go test suite passes;
- live database is not modified by test suite;
- Kernel 76/77 authorization boundaries remain intact;
- `straturli` retains expected visibility and access;
- clean database rebuild produces the expected canonical venue count.

---

# 9. Required evidence

Record:

```bash
cd /opt/victory
git status --short
git rev-parse HEAD
git branch --show-current
```

Before migration, record current live venue rows relevant to the Audition Hall without exposing private user data.

After migration, provide:

- migration number;
- Audition Hall stable ID/slug;
- Location/Lot association;
- visibility/admission policy;
- fresh-database proof;
- live-database proof;
- canonical venue count;
- duplicate check;
- ordinary-user visibility result;
- restricted-venue negative result;
- full test-suite result;
- `git diff --check`.

---

# 10. Required artifacts

## Kernel specification

```text
Construction/Kernels/Kernel 77A — Canonical Venue Seed Repair.md
```

## Venue reconciliation

```text
Construction/Domains/Operations/kernel-77a-canonical-venue-reconciliation.md
```

## Migration

Use the next verified migration number and repository naming convention.

## Reportback

Use the standard kernel reportback template.

The reportback must identify:

- whether only the Audition Hall was missing;
- whether any other canonical venues were repaired;
- whether obsolete seeded venues were discovered;
- exact fresh-install venue count;
- exact visibility result;
- whether historical manual data was merged.

---

# 11. Pass criteria

Kernel 77A passes when:

- Audition Hall is represented by migration/bootstrap truth;
- clean rebuild restores it automatically;
- it appears exactly once;
- authenticated ordinary users can see it;
- venue-request/tutorial code can resolve it;
- restricted venues remain protected;
- any additional canonical venue gaps are either repaired or explicitly documented;
- no live test data is required;
- full tests pass;
- no duplicate or destructive migration behavior exists.

---

# 12. Non-goals

This mini-kernel does not:

- redesign the map;
- redesign the Audition Hall;
- add new tutorial content;
- create every venue found in historical data;
- preserve obsolete test fixtures;
- broaden anonymous access;
- rewrite the venue permission system;
- begin Victory Documents;
- reopen the full security audit.

---

# 13. Kernel-maker note

Grant's suggestion to "check all current venues" is directionally correct but must be constrained.

The correct rule is:

> Audit every venue the current application treats as built-in, then make the canonical installed venue set reproducible from migrations.

The incorrect rule is:

> Copy every row currently present in the live database into a migration.

The first repairs architecture. The second fossilizes accidents.
