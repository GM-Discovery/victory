package characters

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CharacterSkill mirrors a character_skills row (Kernel 60 §5). Every row
// this kernel writes is a full skill (IsHelper always false) -- the column
// and the half-unit capacity formula exist so a later kernel can populate
// real helper rows without a migration (Kernel 60 §5, §11 item 6).
type CharacterSkill struct {
	ID               string `json:"id"`
	CharacterCardID  string `json:"character_card_id"`
	SkillID          string `json:"skill_id"`
	SkillName        string `json:"skill_name"`
	AttributeName    string `json:"attribute_name"`
	LadderStep       int    `json:"ladder_step"`
	IsHelper         bool   `json:"is_helper"`
	Source           string `json:"source"`
	ImprovementCount int    `json:"improvement_count"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// CustomSkillDetail is the input for a player-authored skill (Kernel 60 §6
// /char add skill --custom).
type CustomSkillDetail struct {
	Name          string
	Description   string
	AttributeName string
}

func skillUnits(isHelper bool) int {
	if isHelper {
		return 1
	}
	return 2
}

// resolveSkillByNameOrPrefix looks up a catalogue skill by exact name, or
// falls back to an unambiguous case-insensitive prefix match. Kernel 60 §5:
// "exact or unambiguous-prefix match; ambiguous -> ambiguous_skill_name".
func resolveSkillByNameOrPrefix(rawName string) (Chapter4Skill, error) {
	name := strings.TrimSpace(rawName)
	if name == "" {
		return Chapter4Skill{}, errors.New("skill_name_required")
	}
	if skill, ok := Chapter4SkillByName(name); ok {
		return skill, nil
	}

	lowered := strings.ToLower(name)
	var matches []Chapter4Skill
	for _, s := range Chapter4Skills {
		if strings.HasPrefix(strings.ToLower(s.Name), lowered) {
			matches = append(matches, s)
		}
	}
	switch len(matches) {
	case 0:
		return Chapter4Skill{}, errors.New("unknown_skill")
	case 1:
		return matches[0], nil
	default:
		return Chapter4Skill{}, errors.New("ambiguous_skill_name")
	}
}

// ensureChapter4SkillBackfill inserts the Chapter-4 one-time first skill
// into character_skills if it isn't there yet (Kernel 60 §5: "backfilled...
// on first read... idempotent and never duplicates"). step 1 = d6, since the
// Chapter-4 skill is already trained.
func ensureChapter4SkillBackfill(ctx context.Context, q pgxQuerier, cardID string) error {
	var wbContextRaw []byte
	if err := q.QueryRow(ctx, `
		SELECT workbook_context FROM character_cards WHERE id = $1 AND is_deleted = FALSE
	`, cardID).Scan(&wbContextRaw); err != nil {
		return err
	}
	wbContext := decodeJSONMap(wbContextRaw)
	fact, confirmed := loadChapter4Fact(wbContext)
	if !confirmed {
		return nil
	}
	_, err := q.Exec(ctx, `
		INSERT INTO character_skills (character_card_id, skill_id, skill_name, attribute_name, ladder_step, is_helper, source)
		VALUES ($1, $2, $3, $4, 1, FALSE, 'chapter4_first')
		ON CONFLICT (character_card_id, skill_id) DO NOTHING
	`, cardID, fact.SkillStableID, fact.SkillName, fact.AttributeName)
	return err
}

// pgxQuerier is the subset of pgx.Tx/pgxpool.Pool this file needs; satisfied
// by both so backfill can run inside or outside a transaction.
type pgxQuerier interface {
	characterQuerier
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// ListCharacterSkills backfills the Chapter-4 first skill if needed, then
// returns the character's full skill list ordered by attribute then name.
func ListCharacterSkills(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID string) ([]CharacterSkill, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	cardID = strings.TrimSpace(cardID)
	if actorUserID == "" {
		return nil, errors.New("not_authenticated")
	}
	if cardID == "" {
		return nil, errors.New("character_card_id_required")
	}
	allowed, err := CanEditCard(ctx, pool, actorUserID, cardID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errors.New("forbidden")
	}

	if err := ensureChapter4SkillBackfill(ctx, pool, cardID); err != nil {
		return nil, err
	}

	rows, err := pool.Query(ctx, `
		SELECT id::text, character_card_id::text, skill_id, skill_name, attribute_name, ladder_step, is_helper, source, improvement_count, created_at, updated_at
		FROM character_skills
		WHERE character_card_id = $1
		ORDER BY attribute_name, skill_name
	`, cardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []CharacterSkill{}
	for rows.Next() {
		var s CharacterSkill
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&s.ID, &s.CharacterCardID, &s.SkillID, &s.SkillName, &s.AttributeName, &s.LadderStep, &s.IsHelper, &s.Source, &s.ImprovementCount, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		s.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		s.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		out = append(out, s)
	}
	return out, rows.Err()
}

// AddCharacterSkill adds a catalogue or custom skill to the character at
// ladder step 0 (d4), enforcing half-slot capacity (Kernel 60 §5, §6). It
// returns the new skill row and the ID of the History entry it recorded, so
// the caller can link a Game Event mirror to it via source_character_event_id
// (Kernel 60 §9).
func AddCharacterSkill(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID, rawName string, custom *CustomSkillDetail) (CharacterSkill, string, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	cardID = strings.TrimSpace(cardID)
	if actorUserID == "" {
		return CharacterSkill{}, "", errors.New("not_authenticated")
	}
	if cardID == "" {
		return CharacterSkill{}, "", errors.New("character_card_id_required")
	}

	allowed, err := CanEditCard(ctx, pool, actorUserID, cardID)
	if err != nil {
		return CharacterSkill{}, "", err
	}
	if !allowed {
		return CharacterSkill{}, "", errors.New("forbidden")
	}

	var skillID, skillName, attributeName string
	if custom != nil {
		name := strings.TrimSpace(custom.Name)
		desc := strings.TrimSpace(custom.Description)
		attrName := strings.TrimSpace(custom.AttributeName)
		if name == "" {
			return CharacterSkill{}, "", errors.New("custom_skill_name_required")
		}
		if desc == "" {
			return CharacterSkill{}, "", errors.New("custom_skill_description_required")
		}
		attr, ok := Chapter4AttributeByName(attrName)
		if !ok {
			return CharacterSkill{}, "", errors.New("attribute_not_found")
		}
		skillID = customStableID("skill")
		skillName = name
		attributeName = attr.Name
	} else {
		skill, err := resolveSkillByNameOrPrefix(rawName)
		if err != nil {
			return CharacterSkill{}, "", err
		}
		skillID = skill.ID
		skillName = skill.Name
		attributeName = skill.AttributeName
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return CharacterSkill{}, "", err
	}
	defer tx.Rollback(ctx)

	if err := ensureChapter4SkillBackfill(ctx, tx, cardID); err != nil {
		return CharacterSkill{}, "", err
	}

	// Serialize concurrent adds against the same character+attribute so the
	// capacity check below can't race (Kernel 60 §5: "concurrency: capacity
	// check inside the insert transaction").
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, cardID+"|"+attributeName); err != nil {
		return CharacterSkill{}, "", err
	}

	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM character_skills WHERE character_card_id = $1 AND skill_id = $2)
	`, cardID, skillID).Scan(&exists); err != nil {
		return CharacterSkill{}, "", err
	}
	if exists {
		return CharacterSkill{}, "", errors.New("skill_already_known")
	}

	var wbContextRaw []byte
	var cardName string
	if err := tx.QueryRow(ctx, `
		SELECT workbook_context, name FROM character_cards WHERE id = $1 AND is_deleted = FALSE
	`, cardID).Scan(&wbContextRaw, &cardName); err != nil {
		return CharacterSkill{}, "", err
	}
	attrTotals := loadChapter2Attributes(decodeJSONMap(wbContextRaw))
	attributeScore := attrTotals[attributeName]
	capacityUnits := attributeScore * 2

	var usedUnits int
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN is_helper THEN 1 ELSE 2 END), 0)
		FROM character_skills
		WHERE character_card_id = $1 AND attribute_name = $2
	`, cardID, attributeName).Scan(&usedUnits); err != nil {
		return CharacterSkill{}, "", err
	}
	if usedUnits+skillUnits(false) > capacityUnits {
		return CharacterSkill{}, "", errors.New("skill_capacity_reached")
	}

	var out CharacterSkill
	var createdAt, updatedAt time.Time
	if err := tx.QueryRow(ctx, `
		INSERT INTO character_skills (character_card_id, skill_id, skill_name, attribute_name, ladder_step, is_helper, source)
		VALUES ($1, $2, $3, $4, 0, FALSE, 'player_added')
		RETURNING id::text, character_card_id::text, skill_id, skill_name, attribute_name, ladder_step, is_helper, source, improvement_count, created_at, updated_at
	`, cardID, skillID, skillName, attributeName).Scan(
		&out.ID, &out.CharacterCardID, &out.SkillID, &out.SkillName, &out.AttributeName,
		&out.LadderStep, &out.IsHelper, &out.Source, &out.ImprovementCount, &createdAt, &updatedAt,
	); err != nil {
		return CharacterSkill{}, "", err
	}
	out.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	out.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)

	if err := tx.Commit(ctx); err != nil {
		return CharacterSkill{}, "", err
	}

	displayName := strings.TrimSpace(cardName)
	if displayName == "" {
		displayName = "The character"
	}
	displayExpr, _ := StepExpression(0, true)
	historyResult, err := RecordWorkbookEvents(ctx, pool, actorUserID, WorkbookEventRequest{
		CharacterCardID: cardID,
		Entries: []WorkbookEventInput{
			{
				PageKey:   "history",
				EntryType: "character_skill_added",
				Title:     "Skill added",
				Body:      fmt.Sprintf("%s trained %s (%s).", displayName, out.SkillName, displayExpr),
				Payload: map[string]any{
					"skill_id":       out.SkillID,
					"skill_name":     out.SkillName,
					"attribute_name": out.AttributeName,
					"ladder_step":    out.LadderStep,
				},
			},
		},
	})
	if err != nil {
		return CharacterSkill{}, "", err
	}

	historyEntryID := ""
	if entries, ok := historyResult["entries"].([]CharacterWorkbookEntry); ok && len(entries) > 0 {
		historyEntryID = entries[0].ID
	}

	return out, historyEntryID, nil
}

// FindCharacterSkillByName resolves one of the character's own equipped
// skills by exact or unambiguous-prefix name match (Kernel 60 §7 -- /char
// advance <skill> targets a skill the character already has, not the full
// catalogue).
func FindCharacterSkillByName(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID, rawName string) (CharacterSkill, error) {
	name := strings.TrimSpace(rawName)
	if name == "" {
		return CharacterSkill{}, errors.New("skill_name_required")
	}

	skills, err := ListCharacterSkills(ctx, pool, actorUserID, cardID)
	if err != nil {
		return CharacterSkill{}, err
	}

	for _, s := range skills {
		if strings.EqualFold(s.SkillName, name) {
			return s, nil
		}
	}

	lowered := strings.ToLower(name)
	var matches []CharacterSkill
	for _, s := range skills {
		if strings.HasPrefix(strings.ToLower(s.SkillName), lowered) {
			matches = append(matches, s)
		}
	}
	switch len(matches) {
	case 0:
		return CharacterSkill{}, errors.New("unknown_skill")
	case 1:
		return matches[0], nil
	default:
		return CharacterSkill{}, errors.New("ambiguous_skill_name")
	}
}

// SkillAdvancementOutcome is the result of a single /char advance attempt
// (Kernel 60 §7).
type SkillAdvancementOutcome struct {
	SkillID    string `json:"skill_id"`
	SkillName  string `json:"skill_name"`
	Expression string `json:"expression"`
	Total      int    `json:"total"`
	Improved   bool   `json:"improved"`
	OldStep    int    `json:"old_step"`
	NewStep    int    `json:"new_step"`
}

// RecordSkillAdvancement applies the roll-under-10 result (Kernel 60 §7):
// total < 10 steps the skill up one rung and bumps improvement_count;
// either way it records a History entry (success and failure are both
// public table moments, per §9). Callers must check skill_at_ladder_cap
// (via the skill's current ladder_step) before rolling -- rolling an
// already-capped skill is pointless -- but this function re-checks under
// FOR UPDATE so a concurrent advance can't push a skill past the cap.
func RecordSkillAdvancement(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID, skillID, expression string, total int) (SkillAdvancementOutcome, string, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	cardID = strings.TrimSpace(cardID)
	skillID = strings.TrimSpace(skillID)
	if actorUserID == "" {
		return SkillAdvancementOutcome{}, "", errors.New("not_authenticated")
	}
	if cardID == "" {
		return SkillAdvancementOutcome{}, "", errors.New("character_card_id_required")
	}
	if skillID == "" {
		return SkillAdvancementOutcome{}, "", errors.New("unknown_skill")
	}

	allowed, err := CanEditCard(ctx, pool, actorUserID, cardID)
	if err != nil {
		return SkillAdvancementOutcome{}, "", err
	}
	if !allowed {
		return SkillAdvancementOutcome{}, "", errors.New("forbidden")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return SkillAdvancementOutcome{}, "", err
	}
	defer tx.Rollback(ctx)

	var skillName string
	var oldStep int
	if err := tx.QueryRow(ctx, `
		SELECT skill_name, ladder_step FROM character_skills
		WHERE character_card_id = $1 AND skill_id = $2
		FOR UPDATE
	`, cardID, skillID).Scan(&skillName, &oldStep); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SkillAdvancementOutcome{}, "", errors.New("unknown_skill")
		}
		return SkillAdvancementOutcome{}, "", err
	}
	if oldStep >= SkillLadderMaxStep {
		return SkillAdvancementOutcome{}, "", errors.New("skill_at_ladder_cap")
	}

	improved := total < 10
	newStep := oldStep
	if improved {
		newStep = oldStep + 1
		if _, err := tx.Exec(ctx, `
			UPDATE character_skills
			SET ladder_step = $3, improvement_count = improvement_count + 1, updated_at = NOW()
			WHERE character_card_id = $1 AND skill_id = $2
		`, cardID, skillID, newStep); err != nil {
			return SkillAdvancementOutcome{}, "", err
		}
	}

	var cardName string
	if err := tx.QueryRow(ctx, `
		SELECT name FROM character_cards WHERE id = $1 AND is_deleted = FALSE
	`, cardID).Scan(&cardName); err != nil {
		return SkillAdvancementOutcome{}, "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return SkillAdvancementOutcome{}, "", err
	}

	displayName := strings.TrimSpace(cardName)
	if displayName == "" {
		displayName = "The character"
	}

	entryType := "character_skill_advance_failed"
	body := fmt.Sprintf("%s tried to advance %s (rolled %d on %s) -- no improvement.", displayName, skillName, total, expression)
	if improved {
		newExpr, _ := StepExpression(newStep, true)
		entryType = "character_skill_advanced"
		body = fmt.Sprintf("%s rolled %d on %s -- %s improves to %s!", displayName, total, expression, skillName, newExpr)
	}

	historyResult, err := RecordWorkbookEvents(ctx, pool, actorUserID, WorkbookEventRequest{
		CharacterCardID: cardID,
		Entries: []WorkbookEventInput{
			{
				PageKey:   "history",
				EntryType: entryType,
				Title:     "Skill advancement attempt",
				Body:      body,
				Payload: map[string]any{
					"skill_id":   skillID,
					"skill_name": skillName,
					"expression": expression,
					"total":      total,
					"improved":   improved,
					"old_step":   oldStep,
					"new_step":   newStep,
				},
			},
		},
	})
	if err != nil {
		return SkillAdvancementOutcome{}, "", err
	}
	historyEntryID := ""
	if entries, ok := historyResult["entries"].([]CharacterWorkbookEntry); ok && len(entries) > 0 {
		historyEntryID = entries[0].ID
	}

	return SkillAdvancementOutcome{
		SkillID:    skillID,
		SkillName:  skillName,
		Expression: expression,
		Total:      total,
		Improved:   improved,
		OldStep:    oldStep,
		NewStep:    newStep,
	}, historyEntryID, nil
}
