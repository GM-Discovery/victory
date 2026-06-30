package characters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dice"
)

// chapter2RollEventKey returns the canonical character_workbook_rolls event_key
// for a given stage (1-10) and die type ("d4" or "d6").
// We store chapter 2 rolls in the same character_workbook_rolls table as
// parentage rolls, using the character_card_id as the draft_token field
// (the character always exists before Chapter 2 begins).
func chapter2RollEventKey(stageNumber int, dieType string) string {
	return fmt.Sprintf("c2.s%02d.%s", stageNumber, dieType)
}

type chapter2SingleRoll struct {
	Value int `json:"value"`
}

func loadChapter2Roll(ctx context.Context, pool *pgxpool.Pool, ownerUserID, cardID string, stageNumber int, dieType string) (int, bool, error) {
	eventKey := chapter2RollEventKey(stageNumber, dieType)
	var raw []byte
	var total int
	err := pool.QueryRow(ctx, `
		SELECT dice_payload, roll_total
		FROM character_workbook_rolls
		WHERE owner_user_id = $1 AND draft_token = $2 AND event_key = $3
	`, ownerUserID, cardID, eventKey).Scan(&raw, &total)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return total, true, nil
}

func storeChapter2Roll(ctx context.Context, pool *pgxpool.Pool, ownerUserID, cardID string, stageNumber int, dieType string, value int) error {
	eventKey := chapter2RollEventKey(stageNumber, dieType)
	payload, _ := json.Marshal(chapter2SingleRoll{Value: value})
	_, err := pool.Exec(ctx, `
		INSERT INTO character_workbook_rolls (owner_user_id, draft_token, event_key, dice_payload, roll_total)
		VALUES ($1, $2, $3, $4::jsonb, $5)
		ON CONFLICT (owner_user_id, draft_token, event_key) DO NOTHING
	`, ownerUserID, cardID, eventKey, string(payload), value)
	return err
}

// RequestChapter2Roll generates or retrieves a server-locked die roll for a
// Chapter 2 life stage. dieType must be "d4" or "d6". A d6 may only be
// requested after a d4 has been locked for the same stage. The result is
// idempotent: repeated requests return the same value.
func RequestChapter2Roll(ctx context.Context, pool *pgxpool.Pool, ownerUserID, cardID string, stageNumber int, dieType string) (int, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	cardID = strings.TrimSpace(cardID)
	dieType = strings.TrimSpace(dieType)

	if ownerUserID == "" {
		return 0, errors.New("not_authenticated")
	}
	if cardID == "" {
		return 0, errors.New("character_card_id_required")
	}
	if stageNumber < 1 || stageNumber > 10 {
		return 0, errors.New("invalid_stage_number")
	}
	if dieType != "d4" && dieType != "d6" {
		return 0, errors.New("invalid_die_type")
	}

	// A d6 enhancement requires the d4 to already be locked.
	if dieType == "d6" {
		if _, d4Exists, err := loadChapter2Roll(ctx, pool, ownerUserID, cardID, stageNumber, "d4"); err != nil {
			return 0, err
		} else if !d4Exists {
			return 0, errors.New("d4_must_be_rolled_first")
		}
	}

	// Return existing value if already committed (idempotent).
	if existing, ok, err := loadChapter2Roll(ctx, pool, ownerUserID, cardID, stageNumber, dieType); err != nil {
		return 0, err
	} else if ok {
		return existing, nil
	}

	// Generate a new cryptographically random roll.
	sides := 4
	if dieType == "d6" {
		sides = 6
	}
	raw, err := dice.CryptoSource{}.Intn(sides)
	if err != nil {
		return 0, err
	}
	value := raw + 1

	if err := storeChapter2Roll(ctx, pool, ownerUserID, cardID, stageNumber, dieType, value); err != nil {
		return 0, err
	}

	// Load back in case of a concurrent write (ON CONFLICT DO NOTHING).
	if committed, ok, err := loadChapter2Roll(ctx, pool, ownerUserID, cardID, stageNumber, dieType); err != nil {
		return 0, err
	} else if ok {
		return committed, nil
	}
	return value, nil
}

// Chapter2RollResult holds the rolls for one stage with derived values.
type Chapter2RollResult struct {
	StageNumber      int    `json:"stage_number"`
	D4               int    `json:"d4"`
	D6               *int   `json:"d6,omitempty"`
	FinalRoll        int    `json:"final_roll"`
	EnhancementUsed  bool   `json:"enhancement_used"`
	PrimaryAttribute string `json:"primary_attribute"`
}

// LoadChapter2StageRolls returns all committed roll results for a stage, or
// nil if the d4 has not been rolled yet.
func LoadChapter2StageRolls(ctx context.Context, pool *pgxpool.Pool, ownerUserID, cardID string, stageNumber int) (*Chapter2RollResult, error) {
	d4Val, d4Ok, err := loadChapter2Roll(ctx, pool, ownerUserID, cardID, stageNumber, "d4")
	if err != nil || !d4Ok {
		return nil, err
	}

	stage, ok := Chapter2StageForNumber(stageNumber)
	if !ok {
		return nil, errors.New("invalid_stage_number")
	}

	result := &Chapter2RollResult{
		StageNumber:      stageNumber,
		D4:               d4Val,
		FinalRoll:        d4Val,
		PrimaryAttribute: stage.PrimaryAttribute,
	}

	d6Val, d6Ok, err := loadChapter2Roll(ctx, pool, ownerUserID, cardID, stageNumber, "d6")
	if err != nil {
		return nil, err
	}
	if d6Ok {
		result.D6 = &d6Val
	}

	return result, nil
}

// ApplyChapter2Enhancement resolves the final roll for a stage given whether
// the enhancement is accepted. The d6 must have been locked server-side first.
func ApplyChapter2Enhancement(rolls *Chapter2RollResult, useEnhancement bool) int {
	if rolls == nil {
		return 0
	}
	if useEnhancement && rolls.D6 != nil && *rolls.D6 > rolls.D4 {
		return *rolls.D6
	}
	return rolls.D4
}
