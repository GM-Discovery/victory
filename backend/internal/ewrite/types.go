package ewrite

import "time"

// Collection is one node of the organizational tree above publications:
// ruleset -> series -> module. Parent-kind rules live in the store
// (validateParentKind); the visible product hierarchy is fixed even though
// the storage is one typed tree.
type Collection struct {
	ID         string    `json:"id"`
	LocationID string    `json:"location_id"`
	ParentID   string    `json:"parent_id,omitempty"`
	Kind       string    `json:"kind"`
	Title      string    `json:"title"`
	Slug       string    `json:"slug"`
	Summary    string    `json:"summary"`
	SortOrder  int       `json:"sort_order"`
	Visibility string    `json:"visibility"`
	CreatedBy  string    `json:"created_by,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Publication is one eWriting. SourceMarkdown is the single source of
// truth; RenderedHTML is the sanitized cache regenerated in the same
// transaction as every source write.
type Publication struct {
	ID                string     `json:"id"`
	CollectionID      string     `json:"collection_id"`
	LocationID        string     `json:"location_id"`
	Title             string     `json:"title"`
	Slug              string     `json:"slug"`
	Summary           string     `json:"summary"`
	SourceMarkdown    string     `json:"source_markdown,omitempty"`
	RenderedHTML      string     `json:"rendered_html,omitempty"`
	WordCount         int        `json:"word_count"`
	Status            string     `json:"status"`
	Visibility        string     `json:"visibility"`
	CurrentRevisionID string     `json:"current_revision_id,omitempty"`
	CreatedBy         string     `json:"created_by,omitempty"`
	UpdatedBy         string     `json:"updated_by,omitempty"`
	PublishedAt       *time.Time `json:"published_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// Revision is one append-forward save of a publication's source.
type Revision struct {
	ID             string    `json:"id"`
	PublicationID  string    `json:"publication_id"`
	RevisionNumber int       `json:"revision_number"`
	SourceMarkdown string    `json:"source_markdown,omitempty"`
	BaseRevisionID string    `json:"base_revision_id,omitempty"`
	CreatedBy      string    `json:"created_by,omitempty"`
	CreatedByName  string    `json:"created_by_name,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	ByteSize       int       `json:"byte_size"`
}

// Section is one stable-identity heading row derived from the source.
type Section struct {
	ID              string `json:"id"`
	PublicationID   string `json:"publication_id"`
	ParentSectionID string `json:"parent_section_id,omitempty"`
	HeadingLevel    int    `json:"heading_level"`
	Title           string `json:"title"`
	Anchor          string `json:"anchor"`
	AnchorExplicit  bool   `json:"anchor_explicit"`
	SortOrder       int    `json:"sort_order"`
}

// EditorGrant is a named per-publication authority grant.
type EditorGrant struct {
	ID            string    `json:"id"`
	PublicationID string    `json:"publication_id"`
	UserID        string    `json:"user_id"`
	UserHandle    string    `json:"user_handle,omitempty"`
	GrantKind     string    `json:"grant_kind"`
	GrantedBy     string    `json:"granted_by,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// ObjectLink binds an existing Victory object to an exact rule section.
// Exactly one of the *ID fields is set, matching ObjectType (kernel Goal C,
// spec 8.4 -- one reusable link shape for every linkable object).
type ObjectLink struct {
	ID                 string `json:"id"`
	ObjectType         string `json:"object_type"`
	EquipmentItemID    string `json:"equipment_item_id,omitempty"`
	CueID              string `json:"cue_id,omitempty"`
	IndexCardElementID string `json:"index_card_element_id,omitempty"`
	SceneElementID     string `json:"scene_element_id,omitempty"`
	DialogueTopicID    string `json:"dialogue_topic_id,omitempty"`
	PublicationID      string `json:"publication_id"`
	SectionID          string `json:"section_id,omitempty"`
	SectionAnchor      string `json:"section_anchor,omitempty"`
	SectionTitle       string `json:"section_title,omitempty"`
}

// Directory is compact navigational metadata for a dense rules domain --
// it never copies the canonical rule text (spec 5.1, 5.4). Scoped to a
// Ruleset collection; DirectoryType is a CHECK enum so future directories
// (actions, health systems, equipment, ...) add an arm, not a new shape.
type Directory struct {
	ID            string    `json:"id"`
	CollectionID  string    `json:"collection_id"`
	DirectoryType string    `json:"directory_type"`
	Title         string    `json:"title"`
	Slug          string    `json:"slug"`
	Summary       string    `json:"summary"`
	EntryCount    int       `json:"entry_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// DirectoryEntry is one compact index row pointing at (at most) one exact
// eWrite section. LinkStatus is computed at read time, never stored:
//   - "unlinked": no target has been curated yet.
//   - "hidden": a target exists but the requesting reader cannot read it
//     (spec 5.3 -- an entry must not leak a hidden target's title).
//   - "linked": target resolved and readable; Target* fields are filled.
type DirectoryEntry struct {
	ID                     string   `json:"id"`
	DirectoryID            string   `json:"directory_id"`
	ExternalRef            string   `json:"external_ref"`
	CanonicalName          string   `json:"canonical_name"`
	Aliases                []string `json:"aliases,omitempty"`
	CompactSummary         string   `json:"compact_summary,omitempty"`
	Category               string   `json:"category,omitempty"`
	SortKey                int      `json:"sort_key"`
	LinkStatus             string   `json:"link_status"`
	TargetPublicationID    string   `json:"target_publication_id,omitempty"`
	TargetPublicationTitle string   `json:"target_publication_title,omitempty"`
	TargetSectionID        string   `json:"target_section_id,omitempty"`
	TargetSectionAnchor    string   `json:"target_section_anchor,omitempty"`
	TargetSectionTitle     string   `json:"target_section_title,omitempty"`
}

// ImportReport is returned by import/preview so the author sees exactly
// what the recognizer did before (or after) committing (kernel spec 7.4).
type ImportReport struct {
	SourceFilename   string    `json:"source_filename,omitempty"`
	ByteSize         int       `json:"byte_size"`
	WordCount        int       `json:"word_count"`
	HeadingCount     int       `json:"heading_count"`
	SectionCount     int       `json:"section_count"`
	SubsectionCount  int       `json:"subsection_count"`
	SpacerCount      int       `json:"spacer_count"`
	ExplicitAnchors  int       `json:"explicit_anchors"`
	GeneratedAnchors int       `json:"generated_anchors"`
	InternalLinks    int       `json:"internal_links"`
	ExternalLinks    int       `json:"external_links"`
	ImageRefs        int       `json:"image_refs"`
	RawHTMLCount     int       `json:"raw_html_count"`
	Warnings         []string  `json:"warnings"`
	Outline          []Heading `json:"outline"`
}

func buildImportReport(source string, res *RenderResult) ImportReport {
	rep := ImportReport{
		ByteSize:      len(source),
		WordCount:     res.WordCount,
		HeadingCount:  len(res.Outline) + res.SpacerCount,
		SpacerCount:   res.SpacerCount,
		InternalLinks: len(res.InternalLinks),
		ExternalLinks: len(res.ExternalLinks),
		ImageRefs:     len(res.ImageRefs),
		RawHTMLCount:  res.RawHTMLCount,
		Warnings:      res.Warnings,
		Outline:       res.Outline,
	}
	if rep.Warnings == nil {
		rep.Warnings = []string{}
	}
	for _, h := range res.Outline {
		if h.Explicit {
			rep.ExplicitAnchors++
		} else {
			rep.GeneratedAnchors++
		}
		// Product language: level 2 headings are Sections, level 3+ are
		// Subsections; the level-1 publication title is neither.
		switch {
		case h.Level == 2:
			rep.SectionCount++
		case h.Level >= 3:
			rep.SubsectionCount++
		}
	}
	return rep
}
