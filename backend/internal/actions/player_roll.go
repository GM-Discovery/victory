package actions

// Kernel 88 §11 §13.5: Player-initiated canonical rolls. Mirrors ic_chat.go's
// no-client-character_id pattern exactly -- PlayerMechanicRollRequest has no
// character field at all; the Character is always resolved server-side from
// the caller's current Show-Run roster selection (resolveSpeakerCharacter,
// shared with In Character chat). The skill rolled must belong to that
// resolved Character's own sheet (characters.ListCharacterSkills, re-checked
// here, never trusted from the client), and its dice expression comes from
// the skill's own ladder step (characters.StepExpression) -- Victory never
// invents a mechanic or infers an explosion; explode=true here is simply
// this codebase's existing "explode" default for skill checks (already used
// by characters/venue_sheet.go's own display of the same expression).
//
// If PrimaryActionID is set, this roll is understood to be resolving that
// pending primary action (socio's Interrupt/Help stack, §13): any overage
// banked by that action's resolved child interrupts is consumed
// (socio.ConsumeOverageFor) and applied as a flat "+N" situational modifier
// -- this is the Director-situational-modifier delivery mechanism the spec
// asks for in §11.3, sourced entirely from server-held state, never a
// client-supplied modifier value.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/characters"
	"victory/backend/internal/dice"
	"victory/backend/internal/rollaudience"
	"victory/backend/internal/showings"
)

type PlayerMechanicRollRequest struct {
	SessionID       string `json:"session_id"`
	ActorID         string `json:"actor_id"`
	RequestID       string `json:"request_id"`
	SkillID         string `json:"skill_id"`
	PrimaryActionID string `json:"primary_action_id"`
	Visibility      string `json:"visibility"`
	Label           string `json:"label"`
}

// loadOpenPrimaryActionActor and consumePendingActionOverage query
// socio_pending_actions (Kernel 88's Interrupt/Help stack, owned by
// backend/internal/socio) directly rather than importing that package --
// importing it here would close network -> actions -> socio -> cohorts ->
// network into an import cycle (cohorts/http.go imports network). This is
// the same narrow-duplicated-query tradeoff rollaudience.go's header
// comment documents for the same reason; socio remains the single writer
// of this table's other columns, these two queries only read/consume the
// overage a roll needs.
func loadOpenPrimaryActionActor(ctx context.Context, q actionQuerier, primaryActionID string) (string, error) {
	var kind, status, actorCharacterID string
	err := q.QueryRow(ctx, `
		SELECT kind, status, actor_character_card_id::text
		FROM socio_pending_actions
		WHERE id = $1
	`, primaryActionID).Scan(&kind, &status, &actorCharacterID)
	if err != nil {
		return "", err
	}
	if kind != "primary" {
		return "", errors.New("not_a_primary_action")
	}
	if status != "open" {
		return "", errors.New("primary_action_not_open")
	}
	return actorCharacterID, nil
}

