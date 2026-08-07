-- Kernel 81: Storyboards Presentation Rework
--
-- Adds one optional pinned image reference per card (spec 2.6/9.1).
--
-- Deliberately NOT a foreign key. Two existing asset lifecycle paths would
-- otherwise erase the "an image was once here" state that the gravestone
-- requirement (spec 9.5) depends on:
--   - identity/account_deletion.go hard-DELETEs an uploader's own assets
--     that aren't in active use elsewhere (no ON DELETE clause could
--     preserve the id through that).
--   - assets/warehouse.go's tombstoneWarehouseAsset soft-deletes (sets
--     is_deleted/deleted_at, clears storage) but keeps the row -- a real FK
--     would survive this one, but not the hard-delete path above.
-- Storing image_asset_id as a plain, unconstrained id column means both
-- paths leave the card's reference untouched; GET /api/assets/{id} then
-- reports missing_asset=true (tombstoned) or 404s (hard-deleted), and the
-- frontend renders a gravestone placeholder in either case rather than the
-- image silently disappearing.
BEGIN;

ALTER TABLE storyboard_cards
  ADD COLUMN IF NOT EXISTS image_asset_id UUID;

COMMIT;
