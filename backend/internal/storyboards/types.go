// Package storyboards implements Kernel 80: a reusable, server-authoritative,
// shareable grid board -- ordered columns, ordered rows grouped into bands,
// cells at row x column holding zero or more ordered cards.
package storyboards

import "time"

// Viewer tiers, in ascending capability order. These are not stored
// anywhere -- resolveViewerTier (authority.go) derives one per request from
// board ownership, operator status, and storyboard_grants. "owner" and
// "operator" are not location_role values; every other tier name is a
// direct location_role enum value (see migration 090's comment on why
// granted_role reuses that enum rather than inventing new names).
const (
	TierNone     = ""
	TierAudience = "audience"
	TierCast     = "cast"
	TierCrew     = "crew"
	TierDirector = "director"
	TierProducer = "producer"
	TierOwner    = "owner"
	TierOperator = "operator"
)

// Board modes (Kernel 82). Mode/TemplateVersion are recorded once at
// creation from the instantiated template and never dynamically re-linked
// to it -- see timeline_template.go.
const (
	ModeBlank    = "blank"
	ModeTimeline = "timeline"
)

// Column roles (Kernel 82). Every Blank-mode column is ColumnRoleOrdinary;
// Beginning/Ending are Timeline-mode structural boundaries, not labels --
// see columns.go's boundary-protection logic.
const (
	ColumnRoleBeginning = "beginning"
	ColumnRoleOrdinary  = "ordinary"
	ColumnRoleEnding    = "ending"
)

// Reference Panel field types (Kernel 82). See reference_panel.go.
const (
	FieldTypeShortText  = "short_text"
	FieldTypeLongText   = "long_text"
	FieldTypeList       = "list"
	FieldTypePairedList = "paired_list"
)

// Reference item sides (Kernel 82). ItemSideSingle is used by a plain
// `list` field; ItemSideA/ItemSideB select a `paired_list` field's two
// independently-ordered sublists.
const (
	ItemSideSingle = "single"
	ItemSideA      = "a"
	ItemSideB      = "b"
)

// Storyboard is one board. OwnerUserID is checked separately from
// storyboard_grants -- see authority.go and migration 090's comment for why
// ownership is never a sentinel grant row. Mode/TemplateVersion (Kernel
// 82) record which built-in template, if any, this board was instantiated
// from -- a one-time snapshot fact, not a live link (spec 4.3).
type Storyboard struct {
	ID              string     `json:"id"`
	OwnerUserID     string     `json:"owner_user_id"`
	OwnerHandle     string     `json:"owner_handle,omitempty"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Mode            string     `json:"mode"`
	TemplateVersion *int       `json:"template_version,omitempty"`
	ArchivedAt      *time.Time `json:"archived_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// StoryboardGrant is one explicit per-user access grant. GrantedRole is one
// of the five location_role values -- never "owner".
type StoryboardGrant struct {
	ID           string    `json:"id"`
	StoryboardID string    `json:"storyboard_id"`
	UserID       string    `json:"user_id"`
	UserHandle   string    `json:"user_handle,omitempty"`
	GrantedRole  string    `json:"granted_role"`
	GrantedBy    string    `json:"granted_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// StoryboardColumn is one ordered vertical division of the board. Slug is
// a deterministic, board-unique serialized key derived from Title at
// creation time (Kernel 81A) -- stable across renames/reorders/reloads;
// see slugs.go. ColumnRole (Kernel 82) is ColumnRoleOrdinary for every
// Blank-mode column; Timeline mode's Beginning/Ending columns carry the
// other two roles, which are structural (protected from reorder/removal
// in columns.go), not mere labels.
type StoryboardColumn struct {
	ID           string    `json:"id"`
	StoryboardID string    `json:"storyboard_id"`
	Title        string    `json:"title"`
	Slug         string    `json:"slug"`
	ColumnRole   string    `json:"column_role"`
	SortOrder    int       `json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// StoryboardBand groups one or more adjacent rows across all columns.
// Adjacency/non-overlap is an application-level invariant derived from
// StoryboardRow.SortOrder, not a DB constraint -- see migration 090. Slug
// is a deterministic, board-unique serialized key derived from Label at
// creation time (Kernel 81A) -- see slugs.go.
type StoryboardBand struct {
	ID           string    `json:"id"`
	StoryboardID string    `json:"storyboard_id"`
	Label        string    `json:"label"`
	Slug         string    `json:"slug"`
	Description  string    `json:"description"`
	SortOrder    int       `json:"sort_order"`
	IsCollapsed  bool      `json:"is_collapsed"`
	IsLocked     bool      `json:"is_locked"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// StoryboardRow is one ordered horizontal division, belonging to exactly
// one band. SortOrderInBand orders a row only within its own band -- full
// visual row order is (band.SortOrder, row.SortOrderInBand); see migration
// 090's comment for why this two-level scheme replaces a single global
// order. Slug is a deterministic, *band*-unique (not board-unique)
// serialized key derived from Label at creation time (Kernel 81A) -- see
// slugs.go.
type StoryboardRow struct {
	ID              string    `json:"id"`
	StoryboardID    string    `json:"storyboard_id"`
	BandID          string    `json:"band_id"`
	Label           string    `json:"label"`
	Slug            string    `json:"slug"`
	Description     string    `json:"description"`
	SortOrderInBand int       `json:"sort_order_in_band"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ReferenceField is one Reference Panel field (Kernel 82, spec 3). Slug
// follows the same board-scoped, assigned-once-at-creation contract as
// column/band/row slugs (Kernel 81A) -- see slugs.go. TextContent is used
// only by short_text/long_text fields; SublabelA/SublabelB only by
// paired_list. A list/paired_list field's actual entries live in
// ReferenceItem rows, loaded separately and grouped by FieldID.
type ReferenceField struct {
	ID           string    `json:"id"`
	StoryboardID string    `json:"storyboard_id"`
	Slug         string    `json:"slug"`
	Label        string    `json:"label"`
	FieldType    string    `json:"field_type"`
	SortOrder    int       `json:"sort_order"`
	TextContent  string    `json:"text_content,omitempty"`
	SublabelA    string    `json:"sublabel_a,omitempty"`
	SublabelB    string    `json:"sublabel_b,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ReferenceItem is one entry in a `list` field (Side == ItemSideSingle)
// or one entry in one side of a `paired_list` field (Side ==
// ItemSideA/ItemSideB).
type ReferenceItem struct {
	ID        string    `json:"id"`
	FieldID   string    `json:"field_id"`
	Side      string    `json:"side"`
	SortOrder int       `json:"sort_order"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// StoryboardCard is one content object placed inside one cell
// (RowID, ColumnID), with a stable explicit order within that cell.
type StoryboardCard struct {
	ID                 string    `json:"id"`
	StoryboardID       string    `json:"storyboard_id"`
	RowID              string    `json:"row_id"`
	ColumnID           string    `json:"column_id"`
	SortOrderInCell    int       `json:"sort_order_in_cell"`
	Title              string    `json:"title"`
	FrontText          string    `json:"front_text"`
	BackText           string    `json:"back_text"`
	Category           string    `json:"category"`
	ColorToken         string    `json:"color_token"`
	ImageAssetID       string    `json:"image_asset_id,omitempty"`
	HiddenFromAudience bool      `json:"hidden_from_audience"`
	IsLocked           bool      `json:"is_locked"`
	AuthorUserID       string    `json:"author_user_id,omitempty"`
	AuthorHandle       string    `json:"author_handle,omitempty"`
	Version            int       `json:"version"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
