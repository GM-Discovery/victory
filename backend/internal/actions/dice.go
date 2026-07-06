package actions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dice"
	"victory/backend/internal/showings"
)

type DiceRollRequest struct {
	SessionID  string `json:"session_id"`
	ActorID    string `json:"actor_id"`
	RequestID  string `json:"request_id"`
	Expression string `json:"expression"`
	Visibility string `json:"visibility"`
	Label      string `json:"label"`
	SkillID    string `json:"skill_id"`
}

type diceRollLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	limit   int
	records map[string][]time.Time
}

func newDiceRollLimiter(window time.Duration, limit int) *diceRollLimiter {
	return &diceRollLimiter{
		window:  window,
		limit:   limit,
		records: make(map[string][]time.Time),
	}
}

func (l *diceRollLimiter) Allow(key string, now time.Time) bool {
	if l == nil {
		return true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := now.Add(-l.window)
	history := l.records[key][:0]
	for _, ts := range l.records[key] {
		if ts.After(cutoff) {
			history = append(history, ts)
		}
	}
	if len(history) >= l.limit {
		l.records[key] = history
		return false
	}
	l.records[key] = append(history, now)
	return true
}

var rollDiceLimiter = newDiceRollLimiter(10*time.Second, 8)

// SkillRolledThisSession reports whether actorID has a roll/dice action in
// sessionID whose payload carries this skillID (Kernel 60 §7: "untagged
// rolls do not count -- only the skill-tagged paths... count as verifiable
// use").
func SkillRolledThisSession(ctx context.Context, pool *pgxpool.Pool, sessionID, actorID, skillID string) (bool, error) {
	sessionID = strings.TrimSpace(sessionID)
	actorID = strings.TrimSpace(actorID)
	skillID = strings.TrimSpace(skillID)
	if sessionID == "" || actorID == "" || skillID == "" {
		return false, nil
	}

	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM actions
			WHERE session_id = $1
			  AND actor_id = $2
			  AND type = 'roll/dice'
			  AND payload ->> 'skill_id' = $3
		)
	`, sessionID, actorID, skillID).Scan(&exists)
	return exists, err
}

func normalizeDiceVisibilityMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "public":
		return "public"
	default:
		return ""
	}
}

func StoreDiceRoll(ctx context.Context, pool *pgxpool.Pool, req DiceRollRequest) (*StoredAction, error) {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ActorID = strings.TrimSpace(req.ActorID)
	req.RequestID = strings.TrimSpace(req.RequestID)
	req.Expression = strings.TrimSpace(req.Expression)
	req.Visibility = normalizeDiceVisibilityMode(req.Visibility)
	req.Label = strings.TrimSpace(req.Label)
	req.SkillID = strings.TrimSpace(req.SkillID)

	if req.SessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if req.ActorID == "" {
		return nil, errors.New("actor_id is required")
	}
	if req.RequestID == "" {
		return nil, errors.New("request_id is required")
	}
	if req.Expression == "" {
		return nil, errors.New("expression_required")
	}
	if len(req.Expression) > dice.MaxExpressionLength {
		return nil, errors.New("expression_too_long")
	}
	if req.Label != "" && len(req.Label) > 120 {
		return nil, errors.New("label_too_long")
	}
	if len(req.SkillID) > 120 {
		return nil, errors.New("skill_id_too_long")
	}
	if req.Visibility == "" {
		return nil, errors.New("unsupported_visibility_mode")
	}

	spec, expression, err := dice.ParseExpression(req.Expression)
	if err != nil {
		return nil, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	decision, err := CanAct(ctx, tx, req.ActorID, "roll/dice", req.SessionID, ActionTarget{Kind: "session"})
	if err != nil {
		return nil, err
	}
	if !decision.Allowed {
		return nil, &ActionDeniedError{Reason: decision.Reason}
	}

	showing, err := showings.EnsureForSession(ctx, tx, req.SessionID, req.ActorID)
	if err != nil {
		return nil, err
	}

	displayName, handle, role, persona, err := loadActorIdentity(ctx, tx, req.SessionID, req.ActorID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if !rollDiceLimiter.Allow(req.SessionID+":"+req.ActorID, now) {
		return nil, errors.New("roll_rate_limited")
	}

	rolled, err := dice.Roll(ctx, spec, dice.CryptoSource{})
	if err != nil {
		return nil, err
	}
	rolled.Expression = expression

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
	payload := map[string]any{
		"request_id":      req.RequestID,
		"expression":      rolled.Expression,
		"spec":            rolled.Spec,
		"dice":            rolled.Dice,
		"explosion_count": rolled.ExplosionCount,
		"modifier":        rolled.Modifier,
		"total":           rolled.Total,
		"roll_version":    rolled.RollVersion,
		"visibility_mode": req.Visibility,
		"label":           req.Label,
		"actor_persona":   persona,
	}
	if req.SkillID != "" {
		payload["skill_id"] = req.SkillID
	}
	scope := map[string]any{
		"surfaces":         []string{"dice"},
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
			session_id,
			moment_id,
			actor_id,
			type,
			target,
			payload,
			scope,
			visibility,
			showing_id,
			recorded
		)
		VALUES ($1, $2, $3, 'roll/dice', $4, $5, $6, $7, $8, TRUE)
		RETURNING id, ts
	`, req.SessionID, nextMoment, req.ActorID, targetJSON, payloadJSON, scopeJSON, visibilityJSON, showing.ID).Scan(&out.ID, &ts); err != nil {
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
	out.Type = "roll/dice"
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}
