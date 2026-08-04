-- Kernel 78: eWrite foundation schema -- Victory's first rules-native
-- writing/publication/reading system (Writer's Room authors, Library reads).
--
-- Design decisions recorded here so the schema can be read without the kernel:
--
-- * One typed collection tree (ruleset/series/module) instead of three rigid
--   tables: the visible hierarchy is a product-language contract, not a
--   storage contract (kernel spec 1.3/25.2). Parent-KIND rules (series under
--   ruleset, module under series) are enforced in the Go store because a
--   CHECK constraint cannot inspect the parent row's kind. What CHECK can
--   see is enforced here: a ruleset is always a root.
--
-- * Publications may attach to a collection of ANY kind. A short standalone
--   article must not require a fake Series and Module (spec 1.2 "must not
--   require every integration to use all levels").
--
-- * publications.location_id is denormalized from the collection root so
--   authority checks and search visibility filters never need a recursive
--   tree walk. The Go store keeps it consistent on create/move.
--
-- * rendered_html is a CACHE of bluemonday-sanitized output, regenerated in
--   the same transaction as every source write (backend/internal/ewrite).
--   It is never accepted from a client (spec 16: "no raw HTML stored as
--   trusted rendered output without sanitization guarantees").
--
-- * search_tsv is a stored generated column (explicit 'english' regconfig --
--   generated columns require an immutable expression, so the two-argument
--   to_tsvector form is mandatory). search_text is plain text extracted from
--   the Markdown AST server-side ({#anchors}, escapes and link targets
--   stripped), which searches far cleaner than raw Markdown. left(...,
--   800000) guards the 1MB tsvector input limit; Sociov1_1.md (357KB) fits
--   comfortably.
--
-- * ewrite_object_links copies the stage_element_bindings idiom from
--   migration 064: one REAL foreign-key column per object type plus a
--   CHECK-constrained discriminator. The FK-free source_kind/source_ref
--   idiom (migration 070) is explicitly ruled out here by its own comment:
--   these links are rendered to other users, so referential integrity
--   matters. A second object type later adds a nullable FK column and a
--   CHECK arm -- never a bare untyped object_id.
--
-- * FK deletion behavior, deliberately chosen (spec 12.4/16):
--     - collection parents and publication containers RESTRICT: deleting a
--       non-empty container must fail loudly, never cascade a book away.
--     - revisions/sections/aliases/editor-grants/object-links CASCADE with
--       their publication: they are satellite rows of one work.
--     - every *_by user reference is ON DELETE SET NULL as a backstop only;
--       account deletion (backend/internal/identity/account_deletion.go)
--       explicitly reassigns surviving works to the tombstone user first,
--       inside its own transaction, exactly like the existing reassign slice.
--     - object_links.section_id SET NULL: removing a heading degrades an
--       equipment "view rule" link to the publication top instead of
--       deleting the link (spec 25.7: stable identity over pretty slugs).
--
-- * Visibility enum includes 'public' but Kernel 78 serves public content to
--   authenticated readers only -- the first anonymous content API is
--   deliberately deferred (operator decision recorded in the kernel spec
--   amendments). The schema is ready; the route is not.

CREATE TABLE IF NOT EXISTS ewrite_collections (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE RESTRICT,
  parent_id UUID REFERENCES ewrite_collections(id) ON DELETE RESTRICT,
  kind TEXT NOT NULL,
  title TEXT NOT NULL,
  slug TEXT NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  sort_order INT NOT NULL DEFAULT 0,
  visibility TEXT NOT NULL DEFAULT 'production',
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT ewrite_collections_kind_check
    CHECK (kind IN ('ruleset', 'series', 'module')),
  CONSTRAINT ewrite_collections_visibility_check
    CHECK (visibility IN ('public', 'authenticated', 'production')),
  CONSTRAINT ewrite_collections_ruleset_is_root
    CHECK (kind <> 'ruleset' OR parent_id IS NULL),
  UNIQUE (location_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_ewrite_collections_parent
  ON ewrite_collections(parent_id, sort_order);

CREATE TABLE IF NOT EXISTS ewrite_publications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  collection_id UUID NOT NULL REFERENCES ewrite_collections(id) ON DELETE RESTRICT,
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE RESTRICT,
  title TEXT NOT NULL,
  slug TEXT NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  source_markdown TEXT NOT NULL DEFAULT '',
  rendered_html TEXT NOT NULL DEFAULT '',
  search_text TEXT NOT NULL DEFAULT '',
  word_count INT NOT NULL DEFAULT 0,
  sort_order INT NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'draft',
  visibility TEXT NOT NULL DEFAULT 'production',
  current_revision_id UUID,
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
  published_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  search_tsv tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(summary, '')), 'B') ||
    setweight(to_tsvector('english', left(coalesce(search_text, ''), 800000)), 'C')
  ) STORED,
  CONSTRAINT ewrite_publications_status_check
    CHECK (status IN ('draft', 'published', 'archived')),
  CONSTRAINT ewrite_publications_visibility_check
    CHECK (visibility IN ('public', 'authenticated', 'production')),
  UNIQUE (collection_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_ewrite_publications_search
  ON ewrite_publications USING GIN (search_tsv);
CREATE INDEX IF NOT EXISTS idx_ewrite_publications_collection
  ON ewrite_publications(collection_id, status);
CREATE INDEX IF NOT EXISTS idx_ewrite_publications_location_status
  ON ewrite_publications(location_id, status);

CREATE TABLE IF NOT EXISTS ewrite_revisions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  publication_id UUID NOT NULL REFERENCES ewrite_publications(id) ON DELETE CASCADE,
  revision_number INT NOT NULL,
  source_markdown TEXT NOT NULL,
  base_revision_id UUID REFERENCES ewrite_revisions(id) ON DELETE SET NULL,
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (publication_id, revision_number)
);

-- publications.current_revision_id -> ewrite_revisions(id): added after both
-- tables exist. ALTER TABLE ... ADD CONSTRAINT has no IF NOT EXISTS, so the
-- idempotency requirement (migrations run as one Exec, re-runnable on a
-- database where they already applied) forces the pg_constraint guard.
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'ewrite_publications_current_revision_fk'
  ) THEN
    ALTER TABLE ewrite_publications
      ADD CONSTRAINT ewrite_publications_current_revision_fk
      FOREIGN KEY (current_revision_id) REFERENCES ewrite_revisions(id)
      ON DELETE SET NULL;
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS ewrite_sections (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  publication_id UUID NOT NULL REFERENCES ewrite_publications(id) ON DELETE CASCADE,
  parent_section_id UUID REFERENCES ewrite_sections(id) ON DELETE CASCADE,
  heading_level INT NOT NULL CHECK (heading_level BETWEEN 1 AND 6),
  title TEXT NOT NULL,
  anchor TEXT NOT NULL,
  anchor_explicit BOOLEAN NOT NULL DEFAULT FALSE,
  sort_order INT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (publication_id, anchor)
);

CREATE INDEX IF NOT EXISTS idx_ewrite_sections_publication
  ON ewrite_sections(publication_id, sort_order);

CREATE TABLE IF NOT EXISTS ewrite_anchor_aliases (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  publication_id UUID NOT NULL REFERENCES ewrite_publications(id) ON DELETE CASCADE,
  alias_anchor TEXT NOT NULL,
  section_id UUID NOT NULL REFERENCES ewrite_sections(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (publication_id, alias_anchor)
);

CREATE TABLE IF NOT EXISTS ewrite_editors (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  publication_id UUID NOT NULL REFERENCES ewrite_publications(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  grant_kind TEXT NOT NULL,
  granted_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT ewrite_editors_kind_check
    CHECK (grant_kind IN ('edit', 'publish', 'manage_editors')),
  UNIQUE (publication_id, user_id, grant_kind)
);

CREATE TABLE IF NOT EXISTS ewrite_object_links (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  object_type TEXT NOT NULL DEFAULT 'equipment_item',
  equipment_item_id UUID REFERENCES equipment_items(id) ON DELETE CASCADE,
  publication_id UUID NOT NULL REFERENCES ewrite_publications(id) ON DELETE CASCADE,
  section_id UUID REFERENCES ewrite_sections(id) ON DELETE SET NULL,
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT ewrite_object_links_type_check
    CHECK (object_type IN ('equipment_item')),
  CONSTRAINT ewrite_object_links_fk_shape
    CHECK (object_type <> 'equipment_item' OR equipment_item_id IS NOT NULL),
  UNIQUE (equipment_item_id, object_type)
);
