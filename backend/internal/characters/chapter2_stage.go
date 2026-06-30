package characters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Chapter2StageInput is the client-supplied stage completion payload.
type Chapter2StageInput struct {
	CharacterCardID    string         `json:"character_card_id"`
	StageNumber        int            `json:"stage_number"`
	UseEnhancement     bool           `json:"use_enhancement"`
	BonusChoiceID      string         `json:"bonus_choice_id"`
	CompanionPurchased bool           `json:"companion_purchased"`
	CompanionTarget    string         `json:"companion_target"`
	TraitIDs           []string       `json:"trait_ids"`
	Allocation         map[string]int `json:"allocation"`           // Stage 9 only
	RepresentationText string         `json:"representation_text"`  // Stage 2 only
	DirectorFateAdjust int            `json:"director_fate_adjust"` // optional Director ± adjustment
}

// Chapter2StageResult is returned after a stage is committed.
type Chapter2StageResult struct {
	StageNumber     int            `json:"stage_number"`
	FinalRoll       int            `json:"final_roll"`
	Enhancement     bool           `json:"enhancement_used"`
	BonusChoiceID   string         `json:"bonus_choice_id"`
	AttributeDeltas map[string]int `json:"attribute_deltas"`
	FPSpent         int            `json:"fp_spent"`
	FPBalance       int            `json:"fp_balance"`
	Completed       bool           `json:"completed"`
	NextStage       int            `json:"next_stage"` // 0 when chapter complete
}

// chapter2State is the in-workbook JSON structure for Chapter 2 progress.
type chapter2State struct {
	Version      string                   `json:"version"`
	StartingFP   int                      `json:"starting_fp"`
	FPBalance    int                      `json:"fp_balance"`
	CurrentStage int                      `json:"current_stage"`
	Attributes   map[string]int           `json:"attributes"`
	Stages       map[string]chapter2Stage `json:"stages"`
}

type chapter2Stage struct {
	FinalRoll          int            `json:"final_roll"`
	EnhancementUsed    bool           `json:"enhancement_used"`
	BonusChoiceID      string         `json:"bonus_choice_id"`
	CompanionPurchased bool           `json:"companion_purchased"`
	CompanionTarget    string         `json:"companion_target,omitempty"`
	TraitIDs           []string       `json:"trait_ids"`
	Allocation         map[string]int `json:"allocation,omitempty"`
	RepresentationText string         `json:"representation_text,omitempty"`
	FPSpent            int            `json:"fp_spent"`
	Completed          bool           `json:"completed"`
}

func loadChapter2State(workbookContext map[string]any) chapter2State {
	rawBytes, _ := json.Marshal(workbookContext["chapter2"])
	var s chapter2State
	if err := json.Unmarshal(rawBytes, &s); err != nil || s.StartingFP == 0 {
		s = chapter2State{
			Version:      Chapter2Version,
			StartingFP:   Chapter2StartingFP,
			FPBalance:    Chapter2StartingFP,
			CurrentStage: 1,
			Attributes:   make(map[string]int),
			Stages:       make(map[string]chapter2Stage),
		}
		for _, attr := range AllChapter2Attributes {
			s.Attributes[attr] = 0
		}
	}
	if s.Stages == nil {
		s.Stages = make(map[string]chapter2Stage)
	}
	if s.Attributes == nil {
		s.Attributes = make(map[string]int)
		for _, attr := range AllChapter2Attributes {
			s.Attributes[attr] = 0
		}
	}
	return s
}

