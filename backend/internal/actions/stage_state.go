package actions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

type venueLayoutElementState struct {
	VenueID      string
	VenueSlug    string
	ElementID    string
	ElementSlug  string
	ElementType  string
	ContextClass string
	Surface      string
	Visibility   map[string]any
}

func resolveVenueLayoutElementState(ctx context.Context, q actionQuerier, sessionID, elementID, elementSlug string) (venueLayoutElementState, error) {
	sessionID = strings.TrimSpace(sessionID)
	elementID = strings.TrimSpace(elementID)
	elementSlug = strings.TrimSpace(elementSlug)
	if sessionID == "" || (elementID == "" && elementSlug == "") {
		return venueLayoutElementState{}, errors.New("element_id or element_slug is required")
	}

	base := `
		SELECT
			v.id::text,
			v.slug,
			e.id::text,
			e.slug,
			e.element_type,
			COALESCE(NULLIF(e.context_class, ''), '') AS context_class,
			vle.surface,
			vle.visibility
		FROM sessions s
		JOIN venues v ON v.id = s.venue_id
		JOIN venue_layout_elements vle ON vle.venue_id = v.id
		JOIN elements e ON e.id = vle.element_id
		WHERE s.id = $1
		  AND COALESCE(e.state::text, '') <> 'deleted'
	`

	var (
		venueID      string
		venueSlug    string
		resolvedID   string
		resolvedSlug string
		elementType  string
		contextClass string
		surface      string
		visRaw       []byte
	)

	var err error
	switch {
	case elementID != "":
		err = q.QueryRow(ctx, base+` AND e.id = $2 LIMIT 1`, sessionID, elementID).Scan(
			&venueID,
			&venueSlug,
			&resolvedID,
			&resolvedSlug,
			&elementType,
			&contextClass,
			&surface,
			&visRaw,
		)
	case elementSlug != "":
		err = q.QueryRow(ctx, base+` AND e.slug = $2 LIMIT 1`, sessionID, elementSlug).Scan(
			&venueID,
			&venueSlug,
			&resolvedID,
			&resolvedSlug,
			&elementType,
			&contextClass,
			&surface,
			&visRaw,
		)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return venueLayoutElementState{}, errors.New("index_card_not_found")
		}
		return venueLayoutElementState{}, err
	}

	state := venueLayoutElementState{
		VenueID:      venueID,
		VenueSlug:    venueSlug,
		ElementID:    resolvedID,
		ElementSlug:  resolvedSlug,
		ElementType:  elementType,
		ContextClass: strings.ToLower(strings.TrimSpace(contextClass)),
		Surface:      surface,
		Visibility:   decodeStageJSONMap(visRaw),
	}
	if state.ContextClass == "" {
		state.ContextClass = deriveElementContextClass(elementType, resolvedSlug, nil)
	}

	return state, nil
}

func resolvePlaceableElement(ctx context.Context, q actionQuerier, elementID, elementSlug string) (resolvedID, resolvedSlug string, err error) {
	base := `
		SELECT e.id::text, e.slug
		FROM elements e
		WHERE COALESCE(e.state::text, '') <> 'deleted'
	`

	switch {
	case strings.TrimSpace(elementID) != "":
		err = q.QueryRow(ctx, base+` AND e.id = $1 LIMIT 1`, strings.TrimSpace(elementID)).Scan(&resolvedID, &resolvedSlug)
	case strings.TrimSpace(elementSlug) != "":
		err = q.QueryRow(ctx, base+` AND e.slug = $1 LIMIT 1`, strings.TrimSpace(elementSlug)).Scan(&resolvedID, &resolvedSlug)
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

func visibilityBool(visibility map[string]any, key string, fallback bool) bool {
	if visibility == nil {
		return fallback
	}
	raw, ok := visibility[key]
	if !ok {
		return fallback
	}
	switch value := raw.(type) {
	case bool:
		return value
	case string:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return fallback
}

func mergeVisibilityState(visibility map[string]any, updates map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range visibility {
		out[k] = v
	}
	for k, v := range updates {
		out[k] = v
	}
	return out
}

func marshalVisibilityState(visibility map[string]any) []byte {
	if visibility == nil {
		visibility = map[string]any{}
	}
	raw, err := json.Marshal(visibility)
	if err != nil {
		return []byte(`{}`)
	}
	return raw
}

func decodeStageJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func deriveElementContextClass(elementType, slug string, data map[string]any) string {
	if data != nil {
		if raw, ok := data["context_class"].(string); ok {
			if value := strings.TrimSpace(strings.ToLower(raw)); value != "" {
				return value
			}
		}
	}

	switch strings.ToLower(strings.TrimSpace(elementType)) {
	case "index_card":
		return "card"
	case "prop":
		return "prop"
	case "scenery":
		return "scenery"
	case "surface":
		return "surface"
	case "actor":
		return "actor"
	case "media":
		return "media"
	}

	switch strings.ToLower(strings.TrimSpace(slug)) {
	case "first-fire":
		return "scenery"
	}

	return "system"
}