func consumePendingActionOverage(ctx context.Context, pool *pgxpool.Pool, primaryActionID string) (int, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, overage_bonus FROM socio_pending_actions
		WHERE parent_id = $1 AND status = 'resolved' AND overage_consumed_at IS NULL AND overage_bonus > 0
	`, primaryActionID)
	if err != nil {
		return 0, err
	}
	var ids []string
	total := 0
	for rows.Next() {
		var id string
		var bonus int
		if err := rows.Scan(&id, &bonus); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
		total += bonus
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	if _, err := pool.Exec(ctx, `
		UPDATE socio_pending_actions SET overage_consumed_at = NOW() WHERE id = ANY($1::uuid[])
	`, ids); err != nil {
		return 0, err
	}
	return total, nil
}

// StorePlayerMechanicRoll resolves the caller's own selected Character and
// one of that Character's own canonical skills server-side, rolls it, and
// stores it as an ordinary 'roll/dice' action (payload.player_initiated =
// true, same downstream consumers as any other roll: SkillRolledThisSession,
// theatrical stage-effect projection, Story So Far) so it plays out on
// stage for the Audience the same way a Director-triggered roll does.
func StorePlayerMechanicRoll(ctx context.Context, pool *pgxpool.Pool, req PlayerMechanicRollRequest) (*StoredAction, error) {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ActorID = strings.TrimSpace(req.ActorID)
	req.RequestID = strings.TrimSpace(req.RequestID)
	req.SkillID = strings.TrimSpace(req.SkillID)
	req.PrimaryActionID = strings.TrimSpace(req.PrimaryActionID)
	req.Visibility = normalizeDiceVisibilityMode(req.Visibility)
	req.Label = strings.TrimSpace(req.Label)

	if req.SessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if req.ActorID == "" {
		return nil, errors.New("actor_id is required")
	}
	if req.RequestID == "" {
		return nil, errors.New("request_id is required")
	}
	if req.SkillID == "" {
		return nil, errors.New("skill_id is required")
	}
	if req.Visibility == "" {
		return nil, errors.New("unsupported_visibility_mode")
	}

	decision, err := CanAct(ctx, pool, req.ActorID, "roll/dice_own_mechanic", req.SessionID, ActionTarget{Kind: "session"})
	if err != nil {
		return nil, err
	}
	if !decision.Allowed {
		return nil, &ActionDeniedError{Reason: decision.Reason}
	}

	characterID, _, _, err := resolveSpeakerCharacter(ctx, pool, req.SessionID, req.ActorID)
	if err != nil {
		return nil, err
	}

	skills, err := characters.ListCharacterSkills(ctx, pool, req.ActorID, characterID)
	if err != nil {
		return nil, err
	}
	var skill *characters.CharacterSkill
	for i := range skills {
		if skills[i].SkillID == req.SkillID {
			skill = &skills[i]
			break
		}
	}
	if skill == nil {
		return nil, &ActionDeniedError{Reason: "forbidden"}
	}

	baseExpr, err := characters.StepExpression(skill.LadderStep, true)
	if err != nil {
		return nil, err
	}

	var primaryActionID string
	modifier := 0
	if req.PrimaryActionID != "" {
		actorCharacterID, err := loadOpenPrimaryActionActor(ctx, pool, req.PrimaryActionID)
		if err != nil {
			return nil, err
		}
		if actorCharacterID != characterID {
			return nil, &ActionDeniedError{Reason: "forbidden"}
		}
		modifier, err = consumePendingActionOverage(ctx, pool, req.PrimaryActionID)
		if err != nil {
			return nil, err
		}
		primaryActionID = req.PrimaryActionID
	}

	expression := baseExpr
	if modifier != 0 {
		expression += fmt.Sprintf("%+d", modifier)
	}

	spec, expression, err := dice.ParseExpression(expression)
	if err != nil {
		return nil, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	audience, err := rollaudience.Resolve(ctx, tx, req.SessionID, req.ActorID, req.Visibility)
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

	target := map[string]any{"kind": "session", "id": req.SessionID}
	payload := map[string]any{
		"request_id":       req.RequestID,
		"expression":       rolled.Expression,
		"spec":             rolled.Spec,
		"dice":             rolled.Dice,
		"explosion_count":  rolled.ExplosionCount,
		"modifier":         rolled.Modifier,
		"total":            rolled.Total,
		"roll_version":     rolled.RollVersion,
		"visibility_mode":  audience.Mode,
		"label":            req.Label,
		"actor_persona":    persona,
		"skill_id":         req.SkillID,
		"character_id":     characterID,
		"player_initiated": true,
	}
	if req.PrimaryActionID != "" {
		payload["primary_action_id"] = req.PrimaryActionID
		payload["situational_modifier"] = modifier
	}
	scope := map[string]any{"surfaces": []string{"dice"}, "audienceSegments": []string{"all"}}
	visibility := map[string]any{
		"toRoles":      rolesForAudienceMode(audience.Mode),
		"privateTo":    []string{},
		"audienceMode": audience.Mode,
		"cohortId":     audience.CohortID,
		"showId":       audience.ShowID,
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
		VALUES ($1, $2, $3, 'roll/dice', $4, $5, $6, $7, $8, TRUE)
		RETURNING id, ts
	`, req.SessionID, nextMoment, req.ActorID, targetJSON, payloadJSON, scopeJSON, visibilityJSON, showing.ID).
		Scan(&out.ID, &ts); err != nil {
		return nil, err
	}

	if primaryActionID != "" {
		if _, err := tx.Exec(ctx, `
			UPDATE socio_pending_actions
			SET status = 'resolved', roll_action_id = $2, roll_total = $3, resolved_at = NOW()
			WHERE id = $1
		`, primaryActionID, out.ID, rolled.Total); err != nil {
			return nil, err
		}
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
		"user_id": req.ActorID, "handle": handle, "display_name": displayName, "role": role, "persona": persona,
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
