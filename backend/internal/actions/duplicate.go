package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showings"
)

type DuplicateElementRequest struct {
	SessionID   string  `json:"session_id"`
	ActorID     string  `json:"actor_id"`
	ElementID   string  `json:"element_id"`
	ElementSlug string  `json:"element_slug"`
	VenueSlug   string  `json:"venue_slug"`
	Layer       string  `json:"layer"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Order       int     `json:"order"`
}

func StoreDuplicateElement(ctx context.Context, pool *pgxpool.Pool, req DuplicateElementRequest) (*StoredAction, error) {
	req = sanitizeDuplicateElementRequest(req)
	if req.SessionID == "" {
		return nil, errors.New("session_id is required")
	}
	if req.ActorID == "" {
		return nil, errors.New("actor_id is required")
	}
	if req.ElementID == "" && req.ElementSlug == "" {
		return nil, errors.New("element_id or element_slug is required")
	}
	if req.VenueSlug == "" {
		return nil, errors.New("venue_slug is required")
	}
	if req.Layer != "stage" {
		return nil, errors.New("invalid_layer")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	decision, err := CanAct(ctx, tx, req.ActorID, "act/duplicate_element", req.SessionID, ActionTarget{
		Kind:        "element",
		ElementID:   req.ElementID,
		ElementSlug: req.ElementSlug,
		VenueSlug:   req.VenueSlug,
		Layer:       req.Layer,
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

	source, err := resolveVenueLayoutElementState(ctx, tx, req.SessionID, req.ElementID, req.ElementSlug)
	if err != nil {
		return nil, err
	}
	if source.VenueSlug != "the-cave" || strings.ToLower(strings.TrimSpace(source.Surface)) != "stage" {
		return nil, &ActionDeniedError{Reason: "unknown_target"}
	}
	if visibilityBool(source.Visibility, "locked", false) {
		return nil, &ActionDeniedError{Reason: "locked"}
	}
	if !isDuplicableStageElement(source.ElementType, source.ContextClass) {
		return nil, &ActionDeniedError{Reason: "unknown_target"}
	}

	sourceRow, err := loadElementForDuplication(ctx, tx, source.ElementID)
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

	now := time.Now().UTC().Format(time.RFC3339)
	newSlug := uniqueDuplicateSlug(sourceRow.Slug, req.ActorID)
	newName := duplicateElementName(sourceRow.Name, sourceRow.ElementType, sourceRow.Data)
	newData := cloneStringMap(sourceRow.Data)
	if newData == nil {
		newData = map[string]any{}
	}
	newData["duplicated_from_element_id"] = sourceRow.ID
	newData["duplicated_from_element_slug"] = sourceRow.Slug
	newData["duplicated_by"] = req.ActorID
	newData["duplicated_at"] = now
	newData["updated_at"] = now
	if strings.EqualFold(sourceRow.ElementType, "index_card") {
		newData["created_by"] = req.ActorID
		newData["created_by_display_name"] = displayName
		newData["created_by_handle"] = handle
		newData["created_by_role"] = role
	}

	newDataJSON, _ := json.Marshal(newData)
	var newElementID string
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
		VALUES ($1, $2, $3, $4, $5, 'library', $6)
		RETURNING id::text
	`, sourceRow.LibraryID, newName, newSlug, sourceRow.ElementType, sourceRow.ContextClass, newDataJSON).Scan(&newElementID); err != nil {
		return nil, err
	}

	venueID, enabled, err := resolvePlacementVenue(ctx, tx, req.VenueSlug)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, &ActionDeniedError{Reason: "policy_denied"}
	}

	placementVisibility := mergeVisibilityState(source.Visibility, map[string]any{
		"toRoles":           []string{"director", "producer"},
		"privateTo":         []string{},
		"visible":           true,
		"nameplate_visible": visibilityBool(source.Visibility, "nameplate_visible", true),
		"locked":            false,
	})
	positionJSON, _ := json.Marshal(map[string]any{
		"anchor": req.Layer,
		"x":      req.X,
		"y":      req.Y,
		"z":      0,
		"order":  req.Order,
		"frame":  "top-left",
	})
	visibilityJSON := marshalVisibilityState(placementVisibility)

	if _, err := tx.Exec(ctx, `
		INSERT INTO venue_layout_elements (
			venue_id,
			element_id,
			surface,
			position,
			visibility,
			is_default
		)
		VALUES ($1, $2, $3, $4, $5, FALSE)
		ON CONFLICT (venue_id, element_id, surface) DO UPDATE
		SET position = EXCLUDED.position,
		    visibility = EXCLUDED.visibility,
		    is_default = FALSE
	`, venueID, newElementID, req.Layer, positionJSON, visibilityJSON); err != nil {
		return nil, err
	}

	target := map[string]any{
		"kind":                    "element",
		"element_id":              source.ElementID,
		"element_slug":            source.ElementSlug,
		"venue_slug":              req.VenueSlug,
		"layer":                   req.Layer,
		"source_element_id":       source.ElementID,
		"source_element_slug":     source.ElementSlug,
		"new_element_id":          newElementID,
		"new_element_slug":        newSlug,
		"duplicated_element_id":   newElementID,
		"duplicated_element_slug": newSlug,
	}
	payload := map[string]any{
		"venue_slug":          req.VenueSlug,
		"layer":               req.Layer,
		"x":                   req.X,
		"y":                   req.Y,
		"order":               req.Order,
		"source_element_id":   source.ElementID,
		"source_element_slug": source.ElementSlug,
		"new_element_id":      newElementID,
		"new_element_slug":    newSlug,
		"actor_persona":       persona,
	}
	scope := map[string]any{
		"surfaces":         []string{req.Layer},
		"audienceSegments": []string{"director", "producer"},
	}
	visibility := map[string]any{
		"toRoles":   []string{"director", "producer"},
		"privateTo": []string{},
	}

	targetJSON, _ := json.Marshal(target)
	payloadJSON, _ := json.Marshal(payload)
	scopeJSON, _ := json.Marshal(scope)
	visibilityJSON2, _ := json.Marshal(visibility)

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
		VALUES ($1, $2, $3, 'act/duplicate_element', $4, $5, $6, $7, $8, TRUE)
		RETURNING id, ts
	`, req.SessionID, nextMoment, req.ActorID, targetJSON, payloadJSON, scopeJSON, visibilityJSON2, showing.ID).Scan(&out.ID, &ts); err != nil {
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
	out.Type = "act/duplicate_element"
	out.Target = target
	out.Payload = payload
	out.Scope = scope
	out.Visibility = visibility
	out.Timestamp = ts.UTC().Format(time.RFC3339)

	return &out, nil
}

type duplicatedElementRow struct {
	ID           string
	LibraryID    string
	Name         string
	Slug         string
	ElementType  string
	ContextClass string
	Data         map[string]any
}

func loadElementForDuplication(ctx context.Context, tx pgx.Tx, elementID string) (duplicatedElementRow, error) {
	var row duplicatedElementRow
	var dataRaw []byte
	if err := tx.QueryRow(ctx, `
		SELECT
			e.id::text,
			e.library_id::text,
			e.name,
			e.slug,
			e.element_type,
			COALESCE(NULLIF(e.context_class, ''), '') AS context_class,
			e.data
		FROM elements e
		WHERE e.id = $1
		  AND COALESCE(e.state::text, '') <> 'deleted'
		LIMIT 1
	`, elementID).Scan(&row.ID, &row.LibraryID, &row.Name, &row.Slug, &row.ElementType, &row.ContextClass, &dataRaw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return duplicatedElementRow{}, errors.New("index_card_not_found")
		}
		return duplicatedElementRow{}, err
	}
	row.Data = decodeStageJSONMap(dataRaw)
	if row.ContextClass == "" {
		row.ContextClass = deriveElementContextClass(row.ElementType, row.Slug, row.Data)
	}
	return row, nil
}

func sanitizeDuplicateElementRequest(req DuplicateElementRequest) DuplicateElementRequest {
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ActorID = strings.TrimSpace(req.ActorID)
	req.ElementID = strings.TrimSpace(req.ElementID)
	req.ElementSlug = strings.TrimSpace(req.ElementSlug)
	req.VenueSlug = strings.ToLower(strings.TrimSpace(req.VenueSlug))
	req.Layer = strings.ToLower(strings.TrimSpace(req.Layer))
	if req.Layer == "" {
		req.Layer = "stage"
	}
	return req
}

func isDuplicableStageElement(elementType, contextClass string) bool {
	switch strings.ToLower(strings.TrimSpace(contextClass)) {
	case "card", "prop":
		return true
	}

	switch strings.ToLower(strings.TrimSpace(elementType)) {
	case "index_card", "prop":
		return true
	}

	return false
}

func uniqueDuplicateSlug(baseSlug, actorID string) string {
	baseSlug = strings.TrimSpace(baseSlug)
	if baseSlug == "" {
		baseSlug = "element"
	}
	baseSlug = strings.ToLower(baseSlug)
	baseSlug = strings.Map(func(r rune) rune {
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
	}, baseSlug)
	baseSlug = strings.Trim(baseSlug, "-")
	if baseSlug == "" {
		baseSlug = "element"
	}
	actorID = strings.ToLower(strings.TrimSpace(actorID))
	if len(actorID) > 8 {
		actorID = actorID[:8]
	}
	if actorID == "" {
		actorID = "user"
	}
	return fmt.Sprintf("%s-copy-%s-%d", baseSlug, actorID, time.Now().UTC().UnixNano())
}

func duplicateElementName(name, elementType string, data map[string]any) string {
	name = strings.TrimSpace(name)
	if strings.EqualFold(strings.TrimSpace(elementType), "index_card") {
		if front := strings.TrimSpace(stringValue(data["front_text"])); front != "" {
			return front
		}
	}
	if name == "" {
		return "Duplicated Element"
	}
	if strings.HasSuffix(strings.ToLower(name), " copy") {
		return name
	}
	return name + " Copy"
}

func cloneStringMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
