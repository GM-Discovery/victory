package playerrelationships

import "time"

// Relationship is one private, directional relationship record
// (Kernel 62 §5.1). Raw account UUIDs are deliberately never serialized --
// the subject is exposed only as their opaque Player Workbook / profile ID
// (Kernel 62 §10).
type Relationship struct {
	ID                     string     `json:"id"`
	ObserverUserID         string     `json:"-"`
	SubjectUserID          string     `json:"-"`
	PrivateNickname        string     `json:"private_nickname"`
	RelationshipState      string     `json:"relationship_state"`
	TrustLevel             string     `json:"trust_level"`
	ClosenessLevel         string     `json:"closeness_level"`
	ReliabilityLevel       string     `json:"reliability_level"`
	CommunicationEaseLevel string     `json:"communication_ease_level"`
	ArchivedAt             *time.Time `json:"archived_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// Category is one private checkbox category on a relationship
// (Kernel 62 §5.2). Labels only -- never authority.
type Category struct {
	CategoryKey string `json:"category_key"`
	CustomLabel string `json:"custom_label,omitempty"`
}

// RelationshipFact is the current effective value for one workbook field
// (Kernel 62 §5.3).
type RelationshipFact struct {
	FieldKey      string    `json:"field_key"`
	ValueJSON     any       `json:"value"`
	DisplayValue  string    `json:"display_value"`
	SourceEventID string    `json:"source_event_id,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// RelationshipEvent is one observer-private history row (Kernel 62 §5.4).
type RelationshipEvent struct {
	ID           string         `json:"id"`
	EventType    string         `json:"event_type"`
	PageKey      string         `json:"page_key"`
	Payload      map[string]any `json:"payload,omitempty"`
	HumanSummary string         `json:"human_summary"`
	CreatedAt    time.Time      `json:"created_at"`
}

// JournalEntry is one private dated note about this person (Kernel 62 §5.5).
type JournalEntry struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Body         string    `json:"body"`
	EntryDate    string    `json:"entry_date,omitempty"`
	NoteCategory string    `json:"note_category"`
	Tags         []string  `json:"tags"`
	ProductionID string    `json:"production_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// FollowUp is one stored private intention. No reminder, notification, or
// scheduler surface exists for these rows (Kernel 62 §4.5, §5.6).
type FollowUp struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Notes       string     `json:"notes"`
	TargetDate  string     `json:"target_date,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// SharedProduction is one production Victory can verify both users belong to
// (Kernel 62 §5.7, §13.1).
type SharedProduction struct {
	ProductionName string    `json:"production_name"`
	ObserverRoles  []string  `json:"observer_roles"`
	SubjectRoles   []string  `json:"subject_roles"`
	Since          time.Time `json:"since"`
}

// SharedContext is the read-only derived projection of reliably-known shared
// data. It never infers trust, closeness, or meaning (Kernel 62 §5.7).
type SharedContext struct {
	Productions []SharedProduction `json:"productions"`
}

// SubjectHeader is the small compiled public-in-installation presentation of
// the subject shown alongside private notes. It carries only what the
// subject's own Trailer Face already exposes.
type SubjectHeader struct {
	SubjectProfileID string `json:"subject_profile_id"`
	StageName        string `json:"stage_name"`
	PortraitURL      string `json:"portrait_url,omitempty"`
}

// ListItem is one row of the My People list (Kernel 62 §8.2).
type ListItem struct {
	Relationship
	Subject               SubjectHeader `json:"subject"`
	Categories            []Category    `json:"categories"`
	Archived              bool          `json:"archived"`
	LastJournalAt         *time.Time    `json:"last_journal_at,omitempty"`
	OpenFollowupCount     int           `json:"open_followup_count"`
	SharedProductionCount int           `json:"shared_production_count"`
}

// Detail is the full relationship view for the owning observer only
// (Kernel 62 §8.3).
type Detail struct {
	ListItem
	Facts         map[string]RelationshipFact `json:"facts"`
	Events        []RelationshipEvent         `json:"events"`
	SharedContext SharedContext               `json:"shared_context"`
}
