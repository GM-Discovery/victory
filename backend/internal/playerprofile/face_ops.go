package playerprofile

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SetFaceVisibility lets the owner show or hide an eligible fact on Face
// (Kernel 61 §6.6, §9.4, AC-21). Stage name and any reserved account field
// can never be hidden or targeted here (Kernel 61 §6.6, §6.8).
func SetFaceVisibility(ctx context.Context, pool *pgxpool.Pool, userID, fieldKey, mode string) error {
	userID = strings.TrimSpace(userID)
	fieldKey = strings.TrimSpace(fieldKey)
	mode = strings.TrimSpace(strings.ToLower(mode))

	if userID == "" {
		return errors.New("not_authenticated")
	}
	if IsReservedFieldKey(fieldKey) {
		return errors.New("field_not_face_eligible")
	}
	if mode != VisibilityShown && mode != VisibilityHidden {
		return errors.New("invalid_visibility_mode")
	}

	cat, err := LoadCatalogue()
	if err != nil {
		return err
	}
	field, ok := cat.FieldByKey(fieldKey)
	if !ok || !field.FaceEligible {
		return errors.New("field_not_face_eligible")
	}

	wb, err := EnsureWorkbook(ctx, pool, userID)
	if err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO player_profile_face_overrides (workbook_id, field_key, visibility_mode)
		VALUES ($1, $2, $3)
		ON CONFLICT (workbook_id, field_key)
		DO UPDATE SET visibility_mode = EXCLUDED.visibility_mode, updated_at = NOW()
	`, wb.ID, fieldKey, mode); err != nil {
		return err
	}

	return touchWorkbookProjection(ctx, pool, wb.ID)
}

// ReturnFaceVisibilityToInferred resets a fact's Face visibility to
// contract-inferred behavior (Kernel 61 §6.6, AC-21).
func ReturnFaceVisibilityToInferred(ctx context.Context, pool *pgxpool.Pool, userID, fieldKey string) error {
	userID = strings.TrimSpace(userID)
	fieldKey = strings.TrimSpace(fieldKey)
	if userID == "" {
		return errors.New("not_authenticated")
	}

	wb, err := EnsureWorkbook(ctx, pool, userID)
	if err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		UPDATE player_profile_face_overrides
		SET visibility_mode = 'inferred', updated_at = NOW()
		WHERE workbook_id = $1 AND field_key = $2
	`, wb.ID, fieldKey); err != nil {
		return err
	}

	return touchWorkbookProjection(ctx, pool, wb.ID)
}

// SetFacePriority sets a manual signed-integer priority for an eligible
// fact (Kernel 61 §6.6, §9.4, AC-22).
func SetFacePriority(ctx context.Context, pool *pgxpool.Pool, userID, fieldKey string, score int) error {
	userID = strings.TrimSpace(userID)
	fieldKey = strings.TrimSpace(fieldKey)
	if userID == "" {
		return errors.New("not_authenticated")
	}
	if IsReservedFieldKey(fieldKey) {
		return errors.New("field_not_face_eligible")
	}

	cat, err := LoadCatalogue()
	if err != nil {
		return err
	}
	field, ok := cat.FieldByKey(fieldKey)
	if !ok || !field.FaceEligible {
		return errors.New("field_not_face_eligible")
	}

	wb, err := EnsureWorkbook(ctx, pool, userID)
	if err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO player_profile_face_overrides (workbook_id, field_key, priority_mode, priority_score)
		VALUES ($1, $2, 'manual', $3)
		ON CONFLICT (workbook_id, field_key)
		DO UPDATE SET priority_mode = 'manual', priority_score = EXCLUDED.priority_score, updated_at = NOW()
	`, wb.ID, fieldKey, score); err != nil {
		return err
	}

	return touchWorkbookProjection(ctx, pool, wb.ID)
}

// ReturnFacePriorityToInferred resets a fact's priority to the
// catalogue-defined default (Kernel 61 §6.6, AC-22).
func ReturnFacePriorityToInferred(ctx context.Context, pool *pgxpool.Pool, userID, fieldKey string) error {
	userID = strings.TrimSpace(userID)
	fieldKey = strings.TrimSpace(fieldKey)
	if userID == "" {
		return errors.New("not_authenticated")
	}

	wb, err := EnsureWorkbook(ctx, pool, userID)
	if err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		UPDATE player_profile_face_overrides
		SET priority_mode = 'inferred', priority_score = NULL, updated_at = NOW()
		WHERE workbook_id = $1 AND field_key = $2
	`, wb.ID, fieldKey); err != nil {
		return err
	}

	return touchWorkbookProjection(ctx, pool, wb.ID)
}

func loadFaceOverrides(ctx context.Context, pool *pgxpool.Pool, workbookID string) (map[string]FaceOverride, error) {
	rows, err := pool.Query(ctx, `
		SELECT field_key, visibility_mode, priority_mode, COALESCE(priority_score, 0)
		FROM player_profile_face_overrides
		WHERE workbook_id = $1
	`, workbookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]FaceOverride{}
	for rows.Next() {
		var o FaceOverride
		if err := rows.Scan(&o.FieldKey, &o.VisibilityMode, &o.PriorityMode, &o.PriorityScore); err != nil {
			return nil, err
		}
		out[o.FieldKey] = o
	}
	return out, rows.Err()
}
