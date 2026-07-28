package merchant

// Director-authored Character moments (kernel-75 S1.9, S6.4).
//
// S1.9 requires reusing the existing /journal system rather than building a
// second Director-note store, and the smallest compatible extension is a
// separate ROUTE over the same table -- not a widening of the Player's own
// journal handler, whose "your active Character only" rule is correct and
// should stay exactly as strict as it is.
//
// Lives in merchant rather than characters because the authority helpers and
// the writeOK/writeError envelope are here, and because characters must not
// grow a dependency on showruns.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
	"victory/backend/internal/storysofar"
)

// HandleDirectorCharacterJournal handles POST
// /api/characters/{character_card_id}/director-journal.
//
// Gate: showruns.CanManageShowRun at the CHARACTER'S Location. A Director at
// another Location is refused, and a Player cannot reach it at all.
//
// What it preserves (S6.4): the Director as author, the Character as target,
// Show/Session context, the text, a timestamp, and private-by-default state.
//
// What it deliberately cannot do: touch anything the Player wrote. This adds
// a row; it never updates one. UpdateCharacterJournal and
// ArchiveCharacterJournal remain author-scoped, so "Directors may not
// silently rewrite Player-authored material" holds because no route exists
// that could.
func HandleDirectorCharacterJournal(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		characterCardID := strings.TrimSpace(r.PathValue("character_card_id"))
		if characterCardID == "" {
			writeError(w, errors.New("no_character_selected"))
			return
		}

		var ownerUserID, locationID string
		if err := pool.QueryRow(ctx, `
			SELECT owner_user_id::text, location_id::text
			FROM character_cards WHERE id = $1 AND is_deleted = FALSE
		`, characterCardID).Scan(&ownerUserID, &locationID); err != nil {
			writeError(w, errors.New("character_card_not_found"))
			return
		}

		allowed, err := showruns.CanManageShowRun(ctx, pool, userID, locationID)
		if err != nil {
			writeError(w, err)
			return
		}
		if !allowed {
			writeError(w, errors.New("not_authorized"))
			return
		}

		var body struct {
			Body   string `json:"body"`
			Title  string `json:"title"`
			ShowID string `json:"show_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		text := strings.TrimSpace(body.Body)
		if text == "" {
			writeError(w, errors.New("journal_body_required"))
			return
		}

		showID := strings.TrimSpace(body.ShowID)
		var sessionID, showRunID, placementID string
		if showID != "" {
			// Best effort context. A Director writing a note about a Show
			// that has since ended still gets the note; the context columns
			// simply stay NULL.
			_ = pool.QueryRow(ctx, `
				SELECT s.show_run_id::text,
				       COALESCE(s.current_show_scene_placement_id::text, ''),
				       COALESCE((SELECT id::text FROM sessions WHERE show_id = s.id
				                 ORDER BY started_at DESC LIMIT 1), '')
				FROM shows s WHERE s.id = $1
			`, showID).Scan(&showRunID, &placementID, &sessionID)
		}

		var journalID string
		var createdAt time.Time
		if err := pool.QueryRow(ctx, `
			INSERT INTO character_journals (
				character_card_id, author_user_id, visibility, body,
				session_id, show_id, show_scene_placement_id, source
			)
			VALUES ($1, $2, 'private', $3, $4::uuid, $5::uuid, $6::uuid, 'director_journal')
			RETURNING id::text, created_at
		`, characterCardID, userID, text,
			attemptNullableID(sessionID), attemptNullableID(showID),
			attemptNullableID(placementID)).Scan(&journalID, &createdAt); err != nil {
			writeError(w, err)
			return
		}

		// Mirror into Story So Far so the moment appears in the Character's
		// history alongside generated events -- distinguishable by its
		// author, never merged anonymously into them (S1.8).
		title := strings.TrimSpace(body.Title)
		if title == "" {
			title = "A moment your Director recorded"
		}
		if err := storysofar.RecordAuthored(ctx, pool, storysofar.Ref{
			CharacterCardID: characterCardID,
			OwnerUserID:     ownerUserID,
			ShowRunID:       showRunID,
			ShowID:          showID,
			SessionID:       sessionID,
			PlacementID:     placementID,
		}, userID, storysofar.DraftEvent{
			EventType:  storysofar.EventDirectorMoment,
			Title:      title,
			Summary:    text,
			SourceKind: storysofar.SourceCharacterJournal,
			SourceRef:  journalID,
			OccurredAt: createdAt,
		}); err != nil {
			writeError(w, err)
			return
		}

		writeOK(w, map[string]any{
			"journal_id": journalID,
			"created_at": createdAt,
			"source":     "director_journal",
		})
	}
}
