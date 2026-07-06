package network

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/characters"
)

// HandleVenueCharacterSheet serves the compact, right-tray projection of the
// caller's active character (Kernel 60 §8). GET
// /api/characters/venue-sheet?session_id=
func HandleVenueCharacterSheet(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := currentCommandsUser(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
		cardID, noActive := resolveCardIDOrWriteError(ctx, w, pool, userID, sessionID)
		if noActive {
			return
		}

		sheet, err := characters.BuildVenueCharacterSheet(ctx, pool, userID, cardID)
		if err != nil {
			writeCommandsError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": map[string]any{"sheet": sheet}})
	}
}
