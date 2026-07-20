package merchant

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// characterActiveAndOwnedBy is a deliberately narrower check than
// characters.CanEditCard: CanEditCard also requires CanDraftCharacter (a
// location_memberships role of producer/director/cast/crew), which a
// ticket-only Show Run Player -- Kernel 71's whole point -- never has.
// showruns.SelectCharacter (roster.go) already recognized this and does its
// own direct owner_user_id + is_deleted check rather than reusing
// CanEditCard; every merchant ownership check follows that same precedent,
// not CanEditCard's stricter editing gate.
func characterActiveAndOwnedBy(ctx context.Context, pool *pgxpool.Pool, userID, cardID string) (bool, error) {
	userID = strings.TrimSpace(userID)
	cardID = strings.TrimSpace(cardID)
	if userID == "" || cardID == "" {
		return false, nil
	}
	var ownerUserID string
	var isDeleted bool
	err := pool.QueryRow(ctx, `
		SELECT owner_user_id::text, is_deleted FROM character_cards WHERE id = $1
	`, cardID).Scan(&ownerUserID, &isDeleted)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, errors.New("character_not_found")
	}
	if err != nil {
		return false, err
	}
	if isDeleted {
		return false, nil
	}
	return ownerUserID == userID, nil
}

const inventoryItemColumns = `
	id::text, character_card_id::text, equipment_item_id::text, quantity,
	acquired_by_user_id::text, COALESCE(source_show_run_id::text, ''),
	COALESCE(source_show_id::text, ''), COALESCE(source_session_id::text, ''),
	COALESCE(source_scene_placement_id::text, ''), source_interaction_key,
	first_acquired_at, updated_at
`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanInventoryEntry(row rowScanner) (InventoryEntry, error) {
	var e InventoryEntry
	if err := row.Scan(
		&e.ID, &e.CharacterCardID, &e.EquipmentItemID, &e.Quantity,
		&e.AcquiredByUserID, &e.SourceShowRunID, &e.SourceShowID, &e.SourceSessionID,
		&e.SourceScenePlacementID, &e.SourceInteractionKey, &e.FirstAcquiredAt, &e.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return InventoryEntry{}, errors.New("inventory_item_not_found")
		}
		return InventoryEntry{}, err
	}
	return e, nil
}

