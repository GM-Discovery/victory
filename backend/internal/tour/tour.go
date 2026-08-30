// Package tour is Kernel 91's guided-tour completion record: a small,
// bounded set of campus/venue/role tours, recorded idempotently per user and
// read back to decide what a user has already been shown.
//
// This is deliberately NOT a generic LMS/course-builder (kernel-91 S30, S53):
// there is no tour-authoring surface, no quiz/scoring model, and no branching
// step logic. The tour set is a CHECK constraint in migration 107 and the
// mirrored constant list below, following tutorial.IsMilestone's precedent
// (backend/internal/tutorial/progress.go) -- adding a tour is a migration,
// on purpose.
//
// This package performs no authorization of its own. Callers resolve role
// eligibility via EligibleTours (eligibility.go), which is the one place
// server-authoritative role/venue checks happen; RecordCompletion/
// HasCompletion/LoadCompletions here only read and write the ledger for an
// already-authenticated Subject.
package tour

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	KeyCampusMandatory               = "campus_mandatory"
	KeyCampusContinuation            = "campus_continuation"
	KeyGreenroomIntro                = "greenroom_intro"
	KeyCatharsisCast                 = "catharsis_cast"
	KeyDirectorsChairDirectorToolbox = "directors_chair_director_toolbox"

	StatusCompleted = "completed"
	StatusSkipped   = "skipped"
)

var allTourKeys = []string{
	KeyCampusMandatory,
	KeyCampusContinuation,
	KeyGreenroomIntro,
	KeyCatharsisCast,
	KeyDirectorsChairDirectorToolbox,
}

// IsTourKey mirrors tour_completions_tour_key_check in migration 107, so an
// unrecognized key fails as a clean error rather than a constraint
// violation surfacing as a 500.
func IsTourKey(key string) bool {
	key = strings.TrimSpace(key)
	for _, k := range allTourKeys {
		if k == key {
			return true
		}
	}
	return false
}

// Subject is the already-authorized identity a tour completion belongs to.
// Callers derive it from the session cookie (see http.go); this package
// never derives identity from a client-supplied payload (kernel-91 S46: "one
// user cannot alter another user's tour completion").
type Subject struct {
	UserID string
}

func (s Subject) valid() error {
	if strings.TrimSpace(s.UserID) == "" {
		return errors.New("not_authenticated")
	}
	return nil
}

func nullableString(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}

// Completion is one recorded row, for the account-page replay list.
type Completion struct {
	TourKey     string `json:"tour_key"`
	VenueSlug   string `json:"venue_slug,omitempty"`
	RoleKey     string `json:"role_key,omitempty"`
	Status      string `json:"status"`
	StepReached string `json:"step_reached,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// RecordCompletion writes a completed/skipped row idempotently: a repeated
// call (double Next-press, retry, reconnect) is a no-op rather than a
// duplicate row, enforced by the UNIQUE constraint rather than a
// read-then-write race. A row is a durable historical fact -- replaying a
// tour never updates or re-inserts it (see EligibleTours in eligibility.go).
func RecordCompletion(ctx context.Context, pool *pgxpool.Pool, subj Subject, tourKey, venueSlug, roleKey, status, stepReached string) error {
	if err := subj.valid(); err != nil {
		return err
	}
	tourKey = strings.TrimSpace(tourKey)
	if !IsTourKey(tourKey) {
		return errors.New("invalid_tour_key")
	}
	status = strings.TrimSpace(status)
	if status != StatusCompleted && status != StatusSkipped {
		return errors.New("invalid_status")
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO tour_completions (user_id, tour_key, venue_slug, role_key, status, step_reached)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, tour_key, COALESCE(venue_slug, ''), COALESCE(role_key, '')) DO NOTHING
	`, subj.UserID, tourKey, nullableString(venueSlug), nullableString(roleKey), status, strings.TrimSpace(stepReached))
	if err != nil {
		return err
	}
	// The resumable cursor's job ends once the tour is actually finished --
	// leaving it behind is harmless (EligibleTours never offers a completed
	// tour again) but pointless to keep.
	_, err = pool.Exec(ctx, `
		DELETE FROM tour_progress
		WHERE user_id = $1 AND tour_key = $2
		  AND COALESCE(venue_slug, '') = COALESCE($3, '')
		  AND COALESCE(role_key, '') = COALESCE($4, '')
	`, subj.UserID, tourKey, nullableString(venueSlug), nullableString(roleKey))
	return err
}

