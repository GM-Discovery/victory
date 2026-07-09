package playerrelationships

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CommitRelationshipPage validates and stores one workbook page's answers as
// a typed relationship event, then recomputes effective facts. A commit that
// produces no material change creates no event (Kernel 62 §9.2) -- `changed`
// reports which happened. Ownership is proven before anything is read.
func CommitRelationshipPage(ctx context.Context, pool *pgxpool.Pool, observerUserID, relationshipID, pageKey string, answers map[string]any) (RelationshipEvent, bool, error) {
	pageKey = strings.TrimSpace(pageKey)
	if pageKey == "" {
		return RelationshipEvent{}, false, errors.New("page_key_required")
	}

	if _, err := loadRelationshipOwned(ctx, pool, observerUserID, relationshipID); err != nil {
		return RelationshipEvent{}, false, err
	}

	cat, err := LoadCatalogue()
	if err != nil {
		return RelationshipEvent{}, false, err
	}
	page, ok := cat.PageByKey(pageKey)
	if !ok {
		return RelationshipEvent{}, false, errors.New("unknown_page")
	}

	sanitized, err := ValidatePageAnswers(page, answers)
	if err != nil {
		return RelationshipEvent{}, false, err
	}

	events, err := loadRelationshipEvents(ctx, pool, relationshipID)
	if err != nil {
		return RelationshipEvent{}, false, err
	}
	currentFacts := DeriveEffectiveFacts(events)

	if !PageAnswersDifferFromFacts(sanitized, currentFacts) {
		return RelationshipEvent{}, false, nil
	}

	summary := BuildPageCommitSummary(page, sanitized)
	newEvent, err := insertRelationshipEvent(ctx, pool, relationshipID, "page_commit", pageKey, sanitized, summary, observerUserID)
	if err != nil {
		return RelationshipEvent{}, false, err
	}

	if err := RecomputeRelationshipFacts(ctx, pool, relationshipID); err != nil {
		return RelationshipEvent{}, false, err
	}
	if err := touchRelationship(ctx, pool, relationshipID); err != nil {
		return RelationshipEvent{}, false, err
	}

	return newEvent, true, nil
}

// DeleteRelationshipEvent removes one ordinary relationship event owned by
// the observer and recomputes effective facts (Kernel 62 §5.4, §9.2). A hard
// row delete, matching the Kernel 61 player-profile convention.
func DeleteRelationshipEvent(ctx context.Context, pool *pgxpool.Pool, observerUserID, relationshipID, eventID string) error {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return errors.New("event_id_required")
	}

	if _, err := loadRelationshipOwned(ctx, pool, observerUserID, relationshipID); err != nil {
		return err
	}

	tag, err := pool.Exec(ctx, `
		DELETE FROM player_relationship_events
		WHERE id = $1 AND relationship_id = $2
	`, eventID, relationshipID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("event_not_found")
	}

	if err := RecomputeRelationshipFacts(ctx, pool, relationshipID); err != nil {
		return err
	}
	return touchRelationship(ctx, pool, relationshipID)
}

// RecomputeRelationshipFacts rebuilds player_relationship_facts from every
// remaining event -- a full replace inside one transaction so there is no
// drift between events and facts (Kernel 62 §5.3, §9.2).
func RecomputeRelationshipFacts(ctx context.Context, pool *pgxpool.Pool, relationshipID string) error {
	events, err := loadRelationshipEvents(ctx, pool, relationshipID)
	if err != nil {
		return err
	}
	facts := DeriveEffectiveFacts(events)

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM player_relationship_facts WHERE relationship_id = $1`, relationshipID); err != nil {
		return err
	}

	for _, fact := range facts {
		encoded, err := jsonEncode(fact.ValueJSON)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO player_relationship_facts
				(relationship_id, field_key, value_json, display_value, source_event_id, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, relationshipID, fact.FieldKey, encoded, fact.DisplayValue, fact.SourceEventID, fact.UpdatedAt); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func loadRelationshipEvents(ctx context.Context, pool *pgxpool.Pool, relationshipID string) ([]RelationshipEvent, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, event_type, page_key, payload, human_summary, created_at
		FROM player_relationship_events
		WHERE relationship_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
	`, relationshipID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RelationshipEvent
	for rows.Next() {
		var e RelationshipEvent
		var payload map[string]any
		if err := rows.Scan(&e.ID, &e.EventType, &e.PageKey, &payload, &e.HumanSummary, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Payload = payload
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func insertRelationshipEvent(ctx context.Context, pool *pgxpool.Pool, relationshipID, eventType, pageKey string, payload map[string]any, summary, createdByUserID string) (RelationshipEvent, error) {
	var e RelationshipEvent
	if err := pool.QueryRow(ctx, `
		INSERT INTO player_relationship_events (relationship_id, event_type, page_key, payload, human_summary, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, event_type, page_key, payload, human_summary, created_at
	`, relationshipID, eventType, pageKey, payload, summary, createdByUserID).Scan(
		&e.ID, &e.EventType, &e.PageKey, &e.Payload, &e.HumanSummary, &e.CreatedAt,
	); err != nil {
		return RelationshipEvent{}, err
	}
	return e, nil
}

func loadCurrentFacts(ctx context.Context, pool *pgxpool.Pool, relationshipID string) (map[string]RelationshipFact, error) {
	rows, err := pool.Query(ctx, `
		SELECT field_key, value_json, display_value, COALESCE(source_event_id::text, ''), updated_at
		FROM player_relationship_facts
		WHERE relationship_id = $1
	`, relationshipID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]RelationshipFact{}
	for rows.Next() {
		var f RelationshipFact
		if err := rows.Scan(&f.FieldKey, &f.ValueJSON, &f.DisplayValue, &f.SourceEventID, &f.UpdatedAt); err != nil {
			return nil, err
		}
		out[f.FieldKey] = f
	}
	return out, rows.Err()
}

// jsonEncode always produces pre-encoded JSON bytes for a value_json column
// -- pgx's jsonb codec treats a bare Go string as already-valid JSON text,
// which fails for unquoted values (same footgun note as playerprofile).
func jsonEncode(v any) ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}
	return json.Marshal(v)
}
