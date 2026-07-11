package thirdplace

import "time"

// Headshot is one row of third_place_headshots -- either the caller's
// current active row, or one historic placement/removal record. It never
// carries any Trailer Face content; that is always projected live
// (Kernel 65 §3.5, §5.2).
type Headshot struct {
	ID        string
	UserID    string
	Status    string
	PlacedAt  time.Time
	RemovedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// HeadlineFact is one small, Face-visible fact shown on a Headshot card
// (Kernel 65 §5.4) -- never the full Trailer Face.
type HeadlineFact struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// HeadshotProjection is the public shape returned to any authenticated
// viewer (Kernel 65 §5.3). It deliberately excludes the owner's raw account
// UUID, email, handle, Player Workbook source answers, Trailer history,
// stage-name ledger, and any private relationship data belonging to anyone
// but the current viewer (Kernel 65 §9.3, §9.4).
type HeadshotProjection struct {
	HeadshotID                 string         `json:"headshot_id"`
	ProfileID                  string         `json:"profile_id"`
	PlacedAt                   time.Time      `json:"placed_at"`
	StageName                  string         `json:"stage_name"`
	PortraitURL                string         `json:"portrait_url,omitempty"`
	HeadlineFacts              []HeadlineFact `json:"headline_facts"`
	TrailerURL                 string         `json:"trailer_url"`
	RelationshipStateForViewer string         `json:"relationship_state_for_viewer"`
	CanAddToMyPeople           bool           `json:"can_add_to_my_people"`
	CanOpenMyNotes             bool           `json:"can_open_my_notes"`
	RelationshipID             string         `json:"relationship_id,omitempty"`
	IsYou                      bool           `json:"is_you"`
}

// HistoryEntry is one owner-only placement/removal ledger row (Kernel 65
// §3.5, §6.4). No Face content is ever included here -- old Trailer Face
// snapshots deliberately do not exist anywhere in this package.
type HistoryEntry struct {
	ID        string     `json:"id"`
	Status    string     `json:"status"`
	PlacedAt  time.Time  `json:"placed_at"`
	RemovedAt *time.Time `json:"removed_at,omitempty"`
}
