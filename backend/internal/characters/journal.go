package characters

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CharacterJournalEntry struct {
	ID               string `json:"id"`
	CharacterCardID  string `json:"character_card_id"`
	ModuleInstanceID string `json:"module_instance_id,omitempty"`
	AuthorUserID     string `json:"author_user_id"`
	Visibility       string `json:"visibility"`
	Body             string `json:"body"`
	VenueID          string `json:"venue_id,omitempty"`
	SessionID        string `json:"session_id,omitempty"`
	ShowingID        string `json:"showing_id,omitempty"`
	ArchivedAt       string `json:"archived_at,omitempty"`
	DeletedAt        string `json:"deleted_at,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type journalInput struct {
	ID         string `json:"id"`
	Body       string `json:"body"`
	Visibility string `json:"visibility"`
}

func HandleCharacterJournals(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireUser(ctx, pool, r)
		if err != nil {
			writeAuthError(w, err)
			return
		}

		switch r.Method {
		case http.MethodGet:
			entries, err := ListCharacterJournals(ctx, pool, userID)
			if err != nil {
				writeCharacterError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, response{Ok: true, Data: map[string]any{"entries": entries}})
		case http.MethodPost:
			var input journalInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": "invalid_json"}})
				return
			}
			entry, err := SaveCharacterJournal(ctx, pool, userID, input.Body, input.Visibility, "", "")
			if err != nil {
				writeCharacterError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, response{Ok: true, Data: entry})
		case http.MethodPatch:
			var input journalInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": "invalid_json"}})
				return
			}
			entry, err := UpdateCharacterJournal(ctx, pool, userID, input.ID, input.Body)
			if err != nil {
				writeCharacterError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, response{Ok: true, Data: entry})
		case http.MethodDelete:
			var input journalInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": "invalid_json"}})
				return
			}
			if err := ArchiveCharacterJournal(ctx, pool, userID, input.ID); err != nil {
				writeCharacterError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, response{Ok: true, Data: map[string]any{"id": input.ID}})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false, Data: map[string]any{"error": "method_not_allowed"}})
		}
	}
}

func SaveCharacterJournal(ctx context.Context, pool *pgxpool.Pool, authorUserID, body, visibility, venueID, sessionID string) (CharacterJournalEntry, error) {
	authorUserID = strings.TrimSpace(authorUserID)
	body = strings.TrimSpace(body)
	visibility = strings.TrimSpace(visibility)
	venueID = strings.TrimSpace(venueID)
	sessionID = strings.TrimSpace(sessionID)
	if authorUserID == "" {
		return CharacterJournalEntry{}, errors.New("not_authenticated")
	}
	if body == "" {
		return CharacterJournalEntry{}, errors.New("journal_body_required")
	}
	if visibility == "" {
		visibility = "private"
	}

	activeCharacter, err := ActiveCharacterForUser(ctx, pool, authorUserID)
	if err != nil {
		return CharacterJournalEntry{}, err
	}
	if activeCharacter == nil {
		return CharacterJournalEntry{}, errors.New("no_active_character")
	}

	characterID := strings.TrimSpace(stringValue(activeCharacter["character_card_id"]))
	if characterID == "" {
		return CharacterJournalEntry{}, errors.New("no_active_character")
	}

	moduleID, _ := firstModuleForCharacter(ctx, pool, characterID)

	var entry CharacterJournalEntry
	var createdAt, updatedAt time.Time
	var archivedAt, deletedAt *time.Time
	err = pool.QueryRow(ctx, `
		INSERT INTO character_journals (
			character_card_id,
			module_instance_id,
			author_user_id,
			visibility,
			body,
			venue_id,
			session_id
		)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, NULLIF($6, '')::uuid, NULLIF($7, '')::uuid)
		RETURNING id::text, character_card_id::text, COALESCE(module_instance_id::text, ''), author_user_id::text, visibility, body, COALESCE(venue_id::text, ''), COALESCE(session_id::text, ''), COALESCE(showing_id::text, ''), archived_at, deleted_at, created_at, updated_at
	`, characterID, moduleID, authorUserID, visibility, body, venueID, sessionID).Scan(&entry.ID, &entry.CharacterCardID, &entry.ModuleInstanceID, &entry.AuthorUserID, &entry.Visibility, &entry.Body, &entry.VenueID, &entry.SessionID, &entry.ShowingID, &archivedAt, &deletedAt, &createdAt, &updatedAt)
	if err != nil {
		return CharacterJournalEntry{}, err
	}

	if archivedAt != nil {
		entry.ArchivedAt = archivedAt.UTC().Format(time.RFC3339)
	}
	if deletedAt != nil {
		entry.DeletedAt = deletedAt.UTC().Format(time.RFC3339)
	}
	entry.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	entry.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return entry, nil
}

// UpdateCharacterJournal lets the author edit their own journal entry body.
// Journal edits never touch canonical character history.
func UpdateCharacterJournal(ctx context.Context, pool *pgxpool.Pool, authorUserID, entryID, body string) (CharacterJournalEntry, error) {
	authorUserID = strings.TrimSpace(authorUserID)
	entryID = strings.TrimSpace(entryID)
	body = strings.TrimSpace(body)
	if authorUserID == "" {
		return CharacterJournalEntry{}, errors.New("not_authenticated")
	}
	if entryID == "" {
		return CharacterJournalEntry{}, errors.New("journal_id_required")
	}
	if body == "" {
		return CharacterJournalEntry{}, errors.New("journal_body_required")
	}

	var entry CharacterJournalEntry
	var createdAt, updatedAt time.Time
	var archivedAt, deletedAt *time.Time
	err := pool.QueryRow(ctx, `
		UPDATE character_journals
		SET body = $3, updated_at = NOW()
		WHERE id = $1::uuid AND author_user_id = $2 AND deleted_at IS NULL
		RETURNING id::text, character_card_id::text, COALESCE(module_instance_id::text, ''), author_user_id::text, visibility, body, COALESCE(venue_id::text, ''), COALESCE(session_id::text, ''), COALESCE(showing_id::text, ''), archived_at, deleted_at, created_at, updated_at
	`, entryID, authorUserID, body).Scan(&entry.ID, &entry.CharacterCardID, &entry.ModuleInstanceID, &entry.AuthorUserID, &entry.Visibility, &entry.Body, &entry.VenueID, &entry.SessionID, &entry.ShowingID, &archivedAt, &deletedAt, &createdAt, &updatedAt)
	if err != nil {
		return CharacterJournalEntry{}, err
	}

	if archivedAt != nil {
		entry.ArchivedAt = archivedAt.UTC().Format(time.RFC3339)
	}
	if deletedAt != nil {
		entry.DeletedAt = deletedAt.UTC().Format(time.RFC3339)
	}
	entry.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	entry.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return entry, nil
}

// ArchiveCharacterJournal soft-deletes the author's own journal entry.
func ArchiveCharacterJournal(ctx context.Context, pool *pgxpool.Pool, authorUserID, entryID string) error {
	authorUserID = strings.TrimSpace(authorUserID)
	entryID = strings.TrimSpace(entryID)
	if authorUserID == "" {
		return errors.New("not_authenticated")
	}
	if entryID == "" {
		return errors.New("journal_id_required")
	}

	tag, err := pool.Exec(ctx, `
		UPDATE character_journals
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1::uuid AND author_user_id = $2 AND deleted_at IS NULL
	`, entryID, authorUserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func ListCharacterJournals(ctx context.Context, pool *pgxpool.Pool, authorUserID string) ([]CharacterJournalEntry, error) {
	activeCharacter, err := ActiveCharacterForUser(ctx, pool, authorUserID)
	if err != nil {
		return nil, err
	}
	if activeCharacter == nil {
		return []CharacterJournalEntry{}, nil
	}

	characterID := strings.TrimSpace(stringValue(activeCharacter["character_card_id"]))
	if characterID == "" {
		return []CharacterJournalEntry{}, nil
	}

	rows, err := pool.Query(ctx, `
		SELECT id::text, character_card_id::text, COALESCE(module_instance_id::text, ''), author_user_id::text, visibility, body, COALESCE(venue_id::text, ''), COALESCE(session_id::text, ''), COALESCE(showing_id::text, ''), archived_at, deleted_at, created_at, updated_at
		FROM character_journals
		WHERE character_card_id = $1
		  AND author_user_id = $2
		  AND deleted_at IS NULL
		ORDER BY created_at DESC
	`, characterID, authorUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []CharacterJournalEntry{}
	for rows.Next() {
		var entry CharacterJournalEntry
		var archivedAt, deletedAt *time.Time
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&entry.ID, &entry.CharacterCardID, &entry.ModuleInstanceID, &entry.AuthorUserID, &entry.Visibility, &entry.Body, &entry.VenueID, &entry.SessionID, &entry.ShowingID, &archivedAt, &deletedAt, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		if archivedAt != nil {
			entry.ArchivedAt = archivedAt.UTC().Format(time.RFC3339)
		}
		if deletedAt != nil {
			entry.DeletedAt = deletedAt.UTC().Format(time.RFC3339)
		}
		entry.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		entry.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		out = append(out, entry)
	}
	return out, rows.Err()
}

func firstModuleForCharacter(ctx context.Context, q characterQuerier, characterCardID string) (string, error) {
	var moduleID string
	err := q.QueryRow(ctx, `
		SELECT id::text
		FROM character_workbook_modules
		WHERE character_card_id = $1
		ORDER BY created_at ASC
		LIMIT 1
	`, characterCardID).Scan(&moduleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return moduleID, err
}
