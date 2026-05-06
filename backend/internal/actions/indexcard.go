package actions

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showings"
)

type IndexCardRequest struct {
	SessionID   string `json:"session_id"`
	ActorID     string `json:"actor_id"`
	ElementID   string `json:"element_id"`
	ElementSlug string `json:"element_slug"`
	FrontText   string `json:"front_text"`
	BackText    string `json:"back_text"`
	Color       string `json:"color"`
}

func StoreIndexCardCreate(ctx context.Context, pool *pgxpool.Pool, req IndexCardRequest) (*StoredAction, error) {
	req = sanitizeIndexCardRequest(req)
	if req.SessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if req.ActorID == "" {
		return nil, errors.New("actor_id is required")
	}
	if req.FrontText == "" && req.BackText == "" {
		return nil, errors.New("front_text_or_back_text_required")
	}
	if err := validateIndexCardLength(req.FrontText, req.BackText); err != nil {
		return nil, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	decision, err := CanAct(ctx, tx, req.ActorID, "create/index_card", req.SessionID, ActionTarget{Kind: "index_card"})
	if err != nil {
		return nil, err
	}
	if !decision.Allowed {
		return nil, &ActionDeniedError{Reason: decision.Reason}
	}

	showing, err := showings.EnsureForSession(ctx, tx, req.SessionID, req.ActorID)
	if err != nil {
		return nil, err
	}

	card, productionID, err := createIndexCard(ctx, tx, req)
	if err != nil {
		return nil, err
	}

	displayName, handle, role, persona, err := loadActorIdentity(ctx, tx, req.SessionID, req.ActorID)
	if err != nil {
		return nil, err
	}

	var nextMoment int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(moment_id), 0) + 1
		FROM actions
		WHERE session_id = $1
	`, req.SessionID).Scan(&nextMoment); err != nil {
		return nil, err
	}

	payload := indexCardPayload(card, productionID)
	payload["actor_persona"] = persona
	target := indexCardTarget(card)
	scope := map[string]any{
		"surfaces":         []string{"tray"},
		"audienceSegments": []string{"director", "producer"},
	}
	visibility := map[string]any{
		"toRoles":   []string{"director", "producer"},
		"privateTo": []string{},
	}

	targetJSON, _ := json.Marshal(target)
	payloadJSON, _ := json.Marshal(payload)
	scopeJSON, _ := json.Marshal(scope)
	visibilityJSON, _ := json.Marshal(visibility)

	var out StoredAction
	var ts time.Time
	if err := tx.QueryRow(ctx, `
		INSERT INTO actions (
			session_id,
			moment_id,
			actor_id,
			type,
			target,
			payload,
			scope,
			visibility,
			showing_id,
			recorded
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, TRUE)
		RETURNING id, ts
	`, req.SessionID, nextMoment, req.ActorID, "create/index_card", targetJSON, payloadJSON, scopeJSON, visibilityJSON, showing.ID).
		Scan(&out.ID, &ts); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	out.SessionID = req.SessionID
	out.ShowingID = showing.ID
	out.MomentID = nextMoment
	out.ActorID = req.ActorID
	out.ActorDisplayName = displayName
	out.ActorHandle = handle
	out.ActorRole = role
	out.Actor = map[string]any{
		"user_id":      req.ActorID,
		"handle":       handle,
		"display_name": displayName,
		"role":         role,
		"persona":      persona,
	}
	out.Persona = persona
	out.Type = "create/index_card"
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}

func StoreIndexCardUpdate(ctx context.Context, pool *pgxpool.Pool, req IndexCardRequest) (*StoredAction, error) {
	req = sanitizeIndexCardRequest(req)
	if req.SessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if req.ActorID == "" {
		return nil, errors.New("actor_id is required")
	}
	if req.ElementID == "" && req.ElementSlug == "" {
		return nil, errors.New("element_id or element_slug is required")
	}
	if err := validateIndexCardLength(req.FrontText, req.BackText); err != nil {
		return nil, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	decision, err := CanAct(ctx, tx, req.ActorID, "update/index_card", req.SessionID, ActionTarget{
		Kind:        "index_card",
		ElementID:   req.ElementID,
		ElementSlug: req.ElementSlug,
	})
	if err != nil {
		return nil, err
	}
	if !decision.Allowed {
		return nil, &ActionDeniedError{Reason: decision.Reason}
	}

	showing, err := showings.EnsureForSession(ctx, tx, req.SessionID, req.ActorID)
	if err != nil {
		return nil, err
	}

	card, productionID, err := updateIndexCard(ctx, tx, req)
	if err != nil {
		return nil, err
	}

	displayName, handle, role, persona, err := loadActorIdentity(ctx, tx, req.SessionID, req.ActorID)
	if err != nil {
		return nil, err
	}

	var nextMoment int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(moment_id), 0) + 1
		FROM actions
		WHERE session_id = $1
	`, req.SessionID).Scan(&nextMoment); err != nil {
		return nil, err
	}

	payload := indexCardPayload(card, productionID)
	payload["actor_persona"] = persona
	target := indexCardTarget(card)
	scope := map[string]any{
		"surfaces":         []string{"tray"},
		"audienceSegments": []string{"director", "producer"},
	}
	visibility := map[string]any{
		"toRoles":   []string{"director", "producer"},
		"privateTo": []string{},
	}

	targetJSON, _ := json.Marshal(target)
	payloadJSON, _ := json.Marshal(payload)
	scopeJSON, _ := json.Marshal(scope)
	visibilityJSON, _ := json.Marshal(visibility)

	var out StoredAction
	var ts time.Time
	if err := tx.QueryRow(ctx, `
		INSERT INTO actions (
			session_id,
			moment_id,
			actor_id,
			type,
			target,
			payload,
			scope,
			visibility,
			showing_id,
			recorded
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, TRUE)
		RETURNING id, ts
	`, req.SessionID, nextMoment, req.ActorID, "update/index_card", targetJSON, payloadJSON, scopeJSON, visibilityJSON, showing.ID).
		Scan(&out.ID, &ts); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	out.SessionID = req.SessionID
	out.ShowingID = showing.ID
	out.MomentID = nextMoment
	out.ActorID = req.ActorID
	out.ActorDisplayName = displayName
	out.ActorHandle = handle
	out.ActorRole = role
	out.Actor = map[string]any{
		"user_id":      req.ActorID,
		"handle":       handle,
		"display_name": displayName,
		"role":         role,
		"persona":      persona,
	}
	out.Persona = persona
	out.Type = "update/index_card"
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}

