# eWrite Image Visibility (Kernel 79)

## The gap (Kernel 78)

`backend/internal/assets/read.go`'s asset-serving endpoint
(`/api/assets/{id}/content` and the metadata route) authorized purely on
**location membership / ownership** (`userCanReadAsset`): producer,
uploader, owner, or an active `producer/director/cast/crew` membership at
the asset's own `location_id`. It never consulted an eWrite publication's
own `visibility`/`status` at all, and — this was the more serious half —
**the content-serving route had no access check whatsoever for any
non-`map` asset type**. Every eWrite-embedded image, regardless of the
publication's visibility, served to any request (including anonymous)
that could resolve the asset ID.

This meant two things were simultaneously true and both wrong:

1. A public/published eWriting's image could still incorrectly 403 for a
   legitimate Library reader who wasn't a member of whatever location the
   image happened to be uploaded to.
2. A draft or Production-restricted eWriting's image had **zero** real
   protection — a copied `/api/assets/{id}/content` URL bypassed the
   publication's visibility entirely for any authenticated user, and for
   fully anonymous requests too.

## The fix

**New table**, `ewrite_publication_assets` (migration 086) — which
publication(s) currently reference which asset, in the same "real FK per
side, CASCADE with the owning row" idiom as `ewrite_object_links`
(migration 084):

```sql
CREATE TABLE ewrite_publication_assets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  publication_id UUID NOT NULL REFERENCES ewrite_publications(id) ON DELETE CASCADE,
  asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (publication_id, asset_id)
);
```

**Reconciliation** (`backend/internal/ewrite/asset_refs.go`): `Render()`
already extracted every image URL a publication's Markdown references
(`RenderResult.ImageRefs`) — Kernel 78 built this but only ever used it for
a UI count. `reconcilePublicationAssetRefs` now turns that into
`ewrite_publication_assets` rows, delete-then-insert per publication,
called from inside `SavePublicationSource`'s existing transaction
(`revisions.go`) — the single write path for both manual saves and
imports, so one hook covers both and the reference set can never go stale
relative to the content it describes.

`BackfillPublicationAssetRefs` covers publications that predate this table
(the live Core Rulebook) — it finds any publication with zero existing
rows, re-renders its current `source_markdown`, and reconciles. Runs once
at every boot; a publication with existing rows is skipped, so it only
ever does real work once per publication.

**Authorization** (`backend/internal/assets/read.go`,
`userCanReadAssetConsideringEwrite`): the new entry point every asset read
(both the content and metadata routes) now goes through.

```
if requester is the asset's producer/uploader/owner:
    allow
else if the asset is referenced by 1+ eWrite publications:
    allow if ANY of those publications' ewrite.CanReadPublication(user) is true
    (this is authority.go's existing, already-correct resolution --
     public/authenticated -> any signed-in reader, production -> active
     membership at the PUBLICATION's own location, draft -> edit authority)
    deny otherwise
else:
    fall through to the original userCanReadAsset (location membership) --
    completely unchanged for non-eWrite-bound assets
```

The key design decision: once an asset is eWrite-bound, its access is
governed **exclusively** by the referencing publication(s)' own visibility,
not by the asset's incidental location membership. This closes both
directions of the gap at once — a public eWriting's image is now readable
by any authenticated reader regardless of location, and a production/draft
eWriting's image is denied to anyone without a real relationship to *that
publication*, even if they happen to hold unrelated membership at the
asset's own storage location. Assets never referenced by any eWrite
publication are completely unaffected — this is strictly additive to the
existing model, not a replacement of it.

**Content-serving route**: previously had no access check at all for
non-map assets; now always calls the new helper before streaming bytes.
This is the actual security hole being closed, not just a refinement of an
existing check.

**Cache headers**: the metadata (JSON) response previously set no
`Cache-Control` at all; now sets the same `private, max-age=0,
must-revalidate` the content route already used, since visibility can
change over time (draft -> published, unpublish, a reference added or
removed) and neither response should be cached past an access-decision
change.

## What this does not cover (deferred)

- **Hierarchical export** (spec 10.5, "unauthorized images omitted with a
  manifest warning") — export itself doesn't exist yet in this kernel;
  deferred with the rest of Goal E.
- A dedicated "shared asset used by publications with different
  visibility" UI warning (spec 10.3 mentions this as one acceptable
  concern) — the chosen model (OR across all referencing publications'
  `CanReadPublication`) already prevents a private use from silently
  becoming public: adding a second, more-restrictive publication reference
  to an already-public asset does not revoke the public access the first
  reference grants, which is the correct behavior (the asset is genuinely
  public because at least one publication says so) but is worth flagging
  as a UX surface for a future kernel if Grant wants an explicit warning
  when authoring a draft that reuses an already-public image.

## Verification

Five new tests in `backend/internal/assets/ewrite_visibility_dbtest_test.go`,
each authoring a real publication (through the actual
`CreateCollection`/`CreatePublication`/`SavePublicationSource`/
`PublishPublication` API, embedding a real asset URL) rather than
hand-inserting rows, so the whole reconcile-to-authorize path is proven,
not just the read-side check in isolation:

- owner is always allowed, regardless of eWrite binding;
- a non-eWrite-bound asset behaves exactly as before (regression check);
- a public/published publication's image is readable by an authenticated
  user with no location membership anywhere (the false-negative half of
  the bug);
- a draft publication's image is denied to a non-editor and allowed to the
  publication's own author;
- a production-visibility publication's image is denied to a user who
  holds unrelated crew membership at the *asset's own* location but no
  relationship to the *publication's* location, and allowed to an
  audience-role member of the publication's own location (the
  false-positive / "unrelated asset access" half of the bug — the core
  scenario the Kernel 79 spec called out).

See the reportback for full command output.
