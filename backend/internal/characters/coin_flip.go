package characters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// coinFlipEventKey returns the canonical character_workbook_rolls event_key
// for a parent's inheritance coin flip, keyed by character_card_id as the
// draft_token (the character already exists by the time a coin flip is
// requested).
func coinFlipEventKey(parentIndex int) string {
	return fmt.Sprintf("coin_flip.parent_%d", parentIndex)
}

type coinFlipPayload struct {
	Value int `json:"value"`
}

func loadCoinFlipRoll(ctx context.Context, pool *pgxpool.Pool, ownerUserID, cardID string, parentIndex int) (int, bool, error) {
	eventKey := coinFlipEventKey(parentIndex)
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

func storeCoinFlipRoll(ctx context.Context, pool *pgxpool.Pool, ownerUserID, cardID string, parentIndex, value int) error {
	eventKey := coinFlipEventKey(parentIndex)
	payload, _ := json.Marshal(coinFlipPayload{Value: value})
	_, err := pool.Exec(ctx, `
		INSERT INTO character_workbook_rolls (owner_user_id, draft_token, event_key, dice_payload, roll_total)
		VALUES ($1, $2, $3, $4::jsonb, $5)
		ON CONFLICT (owner_user_id, draft_token, event_key) DO NOTHING
	`, ownerUserID, cardID, eventKey, string(payload), value)
	return err
}

// CatharsisCoinFlipResult is returned to the client after a coin flip is
// requested or replayed.
type CatharsisCoinFlipResult struct {
	ParentIndex       int    `json:"parent_index"`
	Eligible          bool   `json:"eligible"`
	CoinFlipRoll      int    `json:"coin_flip_roll"`
	CoinFlipResult    string `json:"coin_flip_result"`
	InheritancePassed bool   `json:"inheritance_passed"`
	InheritedWealth   int    `json:"inherited_wealth"`
	StartingWealth    int    `json:"starting_wealth"`
}

// RequestCatharsisCoinFlip resolves (or replays) the inheritance coin flip
// for one parent on an already-created character card. The flip is
// server-authoritative and idempotent: once resolved, repeated requests
// return the same stored result rather than flipping again.
func RequestCatharsisCoinFlip(ctx context.Context, pool *pgxpool.Pool, ownerUserID, cardID string, parentIndex int) (CatharsisCoinFlipResult, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	cardID = strings.TrimSpace(cardID)

	if ownerUserID == "" {
		return CatharsisCoinFlipResult{}, errors.New("not_authenticated")
	}
	if cardID == "" {
		return CatharsisCoinFlipResult{}, errors.New("character_card_id_required")
	}
	if parentIndex < 1 {
		return CatharsisCoinFlipResult{}, errors.New("invalid_parent_index")
	}

	var wbContextRaw []byte
	if err := pool.QueryRow(ctx, `
		SELECT workbook_context FROM character_cards WHERE id = $1 AND is_deleted = FALSE
	`, cardID).Scan(&wbContextRaw); err != nil {
		return CatharsisCoinFlipResult{}, err
	}
	wbContext := decodeJSONMap(wbContextRaw)
	rows := normalizeCatharsisParentageRows(wbContext["socio_parentage_parents"])

	rowIdx := -1
	for i, row := range rows {
		idx, _ := parseCatharsisInt(row["parent_index"])
		if idx == parentIndex {
			rowIdx = i
			break
		}
	}
	if rowIdx == -1 {
		return CatharsisCoinFlipResult{}, errors.New("parent_not_found")
	}

	row := rows[rowIdx]
	startCredit, _ := parseCatharsisInt(row["starting_credit"])
	eligible := startCredit > 50
	if !eligible {
		return CatharsisCoinFlipResult{}, errors.New("not_eligible")
	}

	existingResult := strings.TrimSpace(stringValue(row["coin_flip_result"]))
	if existingResult != "" && existingResult != "pending" {
		// Already resolved: replay the stored outcome rather than reflipping.
		inheritedWealth, _ := parseCatharsisInt(row["inherited_wealth"])
		coinFlipRoll, _ := parseCatharsisInt(row["coin_flip_roll"])
		startingWealth, _ := parseCatharsisInt(wbContext["socio_starting_wealth"])
		return CatharsisCoinFlipResult{
			ParentIndex:       parentIndex,
			Eligible:          true,
			CoinFlipRoll:      coinFlipRoll,
			CoinFlipResult:    existingResult,
			InheritancePassed: existingResult == "retain",
			InheritedWealth:   inheritedWealth,
			StartingWealth:    startingWealth,
		}, nil
	}

	// Resolve a new server-locked coin flip (idempotent against retries via
	// character_workbook_rolls' unique index).
	value := 0
	if existing, ok, err := loadCoinFlipRoll(ctx, pool, ownerUserID, cardID, parentIndex); err != nil {
		return CatharsisCoinFlipResult{}, err
	} else if ok {
		value = existing
	} else {
		value = randomCatharsisD2()
		if err := storeCoinFlipRoll(ctx, pool, ownerUserID, cardID, parentIndex, value); err != nil {
			return CatharsisCoinFlipResult{}, err
		}
		if committed, ok, err := loadCoinFlipRoll(ctx, pool, ownerUserID, cardID, parentIndex); err != nil {
			return CatharsisCoinFlipResult{}, err
		} else if ok {
			value = committed
		}
	}

	coinFlipResult := "lose"
	inheritancePassed := false
	inheritedWealth := 0
	if value == 2 {
		coinFlipResult = "retain"
		inheritancePassed = true
		inheritedWealth = startCredit
	}

	row["coin_flip_roll"] = value
	row["coin_flip_result"] = coinFlipResult
	row["inheritance_passed"] = inheritancePassed
	row["inherited_wealth"] = inheritedWealth
	rows[rowIdx] = row

	// Recompute the overall starting wealth: the highest inherited_wealth
	// among parents whose flip retained, first-resolved wins on ties (this
	// matches the prior synchronous resolution order).
	startingWealth := 0
	var sourceParentIndex, sourceRoll int
	var sourceClass string
	for _, r := range rows {
		passed, _ := r["inheritance_passed"].(bool)
		if !passed {
			continue
		}
		wealth, _ := parseCatharsisInt(r["inherited_wealth"])
		if wealth > startingWealth {
			startingWealth = wealth
			sourceParentIndex, _ = parseCatharsisInt(r["parent_index"])
			sourceRoll, _ = parseCatharsisRoll(r["roll_total"])
			sourceClass = stringValue(r["social_class"])
		}
	}

	wbContext["socio_parentage_parents"] = rows
	wbContext["socio_starting_wealth"] = startingWealth
	wbContext["socio_starting_wealth_source_parent_index"] = sourceParentIndex
	wbContext["socio_starting_wealth_source_roll"] = sourceRoll
	wbContext["socio_starting_wealth_source_class"] = sourceClass

	payloadJSON, err := json.Marshal(wbContext)
	if err != nil {
		return CatharsisCoinFlipResult{}, err
	}
	if _, err := pool.Exec(ctx, `
		UPDATE character_cards SET workbook_context = $2::jsonb, updated_at = NOW()
		WHERE id = $1 AND is_deleted = FALSE
	`, cardID, string(payloadJSON)); err != nil {
		return CatharsisCoinFlipResult{}, err
	}

	return CatharsisCoinFlipResult{
		ParentIndex:       parentIndex,
		Eligible:          true,
		CoinFlipRoll:      value,
		CoinFlipResult:    coinFlipResult,
		InheritancePassed: inheritancePassed,
		InheritedWealth:   inheritedWealth,
		StartingWealth:    startingWealth,
	}, nil
}
