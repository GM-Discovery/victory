package showtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
)

type response struct {
	Ok   bool `json:"ok"`
	Data any  `json:"data,omitempty"`
}

func requireAuthenticatedUser(ctx context.Context, pool *pgxpool.Pool, r *http.Request) (string, error) {
	sessionCookie := ""
	if c, err := r.Cookie("victory_session"); err == nil {
		sessionCookie = c.Value
	}
	userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
	if err != nil || strings.TrimSpace(userID) == "" {
		return "", errors.New("not_authenticated")
	}
	return userID, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, response{Ok: true, Data: data})
}

func writeError(w http.ResponseWriter, err error) {
	code := err.Error()
	status := http.StatusBadRequest
	switch {
	case code == "not_authenticated":
		status = http.StatusUnauthorized
	case code == "not_authorized":
		status = http.StatusForbidden
	case strings.HasSuffix(code, "_not_found"):
		status = http.StatusNotFound
	case code == "show_code_ambiguous":
		status = http.StatusConflict
	}
	writeJSON(w, status, response{Ok: false, Data: map[string]any{"error": code}})
}

// resultPayload formats a StartResult/Status result into the response
// shape the frontend renders as spec §9.3's example block, plus structured
// fields for the busy/needs-venue-choice branches.
func resultPayload(result StartResult) map[string]any {
	if result.VenueBusyWithOtherShow != nil {
		return map[string]any{
			"venue_busy_with_other_show": map[string]any{
				"other_show_id":    result.VenueBusyWithOtherShow.OtherShowID,
				"other_show_title": result.VenueBusyWithOtherShow.OtherShowTitle,
			},
		}
	}
	if result.NeedsVenueChoice {
		return map[string]any{
			"needs_venue_choice": true,
			"candidate_venues":   result.CandidateVenues,
			"show_id":            result.ShowID,
			"show_title":         result.ShowTitle,
		}
	}

	message := fmt.Sprintf("Showtime started: %s [%s]\nVenue: %s\nMic: off", result.ShowTitle, result.ShortCode, orDash(result.VenueName, result.VenueSlug))
	if result.CurrentSceneName != "" {
		message = fmt.Sprintf("Showtime started: %s [%s]\nVenue: %s\nCurrent Scene: %s\nPlayers: %d ready, %d still need%s to choose a Character\nMic: off",
			result.ShowTitle, result.ShortCode, orDash(result.VenueName, result.VenueSlug),
			result.CurrentSceneName, result.PlayersReady, result.PlayersNeedCharacter, pluralSuffix(result.PlayersNeedCharacter))
	}
	message += "\n\nUse /mic hot to mirror or save live chat to the configured Discord channel."

	return map[string]any{
		"show_id":                result.ShowID,
		"show_title":             result.ShowTitle,
		"short_code":             result.ShortCode,
		"venue_slug":             result.VenueSlug,
		"venue_name":             result.VenueName,
		"current_scene_name":     result.CurrentSceneName,
		"players_ready":          result.PlayersReady,
		"players_need_character": result.PlayersNeedCharacter,
		"mic_on":                 result.MicOn,
		"session_id":             result.SessionID,
		"was_resumed":            result.WasResumed,
		"message":                message,
	}
}

func orDash(name, fallback string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return fallback
}

func pluralSuffix(n int) string {
	if n == 1 {
		return "s"
	}
	return ""
}

// HandleShowtimeControl handles POST /api/showtime/control, the bespoke
// endpoint the in-app /showtime legacy command (commands/registry.go) is
// bound to -- mirroring /api/session/control's existing shape exactly.
func HandleShowtimeControl(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}

		var body struct {
			ShortCode     string `json:"short_code"`
			Action        string `json:"action"`
			ForceReattach bool   `json:"force_reattach"`
			VenueSlug     string `json:"venue_slug"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}

		switch strings.ToLower(strings.TrimSpace(body.Action)) {
		case "", "start":
			result, err := Start(ctx, pool, userID, body.ShortCode, body.ForceReattach, body.VenueSlug)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, resultPayload(result))

		case "status":
			result, err := Status(ctx, pool, userID, body.ShortCode)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, resultPayload(result))

		case "end":
			result, err := End(ctx, pool, userID, body.ShortCode)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{
				"show_id":    result.ShowID,
				"show_title": result.ShowTitle,
				"session_id": result.SessionID,
				"message":    fmt.Sprintf("Showtime ended for %s. The Show may be resumed later.", result.ShowTitle),
			})

		default:
			writeError(w, errors.New("unknown_action"))
		}
	}
}
