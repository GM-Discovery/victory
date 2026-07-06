package commands

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/actions"
	"victory/backend/internal/characters"
	"victory/backend/internal/dice"
)

// mirrorGameEvent stores a Game Event mirror of a canonical character
// mutation. Mirror failure never rolls back the mutation that already
// committed (Kernel 59 §16.4, Kernel 60 §9/§12) -- so this swallows its own
// error rather than propagating it to the caller.
func mirrorGameEvent(ctx context.Context, pool *pgxpool.Pool, sessionID, actorID, eventKind, sourceCharacterEventID, cardID string, detail map[string]any) *actions.StoredAction {
	if strings.TrimSpace(sessionID) == "" {
		return nil
	}
	event, err := actions.StoreGameEvent(ctx, pool, actions.GameEventRequest{
		SessionID:              sessionID,
		ActorID:                actorID,
		EventKind:              eventKind,
		SourceCharacterEventID: sourceCharacterEventID,
		CharacterCardID:        cardID,
		Detail:                 detail,
	})
	if err != nil {
		return nil
	}
	return event
}

// ExecuteCharAddSkill wraps characters.AddCharacterSkill and mirrors a
// character_skill_added Game Event (Kernel 60 §6, §9).
func ExecuteCharAddSkill(ctx context.Context, pool *pgxpool.Pool, actorUserID, sessionID, cardID, rawName string, custom *characters.CustomSkillDetail) (characters.CharacterSkill, *actions.StoredAction, error) {
	skill, historyEntryID, err := characters.AddCharacterSkill(ctx, pool, actorUserID, cardID, rawName, custom)
	if err != nil {
		return characters.CharacterSkill{}, nil, err
	}

	event := mirrorGameEvent(ctx, pool, sessionID, actorUserID, "character_skill_added", historyEntryID, cardID, map[string]any{
		"skill_id":       skill.SkillID,
		"skill_name":     skill.SkillName,
		"attribute_name": skill.AttributeName,
		"ladder_step":    skill.LadderStep,
	})

	return skill, event, nil
}

// ExecuteCharSkillsList wraps characters.ListCharacterSkills for /char skills.
func ExecuteCharSkillsList(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID string) ([]characters.CharacterSkill, error) {
	return characters.ListCharacterSkills(ctx, pool, actorUserID, cardID)
}

// ExecuteCharAdvance runs the full /char advance <skill> flow (Kernel 60 §7):
// resolve the skill, verify a skill-tagged roll happened this session,
// reject if the skill is already at the top of the ladder, roll the current
// pool without explosion, apply the roll-under-10 result, and mirror a
// character_skill_advanced/character_skill_advance_failed Game Event.
func ExecuteCharAdvance(ctx context.Context, pool *pgxpool.Pool, actorUserID, sessionID, cardID, rawName string) (characters.SkillAdvancementOutcome, *actions.StoredAction, error) {
	skill, err := characters.FindCharacterSkillByName(ctx, pool, actorUserID, cardID, rawName)
	if err != nil {
		return characters.SkillAdvancementOutcome{}, nil, err
	}

	if skill.LadderStep >= characters.SkillLadderMaxStep {
		return characters.SkillAdvancementOutcome{}, nil, errors.New("skill_at_ladder_cap")
	}

	used, err := actions.SkillRolledThisSession(ctx, pool, sessionID, actorUserID, skill.SkillID)
	if err != nil {
		return characters.SkillAdvancementOutcome{}, nil, err
	}
	if !used {
		return characters.SkillAdvancementOutcome{}, nil, errors.New("skill_not_used_this_session")
	}

	expression, err := characters.StepExpression(skill.LadderStep, false)
	if err != nil {
		return characters.SkillAdvancementOutcome{}, nil, err
	}
	rolled, err := dice.RollExpression(ctx, expression, dice.CryptoSource{})
	if err != nil {
		return characters.SkillAdvancementOutcome{}, nil, err
	}

	outcome, historyEntryID, err := characters.RecordSkillAdvancement(ctx, pool, actorUserID, cardID, skill.SkillID, expression, rolled.Total)
	if err != nil {
		return characters.SkillAdvancementOutcome{}, nil, err
	}

	eventKind := "character_skill_advance_failed"
	if outcome.Improved {
		eventKind = "character_skill_advanced"
	}
	event := mirrorGameEvent(ctx, pool, sessionID, actorUserID, eventKind, historyEntryID, cardID, map[string]any{
		"skill_id":   outcome.SkillID,
		"skill_name": outcome.SkillName,
		"expression": outcome.Expression,
		"total":      outcome.Total,
		"improved":   outcome.Improved,
		"old_step":   outcome.OldStep,
		"new_step":   outcome.NewStep,
	})

	return outcome, event, nil
}
