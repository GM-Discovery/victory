# Reference Panel Contract (Kernel 82)

Generic Storyboards infrastructure (spec §3.1) — not Timeline-specific at the data-model or API level, even though Timeline is the only mode that currently seeds any default fields. A Blank board can have Reference Panel fields added to it through the exact same endpoints; nothing gates field creation on `board.mode`.

## Schema

`storyboard_reference_fields` (id, storyboard_id, slug, label, field_type, sort_order, text_content, sublabel_a, sublabel_b) + `storyboard_reference_items` (id, field_id, side, sort_order, content). One field row per Reference Panel entry regardless of type; `text_content` is used only by `short_text`/`long_text`, `sublabel_a`/`sublabel_b` only by `paired_list`, and a `list`/`paired_list` field's actual entries live in `storyboard_reference_items` rows referencing it.

## Field types (spec 3.3) — exactly four, no form-builder

`short_text`, `long_text`, `list`, `paired_list`. `validFieldType` (`reference_panel.go`) is the single gate; adding a fifth type would require a migration to widen the `field_type` CHECK constraint, a code change to `validFieldType`, and new frontend rendering — there is no dynamic/pluggable field-type mechanism, deliberately, per spec's "do not grow this into a general form-builder kernel."

## Paired list (spec 3.7)

One field, two sublabels (`sublabel_a`/`sublabel_b`), two independently-ordered item lists distinguished by `storyboard_reference_items.side` (`'a'` or `'b'`). A plain `list` field's items all carry `side = 'single'`. `resolveItemSide` (`reference_panel.go`) rejects a side/type mismatch outright (`ErrReferenceItemSideInvalid`) — you cannot add an `'a'`-side item to a plain `list` field or a `'single'`-side item to a `paired_list` field. The default Timeline template's own paired field ("Include / Exclude") is exactly this mechanism with no special-casing: it's stored identically to any paired_list field a Director+ user creates by hand later.

## Serialization (Kernel 81A rules extended, spec 3.5)

Fields get a deterministic, board-scoped slug via the same `allocateUniqueSlug` used for columns/bands/rows — same collision behavior (`premise` → `premise-2` if a second field is literally titled "Premise"), same "assigned once at creation, never rewritten by rename" contract. `RenameReferenceField` only ever updates `label`/`sublabel_a`/`sublabel_b`; nothing in this package ever writes to `slug` after `AddReferenceField`'s `INSERT`.

## Authority (spec 3.4) — reuses the existing two-tier split exactly

No new authority function was needed. Content edits (field text, list/paired-list items — `SetReferenceFieldTextContent`, `AddReferenceItem`, `UpdateReferenceItemContent`, `RemoveReferenceItem`, `ReorderReferenceItems`) require `CanEditCards` (Crew+), the same gate card content editing already uses. Structural edits (`AddReferenceField`, `RenameReferenceField`, `ChangeReferenceFieldType`, `ReorderReferenceFields`, `RemoveReferenceField`) require `CanEditStructure` (Director+/owner/Operator), the same gate column/band/row structure already uses. Audience/Cast can view (Reference Panel content is never hidden-from-audience — spec 3.4 gates *editing* by role, not *viewing*, unlike cards) but every write path rejects them with `ErrNotAuthorized`.

## Type change safety (spec 3.4's "change field type where safe")

`ChangeReferenceFieldType` only succeeds when the field currently has no content (`referenceFieldHasContent` — empty `text_content` for text types, zero items for list types). This is a deliberate, narrower interpretation of "where safe" than attempting to migrate content between shapes (e.g. what would a `paired_list`'s two sides even mean converted to a `short_text`?) — the operator clears the field first (delete its items, or overwrite its text to empty), then changes type. Documented as a specific implementation decision in the Kernel 82 reportback, not silently assumed.

## Deletion confirmation (spec 3.6)

`RemoveReferenceField` takes a `confirmed bool`; a field with content (per the same `referenceFieldHasContent` check) returns `ErrReferenceFieldDeleteRequiresConfirm` unless `confirmed == true`. The HTTP layer reads this from `?confirm=true`; the frontend implements the "try, then explicitly ask, then retry" idiom already established for occupied-column/band/row removal in Kernel 80 rather than a separate always-shown confirmation dialog.

## Snapshot/export shape

`BoardSnapshot.ReferenceFields []ReferenceFieldWithItems` (a field with its items nested under it) is loaded by `projectBoardSnapshotForTier` alongside columns/bands/rows/cards — one request gets the whole board including its panel, no second round-trip. `ExportDocument.ReferenceFields` reuses the identical slice from the snapshot; Kernel 82 needed zero export-specific reference-panel code beyond adding the field to the struct.
