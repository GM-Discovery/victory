package storyboards

// Timeline instantiation (Kernel 82). CreateTimelineBoard is a sibling of
// CreateBoard, not a variant of it -- deliberately not threading a `mode`
// parameter through CreateBoard's existing signature, which every
// existing caller and test already depends on. Both functions build the
// same shape of thing (one transaction, slug-allocated columns/band/row)
// but Timeline additionally seeds boundary column roles and the default
// Reference Panel, neither of which Blank has any use for.

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateTimelineBoard instantiates a new owned Storyboard from the
// built-in Timeline template (spec 4.1): three columns (Beginning/
// Middle/Ending, with Beginning and Ending carrying their protected
// structural roles), one default band and row, and the default Reference
// Panel fields. The template itself (timeline_template.go) is read once
// here and never referenced again -- editing the returned board can never
// mutate it, and no other saved Timeline is affected.
func CreateTimelineBoard(ctx context.Context, pool *pgxpool.Pool, ownerUserID, title, description string) (*Storyboard, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrInvalidTitle
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var boardID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO storyboards (owner_user_id, title, description, mode, template_version)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text
	`, ownerUserID, title, description, ModeTimeline, TimelineTemplateVersion).Scan(&boardID); err != nil {
		return nil, err
	}

	for i, seed := range timelineDefaultColumns {
		if _, err := allocateUniqueSlug(ctx, tx, columnSlugExistsSQL, boardID, seed.title, func(ctx context.Context, slug string) (string, error) {
			var newID string
			err := tx.QueryRow(ctx, `
				INSERT INTO storyboard_columns (storyboard_id, title, slug, column_role, sort_order)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id::text
			`, boardID, seed.title, slug, seed.role, i).Scan(&newID)
			return newID, err
		}); err != nil {
			return nil, err
		}
	}

	var bandID string
	if bandID, err = allocateUniqueSlug(ctx, tx, bandSlugExistsSQL, boardID, timelineDefaultBandLabel, func(ctx context.Context, slug string) (string, error) {
		var newID string
		err := tx.QueryRow(ctx, `
			INSERT INTO storyboard_bands (storyboard_id, label, slug, sort_order)
			VALUES ($1, $2, $3, 0)
			RETURNING id::text
		`, boardID, timelineDefaultBandLabel, slug).Scan(&newID)
		return newID, err
	}); err != nil {
		return nil, err
	}

	if _, err := allocateUniqueSlug(ctx, tx, rowSlugExistsSQL, bandID, timelineDefaultRowLabel, func(ctx context.Context, slug string) (string, error) {
		var newID string
		err := tx.QueryRow(ctx, `
			INSERT INTO storyboard_rows (storyboard_id, band_id, label, slug, sort_order_in_band)
			VALUES ($1, $2, $3, $4, 0)
			RETURNING id::text
		`, boardID, bandID, timelineDefaultRowLabel, slug).Scan(&newID)
		return newID, err
	}); err != nil {
		return nil, err
	}

	for i, field := range timelineDefaultReferenceFields {
		if _, err := allocateUniqueSlug(ctx, tx, referenceFieldSlugExistsSQL, boardID, field.label, func(ctx context.Context, slug string) (string, error) {
			var newID string
			err := tx.QueryRow(ctx, `
				INSERT INTO storyboard_reference_fields
					(storyboard_id, slug, label, field_type, sort_order, sublabel_a, sublabel_b)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				RETURNING id::text
			`, boardID, slug, field.label, field.fieldType, i, field.sublabelA, field.sublabelB).Scan(&newID)
			return newID, err
		}); err != nil {
			return nil, err
		}
	}

	board, err := loadBoardRow(ctx, tx, boardID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return board, nil
}
