package characters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Chapter4GroupResult is returned to the client to render the ten-card
// sheet for the resolved attribute group.
type Chapter4GroupResult struct {
	ArchetypeKey      string          `json:"archetype_key"`
	ArchetypeTitle    string          `json:"archetype_title"`
	KeySkillID        string          `json:"key_skill_id"`
	KeySkillName      string          `json:"key_skill_name"`
	AttributeID       string          `json:"attribute_id"`
	AttributeName     string          `json:"attribute_name"`
	AttributeScore    int             `json:"attribute_score"`
	CurrentSkillCount int             `json:"current_skill_count"`
	Capacity          int             `json:"capacity"`
	Skills            []Chapter4Skill `json:"skills"`
}

// Chapter4Fact is the canonical, effective first-skill selection stored in
// workbook_context["chapter4"].
type Chapter4Fact struct {
	CatalogueVersion   string   `json:"catalogue_version"`
	SkillStableID      string   `json:"skill_stable_id"`
	SkillName          string   `json:"skill_name"`
	AttributeID        string   `json:"attribute_id"`
	AttributeName      string   `json:"attribute_name"`
	AcquisitionSource  string   `json:"acquisition_source"`
	TrainingState      string   `json:"training_state"`
	DieSize            string   `json:"die_size"`
	HelperIDs          []string `json:"helper_ids"`
	Confirmed          bool     `json:"confirmed"`
	ConfirmedAt        string   `json:"confirmed_at"`
	SelectionCount     int      `json:"selection_count"`
	OnboardingComplete bool     `json:"onboarding_complete"`
}

func loadChapter2Attributes(wbContext map[string]any) map[string]int {
	out := map[string]int{}
	raw, ok := wbContext["chapter2"].(map[string]any)
	if !ok {
		return out
	}
	attrs, ok := raw["attributes"].(map[string]any)
	if !ok {
		return out
	}
	for name, value := range attrs {
		if n, ok := parseCatharsisInt(value); ok {
			out[name] = n
		}
	}
	return out
}

func loadChapter4Fact(wbContext map[string]any) (Chapter4Fact, bool) {
	raw, ok := wbContext["chapter4"]
	if !ok {
		return Chapter4Fact{}, false
	}
	rawBytes, err := json.Marshal(raw)
	if err != nil {
		return Chapter4Fact{}, false
	}
	var fact Chapter4Fact
	if err := json.Unmarshal(rawBytes, &fact); err != nil {
		return Chapter4Fact{}, false
	}
	return fact, fact.Confirmed
}

// countSelectedSkillsForAttribute is a placeholder for future multi-skill
// tracking; Kernel 56 only ever selects one first skill, so the count is 0
// or 1 depending on whether a skill in this attribute is already confirmed.
func countSelectedSkillsForAttribute(wbContext map[string]any, attributeID string) int {
	fact, confirmed := loadChapter4Fact(wbContext)
	if confirmed && fact.AttributeID == attributeID {
		return 1
	}
	return 0
}

// ResolveChapter4Group resolves the authoritative ten-card skill group for
// the character's confirmed Chapter 3 archetype, routing by the key skill's
// governing attribute (the routing rule selected for this kernel -- see
// socio-skill-catalogue-audit-v0.1.md section 6, option B). This guarantees
// the key skill always appears in its own displayed group.
func ResolveChapter4Group(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID string) (Chapter4GroupResult, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	cardID = strings.TrimSpace(cardID)

	if actorUserID == "" {
		return Chapter4GroupResult{}, errors.New("not_authenticated")
	}
	if cardID == "" {
		return Chapter4GroupResult{}, errors.New("character_card_id_required")
	}

	allowed, err := CanEditCard(ctx, pool, actorUserID, cardID)
	if err != nil {
		return Chapter4GroupResult{}, err
	}
	if !allowed {
		return Chapter4GroupResult{}, errors.New("forbidden")
	}

	var wbContextRaw []byte
	if err := pool.QueryRow(ctx, `
		SELECT workbook_context FROM character_cards WHERE id = $1 AND is_deleted = FALSE
	`, cardID).Scan(&wbContextRaw); err != nil {
		return Chapter4GroupResult{}, err
	}
	wbContext := decodeJSONMap(wbContextRaw)

	if !chapter2Complete(wbContext) {
		return Chapter4GroupResult{}, errors.New("chapter2_not_complete")
	}
	archetypeFact, archetypeConfirmed := loadChapter3Fact(wbContext)
	if !archetypeConfirmed {
		return Chapter4GroupResult{}, errors.New("chapter3_not_complete")
	}

	keySkill, ok := Chapter4SkillByName(archetypeFact.KeySkill)
	if !ok {
		return Chapter4GroupResult{}, fmt.Errorf("key_skill_not_found")
	}

	group := Chapter4SkillsForAttribute(keySkill.AttributeID)
	if len(group) != 10 {
		return Chapter4GroupResult{}, errors.New("skill_group_invalid")
	}

	attribute, ok := Chapter4AttributeByName(keySkill.AttributeName)
	if !ok {
		return Chapter4GroupResult{}, errors.New("attribute_not_found")
	}

	attrTotals := loadChapter2Attributes(wbContext)
	attributeScore := attrTotals[attribute.Name]
	currentCount := countSelectedSkillsForAttribute(wbContext, attribute.ID)

	return Chapter4GroupResult{
		ArchetypeKey:      archetypeFact.ArchetypeKey,
		ArchetypeTitle:    archetypeFact.ArchetypeTitle,
		KeySkillID:        keySkill.ID,
		KeySkillName:      keySkill.Name,
		AttributeID:       attribute.ID,
		AttributeName:     attribute.Name,
		AttributeScore:    attributeScore,
		CurrentSkillCount: currentCount,
		Capacity:          attributeScore,
		Skills:            group,
	}, nil
}