func StoreIndexCardDelete(ctx context.Context, pool *pgxpool.Pool, req IndexCardRequest) (*StoredAction, error) {
	req = sanitizeIndexCardRequest(req)
	if req.SessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if req.ActorID == "" {
		return nil, errors.New("actor_id is required")
	}
	if req.ElementID == "" && req.ElementSlug == "" {
		return nil, errors.New("element_id or element_slug is required")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	decision, err := CanAct(ctx, tx, req.ActorID, "delete/index_card", req.SessionID, ActionTarget{
		Kind:        "index_card",
		ElementID:   req.ElementID,
		ElementSlug: req.ElementSlug,
	})
	if err != nil {
		return nil, err
	}
	if !decision.Allowed {
		return nil, &ActionDeniedError{Reason: decision.Reason}
	}

	showing, err := showings.EnsureForSession(ctx, tx, req.SessionID, req.ActorID)
	if err != nil {
		return nil, err
	}

	resolvedID, resolvedSlug, err := resolveIndexCardTarget(ctx, tx, req.SessionID, req.ElementID, req.ElementSlug)
	if err != nil {
		return nil, err
	}

	var (
		existingData []byte
		existingName string
	)
	if err := tx.QueryRow(ctx, `
		SELECT
			e.data,
			e.name
		FROM elements e
		WHERE e.id = $1
		  AND e.element_type = 'index_card'
		LIMIT 1
	`, resolvedID).Scan(&existingData, &existingName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("index_card_not_found")
		}
		return nil, err
	}
	_ = existingName

	displayName, handle, role, persona, err := loadActorIdentity(ctx, tx, req.SessionID, req.ActorID)
	if err != nil {
		return nil, err
	}

	deletedAt := time.Now().UTC().Format(time.RFC3339)
	var existing map[string]any
	_ = json.Unmarshal(existingData, &existing)
	if existing == nil {
		existing = map[string]any{}
	}
	existing["deleted_at"] = deletedAt
	existing["deleted_by"] = req.ActorID
	existing["deleted_by_display_name"] = displayName
	existing["deleted_by_handle"] = handle
	existing["deleted_by_role"] = role
	existing["deleted_reason"] = "delete/index_card"
	existing["state"] = "deleted"

	deletedJSON, _ := json.Marshal(existing)

	if _, err := tx.Exec(ctx, `
		UPDATE elements
		SET state = 'deleted',
		    data = $2
		WHERE id = $1
		  AND element_type = 'index_card'
	`, resolvedID, deletedJSON); err != nil {
		return nil, err
	}

	var nextMoment int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(moment_id), 0) + 1
		FROM actions
		WHERE session_id = $1
	`, req.SessionID).Scan(&nextMoment); err != nil {
		return nil, err
	}

	target := map[string]any{
		"kind":         "index_card",
		"element_id":   resolvedID,
		"element_slug": resolvedSlug,
	}
	payload := map[string]any{
		"deleted_at":    deletedAt,
		"deleted_by":    req.ActorID,
		"actor_persona": persona,
	}
	scope := map[string]any{
		"surfaces":         []string{"tray", "stage"},
		"audienceSegments": []string{"director", "producer"},
	}
	visibility := map[string]any{
		"toRoles":   []string{"director", "producer"},
		"privateTo": []string{},
	}

	targetJSON, _ := json.Marshal(target)
	payloadJSON, _ := json.Marshal(payload)
	scopeJSON, _ := json.Marshal(scope)
	visibilityJSON, _ := json.Marshal(visibility)

	var out StoredAction
	var ts time.Time
	if err := tx.QueryRow(ctx, `
		INSERT INTO actions (
			session_id,
			showing_id,
			moment_id,
			actor_id,
			type,
			target,
			payload,
			scope,
			visibility,
			recorded
		)
		VALUES ($1, $2, $3, $4, 'delete/index_card', $5, $6, $7, $8, TRUE)
		RETURNING id, ts
	`, req.SessionID, showing.ID, nextMoment, req.ActorID, targetJSON, payloadJSON, scopeJSON, visibilityJSON).
		Scan(&out.ID, &ts); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	out.SessionID = req.SessionID
	out.ShowingID = showing.ID
	out.MomentID = nextMoment
	out.ActorID = req.ActorID
	out.ActorDisplayName = displayName
	out.ActorHandle = handle
	out.ActorRole = role
	out.Actor = map[string]any{
		"user_id":      req.ActorID,
		"handle":       handle,
		"display_name": displayName,
		"role":         role,
		"persona":      persona,
	}
	out.Persona = persona
	out.Type = "delete/index_card"
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}

type indexCardRecord struct {
	ElementID    string         `json:"element_id"`
	Slug         string         `json:"slug"`
	Name         string         `json:"name"`
	FrontText    string         `json:"front_text"`
	BackText     string         `json:"back_text"`
	Color        string         `json:"color"`
	CreatedBy    string         `json:"created_by"`
	CreatedAt    string         `json:"created_at"`
	UpdatedAt    string         `json:"updated_at"`
	ProductionID string         `json:"production_id"`
	VenueSlug    string         `json:"venue_slug"`
	SessionID    string         `json:"session_id"`
	Data         map[string]any `json:"data"`
}

func createIndexCard(ctx context.Context, tx pgx.Tx, req IndexCardRequest) (indexCardRecord, string, error) {
	locationID, productionID, venueSlug, err := resolveIndexCardScope(ctx, tx, req.SessionID, req.ActorID)
	if err != nil {
		return indexCardRecord{}, "", err
	}
	libraryID, err := resolveIndexCardLibrary(ctx, tx, locationID)
	if err != nil {
		return indexCardRecord{}, "", err
	}
	creatorDisplayName, creatorHandle, creatorRole, _, err := loadActorIdentity(ctx, tx, req.SessionID, req.ActorID)
	if err != nil {
		return indexCardRecord{}, "", err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	slug := uniqueIndexCardSlug(req.FrontText, req.ActorID)
	data := map[string]any{
		"type":                    "index_card",
		"context_class":           "card",
		"front_text":              req.FrontText,
		"back_text":               req.BackText,
		"color":                   req.Color,
		"created_by":              req.ActorID,
		"created_by_display_name": creatorDisplayName,
		"created_by_handle":       creatorHandle,
		"created_by_role":         creatorRole,
		"created_at":              now,
		"updated_at":              now,
		"production_id":           productionID,
		"session_id":              req.SessionID,
		"venue_slug":              venueSlug,
	}

	dataJSON, _ := json.Marshal(data)

	var elementID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO elements (
			library_id,
			name,
			slug,
			element_type,
			context_class,
			state,
			data
		)
		VALUES ($1, $2, $3, 'index_card', 'card', 'library', $4)
		RETURNING id::text
	`, libraryID, indexCardDisplayName(req.FrontText), slug, dataJSON).Scan(&elementID); err != nil {
		return indexCardRecord{}, "", err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO venue_layout_elements (
			venue_id,
			element_id,
			surface,
			position,
			visibility,
			is_default
		)
		SELECT
			v.id,
			$1,
			'tray',
			'{"anchor":"tray","x":0,"y":0,"z":0}'::jsonb,
			'{"toRoles":["director","producer"],"privateTo":[]}'::jsonb,
			FALSE
		FROM sessions s
		JOIN venues v ON v.id = s.venue_id
		WHERE s.id = $2
		  AND v.slug = 'the-cave'
	`, elementID, req.SessionID); err != nil {
		return indexCardRecord{}, "", err
	}

	return indexCardRecord{
		ElementID:    elementID,
		Slug:         slug,
		Name:         indexCardDisplayName(req.FrontText),
		FrontText:    req.FrontText,
		BackText:     req.BackText,
		Color:        req.Color,
		CreatedBy:    req.ActorID,
		CreatedAt:    now,
		UpdatedAt:    now,
		ProductionID: productionID,
		VenueSlug:    venueSlug,
		SessionID:    req.SessionID,
		Data:         data,
	}, productionID, nil
}

func updateIndexCard(ctx context.Context, tx pgx.Tx, req IndexCardRequest) (indexCardRecord, string, error) {
	resolvedID, resolvedSlug, err := resolveIndexCardTarget(ctx, tx, req.SessionID, req.ElementID, req.ElementSlug)
	if err != nil {
		return indexCardRecord{}, "", err
	}

	var (
		existingData []byte
	)
	if err := tx.QueryRow(ctx, `
		SELECT
			e.data
		FROM elements e
		WHERE e.id = $1
		  AND e.element_type = 'index_card'
		LIMIT 1
	`, resolvedID).Scan(&existingData); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return indexCardRecord{}, "", errors.New("index_card_not_found")
		}
		return indexCardRecord{}, "", err
	}

	var existing map[string]any
	_ = json.Unmarshal(existingData, &existing)
	if existing == nil {
		existing = map[string]any{}
	}

	createdBy := stringValue(existing["created_by"])
	createdByDisplayName := stringValue(existing["created_by_display_name"])
	createdByHandle := stringValue(existing["created_by_handle"])
	createdByRole := stringValue(existing["created_by_role"])
	createdAt := stringValue(existing["created_at"])
	venueSlug := stringValue(existing["venue_slug"])
	productionID := stringValue(existing["production_id"])
	if venueSlug == "" {
		venueSlug = "the-cave"
	}
	now := time.Now().UTC().Format(time.RFC3339)

	updatedData := map[string]any{
		"type":                    "index_card",
		"context_class":           "card",
		"front_text":              req.FrontText,
		"back_text":               req.BackText,
		"color":                   req.Color,
		"created_by":              createdBy,
		"created_by_display_name": createdByDisplayName,
		"created_by_handle":       createdByHandle,
		"created_by_role":         createdByRole,
		"created_at":              createdAt,
		"updated_at":              now,
		"production_id":           productionID,
		"session_id":              req.SessionID,
		"venue_slug":              venueSlug,
	}
	dataJSON, _ := json.Marshal(updatedData)

	if _, err := tx.Exec(ctx, `
		UPDATE elements
		SET name = $2,
		    context_class = 'card',
		    data = $3
		WHERE id = $1
		  AND element_type = 'index_card'
	`, resolvedID, indexCardDisplayName(req.FrontText), dataJSON); err != nil {
		return indexCardRecord{}, "", err
	}

	return indexCardRecord{
		ElementID:    resolvedID,
		Slug:         resolvedSlug,
		Name:         indexCardDisplayName(req.FrontText),
		FrontText:    req.FrontText,
		BackText:     req.BackText,
		Color:        req.Color,
		CreatedBy:    createdBy,
		CreatedAt:    createdAt,
		UpdatedAt:    now,
		ProductionID: productionID,
		VenueSlug:    venueSlug,
		SessionID:    req.SessionID,
		Data:         updatedData,
	}, productionID, nil
}

func resolveIndexCardScope(ctx context.Context, tx pgx.Tx, sessionID, actorID string) (locationID, productionID, venueSlug string, err error) {
	err = tx.QueryRow(ctx, `
		SELECT
			l.id::text,
			v.slug
		FROM sessions s
		JOIN venues v ON v.id = s.venue_id
		JOIN lots lo ON lo.id = v.lot_id
		JOIN locations l ON l.id = lo.location_id
		WHERE s.id = $1
		  AND v.slug = 'the-cave'
		LIMIT 1
	`, sessionID).Scan(&locationID, &venueSlug)
	if err != nil {
		return "", "", "", err
	}

	err = tx.QueryRow(ctx, `
		SELECT COALESCE(m.production_id::text, '')
		FROM memberships m
		WHERE m.location_id = (
			SELECT l.id
			FROM sessions s
			JOIN venues v ON v.id = s.venue_id
			JOIN lots lo ON lo.id = v.lot_id
			JOIN locations l ON l.id = lo.location_id
			WHERE s.id = $1
			LIMIT 1
		)
		  AND m.user_id = $2
		  AND m.active = TRUE
		  AND m.production_id IS NOT NULL
		  AND m.role IN ('producer', 'director', 'cast', 'crew')
		ORDER BY
			CASE m.role
				WHEN 'producer' THEN 1
				WHEN 'director' THEN 2
				WHEN 'cast' THEN 3
				WHEN 'crew' THEN 4
				ELSE 99
			END,
			m.created_at ASC
		LIMIT 1
	`, sessionID, actorID).Scan(&productionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			productionID = ""
		} else {
			return "", "", "", err
		}
	}

	return locationID, productionID, venueSlug, nil
}

func resolveIndexCardLibrary(ctx context.Context, tx pgx.Tx, locationID string) (string, error) {
	var libraryID string
	err := tx.QueryRow(ctx, `
		SELECT id::text
		FROM libraries
		WHERE location_id = $1
		  AND name = 'house-library'
		LIMIT 1
	`, locationID).Scan(&libraryID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if err := tx.QueryRow(ctx, `
				INSERT INTO libraries (location_id, name)
				VALUES ($1, 'house-library')
				RETURNING id::text
			`, locationID).Scan(&libraryID); err != nil {
				return "", err
			}
			return libraryID, nil
		}
		return "", err
	}

	return libraryID, nil
}

func resolveIndexCardTarget(ctx context.Context, tx pgx.Tx, sessionID, elementID, elementSlug string) (resolvedID, resolvedSlug string, err error) {
	base := `
		SELECT e.id::text, e.slug
		FROM sessions s
		JOIN venues v ON v.id = s.venue_id
		JOIN venue_layout_elements vle ON vle.venue_id = v.id
		JOIN elements e ON e.id = vle.element_id
		WHERE s.id = $1
		  AND e.element_type = 'index_card'
	`

	switch {
	case elementID != "":
		err = tx.QueryRow(ctx, base+` AND e.id = $2 LIMIT 1`, sessionID, elementID).Scan(&resolvedID, &resolvedSlug)
	case elementSlug != "":
		err = tx.QueryRow(ctx, base+` AND e.slug = $2 LIMIT 1`, sessionID, elementSlug).Scan(&resolvedID, &resolvedSlug)
	default:
		return "", "", errors.New("element_id or element_slug is required")
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", errors.New("index_card_not_found")
		}
		return "", "", err
	}

	return resolvedID, resolvedSlug, nil
}

func indexCardDisplayName(frontText string) string {
	frontText = strings.TrimSpace(frontText)
	if frontText == "" {
		return "Index Card"
	}
	if utf8.RuneCountInString(frontText) <= 48 {
		return frontText
	}
	runes := []rune(frontText)
	if len(runes) <= 48 {
		return frontText
	}
	return strings.TrimSpace(string(runes[:48]))
}

func uniqueIndexCardSlug(frontText, actorID string) string {
	frontText = strings.ToLower(strings.TrimSpace(frontText))
	frontText = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-':
			return r
		default:
			return '-'
		}
	}, frontText)
	frontText = strings.Trim(frontText, "-")
	runes := []rune(frontText)
	if len(runes) > 32 {
		frontText = string(runes[:32])
	}
	actorID = strings.ToLower(strings.TrimSpace(actorID))
	if len(actorID) > 8 {
		actorID = actorID[:8]
	}
	tick := time.Now().UTC().UnixNano()
	if frontText == "" {
		frontText = "card"
	}
	if actorID == "" {
		actorID = "user"
	}
	return "index-card-" + actorID + "-" + frontText + "-" + strconv.FormatInt(tick, 10)
}

func sanitizeIndexCardRequest(req IndexCardRequest) IndexCardRequest {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ActorID = strings.TrimSpace(req.ActorID)
	req.ElementID = strings.TrimSpace(req.ElementID)
	req.ElementSlug = strings.TrimSpace(req.ElementSlug)
	req.FrontText = truncateRunes(strings.TrimSpace(req.FrontText), 2000)
	req.BackText = truncateRunes(strings.TrimSpace(req.BackText), 2000)
	req.Color = normalizeIndexCardColor(req.Color)
	return req
}

func truncateRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) <= max {
		return value
	}

	runes := []rune(value)
	if len(runes) <= max {
		return value
	}

	return string(runes[:max])
}

func validateIndexCardLength(frontText, backText string) error {
	total := utf8.RuneCountInString(frontText) + utf8.RuneCountInString(backText)
	if total > 2000 {
		return errors.New("index_card_content_too_long")
	}
	return nil
}

func normalizeIndexCardColor(color string) string {
	color = strings.TrimSpace(color)
	if color == "" {
		return "#d9c7a6"
	}
	if !strings.HasPrefix(color, "#") {
		return "#d9c7a6"
	}
	if len(color) == 4 || len(color) == 7 {
		return strings.ToLower(color)
	}
	return "#d9c7a6"
}

func indexCardTarget(card indexCardRecord) map[string]any {
	return map[string]any{
		"kind":         "index_card",
		"element_id":   card.ElementID,
		"element_slug": card.Slug,
	}
}

func indexCardPayload(card indexCardRecord, productionID string) map[string]any {
	return map[string]any{
		"front_text":              card.FrontText,
		"back_text":               card.BackText,
		"color":                   card.Color,
		"created_by":              card.CreatedBy,
		"created_by_display_name": stringValue(card.Data["created_by_display_name"]),
		"created_by_handle":       stringValue(card.Data["created_by_handle"]),
		"created_by_role":         stringValue(card.Data["created_by_role"]),
		"created_at":              card.CreatedAt,
		"updated_at":              card.UpdatedAt,
		"production_id":           productionID,
		"session_id":              card.SessionID,
		"venue_slug":              card.VenueSlug,
	}
}

func stringValue(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}
