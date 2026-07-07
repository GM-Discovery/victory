package playerprofile

import "time"

// Reserved field keys can never be Face-eligible or catalogue-driven --
// they belong to the account layer, not the workbook (Kernel 61 §3.2, §6.6).
const (
	ReservedFieldStageName   = "stage_name"
	ReservedFieldHandle      = "handle"
	ReservedFieldEmail       = "email"
	ReservedFieldAccountUUID = "account_uuid"
)

const (
	VisibilityInferred = "inferred"
	VisibilityShown    = "shown"
	VisibilityHidden   = "hidden"

	PriorityInferred = "inferred"
	PriorityManual   = "manual"
)

// Workbook is the Player Workbook root row (Kernel 61 §6.1).
type Workbook struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id,omitempty"`
	CatalogueKey      string    `json:"catalogue_key"`
	CatalogueVersion  string    `json:"catalogue_version"`
	ProjectionVersion string    `json:"projection_version"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ProfileEvent is one typed, timestamped profile-history row (Kernel 61 §6.3).
type ProfileEvent struct {
	ID               string         `json:"id"`
	WorkbookID       string         `json:"workbook_id,omitempty"`
	EventType        string         `json:"event_type"`
	PageKey          string         `json:"page_key"`
	CatalogueVersion string         `json:"catalogue_version,omitempty"`
	Payload          map[string]any `json:"payload,omitempty"`
	HumanSummary     string         `json:"human_summary"`
	CreatedByUserID  string         `json:"created_by_user_id,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
}

// ProfileFact is the current effective value for one field_key (Kernel 61 §6.4).
type ProfileFact struct {
	FieldKey         string    `json:"field_key"`
	ValueJSON        any       `json:"value"`
	DisplayValue     string    `json:"display_value"`
	SourceEventID    string    `json:"source_event_id,omitempty"`
	SourcePageKey    string    `json:"source_page_key,omitempty"`
	CatalogueVersion string    `json:"catalogue_version,omitempty"`
	EffectiveAt      time.Time `json:"effective_at"`
}

// StageNameEntry is one interval in the append-only stage-name ledger
// (Kernel 61 §6.5).
type StageNameEntry struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id,omitempty"`
	StageName       string     `json:"stage_name"`
	NormalizedName  string     `json:"normalized_stage_name,omitempty"`
	StartedAt       time.Time  `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	ChangedByUserID string     `json:"changed_by_user_id,omitempty"`
	Source          string     `json:"source,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// FaceOverride is owner Face curation for one eligible fact (Kernel 61 §6.6).
type FaceOverride struct {
	FieldKey       string `json:"field_key"`
	VisibilityMode string `json:"visibility_mode"`
	PriorityMode   string `json:"priority_mode"`
	PriorityScore  int    `json:"priority_score"`
}

// ProjectedField is one fact as it appears in a projection (owner workbook
// preview or social Face), after overrides and region assignment.
type ProjectedField struct {
	FieldKey           string `json:"field_key"`
	Label              string `json:"label"`
	DisplayValue       string `json:"display_value"`
	Region             string `json:"region"`
	FaceVisibilityMode string `json:"face_visibility_mode"`
	FaceVisible        bool   `json:"face_visible"`
	PriorityMode       string `json:"priority_mode"`
	PriorityScore      int    `json:"priority_score"`
}

// TrailerFace is the compiled social projection another authenticated user
// receives (Kernel 61 §6.8, §8.2). It never includes email, handle, account
// UUID, source pages, event history, or the stage-name ledger.
type TrailerFace struct {
	ProjectionVersion string                      `json:"projection_version"`
	StageName         string                      `json:"stage_name"`
	Regions           map[string][]ProjectedField `json:"regions"`
}
