# Storyboards Card Image Contract (Kernel 81)

One optional pinned image per card (spec §2.6/§9), new in Kernel 81 —
absent from Kernel 80's field list entirely.

## Data model — deliberately not a foreign key

`storyboard_cards.image_asset_id UUID` (migration 092), **no `REFERENCES`
clause.** This is explained at length in the migration's own comment; the
short version: two real asset-lifecycle paths would otherwise erase the
"an image was pinned here" state the gravestone requirement depends on —
`identity/account_deletion.go` hard-`DELETE`s an uploader's own
unused assets, and `assets.tombstoneWarehouseAsset` soft-deletes
(`is_deleted`/`deleted_at`, clears storage) but keeps the row. A real FK
would survive the second path but not the first (`DELETE FROM assets`
would either cascade the reference away or fail the delete outright
depending on the FK action chosen — both wrong). An unconstrained id
column survives both: the reference simply becomes unresolvable, which is
exactly what "gravestone, not silent disappearance" means.

## Upload path — a new, deliberately separate pipeline

`assets.HandleWorkshopUpload`/`HandleTokenUploadAsset` (the two existing
upload endpoints) both gate on `resolveProducerScope`, which requires the
*uploader* to be a Producer somewhere (or Operator). Storyboards Crew+
card editors have no such requirement — Storyboard authority is entirely
per-board (`storyboard_grants`), independent of any location/production
role. Reusing the existing endpoints as-is would have silently locked out
most real Crew editors.

Instead:

- `assets.CreateReferencedImageAsset` (`backend/internal/assets/
  card_image.go`, new, exported) — a leaner sibling of
  `HandleWorkshopUpload`'s core logic (same `sniffAllowedMime`/
  `decodeImage`/`resizeToMax` helpers, same `assets` table insert shape),
  with **no membership gate of its own** — the caller (Storyboards) does
  its own authority check first. One "thumbnail" derivative (480px max
  dimension) rather than three pixel-keyed ones; the original file is
  kept as-is for the lightbox's full-size view.
- `storyboards.resolveCardImageStorageScope` (`backend/internal/
  storyboards/card_image.go`) resolves *whose* storage quota the upload
  counts against: the **board owner's** own producer membership if they
  have one, else the same 'amurray-family' fallback location an Operator
  upload already uses. The *uploading* Crew member's own membership is
  irrelevant — only `canMutateCard`'s Crew+-unless-locked check gates the
  action itself.
- `storyboards.HandleCardImage` (`POST`/`DELETE
  /api/storyboards/{board_id}/cards/{card_id}/image`) ties these
  together: authenticate → `canMutateCard` (via `SetCardImage`) →
  `resolveCardImageStorageScope` → `CreateReferencedImageAsset` →
  `SetCardImage(..., created.AssetID)` → broadcast `card_updated`.

## A real authorization bug this surfaced (fixed in the same kernel)

Because the uploaded asset is stored under the *board owner's* producer
scope, `assets`'s existing content/meta read authorization
(`userCanReadAsset`, gated on `location_memberships` at that scope) denied
a legitimately-authorized Storyboard grant holder who had no location
relationship to the owner at all — found via Playwright with a throwaway
crew test account with zero memberships anywhere, not by inspection.
Fixed with `storyboardCardImageViewerAllowed` in `assets/read.go`: a
raw-SQL check (not an import of `storyboards` — that would close a direct
two-package cycle, since `storyboards` already imports `assets`) mirroring
`CanViewBoard`'s "any resolved tier, owner or grant" rule, inserted as an
early branch in `userCanReadAssetConsideringEwrite` before the
location-membership fallback. Every non-storyboard-card asset's
authorization is completely unaffected (the check returns `false, nil` —
not an error — whenever the asset isn't referenced by any
`storyboard_cards.image_asset_id` at all, falling through to the original
logic unchanged). Covered by
`assets/kernel81_storyboard_card_image_dbtest_test.go`.

## Frontend: thumbnail, full-size, lightbox, gravestone

- Card face: `<img src="/api/assets/{id}/content?variant=thumbnail">`
  inside `.sb-card-thumb-wrap`, click → lightbox.
- Lightbox (`#lightbox`): `<img src=".../content?variant=original">` (no
  matching named variant row exists for "original", so
  `resolveAssetContentPathFromParts` falls through to serving the raw
  stored file — full resolution). Close via ✕ button, click-outside, or
  Escape.
- Card editor modal: same thumbnail + Choose…/Remove buttons, uploads via
  `multipart/form-data` `POST`, `DELETE` clears the reference
  (`SetCardImage(..., "")`) without touching the underlying asset row —
  matches the existing `patchStageElementImage` precedent of "clear the
  reference, don't reach into someone else's storage."
- Gravestone: a background `GET /api/assets/{id}` (meta) fetch, cached
  per-session in `assetMetaCache`, decides via `grid-model.js`'s
  `isImageGravestone(hasImageRef, assetMeta)`:
  - `assetMeta === null` (request failed/404 — hard-deleted asset row) →
    gravestone.
  - `assetMeta.missing_asset === true` (soft-tombstoned) → gravestone.
  - otherwise → render the real thumbnail/original.

  On gravestone, the thumbnail `<img>` is replaced with a plain "Image
  unavailable" label — never a broken-image icon, never silently empty
  space, and never the underlying `original_filename`/path (which
  `GET /api/assets/{id}` does return but the frontend never surfaces).
  `/api/assets/{id}/content` itself always degrades gracefully
  server-side too (`serveConstructionFallback`, pre-existing machinery,
  unchanged) — the gravestone label is a deliberate *additional* UI state
  on top of that, not a replacement for it.

## Verified live (not just by inspection)

Real Playwright proof: uploaded a real PNG, confirmed `thumbnail` and
`original` variants both serve correctly with the right content-type,
opened the lightbox and closed it via Escape, then directly tombstoned
the underlying `assets` row in the database (`is_deleted=TRUE`,
`deleted_at=NOW()`, matching `tombstoneWarehouseAsset`'s exact effect) and
reloaded the board: the card face showed "Image unavailable", the modal
showed "Image unavailable (removed)", and `card.image_asset_id` was
unchanged in the API response throughout — the reference survived exactly
as designed (screenshots
`20-owner-gravestone-on-card-face.png`/`21-owner-gravestone-in-modal.png`).