// Chapter4SelectResult is returned after a first-skill confirmation.
type Chapter4SelectResult struct {
	SkillID            string `json:"skill_id"`
	SkillName          string `json:"skill_name"`
	AttributeName      string `json:"attribute_name"`
	DieSize            string `json:"die_size"`
	Completed          bool   `json:"completed"`
	OnboardingComplete bool   `json:"onboarding_complete"`
	Overridden         bool   `json:"overridden"`
}

// CommitChapter4FirstSkill validates and records the player's (or an
// authorized override actor's) confirmed first trained skill. This is the
// terminal event of onboarding: once confirmed, the character is locked --
// normal player flow cannot reopen Chapter 2, 3, or 4. Any further change
// must go through a Director/permitted-actor append-only override (the
// `/override` command surface is defined in a later kernel; this function
// accepts an `override` flag so that surface has something to call).
func CommitChapter4FirstSkill(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID, skillID string, override bool) (Chapter4SelectResult, map[string]any, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	cardID = strings.TrimSpace(cardID)
	skillID = strings.TrimSpace(skillID)

	if actorUserID == "" {
		return Chapter4SelectResult{}, nil, errors.New("not_authenticated")
	}
	if cardID == "" {
		return Chapter4SelectResult{}, nil, errors.New("character_card_id_required")
	}
	if skillID == "" {
		return Chapter4SelectResult{}, nil, errors.New("skill_id_required")
	}

	skill, ok := Chapter4SkillByID(skillID)
	if !ok {
		return Chapter4SelectResult{}, nil, errors.New("unknown_skill_id")
	}

	allowed, err := CanEditCard(ctx, pool, actorUserID, cardID)
	if err != nil {
		return Chapter4SelectResult{}, nil, err
	}
	if !allowed {
		return Chapter4SelectResult{}, nil, errors.New("forbidden")
	}

	var wbContextRaw []byte
	var cardName string
	if err := pool.QueryRow(ctx, `
		SELECT workbook_context, name FROM character_cards WHERE id = $1 AND is_deleted = FALSE
	`, cardID).Scan(&wbContextRaw, &cardName); err != nil {
		return Chapter4SelectResult{}, nil, err
	}
	wbContext := decodeJSONMap(wbContextRaw)

	if !chapter2Complete(wbContext) {
		return Chapter4SelectResult{}, nil, errors.New("chapter2_not_complete")
	}
	archetypeFact, archetypeConfirmed := loadChapter3Fact(wbContext)
	if !archetypeConfirmed {
		return Chapter4SelectResult{}, nil, errors.New("chapter3_not_complete")
	}

	existing, alreadyConfirmed := loadChapter4Fact(wbContext)
	if alreadyConfirmed && !override {
		return Chapter4SelectResult{
			SkillID:            existing.SkillStableID,
			SkillName:          existing.SkillName,
			AttributeName:      existing.AttributeName,
			DieSize:            existing.DieSize,
			Completed:          true,
			OnboardingComplete: existing.OnboardingComplete,
			Overridden:         false,
		}, wbContext, nil
	}

	keySkill, ok := Chapter4SkillByName(archetypeFact.KeySkill)
	if !ok {
		return Chapter4SelectResult{}, nil, errors.New("key_skill_not_found")
	}
	group := Chapter4SkillsForAttribute(keySkill.AttributeID)
	if len(group) != 10 {
		return Chapter4SelectResult{}, nil, errors.New("skill_group_invalid")
	}
	inGroup := false
	for _, s := range group {
		if s.ID == skill.ID {
			inGroup = true
			break
		}
	}
	if !inGroup {
		return Chapter4SelectResult{}, nil, errors.New("skill_not_in_group")
	}

	attribute, ok := Chapter4AttributeByName(skill.AttributeName)
	if !ok {
		return Chapter4SelectResult{}, nil, errors.New("attribute_not_found")
	}
	attrTotals := loadChapter2Attributes(wbContext)
	capacity := attrTotals[attribute.Name]
	currentCount := countSelectedSkillsForAttribute(wbContext, attribute.ID)
	if !override && currentCount >= capacity {
		return Chapter4SelectResult{}, nil, errors.New("attribute_capacity_full")
	}
	if !override && existing.SkillStableID == skill.ID && existing.Confirmed {
		return Chapter4SelectResult{}, nil, errors.New("skill_already_selected")
	}

	helperIDs := make([]string, 0, len(skill.Helpers))
	for _, h := range skill.Helpers {
		helperIDs = append(helperIDs, h.ID)
	}

	selectionCount := existing.SelectionCount + 1
	fact := Chapter4Fact{
		CatalogueVersion:   Chapter4SkillCatalogueVersion,
		SkillStableID:      skill.ID,
		SkillName:          skill.Name,
		AttributeID:        attribute.ID,
		AttributeName:      attribute.Name,
		AcquisitionSource:  "onboarding_first_skill",
		TrainingState:      "trained",
		DieSize:            "d6",
		HelperIDs:          helperIDs,
		Confirmed:          true,
		ConfirmedAt:        time.Now().UTC().Format(time.RFC3339),
		SelectionCount:     selectionCount,
		OnboardingComplete: true,
	}

	wbContext["chapter4"] = fact
	wbContext["current_stage"] = 5
	wbContext["current_event"] = "chapter4_first_skill_confirmed"
	wbContext["character_onboarding_completed"] = true
	wbContext["onboarding_completed_at"] = fact.ConfirmedAt
	wbContext["active_page"] = "face"

	payloadJSON, err := json.Marshal(wbContext)
	if err != nil {
		return Chapter4SelectResult{}, nil, err
	}
	if _, err := pool.Exec(ctx, `
		UPDATE character_cards SET workbook_context = $2::jsonb, workbook_status = 'complete', updated_at = NOW()
		WHERE id = $1 AND is_deleted = FALSE
	`, cardID, string(payloadJSON)); err != nil {
		return Chapter4SelectResult{}, nil, err
	}

	_, _ = pool.Exec(ctx, `
		UPDATE character_workbook_modules
		SET current_stage = 5, current_event = 'chapter4_first_skill_confirmed', module_status = 'complete', updated_at = NOW()
		WHERE character_card_id = $1 AND ruleset_key = 'socio'
	`, cardID)

	// Equip this character as the user's active persona (existing primitive,
	// same as used at character-card creation time).
	if err := setActiveCharacter(ctx, pool, actorUserID, cardID); err != nil {
		return Chapter4SelectResult{}, nil, err
	}

	displayName := strings.TrimSpace(cardName)
	if displayName == "" {
		displayName = "The character"
	}
	historyBody := fmt.Sprintf("%s selected %s as their first skill.", displayName, fact.SkillName)

	entryType := "first_skill_selection"
	if alreadyConfirmed && override {
		entryType = "first_skill_override"
	}

	if _, err := RecordWorkbookEvents(ctx, pool, actorUserID, WorkbookEventRequest{
		CharacterCardID: cardID,
		ModuleKey:       "socio",
		ModuleStatus:    "complete",
		CurrentStage:    5,
		CurrentEvent:    "chapter4_first_skill_confirmed",
		Entries: []WorkbookEventInput{
			{
				PageKey:     "history",
				EntryType:   entryType,
				Title:       "Chapter IV: First skill selected",
				Body:        historyBody,
				StageNumber: 4,
				SortOrder:   selectionCount,
				Payload: map[string]any{
					"source":          "catharsis",
					"ruleset_key":     "socio",
					"skill_stable_id": fact.SkillStableID,
					"attribute_id":    fact.AttributeID,
					"die_size":        fact.DieSize,
				},
			},
		},
	}); err != nil {
		return Chapter4SelectResult{}, nil, err
	}

	return Chapter4SelectResult{
		SkillID:            fact.SkillStableID,
		SkillName:          fact.SkillName,
		AttributeName:      fact.AttributeName,
		DieSize:            fact.DieSize,
		Completed:          true,
		OnboardingComplete: true,
		Overridden:         alreadyConfirmed && override,
	}, wbContext, nil
}
