package showruns

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
)

const showRunColumns = `
	id::text, location_id::text, production_id::text, title, slug,
	COALESCE(description, ''), show_format, COALESCE(custom_show_format, ''),
	COALESCE(cohort_name, ''), status, audience_self_join_enabled,
	created_by_user_id::text, created_at, updated_at, archived_at
`

func scanShowRun(row pgx.Row) (ShowRun, error) {
	var sr ShowRun
	if err := row.Scan(
		&sr.ID, &sr.LocationID, &sr.ProductionID, &sr.Title, &sr.Slug,
		&sr.Description, &sr.ShowFormat, &sr.CustomShowFormat,
		&sr.CohortName, &sr.Status, &sr.AudienceSelfJoinEnabled,
		&sr.CreatedByUserID, &sr.CreatedAt, &sr.UpdatedAt, &sr.ArchivedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ShowRun{}, errors.New("show_run_not_found")
		}
		return ShowRun{}, err
	}
	return sr, nil
}

// CreateShowRunInput is the caller-supplied subset of a new Show Run.
// LocationID is deliberately absent -- it is always resolved server-side
// from productions.location_id, never trusted from the client.
type CreateShowRunInput struct {
	Title            string
	Slug             string
	Description      string
	ShowFormat       string
	CustomShowFormat string
	CohortName       string
}

// CreateShowRun resolves the run's location from the given Production
// (never trusting a client-supplied location_id), authority-checks the
// actor against that resolved location, then inserts.
func CreateShowRun(ctx context.Context, pool *pgxpool.Pool, actorUserID, productionID string, in CreateShowRunInput) (ShowRun, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	productionID = strings.TrimSpace(productionID)
	if actorUserID == "" {
		return ShowRun{}, errors.New("not_authenticated")
	}
	if productionID == "" {
		return ShowRun{}, errors.New("production_id_required")
	}
	if strings.TrimSpace(in.Title) == "" {
		return ShowRun{}, errors.New("title_required")
	}
	if strings.TrimSpace(in.Slug) == "" {
		return ShowRun{}, errors.New("slug_required")
	}

	var locationID string
	if err := pool.QueryRow(ctx, `SELECT location_id::text FROM productions WHERE id = $1`, productionID).Scan(&locationID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ShowRun{}, errors.New("production_not_found")
		}
		return ShowRun{}, err
	}

	ok, err := CanManageShowRun(ctx, pool, actorUserID, locationID)
	if err != nil {
		return ShowRun{}, err
	}
	if !ok {
		return ShowRun{}, errors.New("not_authorized")
	}

	format := strings.TrimSpace(in.ShowFormat)
	if format == "" {
		format = "one-shot"
	}

	row := pool.QueryRow(ctx, `
		INSERT INTO show_runs (
			location_id, production_id, title, slug, description,
			show_format, custom_show_format, cohort_name, created_by_user_id
		)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, NULLIF($7, ''), NULLIF($8, ''), $9)
		RETURNING `+showRunColumns, locationID, productionID, in.Title, in.Slug, in.Description,
		format, in.CustomShowFormat, in.CohortName, actorUserID)
	return scanShowRun(row)
}

// UpdateShowRunPatch carries only the fields being changed.
type UpdateShowRunPatch struct {
	Title                   *string
	Description             *string
	ShowFormat              *string
	CustomShowFormat        *string
	CohortName              *string
	Status                  *string
	AudienceSelfJoinEnabled *bool
}

// UpdateShowRun authority-checks against the run's existing location, then
// applies only the fields present in the patch.
func UpdateShowRun(ctx context.Context, pool *pgxpool.Pool, actorUserID, showRunID string, patch UpdateShowRunPatch) (ShowRun, error) {
	sr, err := LoadShowRunByID(ctx, pool, showRunID)
	if err != nil {
		return ShowRun{}, err
	}
	ok, err := CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return ShowRun{}, err
	}
	if !ok {
		return ShowRun{}, errors.New("not_authorized")
	}

	title := sr.Title
	if patch.Title != nil {
		title = *patch.Title
	}
	description := sr.Description
	if patch.Description != nil {
		description = *patch.Description
	}
	format := sr.ShowFormat
	if patch.ShowFormat != nil {
		format = *patch.ShowFormat
	}
	customFormat := sr.CustomShowFormat
	if patch.CustomShowFormat != nil {
		customFormat = *patch.CustomShowFormat
	}
	cohort := sr.CohortName
	if patch.CohortName != nil {
		cohort = *patch.CohortName
	}
	status := sr.Status
	if patch.Status != nil {
		status = *patch.Status
	}
	selfJoin := sr.AudienceSelfJoinEnabled
	if patch.AudienceSelfJoinEnabled != nil {
		selfJoin = *patch.AudienceSelfJoinEnabled
	}

	archivedAt := "NULL"
	if status == "archived" {
		archivedAt = "NOW()"
	}

	row := pool.QueryRow(ctx, `
		UPDATE show_runs
		SET title = $2, description = NULLIF($3, ''), show_format = $4,
		    custom_show_format = NULLIF($5, ''), cohort_name = NULLIF($6, ''),
		    status = $7, audience_self_join_enabled = $8,
		    archived_at = `+archivedAt+`, updated_at = NOW()
		WHERE id = $1
		RETURNING `+showRunColumns,
		showRunID, title, description, format, customFormat, cohort, status, selfJoin)
	return scanShowRun(row)
}

