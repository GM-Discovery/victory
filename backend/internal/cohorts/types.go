// Package cohorts is Kernel 85's Show-scoped grouping of participants into
// independently Scene-progressing cohorts. A cohort belongs to exactly one
// Show (kernel-85 S1.4) -- never a global account grouping, My People
// category, or Victory permission role. Ungrouped is never a stored row
// (S3.1): it is computed as every active Player-role roster member of the
// Show's parent Show Run minus whoever has a row in
// show_cohort_assignments.
package cohorts

import "time"

// Cohort is one row of show_cohorts.
type Cohort struct {
	ID                          string     `json:"id"`
	ShowID                      string     `json:"show_id"`
	SerialNumber                int        `json:"serial_number"`
	Slug                        string     `json:"slug"`
	Name                        string     `json:"name"`
	CurrentShowScenePlacementID *string    `json:"current_show_scene_placement_id,omitempty"`
	CreatedByUserID             string     `json:"created_by_user_id"`
	CreatedAt                   time.Time  `json:"created_at"`
	UpdatedAt                   time.Time  `json:"updated_at"`
	ArchivedAt                  *time.Time `json:"archived_at,omitempty"`
}

// Participant is the curated per-person shape used in cohort rosters --
// display-safe (stage name, never handle/raw UUID, matching the rest of
// this codebase's people-display convention -- see thirdplace.
// HeadshotProjection and playerrelationships.ListItem).
type Participant struct {
	UserID          string `json:"user_id"`
	DisplayName     string `json:"display_name,omitempty"`
	CharacterCardID string `json:"character_card_id,omitempty"`
	CharacterName   string `json:"character_name,omitempty"`
}

// CohortWithMembers is one cohort plus its currently assigned participants,
// the shape the Cohort management UI and the Game Status cohort selector
// both consume.
type CohortWithMembers struct {
	Cohort
	Members []Participant `json:"members"`
}

// Roster is the full Show grouping picture: every cohort with its members,
// plus whoever remains Ungrouped.
type Roster struct {
	Cohorts   []CohortWithMembers `json:"cohorts"`
	Ungrouped []Participant       `json:"ungrouped"`
}
