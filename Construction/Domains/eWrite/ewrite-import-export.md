# eWrite Import and Export (Kernel 78)

Implementation: `backend/internal/ewrite/importer.go`, `export.go`, plus the
account-lifecycle hooks in `backend/internal/identity/account_export.go` and
`account_deletion.go`.

## Import

- Inputs: pasted Markdown (JSON) or uploaded `.md`/`.markdown`/`.txt`
  (multipart) via `POST /api/ewrite/publications/{id}/import`.
- Normalization: strict UTF-8 (reject otherwise), CRLF→LF, BOM stripped.
  What is stored is this normalized source — export returns it byte-exact.
- Import IS a save: it runs through `SavePublicationSource` (same conflict
  check, same revision, same section reconcile). There is no second write
  path.
- The `ImportReport` returned (and shown in the Writer's Room) records:
  filename, byte size, word count, heading/section/subsection/spacer
  counts, explicit vs generated anchors, internal/external links, images,
  raw-HTML count, duplicate-anchor disambiguations, and every warning.
  Warnings never silently rewrite content (spec 5.4).
- Caps: 2 MiB per manuscript (own MaxBytesReader inside the global 4 MiB
  body cap).

## Scale proof (measured, real manuscript)

`import_scale_dbtest_test.go` imports the real
`frontend/assets/rulesets/Sociov1_1.md` on every test run:
357,153 bytes · 43,207 words · 763 headings (137 explicit anchors preserved
verbatim, 154 spacers skipped) · render ≈ 136 ms · full save (render +
revision + 763 section rows, one tx) ≈ 0.9 s · re-save with reconcile over
existing rows ≈ 1.5 s. Budget 15 s each, with wide margin.

## Per-publication export

`GET /api/library/publications/{id}/export` → in-memory zip:

- `<slug>.md` — the source Markdown, byte-preserved;
- `metadata.json` — title, hierarchy path (root→collection titles),
  visibility, status (editors only), word count, section map,
  internal/external link manifest, image references, format_version.

Gated by `CanReadPublication`; nothing is ever staged under `storage/` or
`exports/` (the backup `--exclude=exports/` trap).

## Account export (Kernel 77 integration)

`buildExportArchive` gained an `ewrite/` section: every publication the
user **created** (any status — drafts are their work product), as
`<slug>-<id8>/publication.md` + `metadata.json` (including revision history
metadata). Manifest `format_version` bumped 1 → 2;
`counts["ewrite_publications"]` added; README lists the section.

## Account deletion (Kernel 77 integration)

Ordered before the generic reassign slice in `executeDeletion`:

- sole-owned drafts (no other named editor) → hard-deleted (satellites
  cascade);
- everything else — published/archived works, shared drafts, collections,
  revisions, grants, links — → reassigned to the tombstone user; the work
  survives, authorship is anonymized, no ownerless rows remain.

Deletion-plan preview gained `ewritings`. Tests:
`identity/kernel78_ewrite_lifecycle_test.go`.

## Backups

All eWrite data lives in Postgres → covered automatically by the
pre-migration dump and the 15-minute scheduled `pg_dump` (Kernel 72/77).
Zero new backup surface.