// ListInventoryForCharacter is the read path for both the new Inventory page
// and Equip Mode's "you already have this" confirmation -- item details are
// joined in for display, most recently updated first.
func ListInventoryForCharacter(ctx context.Context, pool *pgxpool.Pool, actorUserID, characterCardID string) ([]InventoryEntry, error) {
	characterCardID = strings.TrimSpace(characterCardID)
	if characterCardID == "" {
		return nil, errors.New("character_card_id_required")
	}
	allowed, err := characterActiveAndOwnedBy(ctx, pool, actorUserID, characterCardID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errors.New("forbidden")
	}

	rows, err := pool.Query(ctx, `
		SELECT `+inventoryItemColumns+`
		FROM character_inventory_items
		WHERE character_card_id = $1
		ORDER BY updated_at DESC
	`, characterCardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []InventoryEntry
	var itemIDs []string
	for rows.Next() {
		e, err := scanInventoryEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
		itemIDs = append(itemIDs, e.EquipmentItemID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	items := map[string]EquipmentItem{}
	for _, id := range itemIDs {
		if _, ok := items[id]; ok {
			continue
		}
		item, err := LoadEquipmentItemByID(ctx, pool, id)
		if err != nil {
			return nil, err
		}
		items[id] = item
	}
	for i := range out {
		if item, ok := items[out[i].EquipmentItemID]; ok {
			itemCopy := item
			out[i].Item = &itemCopy
		}
	}
	return out, nil
}

func nullableUUID(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

type PurchaseInput struct {
	ActorUserID            string
	CharacterCardID        string
	EquipmentItemID        string
	IdempotencyKey         string
	SourceShowRunID        string
	SourceShowID           string
	SourceSessionID        string
	SourceScenePlacementID string
	SourceInteractionKey   string
}

// PurchaseEquipment resolves character-at-time-of-purchase (the caller
// passes CharacterCardID already resolved fresh from the roster at request
// time -- see interactions.go's eligibility resolution, spec S2.10),
// validates the character is active/owned and the item is active, then
// creates or increments the holding row. Idempotency follows the
// cue_executions ledger pattern (cues/execute.go): a dedicated attempt
// table, not a field on the holdings row, so a retried request (same
// idempotency_key) is answered from the ledger without double-applying the
// quantity change (spec S5.3). A deliberate second purchase (new
// idempotency_key) is free to increase quantity again.
func PurchaseEquipment(ctx context.Context, pool *pgxpool.Pool, in PurchaseInput) (InventoryEntry, error) {
	in.ActorUserID = strings.TrimSpace(in.ActorUserID)
	in.CharacterCardID = strings.TrimSpace(in.CharacterCardID)
	in.EquipmentItemID = strings.TrimSpace(in.EquipmentItemID)
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if in.ActorUserID == "" {
		return InventoryEntry{}, errors.New("not_authenticated")
	}
	if in.CharacterCardID == "" {
		return InventoryEntry{}, errors.New("character_card_id_required")
	}
	if in.EquipmentItemID == "" {
		return InventoryEntry{}, errors.New("equipment_item_id_required")
	}
	if in.IdempotencyKey == "" {
		return InventoryEntry{}, errors.New("idempotency_key_required")
	}

	allowed, err := characterActiveAndOwnedBy(ctx, pool, in.ActorUserID, in.CharacterCardID)
	if err != nil {
		return InventoryEntry{}, err
	}
	if !allowed {
		return InventoryEntry{}, errors.New("forbidden")
	}

	item, err := LoadEquipmentItemByID(ctx, pool, in.EquipmentItemID)
	if err != nil {
		return InventoryEntry{}, err
	}
	if !item.Active {
		return InventoryEntry{}, errors.New("equipment_item_inactive")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return InventoryEntry{}, err
	}
	defer tx.Rollback(ctx)

	var attemptID string
	insertErr := tx.QueryRow(ctx, `
		INSERT INTO character_inventory_purchase_attempts (
			character_card_id, equipment_item_id, idempotency_key, quantity_delta
		)
		VALUES ($1, $2, $3, 1)
		ON CONFLICT (character_card_id, equipment_item_id, idempotency_key) DO NOTHING
		RETURNING id::text
	`, in.CharacterCardID, in.EquipmentItemID, in.IdempotencyKey).Scan(&attemptID)

	if insertErr != nil && !errors.Is(insertErr, pgx.ErrNoRows) {
		return InventoryEntry{}, insertErr
	}

	if errors.Is(insertErr, pgx.ErrNoRows) {
		// Retry of an already-processed attempt: answer from the ledger
		// without touching quantity again.
		var invID string
		if err := tx.QueryRow(ctx, `
			SELECT resulting_inventory_item_id::text
			FROM character_inventory_purchase_attempts
			WHERE character_card_id = $1 AND equipment_item_id = $2 AND idempotency_key = $3
		`, in.CharacterCardID, in.EquipmentItemID, in.IdempotencyKey).Scan(&invID); err != nil {
			return InventoryEntry{}, err
		}
		entry, err := scanInventoryEntry(tx.QueryRow(ctx, `SELECT `+inventoryItemColumns+` FROM character_inventory_items WHERE id = $1`, invID))
		if err != nil {
			return InventoryEntry{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return InventoryEntry{}, err
		}
		return entry, nil
	}

	// New attempt: unique-mode items clamp at quantity 1 (a repeat purchase
	// is a no-op, not a second unit); stackable items increment.
	var invID string
	var upsertErr error
	if item.QuantityMode == QuantityModeUnique {
		upsertErr = tx.QueryRow(ctx, `
			INSERT INTO character_inventory_items (
				character_card_id, equipment_item_id, quantity, acquired_by_user_id,
				source_show_run_id, source_show_id, source_session_id,
				source_scene_placement_id, source_interaction_key
			)
			VALUES ($1, $2, 1, $3, $4::uuid, $5::uuid, $6::uuid, $7::uuid, $8)
			ON CONFLICT (character_card_id, equipment_item_id) DO UPDATE
			  SET updated_at = NOW()
			RETURNING id::text
		`, in.CharacterCardID, in.EquipmentItemID, in.ActorUserID,
			nullableUUID(in.SourceShowRunID), nullableUUID(in.SourceShowID), nullableUUID(in.SourceSessionID),
			nullableUUID(in.SourceScenePlacementID), in.SourceInteractionKey).Scan(&invID)
	} else {
		upsertErr = tx.QueryRow(ctx, `
			INSERT INTO character_inventory_items (
				character_card_id, equipment_item_id, quantity, acquired_by_user_id,
				source_show_run_id, source_show_id, source_session_id,
				source_scene_placement_id, source_interaction_key
			)
			VALUES ($1, $2, 1, $3, $4::uuid, $5::uuid, $6::uuid, $7::uuid, $8)
			ON CONFLICT (character_card_id, equipment_item_id) DO UPDATE
			  SET quantity = character_inventory_items.quantity + 1,
			      acquired_by_user_id = EXCLUDED.acquired_by_user_id,
			      source_show_run_id = EXCLUDED.source_show_run_id,
			      source_show_id = EXCLUDED.source_show_id,
			      source_session_id = EXCLUDED.source_session_id,
			      source_scene_placement_id = EXCLUDED.source_scene_placement_id,
			      source_interaction_key = EXCLUDED.source_interaction_key,
			      updated_at = NOW()
			RETURNING id::text
		`, in.CharacterCardID, in.EquipmentItemID, in.ActorUserID,
			nullableUUID(in.SourceShowRunID), nullableUUID(in.SourceShowID), nullableUUID(in.SourceSessionID),
			nullableUUID(in.SourceScenePlacementID), in.SourceInteractionKey).Scan(&invID)
	}
	if upsertErr != nil {
		return InventoryEntry{}, upsertErr
	}

	if _, err := tx.Exec(ctx, `
		UPDATE character_inventory_purchase_attempts SET resulting_inventory_item_id = $2 WHERE id = $1
	`, attemptID, invID); err != nil {
		return InventoryEntry{}, err
	}

	entry, err := scanInventoryEntry(tx.QueryRow(ctx, `SELECT `+inventoryItemColumns+` FROM character_inventory_items WHERE id = $1`, invID))
	if err != nil {
		return InventoryEntry{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return InventoryEntry{}, err
	}

	entryCopy := entry
	entryCopy.Item = &item
	return entryCopy, nil
}
