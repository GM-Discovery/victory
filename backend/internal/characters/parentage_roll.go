package characters

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dice"
)

const maxDraftTokenLength = 128

var catharsisParentageEventKeys = map[string]bool{
	"egg_donor":   true,
	"sperm_donor": true,
}

type parentageDie struct {
	Index    int   `json:"index"`
	Chain    []int `json:"chain"`
	Subtotal int   `json:"subtotal"`
}

// rollExplodingD20Single rolls one base d20 via the canonical crypto dice
// source. If the base die lands on 20, exactly one additional d20 is rolled
// and added to the chain; that additional die cannot explode again.
func rollExplodingD20Single(source dice.RandomSource) (parentageDie, error) {
	raw, err := source.Intn(20)
	if err != nil {
		return parentageDie{}, err
	}
	face := raw + 1
	chain := []int{face}
	subtotal := face

	if face == 20 {
		explodedRaw, err := source.Intn(20)
		if err != nil {
			return parentageDie{}, err
		}
		explodedFace := explodedRaw + 1
		chain = append(chain, explodedFace)
		subtotal += explodedFace
	}

	return parentageDie{Chain: chain, Subtotal: subtotal}, nil
}

// rollCanonicalParentage3d20 rolls the canonical 3d20 (each base die may
// explode once) using the shared crypto-secure dice source.
func rollCanonicalParentage3d20() ([]parentageDie, int, error) {
	source := dice.CryptoSource{}
	results := make([]parentageDie, 0, 3)
	total := 0
	for index := 0; index < 3; index++ {
		die, err := rollExplodingD20Single(source)
		if err != nil {
			return nil, 0, err
		}
		die.Index = index
		results = append(results, die)
		total += die.Subtotal
	}
	return results, total, nil
}

type parentageRollRow struct {
	Dice      []parentageDie `json:"dice"`
	RollTotal int            `json:"roll_total"`
}

func loadCatharsisParentageRoll(ctx context.Context, pool *pgxpool.Pool, ownerUserID, draftToken, eventKey string) (parentageRollRow, bool, error) {
	var raw []byte
	var total int
	err := pool.QueryRow(ctx, `
		SELECT dice_payload, roll_total
		FROM character_workbook_rolls
		WHERE owner_user_id = $1 AND draft_token = $2 AND event_key = $3
	`, ownerUserID, draftToken, eventKey).Scan(&raw, &total)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return parentageRollRow{}, false, nil
		}
		return parentageRollRow{}, false, err
	}
	var dice []parentageDie
	_ = json.Unmarshal(raw, &dice)
	return parentageRollRow{Dice: dice, RollTotal: total}, true, nil
}

func composeParentageRollResult(eventKey string, row parentageRollRow) (map[string]any, error) {
	entry, ok := ParentageChartEntryForRoll(row.RollTotal)
	if !ok {
		return nil, errors.New("chart_entry_not_found")
	}
	return map[string]any{
		"event_key":               eventKey,
		"dice":                    row.Dice,
		"roll_total":              row.RollTotal,
		"roll_range":              formatParentageRollRange(entry),
		"social_class":            entry.SocialClass,
		"wealth_kind":             entry.WealthKind,
		"starting_credit":         entry.StartingCredit,
		"description":             entry.Description,
		"parentage_chart_version": ParentageChartVersionV11,
	}, nil
}

