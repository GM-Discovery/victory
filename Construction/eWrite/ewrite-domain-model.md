# eWrite Domain Model (Kernel 78)

Authoritative architecture note for the eWrite schema and its Go package
`backend/internal/ewrite/`. Migration: `084_kernel78_ewrite_schema.sql`
(tables), `085_kernel78_writers_room_venue_seed.sql` (venues).

## Shape

One typed collection tree plus distinct publication/section/revision
structures — the visible six-level hierarchy (Ruleset → Series → Module →
Publication → Section → Subsection) is a product-language contract, not six
tables.

```text
ewrite_collections        typed tree: kind ∈ {ruleset, series, module}
└─ ewrite_publications    one eWriting: source_markdown is truth
   ├─ ewrite_revisions    append-forward saves (never rewritten)
   ├─ ewrite_sections     stable-identity heading rows (UUID survives edits)
   │  └─ ewrite_anchor_aliases   old anchor → section after renames
   ├─ ewrite_editors      named per-publication grants
   └─ ewrite_object_links typed FK bindings (equipment_item today)
```

## Rules that matter

- **Parent-kind rules** (series under ruleset; module under ruleset *or*
  series — the omitted-Series case is deliberate) are enforced in
  `store.go: validateParentKind`, because a CHECK constraint cannot inspect
  the parent row. What CHECK *can* see is in the migration: a ruleset is
  always a root.
- **Publications may attach to any collection kind.** A standalone article
  does not need a ceremonial Series and Module.
- **`publications.location_id` is denormalized** from the collection so
  authority checks and search filters never walk the tree. The store keeps
  it consistent; cross-location collection moves are refused
  (`collection_location_mismatch`) rather than silently re-scoping readers.
- **`rendered_html` is a cache, never truth and never client-supplied.**
  It is regenerated inside the same transaction as every source write
  (`revisions.go: SavePublicationSource` — the single write path), so it
  cannot go stale.
- **`search_tsv`** is a stored generated column over title/summary/
  `search_text` (plain text extracted from the AST server-side). First FTS
  use in Victory; GIN-indexed.
- **FK behavior:** containers RESTRICT (deleting a non-empty collection
  fails loudly); satellites CASCADE with their publication; user refs SET
  NULL as backstop only — account deletion reassigns to the tombstone user
  first (see `identity/account_deletion.go`).
- **`ewrite_object_links`** copies the `stage_element_bindings` idiom
  (migration 064): one real FK column per object type + CHECK discriminator.
  A second object type adds a nullable FK column and CHECK arm — never a
  bare untyped `object_id`. `section_id` is SET NULL so a removed heading
  degrades the link to the publication top instead of killing it.

## Venues

- `library` — seeded since Kernel 16, made migration-canonical by 085,
  surfaced to all authenticated users via the `authenticated_surface` arm in
  `access/visibility.go`. Pages: `frontend/venues/library/` (browse +
  `read.html` reader).
- `writers-room` — new in 085; map-visible only to producer/director/crew
  (`ewrite_author_surface` arm) plus Operator. Pages:
  `frontend/venues/writers-room/` (dashboard + `edit.html` editor).
- Map visibility is never invocation authorization: every `/api/ewrite/*`
  route re-derives authority per request.

## API surface

Authoring `/api/ewrite/*` (tree, collections CRUD, publications CRUD,
`PUT .../source` save with conflict check, publish/unpublish, import,
preview, revisions, editors, object-links) and reading `/api/library/*`
(tree, publication reader payload, search, export). All method-prefixed;
writes behind the shared `actionLimiter`; no GET mutates.
