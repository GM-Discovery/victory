package shows

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
)

const showColumns = `
	id::text, show_run_id::text, slug, title, COALESCE(description, ''),
	COALESCE(audience_title, ''), COALESCE(audience_program_blurb, ''),
	status, current_show_scene_placement_id::text, variables_json::text,
	scheduled_start_at, scheduled_end_at, actual_start_at, actual_end_at,
	created_by_user_id::text, created_at, updated_at, archived_at
`

func scanShow(row pgx.Row) (Show, error) {
	var s Show
	var currentPlacementID *string
	var variablesText string
	if err := row.Scan(
		&s.ID, &s.ShowRunID, &s.Slug, &s.Title, &s.Description,
		&s.AudienceTitle, &s.AudienceProgramBlurb,
		&s.Status, &currentPlacementID, &variablesText,
		&s.ScheduledStartAt, &s.ScheduledEndAt, &s.ActualStartAt, &s.ActualEndAt,
		&s.CreatedByUserID, &s.CreatedAt, &s.UpdatedAt, &s.ArchivedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Show{}, errors.New("show_not_found")
		}
		return Show{}, err
	}
	s.CurrentShowScenePlacementID = currentPlacementID
	if variablesText != "" {
		s.VariablesJSON = json.RawMessage(variablesText)
	}
	return s, nil
}

// CreateShow authority-checks the actor against the parent Show Run's
// location (resolved server-side, never trusted from the client) and
// inserts a new Show defaulting to status "draft".
func CreateShow(ctx context.Context, pool *pgxpool.Pool, actorUserID, showRunID string, in CreateShowInput) (Show, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return Show{}, errors.New("not_authenticated")
	}
	if strings.TrimSpace(in.Title) == "" {
		return Show{}, errors.New("title_required")
	}
	if strings.TrimSpace(in.Slug) == "" {
		return Show{}, errors.New("slug_required")
	}

	sr, err := showruns.LoadShowRunByID(ctx, pool, showRunID)
	if err != nil {
		return Show{}, err
	}
	ok, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return Show{}, err
	}
	if !ok {
		return Show{}, errors.New("not_authorized")
	}

	row := pool.QueryRow(ctx, `
		INSERT INTO shows (show_run_id, slug, title, description, audience_title, audience_program_blurb, created_by_user_id)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), $7)
		RETURNING `+showColumns,
		showRunID, in.Slug, in.Title, in.Description, in.AudienceTitle, in.AudienceProgramBlurb, actorUserID)
	return scanShow(row)
}

// UpdateShow authority-checks against the Show's parent Show Run location,
// then applies only the fields present in the patch.
func UpdateShow(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID string, patch UpdateShowPatch) (Show, error) {
	s, err := LoadShowByID(ctx, pool, showID)
	if err != nil {
		return Show{}, err
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return Show{}, err
	}
	ok, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return Show{}, err
	}
	if !ok {
		return Show{}, errors.New("not_authorized")
	}

	title := s.Title
	if patch.Title != nil {
		title = *patch.Title
	}
	description := s.Description
	if patch.Description != nil {
		description = *patch.Description
	}
	audienceTitle := s.AudienceTitle
	if patch.AudienceTitle != nil {
		audienceTitle = *patch.AudienceTitle
	}
	audienceBlurb := s.AudienceProgramBlurb
	if patch.AudienceProgramBlurb != nil {
		audienceBlurb = *patch.AudienceProgramBlurb
	}
	status := s.Status
	if patch.Status != nil {
		status = *patch.Status
	}
	scheduledStart := timePatchValue(patch.ScheduledStartAt, s.ScheduledStartAt)
	scheduledEnd := timePatchValue(patch.ScheduledEndAt, s.ScheduledEndAt)
	actualStart := timePatchValue(patch.ActualStartAt, s.ActualStartAt)
	actualEnd := timePatchValue(patch.ActualEndAt, s.ActualEndAt)

	archivedAt := "NULL"
	if status == "archived" {
		archivedAt = "NOW()"
	}

	row := pool.QueryRow(ctx, `
		UPDATE shows
		SET title = $2, description = NULLIF($3, ''), audience_title = NULLIF($4, ''),
		    audience_program_blurb = NULLIF($5, ''), status = $6,
		    scheduled_start_at = $7, scheduled_end_at = $8,
		    actual_start_at = $9, actual_end_at = $10,
		    archived_at = `+archivedAt+`, updated_at = NOW()
		WHERE id = $1
		RETURNING `+showColumns,
		showID, title, description, audienceTitle, audienceBlurb, status,
		scheduledStart, scheduledEnd, actualStart, actualEnd)
	return scanShow(row)
}

// timePatchValue resolves an UpdateShowPatch string field (nil = don't
// touch, "" = clear to NULL, non-empty = parse and set) against the show's
// current value for that column, returning what should be bound to the SQL
// parameter.
func timePatchValue(patch *string, current *time.Time) *time.Time {
	if patch == nil {
		return current
	}
	if strings.TrimSpace(*patch) == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, *patch)
	if err != nil {
		return current
	}
	return &parsed
}

// ArchiveShow is a thin wrapper over UpdateShow's status field.
func ArchiveShow(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID string) (Show, error) {
	status := "archived"
	return UpdateShow(ctx, pool, actorUserID, showID, UpdateShowPatch{Status: &status})
}

// LoadShowByID returns the raw row with no authority check -- callers that
// expose this to a viewer must authority-check separately, matching Show
// Run's own "existence isn't secret" convention.
func LoadShowByID(ctx context.Context, pool *pgxpool.Pool, showID string) (Show, error) {
	showID = strings.TrimSpace(showID)
	if showID == "" {
		return Show{}, errors.New("show_not_found")
	}
	row := pool.QueryRow(ctx, `SELECT `+showColumns+` FROM shows WHERE id = $1`, showID)
	return scanShow(row)
}

// ListShowsForRun returns every Show under one Show Run, plus a rough
// lifecycle bucket summary computed in Go over the same result set (no
// separate aggregate query). Callers must already have passed
// showruns.CanViewShowRun for this run's location before calling.
func ListShowsForRun(ctx context.Context, pool *pgxpool.Pool, showRunID string) ([]ShowSummary, ShowRunShowsSummary, error) {
	rows, err := pool.Query(ctx, `
		SELECT `+showColumns+`
		FROM shows
		WHERE show_run_id = $1
		ORDER BY created_at DESC
	`, showRunID)
	if err != nil {
		return nil, ShowRunShowsSummary{}, err
	}
	defer rows.Close()

	var out []ShowSummary
	var summary ShowRunShowsSummary
	now := time.Now()
	for rows.Next() {
		s, err := scanShow(rows)
		if err != nil {
			return nil, ShowRunShowsSummary{}, err
		}
		summary.Total++
		switch {
		case s.Status == "live":
			summary.Live++
		case s.Status == "scheduled" && s.ScheduledStartAt != nil && s.ScheduledStartAt.After(now):
			summary.Upcoming++
		case s.Status == "completed" || s.Status == "cancelled" || s.Status == "archived":
			summary.Completed++
		}
		out = append(out, ShowSummary{
			ID:               s.ID,
			Slug:             s.Slug,
			Title:            s.Title,
			Status:           s.Status,
			ScheduledStartAt: s.ScheduledStartAt,
			ScheduledEndAt:   s.ScheduledEndAt,
			ActualStartAt:    s.ActualStartAt,
			ActualEndAt:      s.ActualEndAt,
			CreatedAt:        s.CreatedAt,
		})
	}
	return out, summary, rows.Err()
}