// ArchiveShowRun and UnarchiveShowRun are thin wrappers over UpdateShowRun's
// status field so callers don't have to build a patch by hand.
func ArchiveShowRun(ctx context.Context, pool *pgxpool.Pool, actorUserID, showRunID string) (ShowRun, error) {
	status := "archived"
	return UpdateShowRun(ctx, pool, actorUserID, showRunID, UpdateShowRunPatch{Status: &status})
}

func UnarchiveShowRun(ctx context.Context, pool *pgxpool.Pool, actorUserID, showRunID string) (ShowRun, error) {
	status := "planning"
	return UpdateShowRun(ctx, pool, actorUserID, showRunID, UpdateShowRunPatch{Status: &status})
}

// LoadShowRunByID returns the raw row with no authority check -- callers
// that expose this to a viewer must authority-check separately, since Show
// Run existence itself is not meant to be secret (unlike Kernel 62's
// subject-invisibility rule).
func LoadShowRunByID(ctx context.Context, pool *pgxpool.Pool, showRunID string) (ShowRun, error) {
	showRunID = strings.TrimSpace(showRunID)
	if showRunID == "" {
		return ShowRun{}, errors.New("show_run_not_found")
	}
	row := pool.QueryRow(ctx, `SELECT `+showRunColumns+` FROM show_runs WHERE id = $1`, showRunID)
	return scanShowRun(row)
}

// ListShowRunsVisibleToUser returns every Show Run the caller may see:
// Operator sees all; everyone else sees runs at locations where they have
// any active membership. CanManage on each summary tells the frontend
// whether to show roster-management affordances for that specific run.
func ListShowRunsVisibleToUser(ctx context.Context, pool *pgxpool.Pool, viewerUserID string) ([]ShowRunSummary, error) {
	viewerUserID = strings.TrimSpace(viewerUserID)
	if viewerUserID == "" {
		return nil, errors.New("not_authenticated")
	}

	isOperator, err := access.IsOperatorUser(ctx, pool, viewerUserID)
	if err != nil {
		return nil, err
	}

	var rows pgx.Rows
	if isOperator {
		rows, err = pool.Query(ctx, `
			SELECT `+showRunColumns+`
			FROM show_runs
			ORDER BY created_at DESC
		`)
	} else {
		// Backstage listing (Kernel 68 §3.6): Producer/Director at the run's
		// location, or an active Show Run crew roster row -- not plain
		// Audience/Player membership. The separate Audience Program route
		// stays reachable through its own direct link regardless.
		rows, err = pool.Query(ctx, `
			SELECT `+showRunColumns+`
			FROM show_runs sr
			WHERE EXISTS (
				SELECT 1 FROM location_memberships lm
				WHERE lm.location_id = sr.location_id
				  AND lm.user_id = $1
				  AND lm.active = TRUE
				  AND lm.role IN ('producer', 'director')
			)
			OR EXISTS (
				SELECT 1 FROM show_run_roster_members rm
				WHERE rm.show_run_id = sr.id
				  AND rm.user_id = $1
				  AND rm.role = 'crew'
				  AND rm.removed_at IS NULL
			)
			ORDER BY created_at DESC
		`, viewerUserID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ShowRunSummary
	for rows.Next() {
		sr, err := scanShowRun(rows)
		if err != nil {
			return nil, err
		}
		canManage := isOperator
		if !canManage {
			canManage, err = CanManageShowRun(ctx, pool, viewerUserID, sr.LocationID)
			if err != nil {
				return nil, err
			}
		}
		out = append(out, ShowRunSummary{
			ID:                      sr.ID,
			Title:                   sr.Title,
			Slug:                    sr.Slug,
			Description:             sr.Description,
			ShowFormat:              sr.ShowFormat,
			CustomShowFormat:        sr.CustomShowFormat,
			CohortName:              sr.CohortName,
			Status:                  sr.Status,
			AudienceSelfJoinEnabled: sr.AudienceSelfJoinEnabled,
			CanManage:               canManage,
			CreatedAt:               sr.CreatedAt,
		})
	}
	return out, rows.Err()
}
