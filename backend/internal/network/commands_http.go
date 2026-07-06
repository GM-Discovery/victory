package network

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
			handleCharExecute(ctx, w, hub, pool, userID, sessionID, args, req.IdempotencyKey)
			return

		case "bio":
			handleFieldSetExecute(ctx, w, hub, pool, userID, sessionID, "bio.set", args, req.IdempotencyKey, []string{"face", "bio"}, commands.ExecuteBioSet)
			return

		case "quote":
			handleFieldSetExecute(ctx, w, hub, pool, userID, sessionID, "quote.set", args, req.IdempotencyKey, []string{"face", "quote"}, commands.ExecuteQuoteSet)
			return

		case "journal":
			handleJournalExecute(ctx, w, pool, userID, args, req.IdempotencyKey)
			return

		default:
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "unknown_command"})
		}
	}
}

func handleCharExecute(ctx context.Context, w http.ResponseWriter, hub *Hub, pool *pgxpool.Pool, userID, sessionID string, args []string, idempotencyKey string) {
	if len(args) == 0 {
		args = []string{"face"}
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))

	switch sub {
	case "add":
		handleCharAddSkillExecute(ctx, w, hub, pool, userID, sessionID, args[1:], idempotencyKey)
		return
	case "skills":
		handleCharSkillsListExecute(ctx, w, pool, userID, sessionID)
		return
	case "advance":
		handleCharAdvanceExecute(ctx, w, hub, pool, userID, sessionID, args[1:])
		return
	}

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

	result, replayed, err := commands.ExecuteIdempotent(ctx, pool, userID, "char.set."+field, idempotencyKey, func(ctx context.Context) (map[string]any, error) {
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
	if !replayed {
		BroadcastCharacterProjectionInvalidation(ctx, hub, pool, cardID, []string{"face", "identity"}, "")
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": result})
}

// broadcastGameEvent mirrors the ooc/chat broadcast pattern for the Game
// Events lane: a nil event (mirror failed, or no session) is a no-op.
func broadcastGameEvent(hub *Hub, event *actions.StoredAction) {
	if event == nil {
		return
	}
	msgOut, _ := json.Marshal(map[string]any{"type": "action", "data": event})
	hub.Broadcast(msgOut)
}

// handleCharAddSkillExecute implements "/char add skill <name>" and
// "/char add skill --custom --name <n> --description <d> --attribute <a>"
// (Kernel 60 §6).
func handleCharAddSkillExecute(ctx context.Context, w http.ResponseWriter, hub *Hub, pool *pgxpool.Pool, userID, sessionID string, args []string, idempotencyKey string) {
	if len(args) == 0 || strings.ToLower(strings.TrimSpace(args[0])) != "skill" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_command_arguments"})
		return
	}

	name, custom, err := parseCharAddSkillArgs(args[1:])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	cardID, noActive := resolveCardIDOrWriteError(ctx, w, pool, userID, sessionID)
	if noActive {
		return
	}

	result, replayed, err := commands.ExecuteIdempotent(ctx, pool, userID, "char.add_skill", idempotencyKey, func(ctx context.Context) (map[string]any, error) {
		skill, event, err := commands.ExecuteCharAddSkill(ctx, pool, userID, sessionID, cardID, name, custom)
		if err != nil {
			return nil, err
		}
		out := map[string]any{"skill": skill}
		if event != nil {
			out["game_event"] = event
		}
		return out, nil
	})
	if err != nil {
		writeCommandsError(w, err)
		return
	}
	if !replayed {
		if event, ok := result["game_event"].(*actions.StoredAction); ok {
			broadcastGameEvent(hub, event)
		}
		BroadcastCharacterProjectionInvalidation(ctx, hub, pool, cardID, []string{"mechanics", "skills"}, "")
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": result})
}

// parseCharAddSkillArgs splits "<name...>" or "--custom --name <n>
// --description <d> --attribute <a>" (each flag's value runs until the next
// --flag token, so multi-word names/descriptions work without quoting).
func parseCharAddSkillArgs(args []string) (string, *characters.CustomSkillDetail, error) {
	if len(args) == 0 {
		return "", nil, errors.New("invalid_command_arguments")
	}
	if strings.ToLower(strings.TrimSpace(args[0])) != "--custom" {
		return strings.TrimSpace(strings.Join(args, " ")), nil, nil
	}

	flags := map[string][]string{}
	current := ""
	for _, tok := range args[1:] {
		lower := strings.ToLower(tok)
		if lower == "--name" || lower == "--description" || lower == "--attribute" {
			current = strings.TrimPrefix(lower, "--")
			continue
		}
		if current == "" {
			continue
		}
		flags[current] = append(flags[current], tok)
	}

	name := strings.TrimSpace(strings.Join(flags["name"], " "))
	description := strings.TrimSpace(strings.Join(flags["description"], " "))
	attribute := strings.TrimSpace(strings.Join(flags["attribute"], " "))
	if name == "" {
		return "", nil, errors.New("custom_skill_name_required")
	}
	if description == "" {
		return "", nil, errors.New("custom_skill_description_required")
	}
	if attribute == "" {
		return "", nil, errors.New("attribute_required")
	}

	return "", &characters.CustomSkillDetail{Name: name, Description: description, AttributeName: attribute}, nil
}

// handleCharSkillsListExecute implements "/char skills" -- a private,
// read-only reply, so it bypasses the idempotency wrapper entirely (Kernel
// 60 §6).
func handleCharSkillsListExecute(ctx context.Context, w http.ResponseWriter, pool *pgxpool.Pool, userID, sessionID string) {
	cardID, noActive := resolveCardIDOrWriteError(ctx, w, pool, userID, sessionID)
	if noActive {
		return
	}
	skills, err := commands.ExecuteCharSkillsList(ctx, pool, userID, cardID)
	if err != nil {
		writeCommandsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": map[string]any{"skills": skills}})
}

// handleCharAdvanceExecute implements "/char advance <skill>" (Kernel 60
// §7). The idempotency key is hardcoded to "attempt" under a
// skill+session-scoped command path -- "once per skill per session" is a
// server-enforced rule (§7 item 5, §11 item 4), not a client-optional
// dedupe, so it must not depend on the client supplying (or reusing) a key.
func handleCharAdvanceExecute(ctx context.Context, w http.ResponseWriter, hub *Hub, pool *pgxpool.Pool, userID, sessionID string, args []string) {
	name := strings.TrimSpace(strings.Join(args, " "))
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_command_arguments"})
		return
	}

	cardID, noActive := resolveCardIDOrWriteError(ctx, w, pool, userID, sessionID)
	if noActive {
		return
	}

	skill, err := characters.FindCharacterSkillByName(ctx, pool, userID, cardID, name)
	if err != nil {
		writeCommandsError(w, err)
		return
	}

	commandPath := fmt.Sprintf("char.advance.%s.%s", skill.SkillID, sessionID)
	result, replayed, err := commands.ExecuteIdempotent(ctx, pool, userID, commandPath, "attempt", func(ctx context.Context) (map[string]any, error) {
		outcome, event, err := commands.ExecuteCharAdvance(ctx, pool, userID, sessionID, cardID, skill.SkillName)
		if err != nil {
			return nil, err
		}
		out := map[string]any{"outcome": outcome}
		if event != nil {
			out["game_event"] = event
		}
		return out, nil
	})
	if err != nil {
		writeCommandsError(w, err)
		return
	}
	if !replayed {
		if event, ok := result["game_event"].(*actions.StoredAction); ok {
			broadcastGameEvent(hub, event)
		}
		BroadcastCharacterProjectionInvalidation(ctx, hub, pool, cardID, []string{"mechanics", "skills"}, "")
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": result})
}

// handleFieldSetExecute is shared by /bio set and /quote set: both replace a
// single character_cards column via a read-modify-write, resolve the active
// character the same way, and require the same "set <text...>" argument
// shape (subcommand + free-text tail).
func handleFieldSetExecute(ctx context.Context, w http.ResponseWriter, hub *Hub, pool *pgxpool.Pool, userID, sessionID, commandPath string, args []string, idempotencyKey string, changedDimensions []string, exec func(context.Context, *pgxpool.Pool, string, string, string) (characters.CharacterCard, error)) {
	if len(args) < 2 || strings.ToLower(strings.TrimSpace(args[0])) != "set" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_command_arguments"})
		return
	}
	text := strings.TrimSpace(strings.Join(args[1:], " "))

	cardID, noActive := resolveCardIDOrWriteError(ctx, w, pool, userID, sessionID)
	if noActive {
		return
	}

	result, replayed, err := commands.ExecuteIdempotent(ctx, pool, userID, commandPath, idempotencyKey, func(ctx context.Context) (map[string]any, error) {
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
	if !replayed {
		BroadcastCharacterProjectionInvalidation(ctx, hub, pool, cardID, changedDimensions, "")
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
