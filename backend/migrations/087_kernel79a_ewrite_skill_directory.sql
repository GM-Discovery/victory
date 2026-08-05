-- Kernel 79A: reusable eWrite directory abstraction, first instance the
-- Skill Directory (kernel spec 5.1-5.2).
--
-- A directory is compact navigational metadata for a dense rules domain --
-- it does not copy the canonical rule text, only points at it (spec 5.4).
-- ewrite_directories is scoped to a Ruleset collection (parallel to how
-- ewrite_object_links satellites a Publication); directory_type is a CHECK
-- enum so future directories (actions, reactions, stances, health systems,
-- equipment, conditions, oracles -- spec 5.5) add a new arm rather than a
-- new table.
--
-- ewrite_directory_entries.target_publication_id/target_section_id are
-- separate nullable FKs (not the ewrite_object_links table) because a
-- directory entry's target is compact-index metadata, not an object
-- binding: it has no owning Victory row of its own to CASCADE against, and
-- an entry must keep existing (unlinked) when no target has been curated
-- yet. target_section_id SET NULL on section delete degrades a curated
-- entry to a publication-level link rather than breaking it (spec 6.4:
-- "removed target" must degrade safely, not error).

CREATE TABLE IF NOT EXISTS ewrite_directories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  collection_id UUID NOT NULL REFERENCES ewrite_collections(id) ON DELETE CASCADE,
  directory_type TEXT NOT NULL,
  title TEXT NOT NULL,
  slug TEXT NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT ewrite_directories_type_check
    CHECK (directory_type IN ('skill')),
  UNIQUE (collection_id, slug)
);

CREATE TABLE IF NOT EXISTS ewrite_directory_entries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  directory_id UUID NOT NULL REFERENCES ewrite_directories(id) ON DELETE CASCADE,
  external_ref TEXT NOT NULL,
  canonical_name TEXT NOT NULL,
  aliases TEXT[] NOT NULL DEFAULT '{}',
  compact_summary TEXT NOT NULL DEFAULT '',
  category TEXT NOT NULL DEFAULT '',
  sort_key INT NOT NULL DEFAULT 0,
  target_publication_id UUID REFERENCES ewrite_publications(id) ON DELETE SET NULL,
  target_section_id UUID REFERENCES ewrite_sections(id) ON DELETE SET NULL,
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (directory_id, external_ref)
);

CREATE INDEX IF NOT EXISTS idx_ewrite_directory_entries_directory
  ON ewrite_directory_entries(directory_id, sort_key);

CREATE INDEX IF NOT EXISTS idx_ewrite_directory_entries_target_publication
  ON ewrite_directory_entries(target_publication_id);
