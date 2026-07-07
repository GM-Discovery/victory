package playerprofile

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CommitPlayerProfilePage validates and stores one page's answers as a
// typed profile event, then recomputes effective facts. A commit that
// produces no material change over current facts creates no event
// (Kernel 61 §6.3, AC-13/AC-14) -- `changed` reports which happened.
func CommitPlayerProfilePage(ctx context.Context, pool *pgxpool.Pool, userID, pageKey string, answers map[string]any) (event ProfileEvent, changed bool, err error) {
	userID = strings.TrimSpace(userID)
	pageKey = strings.TrimSpace(pageKey)
	if userID == "" {
		return ProfileEvent{}, false, errors.New("not_authenticated")
	}
	if pageKey == "" {
		return ProfileEvent{}, false, errors.New("page_key_required")
	}

	cat, err := LoadCatalogue()
	if err != nil {
		return ProfileEvent{}, false, err
	}
	page, ok := cat.PageByKey(pageKey)
	if !ok {
		return ProfileEvent{}, false, errors.New("unknown_page")
	}

	sanitized, err := ValidatePageAnswers(page, answers)
	if err != nil {
		return ProfileEvent{}, false, err
	}

	wb, err := EnsureWorkbook(ctx, pool, userID)
	if err != nil {
		return ProfileEvent{}, false, err
	}

	events, err := loadWorkbookEvents(ctx, pool, wb.ID)
	if err != nil {
		return ProfileEvent{}, false, err
	}
	currentFacts := DeriveEffectiveFacts(cat, events)

	if !PageAnswersDifferFromFacts(sanitized, currentFacts) {
		return ProfileEvent{}, false, nil
	}

	summary := BuildPageCommitSummary(page, sanitized)
	newEvent, err := insertProfileEvent(ctx, pool, wb.ID, "page_commit", pageKey, cat.CatalogueVersion, sanitized, summary, userID)
	if err != nil {
		return ProfileEvent{}, false, err
	}

	if err := RecomputePlayerFacts(ctx, pool, wb.ID); err != nil {
		return ProfileEvent{}, false, err
	}
	if err := touchWorkbookProjection(ctx, pool, wb.ID); err != nil {
		return ProfileEvent{}, false, err
	}

	return newEvent, true, nil
}

// DeletePlayerProfileEvent removes one ordinary profile event owned by
// userID and recomputes effective facts. It cannot target stage-name ledger
// rows -- those live in a separate table with no delete path here
// (Kernel 61 §6.3, §8.5, AC-8, AC-15/AC-16/AC-17).
func DeletePlayerProfileEvent(ctx context.Context, pool *pgxpool.Pool, userID, eventID string) error {
	userID = strings.TrimSpace(userID)
	eventID = strings.TrimSpace(eventID)
	if userID == "" {
		return errors.New("not_authenticated")
	}
	if eventID == "" {
		return errors.New("event_id_required")
	}

	wb, err := EnsureWorkbook(ctx, pool, userID)
	if err != nil {
		return err
	}

	tag, err := pool.Exec(ctx, `
		DELETE FROM player_profile_events
		WHERE id = $1 AND workbook_id = $2
	`, eventID, wb.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("event_not_found")
	}

	if err := RecomputePlayerFacts(ctx, pool, wb.ID); err != nil {
		return err
	}
	return touchWorkbookProjection(ctx, pool, wb.ID)
}

// RecomputePlayerFacts rebuilds player_profile_facts from every remaining
// event for a workbook. It is safe to call after any event insert or
// delete -- it fully replaces the fact set rather than patching it
// incrementally, so there is no drift between events and facts
// (Kernel 61 §6.4).
func RecomputePlayerFacts(ctx context.Context, pool *pgxpool.Pool, workbookID string) error {
	cat, err := LoadCatalogue()
	if err != nil {
		return err
	}

	events, err := loadWorkbookEvents(ctx, pool, workbookID)
	if err != nil {
		return err
	}
	facts := DeriveEffectiveFacts(cat, events)

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM player_profile_facts WHERE workbook_id = $1`, workbookID); err != nil {
		return err
	}

	for _, fact := range facts {
		encoded, err := jsonEncode(fact.ValueJSON)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO player_profile_facts
				(workbook_id, field_key, value_json, display_value, source_event_id, source_page_key, catalogue_version, effective_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, workbookID, fact.FieldKey, encoded, fact.DisplayValue, fact.SourceEventID, fact.SourcePageKey, fact.CatalogueVersion, fact.EffectiveAt); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func loadWorkbookEvents(ctx context.Context, pool *pgxpool.Pool, workbookID string) ([]ProfileEvent, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, workbook_id::text, event_type, page_key, catalogue_version, payload, human_summary, created_by_user_id::text, created_at
		FROM player_profile_events
		WHERE workbook_id = $1
		ORDER BY created_at ASC
	`, workbookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ProfileEvent
	for rows.Next() {
		var e ProfileEvent
		var payload map[string]any
		if err := rows.Scan(&e.ID, &e.WorkbookID, &e.EventType, &e.PageKey, &e.CatalogueVersion, &payload, &e.HumanSummary, &e.CreatedByUserID, &e.CreatedAt); err != nil {
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

func insertProfileEvent(ctx context.Context, pool *pgxpool.Pool, workbookID, eventType, pageKey, catalogueVersion string, payload map[string]any, summary, createdByUserID string) (ProfileEvent, error) {
	var e ProfileEvent
	if err := pool.QueryRow(ctx, `
		INSERT INTO player_profile_events (workbook_id, event_type, page_key, catalogue_version, payload, human_summary, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id::text, workbook_id::text, event_type, page_key, catalogue_version, payload, human_summary, created_by_user_id::text, created_at
	`, workbookID, eventType, pageKey, catalogueVersion, payload, summary, createdByUserID).Scan(
		&e.ID, &e.WorkbookID, &e.EventType, &e.PageKey, &e.CatalogueVersion, &e.Payload, &e.HumanSummary, &e.CreatedByUserID, &e.CreatedAt,
	); err != nil {
		return ProfileEvent{}, err
	}
	return e, nil
}

// jsonEncode always produces pre-encoded JSON bytes for a value_json column.
// pgx's jsonb codec treats a bare Go `string` as already-valid JSON text
// (so an unquoted value like `Straturli` fails to parse as JSON) but
// correctly marshals other types -- explicitly marshaling here avoids that
// string-specific footgun regardless of what a fact's value happens to be.
func jsonEncode(v any) ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}
	return json.Marshal(v)
}

// LoadPlayerProfileEvents returns ordinary history rows for the owner view,
// newest first (Kernel 61 §7.8, §10.6).
func LoadPlayerProfileEvents(ctx context.Context, pool *pgxpool.Pool, userID string) ([]ProfileEvent, error) {
	wb, err := EnsureWorkbook(ctx, pool, userID)
	if err != nil {
		return nil, err
	}
	events, err := loadWorkbookEvents(ctx, pool, wb.ID)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].CreatedAt.After(events[j].CreatedAt) })
	return events, nil
}
