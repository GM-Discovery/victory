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
	"victory/backend/internal/identity"
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
// shape the frontend renders, plus structured fields for the busy/needs-
// venue-choice/already-live branches.
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

	bridgeStatus := "not ready"
	if result.ChatBridgeOn {
		bridgeStatus = "connected"
	}

	var message string
	if result.AlreadyLive {
		message = fmt.Sprintf("You're already live. %s is running at %s.", result.ShowTitle, orDash(result.VenueName, result.VenueSlug))
	} else {
		message = fmt.Sprintf("Showtime! %s is live at %s. Chat Bridge: %s.", result.ShowTitle, orDash(result.VenueName, result.VenueSlug), bridgeStatus)
		if result.CurrentSceneName != "" {
			message = fmt.Sprintf("Showtime! %s is live at %s.\nScene: %s\nCast ready: %d · needs a Character: %d\nChat Bridge: %s.",
				result.ShowTitle, orDash(result.VenueName, result.VenueSlug),
				result.CurrentSceneName, result.PlayersReady, result.PlayersNeedCharacter, bridgeStatus)
		}
	}

	return map[string]any{
		"show_id":                result.ShowID,
		"show_title":             result.ShowTitle,
		"short_code":             result.ShortCode,
		"venue_slug":             result.VenueSlug,
		"venue_name":             result.VenueName,
		"current_scene_name":     result.CurrentSceneName,
		"players_ready":          result.PlayersReady,
		"players_need_character": result.PlayersNeedCharacter,
		"chat_bridge_on":         result.ChatBridgeOn,
		"chat_bridge_message":    result.ChatBridgeMessage,
		"session_id":             result.SessionID,
		"was_resumed":            result.WasResumed,
		"already_live":           result.AlreadyLive,
		"message":                message,
	}
}

func orDash(name, fallback string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return fallback
}

// HandleShowtimeControl handles POST /api/showtime/control, the bespoke
// endpoint the in-app /showtime legacy command (commands/registry.go) and
// the Kernel 92 Showtime popup both call -- one domain operation per
// action, multiple entry points (kernel doc §38).
func HandleShowtimeControl(pool *pgxpool.Pool, discordCfg identity.DiscordServerLinkConfig) http.HandlerFunc {
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
			ShowID        string `json:"show_id"`
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
			result, err := Start(ctx, pool, discordCfg, userID, body.ShortCode, body.ShowID, body.ForceReattach, body.VenueSlug)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, resultPayload(result))

		case "status":
			result, err := Status(ctx, pool, userID, body.ShortCode, body.ShowID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, resultPayload(result))

		case "preflight":
			result, err := Preflight(ctx, pool, userID, body.ShowID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, preflightPayload(result))

		case "end":
			result, err := End(ctx, pool, discordCfg, userID, body.ShortCode, body.ShowID)
			if err != nil {
				writeError(w, err)
				return
			}
			message := fmt.Sprintf("Showtime ended for %s. The Show may be resumed later.", result.ShowTitle)
			if result.AlreadyEnded {
				message = fmt.Sprintf("%s is already off stage.", result.ShowTitle)
			}
			writeOK(w, map[string]any{
				"show_id":                  result.ShowID,
				"show_title":               result.ShowTitle,
				"session_id":               result.SessionID,
				"already_ended":            result.AlreadyEnded,
				"aftercare_eligible_count": result.AftercareEligibleCount,
				"message":                  message,
			})

		default:
			writeError(w, errors.New("unknown_action"))
		}
	}
}

// preflightPayload formats a PreflightResult for the Kernel 92 Showtime
// popup's "Ready for Showtime" summary (kernel doc §12/§39).
func preflightPayload(result PreflightResult) map[string]any {
	return map[string]any{
		"show_id":                    result.ShowID,
		"show_title":                 result.ShowTitle,
		"nickname":                   result.Nickname,
		"short_code":                 result.ShortCode,
		"venue_slug":                 result.VenueSlug,
		"venue_name":                 result.VenueName,
		"current_scene_name":         result.CurrentSceneName,
		"cast_admitted_count":        result.CastAdmittedCount,
		"ticket_count":               result.TicketCount,
		"ready_character_count":      result.ReadyCharacterCount,
		"incomplete_character_count": result.IncompleteCharacterCount,
		"chat_bridge_ready":          result.ChatBridgeReady,
		"is_live":                    result.IsLive,
		"blockers":                   result.Blockers,
		"warnings":                   result.Warnings,
	}
}
