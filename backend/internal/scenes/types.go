package scenes

import (
	"encoding/json"
	"time"
)

// Scene is one row of scenes -- a reusable, location-scoped
// authored/configured playable or viewable unit, comparable to a standing
// set on a studio lot. A Scene may be staged (placed) in zero, one, or many
// Shows under any Production at the same Victory location; it is never a
// child row of a single Show or a single Production (Kernel 70 SS1.2,
// SS3.1, correcting Kernel 69 SS1.2/SS1.3's Production-exclusive rule).
// SourceProductionID remembers where a Scene originated as provenance only
// -- it never gates reuse. Script/hyperlink systems are future work --
// SourceRef and ConfigJSON only reserve room for that, they do not
// implement it (Kernel 69 SS1.6).
type Scene struct {
	ID                 string          `json:"id"`
	LocationID         string          `json:"location_id"`
	SourceProductionID *string         `json:"source_production_id,omitempty"`
	Slug               string          `json:"slug"`
	Title              string          `json:"title"`
	ShortTitle         string          `json:"short_title,omitempty"`
	DefaultVenueID     *string         `json:"default_venue_id,omitempty"`
	AudienceTitle      string          `json:"audience_title,omitempty"`
	AudienceSummary    string          `json:"audience_summary,omitempty"`
	PlayerBrief        string          `json:"player_brief,omitempty"`
	DirectorNotes      string          `json:"director_notes,omitempty"`
	OperatorNotes      string          `json:"operator_notes,omitempty"`
	SourceRef          string          `json:"source_ref,omitempty"`
	Status             string          `json:"status"`
	ConfigJSON         json.RawMessage `json:"config_json,omitempty"`
	CreatedByUserID    *string         `json:"created_by_user_id,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	ArchivedAt         *time.Time      `json:"archived_at,omitempty"`
}

// CreateSceneInput is the caller-supplied subset of a new Scene. ProductionID
// is a separate function argument, not a field here, matching
// showruns.CreateShowRun's own convention.
type CreateSceneInput struct {
	Slug            string
	Title           string
	ShortTitle      string
	DefaultVenueID  string
	AudienceTitle   string
	AudienceSummary string
	PlayerBrief     string
	DirectorNotes   string
	OperatorNotes   string
	SourceRef       string
}

// UpdateScenePatch carries only the fields being changed. A nil pointer
// means "leave as-is"; an empty string on an optional text pointer clears
// the column to NULL, mirroring shows.UpdateShowPatch's convention.
type UpdateScenePatch struct {
	Title           *string
	ShortTitle      *string
	DefaultVenueID  *string
	AudienceTitle   *string
	AudienceSummary *string
	PlayerBrief     *string
	DirectorNotes   *string
	OperatorNotes   *string
	SourceRef       *string
	Status          *string
}

// SceneSummary is the list-view shape returned by ListScenesForLocation --
// omits backstage-only long-form text fields to keep the library listing
// light.
type SceneSummary struct {
	ID                 string    `json:"id"`
	LocationID         string    `json:"location_id"`
	SourceProductionID *string   `json:"source_production_id,omitempty"`
	Slug               string    `json:"slug"`
	Title              string    `json:"title"`
	ShortTitle         string    `json:"short_title,omitempty"`
	DefaultVenueID     *string   `json:"default_venue_id,omitempty"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
}

// ShowScenePlacement is one row of show_scene_placements -- the use of a
// reusable Scene inside one specific Show, with its own ordering, optional
// venue override, and a small set of audience/director overrides. Removing
// or archiving a placement never touches the Scene it points at (Kernel 69
// SS1.3).
type ShowScenePlacement struct {
	ID                      string          `json:"id"`
	ShowID                  string          `json:"show_id"`
	SceneID                 string          `json:"scene_id"`
	VenueID                 *string         `json:"venue_id,omitempty"`
	SortOrder               int             `json:"sort_order"`
	Status                  string          `json:"status"`
	AudienceTitleOverride   string          `json:"audience_title_override,omitempty"`
	AudienceSummaryOverride string          `json:"audience_summary_override,omitempty"`
	DirectorNotesOverride   string          `json:"director_notes_override,omitempty"`
	ConfigJSON              json.RawMessage `json:"config_json,omitempty"`
	CreatedByUserID         *string         `json:"created_by_user_id,omitempty"`
	CreatedAt               time.Time       `json:"created_at"`
	UpdatedAt               time.Time       `json:"updated_at"`
	ArchivedAt              *time.Time      `json:"archived_at,omitempty"`
}

// CreatePlacementInput is the caller-supplied subset of a new placement.
// ShowID is a separate function argument, matching Scene's own convention.
type CreatePlacementInput struct {
	SceneID                 string
	VenueID                 string
	SortOrder               int
	AudienceTitleOverride   string
	AudienceSummaryOverride string
	DirectorNotesOverride   string
}

// UpdatePlacementPatch carries only the fields being changed.
type UpdatePlacementPatch struct {
	VenueID                 *string
	SortOrder               *int
	Status                  *string
	AudienceTitleOverride   *string
	AudienceSummaryOverride *string
	DirectorNotesOverride   *string
}

// PlacementDetail is the backstage listing shape returned for a Show's
// Scenes section -- the placement plus a curated projection of its base
// Scene's own fields (never the Scene's operator_notes).
type PlacementDetail struct {
	Placement ShowScenePlacement `json:"placement"`
	Scene     ScenePlacementBase `json:"scene"`
}

// ScenePlacementBase is the subset of a Scene surfaced alongside a
// placement -- enough for backstage staging decisions, still excluding
// operator_notes (Operator-only, never round-tripped to a placement view).
type ScenePlacementBase struct {
	ID              string  `json:"id"`
	Slug            string  `json:"slug"`
	Title           string  `json:"title"`
	ShortTitle      string  `json:"short_title,omitempty"`
	DefaultVenueID  *string `json:"default_venue_id,omitempty"`
	AudienceTitle   string  `json:"audience_title,omitempty"`
	AudienceSummary string  `json:"audience_summary,omitempty"`
	PlayerBrief     string  `json:"player_brief,omitempty"`
	DirectorNotes   string  `json:"director_notes,omitempty"`
	SourceRef       string  `json:"source_ref,omitempty"`
	Status          string  `json:"status"`
}

// AudienceScenePlacement is the curated Audience Program shape for a Show
// Scene Placement -- only ready, non-archived placements are ever listed
// here, and only audience-safe fields are included (Kernel 69 SS2.6). No
// director_notes, operator_notes, source_ref, config_json, or
// created_by_user_id ever appears on this type.
type AudienceScenePlacement struct {
	Title           string `json:"title"`
	AudienceSummary string `json:"audience_summary,omitempty"`
	SortOrder       int    `json:"sort_order"`
}
