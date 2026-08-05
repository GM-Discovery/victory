-- Kernel 79: eWrite publication-asset references.
--
-- Kernel 78's asset-serving endpoint (backend/internal/assets/read.go)
-- authorizes purely on location-membership/ownership -- it never consults
-- an eWrite publication's own visibility. A public eWriting's embedded
-- image can 403 for a legitimate non-member Library reader, and there is
-- no path that restricts an image to a publication's Production/draft
-- scope if the requester happens to hold unrelated location membership on
-- the asset's own location.
--
-- This table records which asset(s) a publication's current source
-- actually references (reconciled from Render()'s ImageRefs on every save,
-- see backend/internal/ewrite/asset_refs.go), so the asset endpoint can
-- gate an eWrite-bound asset through the publication's own
-- CanReadPublication resolution instead. Same idiom as ewrite_object_links
-- (migration 084): a real FK per side, not a bare untyped pair, since this
-- is a referential-integrity-sensitive link rendered to other users.
--
-- publication_id CASCADEs with the publication (satellite row of one
-- work, same as ewrite_object_links). asset_id CASCADEs with the asset
-- (matching asset_derivatives' own CASCADE choice, migration 005) -- a
-- reference row is meaningless once the asset itself is gone.

CREATE TABLE IF NOT EXISTS ewrite_publication_assets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  publication_id UUID NOT NULL REFERENCES ewrite_publications(id) ON DELETE CASCADE,
  asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (publication_id, asset_id)
);

CREATE INDEX IF NOT EXISTS idx_ewrite_publication_assets_asset
  ON ewrite_publication_assets(asset_id);
