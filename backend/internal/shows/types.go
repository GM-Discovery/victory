package shows

import (
	"encoding/json"
	"time"
)

// Show is one row of shows -- the concrete playable/viewable instance of a
// Show Run (Kernel 67). It is deliberately not a Session (technical live
// runtime window, `sessions` table) and not a Showing (Kernel 22's live 1:1
// audience-visibility wrapper, backend/internal/showings) -- both remain
// untouched. A Show has its own Audience Program content but inherits its
// parent Show Run's roster and authority wholesale; it has no roster table
// of its own.
//
// CurrentShowScenePlacementID is the persistent current-Scene pointer
// (Kernel 70 SS4.1) -- the Show, not the Session, owns the stage. Ending a
// Session never clears it; only an explicit SetCurrentScenePlacement/
// ClearCurrentScenePlacement call (backend/internal/shows/stage.go) does.
// VariablesJSON is a materialized cache of Show variables set by
// set_show_variable Cue actions -- the canonical source of truth is the
// show_id-scoped actions log, not this column (Kernel 70 SS4.2).
//
// Both fields are deliberately `json:"-"` -- this Show struct is also
// serialized wholesale by HandleShowProgram, the Audience-viewable
// endpoint (gated only by CanViewShowRun, which Audience passes). Backstage
// state and Cue-adjacent data must never leak through a struct that's
// reused across both backstage and audience-facing handlers (Kernel 70
// SS9's "audience projections structurally exclude backstage state and Cue
// internals"). Backstage callers that need these fields read them
// explicitly (see shows/http.go's HandleShowByID, which adds them to its
// response map only after its own CanViewBackstage check).
type Show struct {
	ID                          string          `json:"id"`
	ShowRunID                   string          `json:"show_run_id"`
	Slug                        string          `json:"slug"`
	ShortCode                   string          `json:"short_code,omitempty"`
	Title                       string          `json:"title"`
	Description                 string          `json:"description,omitempty"`
	AudienceTitle               string          `json:"audience_title,omitempty"`
	AudienceProgramBlurb        string          `json:"audience_program_blurb,omitempty"`
	Status                      string          `json:"status"`
	CurrentShowScenePlacementID *string         `json:"-"`
	VariablesJSON               json.RawMessage `json:"-"`
	ScheduledStartAt            *time.Time      `json:"scheduled_start_at,omitempty"`
	ScheduledEndAt              *time.Time      `json:"scheduled_end_at,omitempty"`
	ActualStartAt               *time.Time      `json:"actual_start_at,omitempty"`
	ActualEndAt                 *time.Time      `json:"actual_end_at,omitempty"`
	CreatedByUserID             string          `json:"created_by_user_id"`
	CreatedAt                   time.Time       `json:"created_at"`
	UpdatedAt                   time.Time       `json:"updated_at"`
	ArchivedAt                  *time.Time      `json:"archived_at,omitempty"`
}

// ShowSummary is the list-view shape returned by ListShowsForRun.
type ShowSummary struct {
	ID               string     `json:"id"`
	Slug             string     `json:"slug"`
	Title            string     `json:"title"`
	Status           string     `json:"status"`
	ScheduledStartAt *time.Time `json:"scheduled_start_at,omitempty"`
	ScheduledEndAt   *time.Time `json:"scheduled_end_at,omitempty"`
	ActualStartAt    *time.Time `json:"actual_start_at,omitempty"`
	ActualEndAt      *time.Time `json:"actual_end_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// ShowRunShowsSummary buckets a Show Run's Shows by rough lifecycle state,
// computed in Go over one query result rather than a separate aggregate
// query -- this is the "current/upcoming/live/completed indicator" the
// operator spec asks Show Run pages to display.
type ShowRunShowsSummary struct {
	Live      int `json:"live"`
	Upcoming  int `json:"upcoming"`
	Completed int `json:"completed"`
	Total     int `json:"total"`
}

// CreateShowInput is the caller-supplied subset of a new Show. ShowRunID is
// a separate function argument, not a field here, matching CreateShowRun's
// own convention in the showruns package.
type CreateShowInput struct {
	Title                string
	Slug                 string
	Description          string
	AudienceTitle        string
	AudienceProgramBlurb string
}

// UpdateShowPatch carries only the fields being changed. A nil pointer means
// "leave as-is"; the four timestamp fields are raw ISO-8601 strings (an
// empty string clears the column to NULL via the same NULLIF($n, ”)
// pattern this codebase already uses for optional text columns, extended
// with a ::timestamptz cast) rather than **time.Time, so the HTTP layer
// doesn't need a double-pointer decode.
type UpdateShowPatch struct {
	Title                *string
	Description          *string
	AudienceTitle        *string
	AudienceProgramBlurb *string
	Status               *string
	ScheduledStartAt     *string
	ScheduledEndAt       *string
	ActualStartAt        *string
	ActualEndAt          *string
}
