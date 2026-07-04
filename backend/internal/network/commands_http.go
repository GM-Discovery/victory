package network

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/actions"
	"victory/backend/internal/characters"
	"victory/backend/internal/commands"
)

type commandsExecuteRequest struct {
	Path           string   `json:"path"`
	Args           []string `json:"args"`
	VenueSlug      string   `json:"venue_slug"`
	SessionID      string   `json:"session_id"`
	IdempotencyKey string   `json:"idempotency_key"`
}

// HandleCommandsAvailable serves the registry-driven catalogue the frontend
// command palette fetches and caches. GET /api/commands/available?venue=&session_id=
func HandleCommandsAvailable(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := currentCommandsUser(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		venueSlug := strings.TrimSpace(r.URL.Query().Get("venue"))
		sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))

		persona, source, err := commands.ResolveActiveCharacter(ctx, pool, userID, sessionID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "active_character_lookup_failed"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"data": map[string]any{
				"commands":         commands.Available(venueSlug),
				"active_character": persona,
				"active_source":    source,
			},
		})
	}
}

// HandleCommandsPreview validates a command's arguments and returns a
// non-mutating summary. Every command in this pass has RequiresPreview=false,
// so this never mints a confirmation token -- it exists so a future
// /value or /override kernel has one consistent call shape to extend with
// real token-minting, rather than inventing a second preview endpoint later.
func HandleCommandsPreview(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		var req commandsExecuteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_json"})
			return
		}

		cmd, ok := commands.Find(req.Path)
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "unknown_command"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"data": map[string]any{
				"requires_preview": false,
				"execute_directly": true,
				"summary":          cmd.Title + ": " + cmd.Usage,
				"legacy":           cmd.Legacy,
			},
		})
	}
}

// HandleCommandsExecute dispatches non-legacy commands. /mic and /session are
// explicitly rejected here (never proxied) -- clients must keep calling the
// existing bespoke endpoints for those, so Discord-bridge/session-control
// logic is never duplicated or risked.
func HandleCommandsExecute(hub *Hub, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		var req commandsExecuteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_json"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := currentCommandsUser(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		cmd, ok := commands.Find(req.Path)
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "unknown_command"})
			return
		}
		if cmd.Legacy {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":              false,
				"error":           "use_legacy_endpoint",
				"legacy_endpoint": cmd.LegacyEndpoint,
			})
			return
		}

		args := req.Args
		sessionID := strings.TrimSpace(req.SessionID)

		switch cmd.Path {
		case "help":
			topic := ""
			if len(args) > 0 {
				topic = args[0]
			}
			entries, err := commands.Help(topic)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": map[string]any{"entries": entries}})
			return

		case "ooc":
			text := strings.TrimSpace(strings.Join(args, " "))
			result, replayed, err := commands.ExecuteIdempotent(ctx, pool, userID, "ooc.send", req.IdempotencyKey, func(ctx context.Context) (map[string]any, error) {
				stored, err := commands.ExecuteOOC(ctx, pool, userID, sessionID, text)
				if err != nil {
					return nil, err
				}
				return map[string]any{"action": stored}, nil
			})
			if err != nil {
				writeCommandsError(w, err)
				return
			}
			if !replayed {
				if action, ok := result["action"]; ok {
					msgOut, _ := json.Marshal(map[string]any{"type": "action", "data": action})
					hub.Broadcast(msgOut)
				}
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": result})
			return

		case "char":
			handleCharExecute(ctx, w, pool, userID, sessionID, args, req.IdempotencyKey)
			return

		case "bio":
			handleFieldSetExecute(ctx, w, pool, userID, sessionID, "bio.set", args, req.IdempotencyKey, commands.ExecuteBioSet)
			return

		case "quote":
			handleFieldSetExecute(ctx, w, pool, userID, sessionID, "quote.set", args, req.IdempotencyKey, commands.ExecuteQuoteSet)
			return


		case "journal":
			handleJournalExecute(ctx, w, pool, userID, args, req.IdempotencyKey)
			return

		default:
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "unknown_command"})
		}
	}
}