// CommitChapter2Stage validates and records a completed life stage.
func CommitChapter2Stage(ctx context.Context, pool *pgxpool.Pool, actorUserID string, input Chapter2StageInput) (Chapter2StageResult, map[string]any, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	cardID := strings.TrimSpace(input.CharacterCardID)

	if actorUserID == "" {
		return Chapter2StageResult{}, nil, errors.New("not_authenticated")
	}
	if cardID == "" {
		return Chapter2StageResult{}, nil, errors.New("character_card_id_required")
	}
	if input.StageNumber < 1 || input.StageNumber > 10 {
		return Chapter2StageResult{}, nil, errors.New("invalid_stage_number")
	}

	allowed, err := CanEditCard(ctx, pool, actorUserID, cardID)
	if err != nil {
		return Chapter2StageResult{}, nil, err
	}
	if !allowed {
		return Chapter2StageResult{}, nil, errors.New("forbidden")
	}

	// Load current workbook context.
	var wbContextRaw []byte
	err = pool.QueryRow(ctx, `SELECT workbook_context FROM character_cards WHERE id = $1 AND is_deleted = FALSE`, cardID).Scan(&wbContextRaw)
	if err != nil {
		return Chapter2StageResult{}, nil, err
	}
	wbContext := decodeJSONMap(wbContextRaw)
	ch2 := loadChapter2State(wbContext)

	stageKey := fmt.Sprintf("%d", input.StageNumber)

	// Stage must not already be completed.
	if existing, ok := ch2.Stages[stageKey]; ok && existing.Completed {
		res := Chapter2StageResult{
			StageNumber:   input.StageNumber,
			FinalRoll:     existing.FinalRoll,
			Enhancement:   existing.EnhancementUsed,
			BonusChoiceID: existing.BonusChoiceID,
			FPBalance:     ch2.FPBalance,
			Completed:     true,
		}
		if input.StageNumber < 10 {
			res.NextStage = input.StageNumber + 1
		}
		return res, wbContext, nil
	}

	// Load server-locked dice for this stage.
	rolls, err := LoadChapter2StageRolls(ctx, pool, actorUserID, cardID, input.StageNumber)
	if err != nil || rolls == nil {
		return Chapter2StageResult{}, nil, errors.New("stage_roll_required")
	}

	finalRoll := ApplyChapter2Enhancement(rolls, input.UseEnhancement)
	if input.UseEnhancement && rolls.D6 == nil {
		return Chapter2StageResult{}, nil, errors.New("d6_not_rolled")
	}

	// Look up the stage rules.
	stageRules, ok := Chapter2StageForNumber(input.StageNumber)
	if !ok {
		return Chapter2StageResult{}, nil, errors.New("invalid_stage_number")
	}

	// Validate bonus choice.
	bonusID := strings.TrimSpace(input.BonusChoiceID)
	var chosenBonus *Chapter2BonusChoice
	for _, bc := range stageRules.BonusChoices {
		if bc.ID == bonusID {
			chosenBonus = &bc
			break
		}
	}
	if chosenBonus == nil {
		return Chapter2StageResult{}, nil, errors.New("bonus_choice_required")
	}

	// Validate trait IDs belong to this stage and are distinct.
	if len(input.TraitIDs) > 2 {
		return Chapter2StageResult{}, nil, errors.New("max_2_traits_per_stage")
	}
	traitsSeen := map[string]bool{}
	traitFPCost := 0
	for _, tid := range input.TraitIDs {
		if traitsSeen[tid] {
			return Chapter2StageResult{}, nil, errors.New("duplicate_trait_id")
		}
		traitsSeen[tid] = true
		found := false
		for _, t := range stageRules.Traits {
			if t.ID == tid {
				found = true
				traitFPCost += t.CostFP
				break
			}
		}
		if !found {
			return Chapter2StageResult{}, nil, fmt.Errorf("trait_%s_not_in_stage", tid)
		}
	}

	// Compute companion cost.
	companionFPCost := 0
	if input.CompanionPurchased {
		if stageRules.Companion == nil {
			return Chapter2StageResult{}, nil, errors.New("no_companion_for_stage")
		}
		companionFPCost = stageRules.Companion.CostFP
	}

	// Compute FP spent this stage.
	enhancementFPCost := 0
	if input.UseEnhancement {
		enhancementFPCost = Chapter2EnhancementCost
	}
	stageFPSpent := enhancementFPCost + companionFPCost + traitFPCost

	// Apply optional Director adjustment (signed integer, may be negative).
	directorAdjust := input.DirectorFateAdjust

	newBalance := ch2.FPBalance + directorAdjust - stageFPSpent
	if newBalance < 0 {
		return Chapter2StageResult{}, nil, errors.New("insufficient_fate_points")
	}

	// Compute attribute deltas for this stage.
	deltas := make(map[string]int)

	// Stage 9: special allocation — 1 to Craft, rest player-distributed.
	if stageRules.HasSpecialAllocation {
		if len(input.Allocation) == 0 {
			return Chapter2StageResult{}, nil, errors.New("stage9_allocation_required")
		}
		totalAllocated := 0
		for _, v := range input.Allocation {
			totalAllocated += v
		}
		if totalAllocated != finalRoll {
			return Chapter2StageResult{}, nil, fmt.Errorf("stage9_allocation_must_total_%d", finalRoll)
		}
		craftAlloc, hasCraft := input.Allocation["Craft"]
		if !hasCraft || craftAlloc < 1 {
			return Chapter2StageResult{}, nil, errors.New("stage9_craft_requires_1")
		}
		for attr, pts := range input.Allocation {
			if pts > 0 {
				deltas[attr] = pts
			}
		}
	} else {
		// All other stages: entire roll goes to the primary attribute.
		deltas[stageRules.PrimaryAttribute] = finalRoll
	}

	// Add bonus choice delta.
	deltas[chosenBonus.TargetAttribute] += chosenBonus.Modifier

	// Validate attribute cap (hard cap = 10).
	for attr, delta := range deltas {
		current := ch2.Attributes[attr]
		if current+delta > Chapter2AttributeCap {
			return Chapter2StageResult{}, nil, fmt.Errorf("attribute_%s_would_exceed_cap", attr)
		}
	}

	// Apply deltas to running attribute tallies.
	for attr, delta := range deltas {
		ch2.Attributes[attr] += delta
	}

	// Record completed stage state.
	traitIDsCopy := make([]string, len(input.TraitIDs))
	copy(traitIDsCopy, input.TraitIDs)

	ch2.Stages[stageKey] = chapter2Stage{
		FinalRoll:          finalRoll,
		EnhancementUsed:    input.UseEnhancement,
		BonusChoiceID:      bonusID,
		CompanionPurchased: input.CompanionPurchased,
		CompanionTarget:    strings.TrimSpace(input.CompanionTarget),
		TraitIDs:           traitIDsCopy,
		Allocation:         input.Allocation,
		RepresentationText: strings.TrimSpace(input.RepresentationText),
		FPSpent:            stageFPSpent,
		Completed:          true,
	}
	ch2.FPBalance = newBalance
	if input.StageNumber >= ch2.CurrentStage {
		ch2.CurrentStage = input.StageNumber + 1
	}

	// Merge chapter2 state back into workbook context.
	wbContext["chapter2"] = ch2
	nextStage := 0
	if input.StageNumber < 10 {
		nextStage = input.StageNumber + 1
	}
	wbContext["current_stage"] = 2
	if input.StageNumber == 10 {
		wbContext["current_stage"] = 3 // Completed Chapter 2
		wbContext["current_event"] = "chapter2_complete"
	} else {
		wbContext["current_event"] = fmt.Sprintf("chapter2_stage_%d", nextStage)
	}

	result := Chapter2StageResult{
		StageNumber:     input.StageNumber,
		FinalRoll:       finalRoll,
		Enhancement:     input.UseEnhancement,
		BonusChoiceID:   bonusID,
		AttributeDeltas: deltas,
		FPSpent:         stageFPSpent,
		FPBalance:       newBalance,
		Completed:       true,
		NextStage:       nextStage,
	}

	return result, wbContext, nil
}
