package showruns

import "time"

// ShowRun is one row of show_runs -- the bounded run/cohort/campaign/season
// container this kernel implements (the dictionary's pre-existing
// "Production Run" concept; "Show Run" is its product-facing name). It is
// deliberately not a "Showing" (Kernel 22): a Show Run has no notion of a
// live, in-progress moment, only planning/active/paused/completed/archived
// status.
type ShowRun struct {
	ID                      string     `json:"id"`
	LocationID              string     `json:"location_id"`
	ProductionID            string     `json:"production_id"`
	Title                   string     `json:"title"`
	Slug                    string     `json:"slug"`
	Description             string     `json:"description,omitempty"`
	ShowFormat              string     `json:"show_format"`
	CustomShowFormat        string     `json:"custom_show_format,omitempty"`
	CohortName              string     `json:"cohort_name,omitempty"`
	Status                  string     `json:"status"`
	AudienceSelfJoinEnabled bool       `json:"audience_self_join_enabled"`
	CreatedByUserID         string     `json:"created_by_user_id"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
	ArchivedAt              *time.Time `json:"archived_at,omitempty"`
}

// RosterMember is one row of show_run_roster_members -- either the current
// active row for a user on a run, or (with RemovedAt set) a historic one. It
// never carries Trailer Face content; roster cards are always projected live
// from the member's current Trailer Face at read time (mirrors Kernel 65's
// third_place_headshots discipline exactly).
type RosterMember struct {
	ID              string     `json:"id"`
	ShowRunID       string     `json:"show_run_id"`
	UserID          string     `json:"user_id"`
	Role            string     `json:"role"`
	CustomRoleLabel string     `json:"custom_role_label,omitempty"`
	ProgramVisible  bool       `json:"program_visible"`
	AddedByUserID   string     `json:"added_by_user_id"`
	AddedAt         time.Time  `json:"added_at"`
	RemovedAt       *time.Time `json:"removed_at,omitempty"`
}

// AudienceBlock is one row of show_run_audience_blocks -- run-scoped only,
// never a site-wide moderation record.
type AudienceBlock struct {
	ID              string     `json:"id"`
	ShowRunID       string     `json:"show_run_id"`
	UserID          string     `json:"user_id"`
	BlockedByUserID string     `json:"blocked_by_user_id"`
	Reason          string     `json:"reason,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	LiftedAt        *time.Time `json:"lifted_at,omitempty"`
}

// HeadlineFact is one small Face-visible fact shown on a roster card, the
// same shape thirdplace.HeadlineFact uses.
type HeadlineFact struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// RosterMemberProjection is the internal-roster public shape: everything a
// Producer/Director/Operator managing the run is allowed to see. Stage
// name/portrait/headline facts are always re-derived live from the member's
// current Trailer Face -- never stored on the roster row.
type RosterMemberProjection struct {
	MemberID       string         `json:"member_id"`
	ProfileID      string         `json:"profile_id"`
	Role           string         `json:"role"`
	RoleLabel      string         `json:"role_label"`
	ProgramVisible bool           `json:"program_visible"`
	AddedAt        time.Time      `json:"added_at"`
	StageName      string         `json:"stage_name"`
	PortraitURL    string         `json:"portrait_url,omitempty"`
	HeadlineFacts  []HeadlineFact `json:"headline_facts"`
	TrailerURL     string         `json:"trailer_url"`
	IsYou          bool           `json:"is_you"`
}

// AudienceProgramEntry is the curated Audience-facing shape -- deliberately
// narrower than RosterMemberProjection. It omits added_by_user_id and any
// other internal-only roster metadata (Kernel 66 "Audience does not see the
// full internal roster by default").
type AudienceProgramEntry struct {
	Role          string         `json:"role"`
	RoleLabel     string         `json:"role_label"`
	StageName     string         `json:"stage_name"`
	PortraitURL   string         `json:"portrait_url,omitempty"`
	HeadlineFacts []HeadlineFact `json:"headline_facts"`
	TrailerURL    string         `json:"trailer_url"`
	IsYou         bool           `json:"is_you"`
}

// ShowRunSummary is the list-view shape returned by ListShowRunsVisibleToUser.
type ShowRunSummary struct {
	ID                      string    `json:"id"`
	Title                   string    `json:"title"`
	Slug                    string    `json:"slug"`
	Description             string    `json:"description,omitempty"`
	ShowFormat              string    `json:"show_format"`
	CustomShowFormat        string    `json:"custom_show_format,omitempty"`
	CohortName              string    `json:"cohort_name,omitempty"`
	Status                  string    `json:"status"`
	AudienceSelfJoinEnabled bool      `json:"audience_self_join_enabled"`
	CanManage               bool      `json:"can_manage"`
	CreatedAt               time.Time `json:"created_at"`
}
