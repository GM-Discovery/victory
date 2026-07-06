package characters

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FaceOverride is owner curation plus Director override/lock metadata for one
// Face-eligible fact (Kernel 59A §5).
type FaceOverride struct {
	FactKey             string `json:"fact_key"`
	VisibilityMode      string `json:"visibility_mode"`
	VisibilityLocked    bool   `json:"visibility_locked"`
	PriorityMode        string `json:"priority_mode"`
	PriorityScore       int    `json:"priority_score"`
	PriorityLocked      bool   `json:"priority_locked"`
	ValueLocked         bool   `json:"value_locked"`
	ValueOverrideActive bool   `json:"value_override_active"`
	ValueOverride       string `json:"value_override,omitempty"`
}

const (
	faceVisibilityInferred = "inferred"
	faceVisibilityShown    = "shown"
	faceVisibilityHidden   = "hidden"

	facePriorityInferred = "inferred"
	facePriorityManual   = "manual"
)

const (
	faceLockDimensionValue      = "value"
	faceLockDimensionVisibility = "visibility"
	faceLockDimensionPriority   = "priority"
)

// loadFaceOverrides returns every override row for a character, keyed by
// fact_key. Missing keys mean "inferred" -- callers should not assume a
// zero-value FaceOverride is present in the map.
func loadFaceOverrides(ctx context.Context, pool *pgxpool.Pool, cardID string) (map[string]FaceOverride, error) {
	rows, err := pool.Query(ctx, `
		SELECT fact_key, visibility_mode, visibility_locked, priority_mode, COALESCE(priority_score, 0), priority_locked, value_locked, COALESCE(value_override_active, FALSE), COALESCE(value_override, '')
		FROM character_face_overrides
		WHERE character_card_id = $1
	`, cardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]FaceOverride{}
	for rows.Next() {
		var o FaceOverride
		if err := rows.Scan(&o.FactKey, &o.VisibilityMode, &o.VisibilityLocked, &o.PriorityMode, &o.PriorityScore, &o.PriorityLocked, &o.ValueLocked, &o.ValueOverrideActive, &o.ValueOverride); err != nil {
			return nil, err
		}
		out[o.FactKey] = o
	}
	return out, rows.Err()
}

// faceFieldByKey finds an eligible Face field by its Key. Overrides may
// only target facts the current fact contract actually produced this
// request -- Kernel 59A §6.7: "must operate on typed eligible facts."
func faceFieldByKey(fields []WorkbookPageField, key string) (WorkbookPageField, bool) {
	for _, f := range fields {
		if f.Key == key {
			return f, true
		}
	}
	return WorkbookPageField{}, false
}

// resolveFaceEligibility loads the current Face-eligible field list for a
// character so the mutation functions below can validate a fact_key before
// writing an override for it.
func resolveFaceEligibility(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID string) ([]WorkbookPageField, error) {
	card, module, entries, journals, err := loadWorkbookState(ctx, pool, actorUserID, cardID)
	if err != nil {
		return nil, err
	}
	pages := buildWorkbookPages(card, module, entries, journals)
	for _, page := range pages {
		if page.Key == "face" {
			return page.Fields, nil
		}
	}
	return nil, nil
}

func canDirectorLockCard(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID string) (bool, error) {
	var locationID string
	if err := pool.QueryRow(ctx, `
		SELECT location_id::text
		FROM character_cards
		WHERE id = $1
		  AND is_deleted = FALSE
		LIMIT 1
	`, cardID).Scan(&locationID); err != nil {
		return false, err
	}
	return hasAuthorityInLocation(ctx, pool, actorUserID, locationID)
}

func recordFaceOverrideEvent(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID, entryType, title, body string, payload map[string]any) error {
	_, err := RecordWorkbookEvents(ctx, pool, actorUserID, WorkbookEventRequest{
		CharacterCardID: cardID,
		Entries: []WorkbookEventInput{
			{PageKey: "history", EntryType: entryType, Title: title, Body: body, Payload: payload},
		},
	})
	return err
}

func scanFaceOverride(row interface {
	Scan(dest ...any) error
}, out *FaceOverride) error {
	return row.Scan(&out.FactKey, &out.VisibilityMode, &out.VisibilityLocked, &out.PriorityMode, &out.PriorityScore, &out.PriorityLocked, &out.ValueLocked, &out.ValueOverrideActive, &out.ValueOverride)
}

