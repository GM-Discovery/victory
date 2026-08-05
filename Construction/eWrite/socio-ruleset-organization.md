# Socio Ruleset Organization (Kernel 79)

## Before

Kernel 78 seeded exactly one eWriting: a `ruleset` Collection
(`socio-stories-of-us`, "Socio: Stories of Us") with the Core Rulebook
Publication attached **directly** to that ruleset's root — no Series level
existed. This matched the Kernel 79 spec's own description of the starting
state (1.2/4.1): a flat Ruleset -> Publication with no intermediate
organization.

## After

```text
Socio: Stories of Us -- Ruleset (socio-stories-of-us)
├── Core Rulebook -- Series (core-rulebook)
│   └── Core Rulebook -- Publication (core-rulebook), 43,207 words
├── Quickstart -- Series (quickstart)
│   └── Socio-: The Locked Courtyard & Beyond -- Publication (the-locked-courtyard), 10,158 words
└── Niava -- Series (niava)
    └── Niava Setting Supplement -- Publication (niava-setting-supplement), 13,266 words
```

All three Series are `visibility='public'`; all three Publications are
seeded `status='published'`, `visibility='public'`, matching the existing
Core Rulebook default. Grant can change any of these later through Writer's
Room exactly like hand-authored content — nothing about the seed makes them
special or protected beyond the standard "create if absent, never revert an
edit" contract already established by `EnsureCanonicalSocioManuscript`.

## Reparenting the Core Rulebook without losing its identity

The Core Rulebook Publication already existed live (Kernel 78), with real
`ewrite_sections` (anchors) and potential `ewrite_object_links` (equipment
rule links) pointing at it by ID. Kernel 79 could not simply delete and
recreate it under the new Series — that would orphan every existing link
and anchor.

Instead, `EnsureSocioSeriesHierarchy` (backend/internal/ewrite/seed.go)
does a plain `UPDATE ewrite_publications SET collection_id = <series id>
WHERE collection_id = <ruleset id> AND slug = 'core-rulebook'` — the
publication's ID, slug, revisions, sections, anchors, and object-links are
completely untouched. Only which Collection it hangs under changes. This
was verified directly: the publication's UUID is identical before and
after the reorganization, in both the test database and the live
deployment.

## Source manuscripts

- **Quickstart**: Grant's "Socio-: The Locked Courtyard & Beyond" —
  cleaned from `Construction/eWrite/manuscript/Socio-_ The Locked Courtyard
  & Beyond.md` into `backend/internal/ewrite/seed/quickstart-v1.md`:
  - Removed 41 blank Google-Docs spacer heading lines (`^#{1,6}\s*$`,
    scattered through the whole document, not just the front matter) —
    these would otherwise create empty-titled `ewrite_sections` rows.
  - Removed the redundant auto-generated linked table-of-contents block
    (the bracketed `[Title N](#anchor)` list with print page numbers) —
    fully redundant with the Library reader's own generated TOC sidebar.
  - Fixed the 4 (of 47) explicit `{#...}` anchor ids that contained an
    en-dash (–), which is outside `markdown_policy.go`'s
    `anchorIDPattern` charset and would have silently fallen back to a
    generated slug with only a soft warning. The other 43 anchors were
    already valid and left untouched.
- **Niava**: Grant's "Niava Setting Supplement" — copied byte-for-byte
  verbatim from `Construction/eWrite/manuscript/Niava Setting Supplement.md`
  into `backend/internal/ewrite/seed/niava-v1.md`. No blank headings, no
  explicit anchors, no cleanup needed.

The original files in `Construction/eWrite/manuscript/` are Grant's
authoring copies and were left untouched — the cleaned/embedded copies
live only under `backend/internal/ewrite/seed/`, matching the existing
`socio-v1.1.md` precedent.

## A real bug found and fixed along the way

`EnsureCanonicalSocioManuscript`'s existence check joined through
`ewrite_collections` requiring the collection's slug to equal the
*ruleset's* slug. Once `EnsureSocioSeriesHierarchy` reparents the
publication into the Core Rulebook Series (a different collection, slug
`core-rulebook` instead of `socio-stories-of-us`), that join stopped
matching on every subsequent boot — `EnsureCanonicalSocioManuscript` would
have concluded the publication didn't exist and tried to recreate a
duplicate, which then collided with the reparent's own `UPDATE` (unique
constraint violation) on the *third* boot. Caught by actually booting the
compiled binary against the test database twice in a row (not just unit
tests) before calling this done.

Fixed by keying the existence check off `ewrite_publications.location_id`
+ `slug` directly (that column is denormalized from the collection root,
migration 084) instead of joining through a collection with a specific
slug. Regression-tested in `seed_hierarchy_dbtest_test.go`
(`TestSocioSeriesHierarchyAndManuscriptSeedsAreIdempotent`), which runs the
full boot-seed sequence twice and asserts zero duplication.

## What's deliberately not here (Kernel 79A / later)

Quickstart and Niava overlap with the Core Rulebook in places (e.g. the
Quickstart's Social Stances mechanic is a condensed version of the Core
Rulebook's fuller treatment) — per spec 4.4, Kernel 79 does not
automatically deduplicate or cross-link this content. Import curation
tooling (split/merge/reassign Publications, source-lineage tracking) was
also not built — with only 3 real manuscripts and one already-clean
reparent, hand-authored seed functions were sufficient and did not justify
building general curation UI in this pass.
