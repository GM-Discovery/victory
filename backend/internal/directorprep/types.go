// Package directorprep is Kernel 89's Director preparation store: small,
// typed, reusable prepared values a Director authors ahead of time and
// recalls during live play.
//
// The whole package is Director+ only. There is no Player read path, no
// "visible to" column, and no projection tier -- exposing something to a
// Player is always a separate, explicit act through an already-canonical
// model (a participant_interactions row for a merchant, a Stage Effect for
// an announcement). That is why kernel 89 §14's "Director-only
// preparations must not leak" needs no filter here: there is nothing for a
// Player to call.
//
// It is emphatically not a macro engine (kernel 89 §6, §24). One row is one
// understandable prepared value. Nothing in this package composes, chains,
// schedules, branches, or executes; `Recall` returns data and does not act
// on it. If a future kernel wants "award Fate AND announce AND move the
// Cohort", that is three Director decisions, not a new payload key.
package directorprep

import "time"

// Kinds. Closed set, mirrored by the CHECK constraint in migration 105.
const (
	// KindTargetComplexity is kernel 89 §7: a label and a number the
	// Director prepared in advance ("Climb Training Wall", 14). Recalling it
	// does not roll, does not choose a skill, and does not decide whether
	// the fiction permits the attempt -- the Director still adjudicates.
	KindTargetComplexity = "target_complexity"

	// KindAnnouncement is kernel 89 §10.3's saved custom announcement: a
	// style key from backend/internal/announcements plus the Director's own
	// words, kept so a recurring beat is one click rather than retyping.
	KindAnnouncement = "announcement"
)

// Preparation is one prepared record.
type Preparation struct {
	ID              string         `json:"id"`
	ShowID          string         `json:"show_id"`
	Kind            string         `json:"kind"`
	Label           string         `json:"label"`
	Payload         map[string]any `json:"payload"`
	SortOrder       int            `json:"sort_order"`
	CreatedByUserID string         `json:"created_by_user_id,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// TargetComplexity is the decoded payload for KindTargetComplexity.
type TargetComplexity struct {
	Value int    `json:"value"`
	Note  string `json:"note,omitempty"`
}

// Announcement is the decoded payload for KindAnnouncement.
type Announcement struct {
	Style string `json:"style"`
	Text  string `json:"text"`
}

// Bounds. Deliberately generous but finite: a target complexity is a Socio
// number a human reads at a glance, and a label is a menu entry.
const (
	MaxLabelLength      = 80
	MaxNoteLength       = 200
	MinComplexityValue  = 1
	MaxComplexityValue  = 99
)