// SetCharacterFactFaceVisibility lets the owner explicitly show or hide an
// eligible fact on Face (Kernel 59A §3.1, §6.4). mode must be "shown" or
// "hidden" -- use ReturnCharacterFactFaceVisibilityToInferred to reset.
func SetCharacterFactFaceVisibility(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID, factKey, mode string) (FaceOverride, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	cardID = strings.TrimSpace(cardID)
	factKey = strings.TrimSpace(factKey)
	mode = strings.TrimSpace(strings.ToLower(mode))

	if actorUserID == "" {
		return FaceOverride{}, errors.New("not_authenticated")
	}
	if cardID == "" {
		return FaceOverride{}, errors.New("character_card_id_required")
	}
	if factKey == "" {
		return FaceOverride{}, errors.New("fact_key_required")
	}
	if mode != faceVisibilityShown && mode != faceVisibilityHidden {
		return FaceOverride{}, errors.New("invalid_visibility_mode")
	}

	allowed, err := CanEditCard(ctx, pool, actorUserID, cardID)
	if err != nil {
		return FaceOverride{}, err
	}
	if !allowed {
		return FaceOverride{}, errors.New("forbidden")
	}

	fields, err := resolveFaceEligibility(ctx, pool, actorUserID, cardID)
	if err != nil {
		return FaceOverride{}, err
	}
	field, ok := faceFieldByKey(fields, factKey)
	if !ok {
		return FaceOverride{}, errors.New("unknown_fact_key")
	}

	existing, err := loadFaceOverrides(ctx, pool, cardID)
	if err != nil {
		return FaceOverride{}, err
	}
	if prior, ok := existing[factKey]; ok && prior.VisibilityLocked {
		return FaceOverride{}, errors.New("face_visibility_locked")
	}

	var out FaceOverride
	if err := scanFaceOverride(pool.QueryRow(ctx, `
		INSERT INTO character_face_overrides (character_card_id, fact_key, visibility_mode)
		VALUES ($1, $2, $3)
		ON CONFLICT (character_card_id, fact_key)
		DO UPDATE SET visibility_mode = EXCLUDED.visibility_mode, updated_at = NOW()
		RETURNING fact_key, visibility_mode, visibility_locked, priority_mode, COALESCE(priority_score, 0), priority_locked, value_locked, COALESCE(value_override_active, FALSE), COALESCE(value_override, '')
	`, cardID, factKey, mode), &out); err != nil {
		return FaceOverride{}, err
	}

	entryType := "character_face_visibility_set"
	verb := "shown on"
	if mode == faceVisibilityHidden {
		verb = "hidden from"
	}
	if err := recordFaceOverrideEvent(ctx, pool, actorUserID, cardID, entryType, "Face visibility changed",
		strings.TrimSpace(field.Label)+" was "+verb+" Face.",
		map[string]any{"fact_key": factKey, "visibility_mode": mode},
	); err != nil {
		return FaceOverride{}, err
	}

	return out, nil
}

// ReturnCharacterFactFaceVisibilityToInferred resets a fact's Face
// visibility to contract-inferred behavior (Kernel 59A §3.1, §6.4).
func ReturnCharacterFactFaceVisibilityToInferred(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID, factKey string) (FaceOverride, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	cardID = strings.TrimSpace(cardID)
	factKey = strings.TrimSpace(factKey)

	if actorUserID == "" {
		return FaceOverride{}, errors.New("not_authenticated")
	}
	if cardID == "" {
		return FaceOverride{}, errors.New("character_card_id_required")
	}
	if factKey == "" {
		return FaceOverride{}, errors.New("fact_key_required")
	}

	allowed, err := CanEditCard(ctx, pool, actorUserID, cardID)
	if err != nil {
		return FaceOverride{}, err
	}
	if !allowed {
		return FaceOverride{}, errors.New("forbidden")
	}

	existing, err := loadFaceOverrides(ctx, pool, cardID)
	if err != nil {
		return FaceOverride{}, err
	}
	prior, hadOverride := existing[factKey]
	if hadOverride && prior.VisibilityLocked {
		return FaceOverride{}, errors.New("face_visibility_locked")
	}
	if !hadOverride {
		return FaceOverride{FactKey: factKey, VisibilityMode: faceVisibilityInferred, PriorityMode: facePriorityInferred}, nil
	}

	var out FaceOverride
	if err := scanFaceOverride(pool.QueryRow(ctx, `
		UPDATE character_face_overrides
		SET visibility_mode = 'inferred', updated_at = NOW()
		WHERE character_card_id = $1 AND fact_key = $2
		RETURNING fact_key, visibility_mode, visibility_locked, priority_mode, COALESCE(priority_score, 0), priority_locked, value_locked, COALESCE(value_override_active, FALSE), COALESCE(value_override, '')
	`, cardID, factKey), &out); err != nil {
		return FaceOverride{}, err
	}

	if err := recordFaceOverrideEvent(ctx, pool, actorUserID, cardID, "character_face_visibility_returned_to_inferred", "Face visibility reset",
		factKey+" returned to inferred Face visibility.",
		map[string]any{"fact_key": factKey},
	); err != nil {
		return FaceOverride{}, err
	}

	return out, nil
}