// RequestCatharsisParentageRoll resolves the canonical, server-authoritative
// 3d20 parentage roll for one donor event. It is idempotent per
// (owner, draft_token, event_key): a repeated request returns the
// already-locked result instead of rolling again.
func RequestCatharsisParentageRoll(ctx context.Context, pool *pgxpool.Pool, ownerUserID, draftToken, eventKey string) (map[string]any, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	draftToken = strings.TrimSpace(draftToken)
	eventKey = strings.TrimSpace(eventKey)

	if ownerUserID == "" {
		return nil, errors.New("not_authenticated")
	}
	if draftToken == "" {
		return nil, errors.New("draft_token_required")
	}
	if len(draftToken) > maxDraftTokenLength {
		return nil, errors.New("draft_token_required")
	}
	if !catharsisParentageEventKeys[eventKey] {
		return nil, errors.New("event_key_required")
	}

	if existing, ok, err := loadCatharsisParentageRoll(ctx, pool, ownerUserID, draftToken, eventKey); err != nil {
		return nil, err
	} else if ok {
		return composeParentageRollResult(eventKey, existing)
	}

	rolledDice, total, err := rollCanonicalParentage3d20()
	if err != nil {
		return nil, err
	}
	dicePayload, err := json.Marshal(rolledDice)
	if err != nil {
		return nil, err
	}

	var rollID string
	err = pool.QueryRow(ctx, `
		INSERT INTO character_workbook_rolls (owner_user_id, draft_token, event_key, dice_payload, roll_total)
		VALUES ($1, $2, $3, $4::jsonb, $5)
		ON CONFLICT (owner_user_id, draft_token, event_key) DO NOTHING
		RETURNING id::text
	`, ownerUserID, draftToken, eventKey, string(dicePayload), total).Scan(&rollID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			existing, ok, lerr := loadCatharsisParentageRoll(ctx, pool, ownerUserID, draftToken, eventKey)
			if lerr != nil {
				return nil, lerr
			}
			if ok {
				return composeParentageRollResult(eventKey, existing)
			}
			return nil, errors.New("parentage_roll_failed")
		}
		return nil, err
	}

	return composeParentageRollResult(eventKey, parentageRollRow{Dice: rolledDice, RollTotal: total})
}

// loadCatharsisParentageRolls fetches every locked donor roll for a draft
// token, keyed by event key. Missing rolls are simply absent from the map.
func loadCatharsisParentageRolls(ctx context.Context, pool *pgxpool.Pool, ownerUserID, draftToken string) (map[string]parentageRollRow, error) {
	out := map[string]parentageRollRow{}
	for eventKey := range catharsisParentageEventKeys {
		row, ok, err := loadCatharsisParentageRoll(ctx, pool, ownerUserID, draftToken, eventKey)
		if err != nil {
			return nil, err
		}
		if ok {
			out[eventKey] = row
		}
	}
	return out, nil
}

// attachCanonicalCatharsisParentageRows replaces any client-supplied
// socio_parentage_parents/socio_parentage_roll fields with rows built solely
// from the server-locked donor rolls for the request's draft_token. This is
// the security boundary that keeps the parentage roll and Starting Credit
// server-authoritative: the client cannot forge a roll total or chart row by
// including its own values in the request body.
func attachCanonicalCatharsisParentageRows(ctx context.Context, pool *pgxpool.Pool, ownerUserID string, input CharacterCardInput) (CharacterCardInput, error) {
	context := normalizeWorkbookContext(input.WorkbookContext)
	draftToken := strings.TrimSpace(stringValue(context["draft_token"]))
	if draftToken == "" {
		return CharacterCardInput{}, errors.New("draft_token_required")
	}

	rolls, err := loadCatharsisParentageRolls(ctx, pool, ownerUserID, draftToken)
	if err != nil {
		return CharacterCardInput{}, err
	}
	eggRoll, ok := rolls["egg_donor"]
	if !ok {
		return CharacterCardInput{}, errors.New("egg_donor_roll_required")
	}
	spermRoll, ok := rolls["sperm_donor"]
	if !ok {
		return CharacterCardInput{}, errors.New("sperm_donor_roll_required")
	}

	context["socio_parentage_parents"] = []map[string]any{
		{"parent_index": 1, "event_key": "egg_donor", "roll_total": eggRoll.RollTotal, "dice": eggRoll.Dice},
		{"parent_index": 2, "event_key": "sperm_donor", "roll_total": spermRoll.RollTotal, "dice": spermRoll.Dice},
	}
	context["socio_parentage_first_roll"] = eggRoll.RollTotal
	context["socio_parentage_second_roll"] = spermRoll.RollTotal
	context["socio_parentage_roll"] = eggRoll.RollTotal + spermRoll.RollTotal

	input.WorkbookContext = context
	return input, nil
}