func handleCharExecute(ctx context.Context, w http.ResponseWriter, pool *pgxpool.Pool, userID, sessionID string, args []string, idempotencyKey string) {
	if len(args) == 0 {
		args = []string{"face"}
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))
	if sub != "set" {
		page, err := commands.ExecuteCharNavigate(sub)
		if err != nil {
			writeCommandsError(w, err)
			return
		}
		cardID, noActive := resolveCardIDOrWriteError(ctx, w, pool, userID, sessionID)
		if noActive {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"data": map[string]any{
				"result_kind": "navigate",
				"destination": map[string]any{
					"surface":      "greenroom.character",
					"character_id": cardID,
					"page":         page,
				},
			},
		})
		return
	}

	if len(args) < 3 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_command_arguments"})
		return
	}
	field := strings.ToLower(strings.TrimSpace(args[1]))
	value := strings.TrimSpace(strings.Join(args[2:], " "))

	cardID, noActive := resolveCardIDOrWriteError(ctx, w, pool, userID, sessionID)
	if noActive {
		return
	}

	result, _, err := commands.ExecuteIdempotent(ctx, pool, userID, "char.set."+field, idempotencyKey, func(ctx context.Context) (map[string]any, error) {
		card, err := commands.ExecuteCharSet(ctx, pool, userID, cardID, field, value)
		if err != nil {
			return nil, err
		}
		return map[string]any{"character": card}, nil
	})
	if err != nil {
		writeCommandsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": result})
}

// handleFieldSetExecute is shared by /bio set and /quote set: both replace a
// single character_cards column via a read-modify-write, resolve the active
// character the same way, and require the same "set <text...>" argument
// shape (subcommand + free-text tail).
func handleFieldSetExecute(ctx context.Context, w http.ResponseWriter, pool *pgxpool.Pool, userID, sessionID, commandPath string, args []string, idempotencyKey string, exec func(context.Context, *pgxpool.Pool, string, string, string) (characters.CharacterCard, error)) {
	if len(args) < 2 || strings.ToLower(strings.TrimSpace(args[0])) != "set" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_command_arguments"})
		return
	}
	text := strings.TrimSpace(strings.Join(args[1:], " "))

	cardID, noActive := resolveCardIDOrWriteError(ctx, w, pool, userID, sessionID)
	if noActive {
		return
	}

	result, _, err := commands.ExecuteIdempotent(ctx, pool, userID, commandPath, idempotencyKey, func(ctx context.Context) (map[string]any, error) {
		card, err := exec(ctx, pool, userID, cardID, text)
		if err != nil {
			return nil, err
		}
		return map[string]any{"character": card}, nil
	})
	if err != nil {
		writeCommandsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": result})
}

func handleJournalExecute(ctx context.Context, w http.ResponseWriter, pool *pgxpool.Pool, userID string, args []string, idempotencyKey string) {
	if len(args) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_command_arguments"})
		return
	}
	sub := strings.ToLower(strings.TrimSpace(args[0]))
	switch sub {
	case "recent":
		entries, err := commands.ExecuteJournalRecent(ctx, pool, userID)
		if err != nil {
			writeCommandsError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": map[string]any{"entries": entries}})
	case "add":
		text := strings.TrimSpace(strings.Join(args[1:], " "))
		result, _, err := commands.ExecuteIdempotent(ctx, pool, userID, "journal.add", idempotencyKey, func(ctx context.Context) (map[string]any, error) {
			entry, err := commands.ExecuteJournalAdd(ctx, pool, userID, text, "", "")
			if err != nil {
				return nil, err
			}
			return map[string]any{"entry": entry}, nil
		})
		if err != nil {
			writeCommandsError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": result})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_command_arguments"})
	}
}

func resolveCardIDOrWriteError(ctx context.Context, w http.ResponseWriter, pool *pgxpool.Pool, userID, sessionID string) (string, bool) {
	persona, _, err := commands.ResolveActiveCharacter(ctx, pool, userID, sessionID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "active_character_lookup_failed"})
		return "", true
	}
	if persona == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "no_active_character"})
		return "", true
	}
	cardID, _ := persona["character_card_id"].(string)
	if strings.TrimSpace(cardID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "no_active_character"})
		return "", true
	}
	return cardID, false
}

func writeCommandsError(w http.ResponseWriter, err error) {
	var denied *actions.ActionDeniedError
	if errors.As(err, &denied) {
		writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": denied.Reason})
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
}

func currentCommandsUser(ctx context.Context, pool *pgxpool.Pool, r *http.Request) (string, error) {
	sessionCookie := ""
	if c, err := r.Cookie("victory_session"); err == nil {
		sessionCookie = c.Value
	}
	return access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
}