// SetCharacterFactPriority sets a manual signed-integer priority for an
// eligible fact (Kernel 59A §3.2, §6.4).
func SetCharacterFactPriority(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID, factKey string, score int) (FaceOverride, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	cardID = strings.TrimSpace(cardID)
	factKey = strings.TrimSpace(factKey)

	if actorUserID == "" {
		return FaceOverride{}, errors.New("not_authenticated")
	}
	if cardID == "" {
		return FaceOverride{}, errors.New("character_card_id_required")
	}
	if factKey == "" {
		return FaceOverride{}, errors.New("fact_key_required")
	}

	allowed, err := CanEditCard(ctx, pool, actorUserID, cardID)
	if err != nil {
		return FaceOverride{}, err
	}
	if !allowed {
		return FaceOverride{}, errors.New("forbidden")
	}

	fields, err := resolveFaceEligibility(ctx, pool, actorUserID, cardID)
	if err != nil {
		return FaceOverride{}, err
	}
	field, ok := faceFieldByKey(fields, factKey)
	if !ok {
		return FaceOverride{}, errors.New("unknown_fact_key")
	}

	existing, err := loadFaceOverrides(ctx, pool, cardID)
	if err != nil {
		return FaceOverride{}, err
	}
	if prior, ok := existing[factKey]; ok && prior.PriorityLocked {
		return FaceOverride{}, errors.New("face_priority_locked")
	}

	var out FaceOverride
	if err := scanFaceOverride(pool.QueryRow(ctx, `
		INSERT INTO character_face_overrides (character_card_id, fact_key, priority_mode, priority_score)
		VALUES ($1, $2, 'manual', $3)
		ON CONFLICT (character_card_id, fact_key)
		DO UPDATE SET priority_mode = 'manual', priority_score = EXCLUDED.priority_score, updated_at = NOW()
		RETURNING fact_key, visibility_mode, visibility_locked, priority_mode, COALESCE(priority_score, 0), priority_locked, value_locked, COALESCE(value_override_active, FALSE), COALESCE(value_override, '')
	`, cardID, factKey, score), &out); err != nil {
		return FaceOverride{}, err
	}

	if err := recordFaceOverrideEvent(ctx, pool, actorUserID, cardID, "character_face_priority_set", "Face priority changed",
		strings.TrimSpace(field.Label)+" priority set manually.",
		map[string]any{"fact_key": factKey, "priority_score": score},
	); err != nil {
		return FaceOverride{}, err
	}

	return out, nil
}

