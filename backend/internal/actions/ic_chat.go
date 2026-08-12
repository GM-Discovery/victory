package actions

// Kernel 87 §10: In Character chat. The server -- never the client --
// resolves which Character is speaking, using the same canonical source
// world/snapshot.go's resolveTheaterContext already treats as "has this
// Player chosen a Character": the Show-Run-scoped
// show_run_roster_members.character_card_id (Kernel 71). A client may
// send text; it may never send a character_id, and none is accepted here.

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showings"
)

// ErrNoCharacterSelected is returned when the acting user has no valid,
// non-deleted Character selected on their Show Run roster row -- the
// "choose a Character first" case (kernel §10).
var ErrNoCharacterSelected = errors.New("no_character_selected")

type ICChatMessageRequest struct {
	SessionID string `json:"session_id"`
	ActorID   string `json:"actor_id"`
	Text      string `json:"text"`
}

// resolveSpeakerCharacter mirrors world/snapshot.go's resolveTheaterContext
// roster lookup exactly (same table, same is_deleted fallback), duplicated
// here rather than imported to avoid pulling the much larger world package
// into actions' import graph for one lookup.
func resolveSpeakerCharacter(ctx context.Context, q actionQuerier, sessionID, userID string) (characterID, name, portraitURL string, err error) {
	var showID *string
	if err := q.QueryRow(ctx, `SELECT show_id::text FROM sessions WHERE id = $1`, sessionID).Scan(&showID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", "", ErrNoCharacterSelected
		}
		return "", "", "", err
	}
	if showID == nil || *showID == "" {
		return "", "", "", ErrNoCharacterSelected
	}

	var showRunID string
	if err := q.QueryRow(ctx, `SELECT show_run_id::text FROM shows WHERE id = $1`, *showID).Scan(&showRunID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", "", ErrNoCharacterSelected
		}
		return "", "", "", err
	}

	var (
		charID    *string
		charName  string
		portrait  string
		isDeleted *bool
	)
	err = q.QueryRow(ctx, `
		SELECT rm.character_card_id::text, COALESCE(cc.name, ''), COALESCE(cc.portrait_url, ''), cc.is_deleted
		FROM show_run_roster_members rm
		LEFT JOIN character_cards cc ON cc.id = rm.character_card_id
		WHERE rm.show_run_id = $1 AND rm.user_id = $2 AND rm.removed_at IS NULL
		LIMIT 1
	`, showRunID, userID).Scan(&charID, &charName, &portrait, &isDeleted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", "", ErrNoCharacterSelected
		}
		return "", "", "", err
	}
	if charID == nil || strings.TrimSpace(*charID) == "" {
		return "", "", "", ErrNoCharacterSelected
	}
	if isDeleted != nil && *isDeleted {
		return "", "", "", ErrNoCharacterSelected
	}
	return *charID, charName, portrait, nil
}

// StoreICChatMessage stores one In Character chat message. req.Text is the
// only client-authored content; the speaker Character is always resolved
// server-side from the caller's current Show-Run roster selection, never
// accepted from the request. Returns ErrNoCharacterSelected (wrapped) when
// the caller has not chosen a Character yet -- callers should present this
// as "choose a Character first," not a generic error.
func StoreICChatMessage(ctx context.Context, pool *pgxpool.Pool, req ICChatMessageRequest) (*StoredAction, error) {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ActorID = strings.TrimSpace(req.ActorID)
	req.Text = strings.TrimSpace(req.Text)

	if req.SessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if req.ActorID == "" {
		return nil, errors.New("actor_id is required")
	}
	if req.Text == "" {
		return nil, errors.New("text is required")
	}
	if utf8.RuneCountInString(req.Text) > 500 {
		return nil, errors.New("message_too_long")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	decision, err := CanAct(ctx, tx, req.ActorID, "chat/ic_message", req.SessionID, ActionTarget{Kind: "session"})
	if err != nil {
		return nil, err
	}
	if !decision.Allowed {
		return nil, &ActionDeniedError{Reason: decision.Reason}
	}

	characterID, characterName, portraitURL, err := resolveSpeakerCharacter(ctx, tx, req.SessionID, req.ActorID)
	if err != nil {
		return nil, err
	}

	showing, err := showings.EnsureForSession(ctx, tx, req.SessionID, req.ActorID)
	if err != nil {
		return nil, err
	}

	displayName, handle, role, persona, err := loadActorIdentity(ctx, tx, req.SessionID, req.ActorID)
	if err != nil {
		return nil, err
	}

	var nextMoment int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(moment_id), 0) + 1
		FROM actions
		WHERE session_id = $1
	`, req.SessionID).Scan(&nextMoment); err != nil {
		return nil, err
	}

	target := map[string]any{
		"kind": "session",
		"id":   req.SessionID,
	}
	// character_name/character_portrait_url are stamped into the payload
	// at send time so old messages preserve the speaker identity used
	// when sent, even if the Character is later renamed or the user
	// switches to a different Character (kernel §10 "Old messages
	// preserve the Character identity used when sent").
	payload := map[string]any{
		"text":               req.Text,
		"actor_persona":      persona,
		"character_id":       characterID,
		"character_name":     characterName,
		"character_portrait": portraitURL,
	}
	scope := map[string]any{
		"surfaces":         []string{"chat_ic"},
		"audienceSegments": []string{"all"},
	}
	visibility := map[string]any{
		"toRoles":   []string{"audience", "cast", "crew", "director", "producer"},
		"privateTo": []string{},
	}

	targetJSON, _ := json.Marshal(target)
	payloadJSON, _ := json.Marshal(payload)
	scopeJSON, _ := json.Marshal(scope)
	visibilityJSON, _ := json.Marshal(visibility)

	var out StoredAction
	var ts time.Time
	if err := tx.QueryRow(ctx, `
		INSERT INTO actions (
			session_id, moment_id, actor_id, type, target, payload, scope, visibility,
			showing_id, recorded
		)
		VALUES ($1, $2, $3, 'chat/ic_message', $4, $5, $6, $7, $8, TRUE)
		RETURNING id, ts
	`, req.SessionID, nextMoment, req.ActorID, targetJSON, payloadJSON, scopeJSON, visibilityJSON, showing.ID).
		Scan(&out.ID, &ts); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	out.SessionID = req.SessionID
	out.ShowingID = showing.ID
	out.MomentID = nextMoment
	out.ActorID = req.ActorID
	out.ActorDisplayName = displayName
	out.ActorHandle = handle
	out.ActorRole = role
	out.Actor = map[string]any{
		"user_id":      req.ActorID,
		"handle":       handle,
		"display_name": displayName,
		"role":         role,
		"persona":      persona,
	}
	out.Persona = persona
	out.Type = "chat/ic_message"
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}
