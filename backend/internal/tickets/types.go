// Package tickets implements Kernel 71's two-punch Show ticket: the sole
// ordinary product path that creates real Player participation in a Show
// Run. Either side (Player or Director) may punch first; the second punch
// is one atomic transaction that marks the ticket valid and creates or
// reactivates exactly one show_run_roster_members player row -- see
// SecondPunch in tickets.go for the full sequence.
package tickets

import "time"

const (
	RequestedRolePlayer = "player"

	InitiatedBySidePlayer   = "player"
	InitiatedBySideDirector = "director"

	StatusPendingPlayer   = "pending_player"
	StatusPendingDirector = "pending_director"
	StatusValid           = "valid"
	StatusDeclined        = "declined"
	StatusWithdrawn       = "withdrawn"
)

// Ticket is one row of show_run_tickets.
type Ticket struct {
	ID                      string     `json:"id"`
	ShowRunID               string     `json:"show_run_id"`
	UserID                  string     `json:"user_id"`
	RequestedRole           string     `json:"requested_role"`
	InitiatedBySide         string     `json:"initiated_by_side"`
	PlayerPunchedAt         *time.Time `json:"player_punched_at,omitempty"`
	PlayerPunchedByUserID   string     `json:"player_punched_by_user_id,omitempty"`
	DirectorPunchedAt       *time.Time `json:"director_punched_at,omitempty"`
	DirectorPunchedByUserID string     `json:"director_punched_by_user_id,omitempty"`
	Status                  string     `json:"status"`
	Message                 string     `json:"message,omitempty"`
	RosterMembershipID      string     `json:"roster_membership_id,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
	ResolvedAt              *time.Time `json:"resolved_at,omitempty"`

	// ShowRunTitle is populated only by the list endpoints (ListMineAsPlayer,
	// ListIncomingForDirector) -- a display convenience so Audition Hall
	// can show a Player their own pending/valid tickets without a second,
	// separately-authority-gated call to a show-run detail route a
	// ticket-only Player (no location_memberships) might not pass.
	ShowRunTitle string `json:"show_run_title,omitempty"`
}