// SetDirectorCharacterFactLock independently locks or unlocks one fact
// dimension using the lock columns reserved in migration 034.
func SetDirectorCharacterFactLock(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID, factKey, dimension string, locked bool, reason string) (FaceOverride, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	cardID = strings.TrimSpace(cardID)
	factKey = strings.TrimSpace(factKey)
	dimension = strings.TrimSpace(strings.ToLower(dimension))
	reason = strings.TrimSpace(reason)

	if actorUserID == "" {
		return FaceOverride{}, errors.New("not_authenticated")
	}
	if cardID == "" {
		return FaceOverride{}, errors.New("character_card_id_required")
	}
	if factKey == "" {
		return FaceOverride{}, errors.New("fact_key_required")
	}
	if reason == "" {
		return FaceOverride{}, errors.New("director_reason_required")
	}
	switch dimension {
	case faceLockDimensionValue, faceLockDimensionVisibility, faceLockDimensionPriority:
	default:
		return FaceOverride{}, errors.New("invalid_lock_dimension")
	}

	allowed, err := canDirectorLockCard(ctx, pool, actorUserID, cardID)
	if err != nil {
		return FaceOverride{}, err
	}
	if !allowed {
		return FaceOverride{}, errors.New("director_authority_required")
	}

	fields, err := resolveFaceEligibility(ctx, pool, actorUserID, cardID)
	if err != nil {
		return FaceOverride{}, err
	}
	field, ok := faceFieldByKey(fields, factKey)
	if !ok {
		return FaceOverride{}, errors.New("unknown_fact_key")
	}

	var out FaceOverride
	query := `
		INSERT INTO character_face_overrides (character_card_id, fact_key, value_locked)
		VALUES ($1, $2, $3)
		ON CONFLICT (character_card_id, fact_key)
		DO UPDATE SET value_locked = EXCLUDED.value_locked, updated_at = NOW()
		RETURNING fact_key, visibility_mode, visibility_locked, priority_mode, COALESCE(priority_score, 0), priority_locked, value_locked, COALESCE(value_override_active, FALSE), COALESCE(value_override, '')
	`
	switch dimension {
	case faceLockDimensionVisibility:
		query = `
			INSERT INTO character_face_overrides (character_card_id, fact_key, visibility_locked)
			VALUES ($1, $2, $3)
			ON CONFLICT (character_card_id, fact_key)
			DO UPDATE SET visibility_locked = EXCLUDED.visibility_locked, updated_at = NOW()
			RETURNING fact_key, visibility_mode, visibility_locked, priority_mode, COALESCE(priority_score, 0), priority_locked, value_locked, COALESCE(value_override_active, FALSE), COALESCE(value_override, '')
		`
	case faceLockDimensionPriority:
		query = `
			INSERT INTO character_face_overrides (character_card_id, fact_key, priority_locked)
			VALUES ($1, $2, $3)
			ON CONFLICT (character_card_id, fact_key)
			DO UPDATE SET priority_locked = EXCLUDED.priority_locked, updated_at = NOW()
			RETURNING fact_key, visibility_mode, visibility_locked, priority_mode, COALESCE(priority_score, 0), priority_locked, value_locked, COALESCE(value_override_active, FALSE), COALESCE(value_override, '')
		`
	}
	if err := scanFaceOverride(pool.QueryRow(ctx, query, cardID, factKey, locked), &out); err != nil {
		return FaceOverride{}, err
	}

	state := "unlocked"
	if locked {
		state = "locked"
	}
	if err := recordFaceOverrideEvent(ctx, pool, actorUserID, cardID, "character_fact_lock_changed", "Character fact lock changed",
		strings.TrimSpace(field.Label)+" "+dimension+" was "+state+".",
		map[string]any{
			"fact_key":  factKey,
			"dimension": dimension,
			"locked":    locked,
			"reason":    reason,
		},
	); err != nil {
		return FaceOverride{}, err
	}

	return out, nil
}

func SetDirectorCharacterFactValueOverride(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID, factKey, value, reason string) (FaceOverride, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	cardID = strings.TrimSpace(cardID)
	factKey = strings.TrimSpace(factKey)
	value = strings.TrimSpace(value)
	reason = strings.TrimSpace(reason)

	if actorUserID == "" {
		return FaceOverride{}, errors.New("not_authenticated")
	}
	if cardID == "" {
		return FaceOverride{}, errors.New("character_card_id_required")
	}
	if factKey == "" {
		return FaceOverride{}, errors.New("fact_key_required")
	}
	if value == "" {
		return FaceOverride{}, errors.New("override_value_required")
	}
	if reason == "" {
		return FaceOverride{}, errors.New("director_reason_required")
	}

	allowed, err := canDirectorLockCard(ctx, pool, actorUserID, cardID)
	if err != nil {
		return FaceOverride{}, err
	}
	if !allowed {
		return FaceOverride{}, errors.New("director_authority_required")
	}

	fields, err := resolveFaceEligibility(ctx, pool, actorUserID, cardID)
	if err != nil {
		return FaceOverride{}, err
	}
	field, ok := faceFieldByKey(fields, factKey)
	if !ok {
		return FaceOverride{}, errors.New("unknown_fact_key")
	}

	var out FaceOverride
	if err := scanFaceOverride(pool.QueryRow(ctx, `
		INSERT INTO character_face_overrides (character_card_id, fact_key, value_override_active, value_override)
		VALUES ($1, $2, TRUE, $3)
		ON CONFLICT (character_card_id, fact_key)
		DO UPDATE SET value_override_active = TRUE, value_override = EXCLUDED.value_override, updated_at = NOW()
		RETURNING fact_key, visibility_mode, visibility_locked, priority_mode, COALESCE(priority_score, 0), priority_locked, value_locked, COALESCE(value_override_active, FALSE), COALESCE(value_override, '')
	`, cardID, factKey, value), &out); err != nil {
		return FaceOverride{}, err
	}

	if err := recordFaceOverrideEvent(ctx, pool, actorUserID, cardID, "character_fact_director_override", "Character fact overridden",
		strings.TrimSpace(field.Label)+" was overridden by a Director.",
		map[string]any{
			"fact_key":       factKey,
			"previous_value": field.Value,
			"override_value": value,
			"reason":         reason,
		},
	); err != nil {
		return FaceOverride{}, err
	}

	return out, nil
}

