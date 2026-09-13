# Storyboards Export Format (Kernel 80)

Implementation: `backend/internal/storyboards/export.go`.
`GET /api/storyboards/{board_id}/export`.

## Why synchronous, in-memory

Follows `ewrite.HandleLibraryExport`'s pattern, not the async job/polling
machinery in `identity/account_export.go`: a single board's JSON is
small and bounded (≤200 columns, whatever rows/cards exist), so building
job-queue infrastructure for it would itself be unrequested scope beyond
what this kernel needs.

## Authority

`BuildBoardExport` requires `CanExportBoard` (owner or Director+ —
`tierAtLeastDirector`; Crew never exports, spec 5.3-5.5).
`TestExportVersionedFormatAndHiddenAuthority` proves a Crew-tier caller
gets `ErrNotAuthorized`.

Hidden cards are included **only** when the exporter's own tier
authorizes seeing them — `BuildBoardExport` calls
`projectBoardSnapshotForTier` internally (the same hidden-filtering
function `ProjectBoardSnapshot` and the WS layer use) rather than
reimplementing the filter a third time.

## Format

```json
{
  "format": "victory-storyboard",
  "format_version": 1,
  "exported_at": "...",
  "board": { "id", "title", "description", "owner_user_id", "owner_handle", "created_at", "updated_at", "archived_at" },
  "visibility_grants": [ { "user_id", "user_handle", "granted_role", "granted_at" } ],
  "columns": [ ...StoryboardColumn ],
  "bands": [ ...StoryboardBand ],
  "rows": [ ...StoryboardRow ],
  "cards": [ ...StoryboardCard, "ewrite_link": { "publication_id", "section_anchor" } | omitted ],
  "integrity_sha256": "..."
}
```

`format_version` starts at 1; bump it (with a comment explaining what
changed) the next time the shape changes, matching the convention
`identity/account_export.go`'s `manifest["format_version"]` already
uses.

## `ewrite_link` is a reference only

Never resolved publication content — just `publication_id` and
`section_anchor`, resolved via `storyboardCardEwriteLinks` (a batched
query over the `object_type = 'storyboard_card'` rows added by migration
091, same shape as `ewrite.RuleLinksForEquipmentItems`). Because the
export never embeds eWrite content at all, hidden or not, spec 11.4's
"no ... inaccessible eWrite content" is satisfied by construction — there
is no content to leak in the first place.

## No secrets

No session tokens, password hashes, or private My People data appear
anywhere in the document — the export only ever touches
`storyboards`/`storyboard_grants`/`storyboard_columns`/`storyboard_bands`/
`storyboard_rows`/`storyboard_cards`/`ewrite_object_links` tables, none
of which carry that data. `TestExportVersionedFormatAndHiddenAuthority`
includes a substring scan for `password`/`session_token`/
`victory_session` over the serialized bytes as a belt-and-suspenders
check.

## Integrity

`integrity_sha256` is a SHA-256 of the document marshaled *without* that
field (computed, then the field is set and the document returned) — a
tamper-evidence checksum, not a cryptographic signature. Useful for
"did this file change since it was exported," not authentication.

## Malformed / nonexistent board

`BuildBoardExport` returns `ErrBoardNotFound` for a board ID that doesn't
resolve at all (`TestMalformedBoardCannotExport`) — there is no partial
or "best effort" export path.

## Import

Not built in Kernel 80 (spec 11.5 confirms this is acceptable). The
format uses stable IDs and explicit `row_id`/`column_id`/`band_id`
references throughout rather than positional/visual coordinates, so a
later kernel can import without reconstructing meaning from layout.
