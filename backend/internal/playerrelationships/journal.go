package playerrelationships

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxJournalTitleLength = 200
	maxJournalBodyLength  = 50000
	maxJournalTags        = 20
	maxTagLength          = 60
)

// JournalInput carries a create or update of one private journal entry
// (Kernel 62 §5.5, §9.3).
type JournalInput struct {
	Title        string   `json:"title"`
	Body         string   `json:"body"`
	EntryDate    string   `json:"entry_date"`
	NoteCategory string   `json:"note_category"`
	Tags         []string `json:"tags"`
	ProductionID string   `json:"production_id"`
}

func sanitizeJournalInput(input JournalInput) (JournalInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Body = strings.TrimSpace(input.Body)
	input.NoteCategory = strings.TrimSpace(strings.ToLower(input.NoteCategory))
	input.EntryDate = strings.TrimSpace(input.EntryDate)
	input.ProductionID = strings.TrimSpace(input.ProductionID)

	if input.Body == "" {
		return JournalInput{}, errors.New("journal_body_required")
	}
	if len([]rune(input.Title)) > maxJournalTitleLength {
		return JournalInput{}, errors.New("journal_title_too_long")
	}
	if len(input.Body) > maxJournalBodyLength {
		return JournalInput{}, errors.New("journal_body_too_long")
	}
	if input.NoteCategory == "" {
		input.NoteCategory = "general"
	}
	if input.EntryDate != "" {
		if _, err := time.Parse("2006-01-02", input.EntryDate); err != nil {
			return JournalInput{}, errors.New("invalid_entry_date")
		}
	}

	cleaned := make([]string, 0, len(input.Tags))
	seen := map[string]bool{}
	for _, tag := range input.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if len([]rune(tag)) > maxTagLength {
			return JournalInput{}, errors.New("journal_tag_too_long")
		}
		lower := strings.ToLower(tag)
		if seen[lower] {
			continue
		}
		seen[lower] = true
		cleaned = append(cleaned, tag)
	}
	if len(cleaned) > maxJournalTags {
		return JournalInput{}, errors.New("too_many_tags")
	}
	input.Tags = cleaned

	return input, nil
}

// checkProductionContext validates an optional production link: the observer
// must actually hold an active membership in that production (Kernel 62
// §5.5 "optional context links must be authority-checked").
func checkProductionContext(ctx context.Context, pool *pgxpool.Pool, observerUserID, productionID string) error {
	if productionID == "" {
		return nil
	}
	var ok bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM memberships
			WHERE user_id = $1 AND production_id = $2 AND active
		)
	`, observerUserID, productionID).Scan(&ok); err != nil || !ok {
		return errors.New("production_not_found")
	}
	return nil
}

const journalColumns = `
	id::text, title, body,
	COALESCE(to_char(entry_date, 'YYYY-MM-DD'), '') AS entry_date,
	note_category, tags, COALESCE(production_id::text, ''),
	created_at, updated_at