func ClearDirectorCharacterFactValueOverride(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID, factKey, reason string) (FaceOverride, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	cardID = strings.TrimSpace(cardID)
	factKey = strings.TrimSpace(factKey)
	reason = strings.TrimSpace(reason)

	if actorUserID == "" {
		return FaceOverride{}, errors.New("not_authenticated")
	}
	if cardID == "" {
		return FaceOverride{}, errors.New("character_card_id_required")
	}
	if factKey == "" {
		return FaceOverride{}, errors.New("fact_key_required")
	}
	if reason == "" {
		return FaceOverride{}, errors.New("director_reason_required")
	}

	allowed, err := canDirectorLockCard(ctx, pool, actorUserID, cardID)
	if err != nil {
		return FaceOverride{}, err
	}
	if !allowed {
		return FaceOverride{}, errors.New("director_authority_required")
	}

	fields, err := resolveFaceEligibility(ctx, pool, actorUserID, cardID)
	if err != nil {
		return FaceOverride{}, err
	}
	field, ok := faceFieldByKey(fields, factKey)
	if !ok {
		return FaceOverride{}, errors.New("unknown_fact_key")
	}

	var out FaceOverride
	if err := scanFaceOverride(pool.QueryRow(ctx, `
		INSERT INTO character_face_overrides (character_card_id, fact_key, value_override_active, value_override)
		VALUES ($1, $2, FALSE, '')
		ON CONFLICT (character_card_id, fact_key)
		DO UPDATE SET value_override_active = FALSE, value_override = '', updated_at = NOW()
		RETURNING fact_key, visibility_mode, visibility_locked, priority_mode, COALESCE(priority_score, 0), priority_locked, value_locked, COALESCE(value_override_active, FALSE), COALESCE(value_override, '')
	`, cardID, factKey), &out); err != nil {
		return FaceOverride{}, err
	}

	if err := recordFaceOverrideEvent(ctx, pool, actorUserID, cardID, "character_fact_director_override_cleared", "Character fact override cleared",
		strings.TrimSpace(field.Label)+" Director override was cleared.",
		map[string]any{
			"fact_key": factKey,
			"reason":   reason,
		},
	); err != nil {
		return FaceOverride{}, err
	}

	return out, nil
}

// ReturnCharacterFactPriorityToInferred resets a fact's priority to the
// contract-computed default (Kernel 59A §3.2, §6.4).
func ReturnCharacterFactPriorityToInferred(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID, factKey string) (FaceOverride, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	cardID = strings.TrimSpace(cardID)
	factKey = strings.TrimSpace(factKey)

	if actorUserID == "" {
		return FaceOverride{}, errors.New("not_authenticated")
	}
	if cardID == "" {
		return FaceOverride{}, errors.New("character_card_id_required")
	}
	if factKey == "" {
		return FaceOverride{}, errors.New("fact_key_required")
	}

	allowed, err := CanEditCard(ctx, pool, actorUserID, cardID)
	if err != nil {
		return FaceOverride{}, err
	}
	if !allowed {
		return FaceOverride{}, errors.New("forbidden")
	}

	existing, err := loadFaceOverrides(ctx, pool, cardID)
	if err != nil {
		return FaceOverride{}, err
	}
	prior, hadOverride := existing[factKey]
	if hadOverride && prior.PriorityLocked {
		return FaceOverride{}, errors.New("face_priority_locked")
	}
	if !hadOverride {
		return FaceOverride{FactKey: factKey, VisibilityMode: faceVisibilityInferred, PriorityMode: facePriorityInferred}, nil
	}

	var out FaceOverride
	if err := scanFaceOverride(pool.QueryRow(ctx, `
		UPDATE character_face_overrides
		SET priority_mode = 'inferred', priority_score = NULL, updated_at = NOW()
		WHERE character_card_id = $1 AND fact_key = $2
		RETURNING fact_key, visibility_mode, visibility_locked, priority_mode, COALESCE(priority_score, 0), priority_locked, value_locked, COALESCE(value_override_active, FALSE), COALESCE(value_override, '')
	`, cardID, factKey), &out); err != nil {
		return FaceOverride{}, err
	}

	if err := recordFaceOverrideEvent(ctx, pool, actorUserID, cardID, "character_face_priority_returned_to_inferred", "Face priority reset",
		factKey+" returned to inferred priority.",
		map[string]any{"fact_key": factKey},
	); err != nil {
		return FaceOverride{}, err
	}

	return out, nil
}
