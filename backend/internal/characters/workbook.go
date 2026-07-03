package characters

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func normalizeWorkbookContext(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}

	out := make(map[string]any, len(input))
	for key, value := range input {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		out[trimmed] = value
	}
	if out == nil {
		return map[string]any{}
	}
	return out
}

func decodeJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return map[string]any{}
	}
	return out
}

func findDraftCharacterForContext(ctx context.Context, pool *pgxpool.Pool, ownerUserID string, workbookContextJSON []byte) (CharacterCard, bool, error) {
	draftToken := ""
	if len(workbookContextJSON) > 0 {
		var parsed map[string]any
		if err := json.Unmarshal(workbookContextJSON, &parsed); err == nil {
			draftToken = strings.TrimSpace(stringValue(parsed["draft_token"]))
		}
	}
	if draftToken == "" {
		return CharacterCard{}, false, nil
	}

	var card CharacterCard
	var createdAt, updatedAt time.Time
	var sheetLinksRaw, workbookContextRaw []byte
	err := pool.QueryRow(ctx, `
		SELECT id::text, owner_user_id::text, location_id::text, COALESCE(production_id::text, ''), workbook_status, workbook_context, name, pronouns, portrait_url, COALESCE(token_aura, ''), color, tagline, public_description, private_notes, sheet_links, created_at, updated_at
		FROM character_cards
		WHERE owner_user_id = $1
		  AND is_deleted = FALSE
		  AND workbook_status = 'draft'
		  AND workbook_context ->> 'draft_token' = $2
		ORDER BY updated_at DESC, created_at DESC
		LIMIT 1
	`, ownerUserID, draftToken).Scan(&card.ID, &card.OwnerUserID, &card.LocationID, &card.ProductionID, &card.WorkbookStatus, &workbookContextRaw, &card.Name, &card.Pronouns, &card.PortraitURL, &card.TokenAura, &card.Color, &card.Tagline, &card.PublicDescription, &card.PrivateNotes, &sheetLinksRaw, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CharacterCard{}, false, nil
		}
		return CharacterCard{}, false, err
	}

	card.WorkbookContext = decodeJSONMap(workbookContextRaw)
	if err := finishCharacterCard(&card, sheetLinksRaw, createdAt, updatedAt); err != nil {
		return CharacterCard{}, false, err
	}
	return card, true, nil
}

func ensureWorkbookModule(ctx context.Context, pool *pgxpool.Pool, characterCardID string, workbookContext map[string]any, workbookStatus string) error {
	if strings.ToLower(strings.TrimSpace(stringValue(workbookContext["source"]))) != "catharsis" {
		return nil
	}

	moduleContextJSON, err := json.Marshal(normalizeWorkbookContext(workbookContext))
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO character_workbook_modules (
			character_card_id,
			ruleset_key,
			ruleset_version,
			module_status,
			creation_flow_version,
			parentage_chart_version,
			current_stage,
			current_event,
			module_context
		)
		VALUES ($1, 'socio', '1.1', $2, 'v1', '1.1', 1, 'egg_donor_parentage', $3::jsonb)
		ON CONFLICT (character_card_id, ruleset_key)
		DO UPDATE SET
			module_status = EXCLUDED.module_status,
			module_context = EXCLUDED.module_context,
			updated_at = NOW()
	`, characterCardID, strings.TrimSpace(workbookStatus), string(moduleContextJSON))
	return err
}

func setActiveCharacter(ctx context.Context, pool *pgxpool.Pool, userID, characterCardID string) error {
	userID = strings.TrimSpace(userID)
	characterCardID = strings.TrimSpace(characterCardID)
	if userID == "" || characterCardID == "" {
		return nil
	}

	_, err := pool.Exec(ctx, `
		INSERT INTO active_user_characters (user_id, character_card_id, activated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id)
		DO UPDATE SET character_card_id = EXCLUDED.character_card_id, activated_at = EXCLUDED.activated_at
	`, userID, characterCardID)
	return err
}

func ActiveCharacterForUser(ctx context.Context, q characterQuerier, userID string) (map[string]any, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}

	var cardID, name, pronouns, portraitURL, tokenAura, color, tagline, status string
	var contextRaw []byte
	err := q.QueryRow(ctx, `
		SELECT cc.id::text, cc.name, cc.pronouns, cc.portrait_url, COALESCE(cc.token_aura, ''), cc.color, cc.tagline, cc.workbook_status, cc.workbook_context
		FROM active_user_characters auc
		JOIN character_cards cc ON cc.id = auc.character_card_id
		WHERE auc.user_id = $1
		  AND cc.is_deleted = FALSE
		LIMIT 1
	`, userID).Scan(&cardID, &name, &pronouns, &portraitURL, &tokenAura, &color, &tagline, &status, &contextRaw)
	if err == nil {
		return map[string]any{
			"character_card_id": cardID,
			"name":              name,
			"display_name":      name,
			"pronouns":          pronouns,
			"portrait_url":      portraitURL,
			"token_aura":        tokenAura,
			"color":             color,
			"tagline":           tagline,
			"workbook_status":   status,
			"workbook_context":  decodeJSONMap(contextRaw),
		}, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return nil, err
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return ""
	}
}
