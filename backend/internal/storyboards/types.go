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

// Storyboard is one board. OwnerUserID is checked separately from
// storyboard_grants -- see authority.go and migration 090's comment for why
// ownership is never a sentinel grant row.
type Storyboard struct {
	ID          string     `json:"id"`
	OwnerUserID string     `json:"owner_user_id"`
	OwnerHandle string     `json:"owner_handle,omitempty"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
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

// StoryboardColumn is one ordered vertical division of the board.
type StoryboardColumn struct {
	ID           string    `json:"id"`
	StoryboardID string    `json:"storyboard_id"`
	Title        string    `json:"title"`
	SortOrder    int       `json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// StoryboardBand groups one or more adjacent rows across all columns.
// Adjacency/non-overlap is an application-level invariant derived from
// StoryboardRow.SortOrder, not a DB constraint -- see migration 090.
type StoryboardBand struct {
	ID           string    `json:"id"`
	StoryboardID string    `json:"storyboard_id"`
	Label        string    `json:"label"`
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
// order.
type StoryboardRow struct {
	ID              string    `json:"id"`
	StoryboardID    string    `json:"storyboard_id"`
	BandID          string    `json:"band_id"`
	Label           string    `json:"label"`
	Description     string    `json:"description"`
	SortOrderInBand int       `json:"sort_order_in_band"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
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
	HiddenFromAudience bool      `json:"hidden_from_audience"`
	IsLocked           bool      `json:"is_locked"`
	AuthorUserID       string    `json:"author_user_id,omitempty"`
	AuthorHandle       string    `json:"author_handle,omitempty"`
	Version            int       `json:"version"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
