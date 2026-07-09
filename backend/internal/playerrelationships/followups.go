package playerrelationships

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxFollowUpTitleLength = 200
	maxFollowUpNotesLength = 10000
)

// FollowUpInput carries a create or update of one stored follow-up item.
// Follow-ups are records only -- no scheduler, reminder, notification,
// email, or calendar integration exists for them (Kernel 62 §4.5, §5.6).
type FollowUpInput struct {
	Title      string `json:"title"`
	Notes      string `json:"notes"`
	TargetDate string `json:"target_date"`
}

func sanitizeFollowUpInput(input FollowUpInput) (FollowUpInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Notes = strings.TrimSpace(input.Notes)
	input.TargetDate = strings.TrimSpace(input.TargetDate)

	if input.Title == "" {
		return FollowUpInput{}, errors.New("followup_title_required")
	}
	if len([]rune(input.Title)) > maxFollowUpTitleLength {
		return FollowUpInput{}, errors.New("followup_title_too_long")
	}
	if len(input.Notes) > maxFollowUpNotesLength {
		return FollowUpInput{}, errors.New("followup_notes_too_long")
	}
	if input.TargetDate != "" {
		if _, err := time.Parse("2006-01-02", input.TargetDate); err != nil {
			return FollowUpInput{}, errors.New("invalid_target_date")
		}
	}
	return input, nil
}

const followUpColumns = `
	id::text, title, notes,
	COALESCE(to_char(target_date, 'YYYY-MM-DD'), '') AS target_date,
	status, created_at, updated_at, completed_at
`

func scanFollowUp(scan func(dest ...any) error) (FollowUp, error) {
	var f FollowUp
	err := scan(&f.ID, &f.Title, &f.Notes, &f.TargetDate, &f.Status, &f.CreatedAt, &f.UpdatedAt, &f.CompletedAt)
	return f, err
}

// CreateFollowUp stores one private intention (Kernel 62 §9.4).
func CreateFollowUp(ctx context.Context, pool *pgxpool.Pool, observerUserID, relationshipID string, input FollowUpInput) (FollowUp, error) {
	if _, err := loadRelationshipOwned(ctx, pool, observerUserID, relationshipID); err != nil {
		return FollowUp{}, err
	}
	input, err := sanitizeFollowUpInput(input)
	if err != nil {
		return FollowUp{}, err
	}

	row := pool.QueryRow(ctx, `
		INSERT INTO player_relationship_followups (relationship_id, title, notes, target_date)
		VALUES ($1, $2, $3, NULLIF($4, '')::date)
		RETURNING `+followUpColumns+`
	`, relationshipID, input.Title, input.Notes, input.TargetDate)
	f, err := scanFollowUp(row.Scan)
	if err != nil {
		return FollowUp{}, err
	}
	if err := touchRelationship(ctx, pool, relationshipID); err != nil {
		return FollowUp{}, err
	}
	return f, nil
}

// UpdateFollowUp edits title/notes/target date of one open-or-closed item
// (Kernel 62 §9.4). Status transitions go through Complete/Dismiss/Reopen.
func UpdateFollowUp(ctx context.Context, pool *pgxpool.Pool, observerUserID, relationshipID, followupID string, input FollowUpInput) (FollowUp, error) {
	if _, err := loadRelationshipOwned(ctx, pool, observerUserID, relationshipID); err != nil {
		return FollowUp{}, err
	}
	input, err := sanitizeFollowUpInput(input)
	if err != nil {
		return FollowUp{}, err
	}

	row := pool.QueryRow(ctx, `
		UPDATE player_relationship_followups
		SET title = $3, notes = $4, target_date = NULLIF($5, '')::date, updated_at = NOW()
		WHERE id = $1 AND relationship_id = $2
		RETURNING `+followUpColumns+`
	`, followupID, relationshipID, input.Title, input.Notes, input.TargetDate)
	f, err := scanFollowUp(row.Scan)
	if err != nil {
		return FollowUp{}, errors.New("followup_not_found")
	}
	if err := touchRelationship(ctx, pool, relationshipID); err != nil {
		return FollowUp{}, err
	}
	return f, nil
}

func setFollowUpStatus(ctx context.Context, pool *pgxpool.Pool, observerUserID, relationshipID, followupID, status string) (FollowUp, error) {
	if !ValidateFollowUpStatus(status) {
		return FollowUp{}, errors.New("invalid_followup_status")
	}
	if _, err := loadRelationshipOwned(ctx, pool, observerUserID, relationshipID); err != nil {
		return FollowUp{}, err
	}

	row := pool.QueryRow(ctx, `
		UPDATE player_relationship_followups
		SET status = $3,
		    completed_at = CASE WHEN $3 = 'done' THEN NOW() ELSE NULL END,
		    updated_at = NOW()
		WHERE id = $1 AND relationship_id = $2
		RETURNING `+followUpColumns+`
	`, followupID, relationshipID, status)
	f, err := scanFollowUp(row.Scan)
	if err != nil {
		return FollowUp{}, errors.New("followup_not_found")
	}
	if err := touchRelationship(ctx, pool, relationshipID); err != nil {
		return FollowUp{}, err
	}
	return f, nil
}

// CompleteFollowUp marks an item done (Kernel 62 §9.4).
func CompleteFollowUp(ctx context.Context, pool *pgxpool.Pool, observerUserID, relationshipID, followupID string) (FollowUp, error) {
	return setFollowUpStatus(ctx, pool, observerUserID, relationshipID, followupID, FollowUpDone)
}

// DismissFollowUp marks an item dismissed (Kernel 62 §9.4).
func DismissFollowUp(ctx context.Context, pool *pgxpool.Pool, observerUserID, relationshipID, followupID string) (FollowUp, error) {
	return setFollowUpStatus(ctx, pool, observerUserID, relationshipID, followupID, FollowUpDismissed)
}

// ReopenFollowUp returns an item to open. Not required by the kernel but a
// natural inverse the UI can offer safely -- still just a stored record.
func ReopenFollowUp(ctx context.Context, pool *pgxpool.Pool, observerUserID, relationshipID, followupID string) (FollowUp, error) {
	return setFollowUpStatus(ctx, pool, observerUserID, relationshipID, followupID, FollowUpOpen)
}

// ListFollowUps returns every follow-up on the relationship, open first,
// then by target date and creation time (Kernel 62 §9.4).
func ListFollowUps(ctx context.Context, pool *pgxpool.Pool, observerUserID, relationshipID string) ([]FollowUp, error) {
	if _, err := loadRelationshipOwned(ctx, pool, observerUserID, relationshipID); err != nil {
		return nil, err
	}

	rows, err := pool.Query(ctx, `
		SELECT `+followUpColumns+`
		FROM player_relationship_followups
		WHERE relationship_id = $1
		ORDER BY (status = 'open') DESC, target_date ASC NULLS LAST, created_at DESC
	`, relationshipID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []FollowUp{}
	for rows.Next() {
		f, err := scanFollowUp(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