// RecordProgress upserts the resumable cursor for a tour that has not yet
// been completed or skipped (see migration 108's comment for why this
// exists as a separate table from tour_completions). Written via
// navigator.sendBeacon from a click-gated step whose target navigates away
// immediately -- see kernel-91's Audition Hall/Trailer steps -- so an
// ordinary fetch() racing the page unload can no longer silently lose the
// user's place and restart the mandatory tour from step 0.
func RecordProgress(ctx context.Context, pool *pgxpool.Pool, subj Subject, tourKey, venueSlug, roleKey, stepReached string) error {
	if err := subj.valid(); err != nil {
		return err
	}
	tourKey = strings.TrimSpace(tourKey)
	if !IsTourKey(tourKey) {
		return errors.New("invalid_tour_key")
	}
	stepReached = strings.TrimSpace(stepReached)
	if stepReached == "" {
		return errors.New("invalid_step")
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO tour_progress (user_id, tour_key, venue_slug, role_key, step_reached, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (user_id, tour_key, COALESCE(venue_slug, ''), COALESCE(role_key, ''))
		DO UPDATE SET step_reached = EXCLUDED.step_reached, updated_at = NOW()
	`, subj.UserID, tourKey, nullableString(venueSlug), nullableString(roleKey), stepReached)
	return err
}

// LoadProgressStep returns the last step_reached recorded for this scope,
// or "" if there is none.
func LoadProgressStep(ctx context.Context, pool *pgxpool.Pool, subj Subject, tourKey, venueSlug, roleKey string) (string, error) {
	if err := subj.valid(); err != nil {
		return "", err
	}
	var stepReached string
	err := pool.QueryRow(ctx, `
		SELECT step_reached FROM tour_progress
		WHERE user_id = $1 AND tour_key = $2
		  AND COALESCE(venue_slug, '') = COALESCE($3, '')
		  AND COALESCE(role_key, '') = COALESCE($4, '')
	`, subj.UserID, strings.TrimSpace(tourKey), nullableString(venueSlug), nullableString(roleKey)).Scan(&stepReached)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return stepReached, nil
}

// HasCompletion reports whether subj has a completed-or-skipped row for the
// given (tour, venue, role) scope. Both completed and skipped count as
// "already shown this" for eligibility purposes -- callers that need to
// distinguish the two read Status via LoadCompletions instead.
func HasCompletion(ctx context.Context, pool *pgxpool.Pool, subj Subject, tourKey, venueSlug, roleKey string) (bool, error) {
	if err := subj.valid(); err != nil {
		return false, err
	}
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM tour_completions
			WHERE user_id = $1
			  AND tour_key = $2
			  AND COALESCE(venue_slug, '') = COALESCE($3, '')
			  AND COALESCE(role_key, '') = COALESCE($4, '')
		)
	`, subj.UserID, strings.TrimSpace(tourKey), nullableString(venueSlug), nullableString(roleKey)).Scan(&exists)
	return exists, err
}

// LoadCompletions returns every tour row for subj, most recent first, for
// the account-page replay list.
func LoadCompletions(ctx context.Context, pool *pgxpool.Pool, subj Subject) ([]Completion, error) {
	if err := subj.valid(); err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT tour_key, COALESCE(venue_slug, ''), COALESCE(role_key, ''), status, step_reached, created_at::text
		FROM tour_completions
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, subj.UserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Completion{}
	for rows.Next() {
		var c Completion
		if err := rows.Scan(&c.TourKey, &c.VenueSlug, &c.RoleKey, &c.Status, &c.StepReached, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