`

// CreateJournalEntry stores one private dated note (Kernel 62 §9.3). No
// sharing, no notification, nothing reaches the subject.
func CreateJournalEntry(ctx context.Context, pool *pgxpool.Pool, observerUserID, relationshipID string, input JournalInput) (JournalEntry, error) {
	if _, err := loadRelationshipOwned(ctx, pool, observerUserID, relationshipID); err != nil {
		return JournalEntry{}, err
	}
	input, err := sanitizeJournalInput(input)
	if err != nil {
		return JournalEntry{}, err
	}
	if err := checkProductionContext(ctx, pool, observerUserID, input.ProductionID); err != nil {
		return JournalEntry{}, err
	}

	var entry JournalEntry
	if err := pool.QueryRow(ctx, `
		INSERT INTO player_relationship_journal_entries
			(relationship_id, title, body, entry_date, note_category, tags, production_id, created_by_user_id)
		VALUES ($1, $2, $3, NULLIF($4, '')::date, $5, $6, NULLIF($7, '')::uuid, $8)
		RETURNING `+journalColumns+`
	`, relationshipID, input.Title, input.Body, input.EntryDate, input.NoteCategory, input.Tags, input.ProductionID, observerUserID).Scan(
		&entry.ID, &entry.Title, &entry.Body, &entry.EntryDate,
		&entry.NoteCategory, &entry.Tags, &entry.ProductionID,
		&entry.CreatedAt, &entry.UpdatedAt,
	); err != nil {
		return JournalEntry{}, err
	}

	if err := touchRelationship(ctx, pool, relationshipID); err != nil {
		return JournalEntry{}, err
	}
	return entry, nil
}

// UpdateJournalEntry edits one of the observer's own entries (Kernel 62
// §7.6, §9.3).
func UpdateJournalEntry(ctx context.Context, pool *pgxpool.Pool, observerUserID, relationshipID, entryID string, input JournalInput) (JournalEntry, error) {
	if _, err := loadRelationshipOwned(ctx, pool, observerUserID, relationshipID); err != nil {
		return JournalEntry{}, err
	}
	input, err := sanitizeJournalInput(input)
	if err != nil {
		return JournalEntry{}, err
	}
	if err := checkProductionContext(ctx, pool, observerUserID, input.ProductionID); err != nil {
		return JournalEntry{}, err
	}

	var entry JournalEntry
	if err := pool.QueryRow(ctx, `
		UPDATE player_relationship_journal_entries
		SET title = $3, body = $4, entry_date = NULLIF($5, '')::date,
		    note_category = $6, tags = $7, production_id = NULLIF($8, '')::uuid,
		    updated_at = NOW()
		WHERE id = $1 AND relationship_id = $2 AND deleted_at IS NULL
		RETURNING `+journalColumns+`
	`, entryID, relationshipID, input.Title, input.Body, input.EntryDate, input.NoteCategory, input.Tags, input.ProductionID).Scan(
		&entry.ID, &entry.Title, &entry.Body, &entry.EntryDate,
		&entry.NoteCategory, &entry.Tags, &entry.ProductionID,
		&entry.CreatedAt, &entry.UpdatedAt,
	); err != nil {
		return JournalEntry{}, errors.New("entry_not_found")
	}

	if err := touchRelationship(ctx, pool, relationshipID); err != nil {
		return JournalEntry{}, err
	}
	return entry, nil
}

// DeleteJournalEntry soft-deletes one of the observer's own entries
// (Kernel 62 §4.6 allows journal-entry deletion; the relationship itself is
// only ever archived).
func DeleteJournalEntry(ctx context.Context, pool *pgxpool.Pool, observerUserID, relationshipID, entryID string) error {
	if _, err := loadRelationshipOwned(ctx, pool, observerUserID, relationshipID); err != nil {
		return err
	}

	tag, err := pool.Exec(ctx, `
		UPDATE player_relationship_journal_entries
		SET deleted_at = NOW()
		WHERE id = $1 AND relationship_id = $2 AND deleted_at IS NULL
	`, entryID, relationshipID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("entry_not_found")
	}
	return touchRelationship(ctx, pool, relationshipID)
}

// ListJournalEntries returns the observer's non-deleted entries, newest
// first by entry date then creation time (Kernel 62 §9.3). Category/tag
// filtering happens client-side over this private list.
func ListJournalEntries(ctx context.Context, pool *pgxpool.Pool, observerUserID, relationshipID string) ([]JournalEntry, error) {
	if _, err := loadRelationshipOwned(ctx, pool, observerUserID, relationshipID); err != nil {
		return nil, err
	}

	rows, err := pool.Query(ctx, `
		SELECT `+journalColumns+`
		FROM player_relationship_journal_entries
		WHERE relationship_id = $1 AND deleted_at IS NULL
		ORDER BY COALESCE(entry_date, created_at::date) DESC, created_at DESC
	`, relationshipID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []JournalEntry{}
	for rows.Next() {
		var entry JournalEntry
		if err := rows.Scan(
			&entry.ID, &entry.Title, &entry.Body, &entry.EntryDate,
			&entry.NoteCategory, &entry.Tags, &entry.ProductionID,
			&entry.CreatedAt, &entry.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}
