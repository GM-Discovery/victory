package network

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"victory/backend/internal/access"
	"victory/backend/internal/actions"
	"victory/backend/internal/identity"

	"github.com/jackc/pgx/v5/pgxpool"
)

type indexCardSaveRequest struct {
	ActionType  string `json:"action_type"`
	ElementID   string `json:"element_id"`
	ElementSlug string `json:"element_slug"`
	FrontText   string `json:"front_text"`
	BackText    string `json:"back_text"`
	Color       string `json:"color"`
}

func HandleIndexCardSave(hub *Hub, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"ok":    false,
				"error": "method_not_allowed",
			})
			return
		}

		var req indexCardSaveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": "invalid_json",
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		sessionCookie := ""
		if c, err := r.Cookie("victory_session"); err == nil {
			sessionCookie = c.Value
		}

		userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":    false,
				"error": "not_authenticated",
			})
			return
		}

		allowed, err := access.UserCanAccessVenueSlug(ctx, pool, userID, "the-cave")
		if err != nil {
			log.Printf("index card save access check failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "access_check_failed",
			})
			return
		}
		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":    false,
				"error": "forbidden",
			})
			return
		}

		sessionID, err := identity.ResolveActiveCaveSessionID(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":    false,
				"error": "not_session_participant",
			})
			return
		}

		actionType := strings.TrimSpace(req.ActionType)
		if actionType == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": "missing_action_type",
			})
			return
		}

		var storedAction *actions.StoredAction
		switch actionType {
		case "create/index_card":
			storedAction, err = actions.StoreIndexCardCreate(ctx, pool, actions.IndexCardRequest{
				SessionID:   sessionID,
				ActorID:     userID,
				ElementID:   req.ElementID,
				ElementSlug: req.ElementSlug,
				FrontText:   req.FrontText,
				BackText:    req.BackText,
				Color:       req.Color,
			})
		case "update/index_card":
			storedAction, err = actions.StoreIndexCardUpdate(ctx, pool, actions.IndexCardRequest{
				SessionID:   sessionID,
				ActorID:     userID,
				ElementID:   req.ElementID,
				ElementSlug: req.ElementSlug,
				FrontText:   req.FrontText,
				BackText:    req.BackText,
				Color:       req.Color,
			})
		default:
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": "unknown_action",
			})
			return
		}
		if err != nil {
			var denied *actions.ActionDeniedError
			if errors.As(err, &denied) {
				writeJSON(w, http.StatusForbidden, map[string]any{
					"ok":    false,
					"error": denied.Reason,
				})
				return
			}
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": err.Error(),
			})
			return
		}

		msgOut, _ := json.Marshal(map[string]any{
			"type": "action",
			"data": storedAction,
		})
		hub.Broadcast(msgOut)

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"data": storedAction,
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
