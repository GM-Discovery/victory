package world

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Snapshot struct {
	Location string          `json:"location"`
	Lot      string          `json:"lot"`
	Venue    Venue           `json:"venue"`
	Session  Session         `json:"session"`
	Elements []PlacedElement `json:"elements"`
	Actions  []Action        `json:"actions"`
}

type Venue struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Slug   string         `json:"slug"`
	Kind   string         `json:"kind"`
	Config map[string]any `json:"config"`
}

type Session struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at"`
}

type PlacedElement struct {
	ElementID   string         `json:"element_id"`
	Name        string         `json:"name"`
	Slug        string         `json:"slug"`
	ElementType string         `json:"element_type"`
	Surface     string         `json:"surface"`
	Position    map[string]any `json:"position"`
	Visibility  map[string]any `json:"visibility"`
	Data        map[string]any `json:"data"`
}

type Action struct {
	ID         string         `json:"id"`
	MomentID   int64          `json:"moment_id"`
	ActorID    string         `json:"actor_id"`
	Type       string         `json:"type"`
	Target     map[string]any `json:"target"`
	Payload    map[string]any `json:"payload"`
	Scope      map[string]any `json:"scope"`
	Visibility map[string]any `json:"visibility"`
	Timestamp  string         `json:"ts"`
}

func LoadCaveSnapshot(ctx context.Context, pool *pgxpool.Pool, viewerRole string) (*Snapshot, error) {
	var snap Snapshot
	snap.Elements = []PlacedElement{}
	snap.Actions = []Action{}

	var venueConfig []byte
	var startedAt time.Time

	err := pool.QueryRow(ctx, `
		SELECT
			l.name AS location_name,
			lo.name AS lot_name,
			v.id,
			v.name,
			v.slug,
			v.kind,
			v.config,
			s.id,
			s.status,
			s.started_at
		FROM venues v
		JOIN lots lo ON lo.id = v.lot_id
		JOIN locations l ON l.id = lo.location_id
		LEFT JOIN sessions s ON s.venue_id = v.id AND s.status IN ('rehearsal', 'live')
		WHERE v.slug = 'the-cave'
		ORDER BY s.started_at DESC NULLS LAST
		LIMIT 1
	`).Scan(
		&snap.Location,
		&snap.Lot,
		&snap.Venue.ID,
		&snap.Venue.Name,
		&snap.Venue.Slug,
		&snap.Venue.Kind,
		&venueConfig,
		&snap.Session.ID,
		&snap.Session.Status,
		&startedAt,
	)
	if err != nil {
		return nil, err
	}

	snap.Session.StartedAt = startedAt.UTC().Format(time.RFC3339)

	if err := json.Unmarshal(venueConfig, &snap.Venue.Config); err != nil {
		return nil, err
	}

	elementRows, err := pool.Query(ctx, `
		SELECT
			e.id,
			e.name,
			e.slug,
			e.element_type,
			vle.surface,
			vle.position,
			vle.visibility,
			e.data
		FROM venue_layout_elements vle
		JOIN elements e ON e.id = vle.element_id
		JOIN venues v ON v.id = vle.venue_id
		WHERE v.slug = 'the-cave'
		ORDER BY e.name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer elementRows.Close()

	for elementRows.Next() {
		var item PlacedElement
		var posRaw, visRaw, dataRaw []byte

		if err := elementRows.Scan(
			&item.ElementID,
			&item.Name,
			&item.Slug,
			&item.ElementType,
			&item.Surface,
			&posRaw,
			&visRaw,
			&dataRaw,
		); err != nil {
			return nil, err
		}

		item.Position = decodeJSONMap(posRaw)
		item.Visibility = decodeJSONMap(visRaw)
		item.Data = decodeJSONMap(dataRaw)
		snap.Elements = append(snap.Elements, item)
	}

	if err := elementRows.Err(); err != nil {
		return nil, err
	}

	if snap.Session.ID == "" {
		return nil, errors.New("no active session found for the-cave")
	}

	actionRows, err := pool.Query(ctx, `
		SELECT
			id,
			moment_id,
			actor_id,
			type,
			target,
			payload,
			scope,
			visibility,
			ts
		FROM actions
		WHERE session_id = $1
		ORDER BY moment_id DESC
		LIMIT 50
	`, snap.Session.ID)
	if err != nil {
		return nil, err
	}
	defer actionRows.Close()

	for actionRows.Next() {
		var a Action
		var targetRaw, payloadRaw, scopeRaw, visibilityRaw []byte
		var ts time.Time

		if err := actionRows.Scan(
			&a.ID,
			&a.MomentID,
			&a.ActorID,
			&a.Type,
			&targetRaw,
			&payloadRaw,
			&scopeRaw,
			&visibilityRaw,
			&ts,
		); err != nil {
			return nil, err
		}

		a.Target = decodeJSONMap(targetRaw)
		a.Payload = decodeJSONMap(payloadRaw)
		a.Scope = decodeJSONMap(scopeRaw)
		a.Visibility = decodeJSONMap(visibilityRaw)
		a.Timestamp = ts.UTC().Format(time.RFC3339)

		snap.Actions = append(snap.Actions, a)
	}

	if err := actionRows.Err(); err != nil {
		return nil, err
	}

	layerVisibility := deriveElementLayerVisibility(snap.Actions)
	
	filtered := make([]PlacedElement, 0, len(snap.Elements))
	for _, el := range snap.Elements {
		switch normalizeRole(viewerRole) {
		case "producer", "director":
			filtered = append(filtered, el)

		case "actor", "crew", "cast":
			if elementVisibilityForLayer(layerVisibility, el, "actor", false) {
				filtered = append(filtered, el)
			}

		case "audience":
			if elementVisibilityForLayer(layerVisibility, el, "audience", false) {
				filtered = append(filtered, el)
			}

		default:
			if elementVisibilityForLayer(layerVisibility, el, "audience", false) {
				filtered = append(filtered, el)
			}
		}
	}
	snap.Elements = filtered
	return &snap, nil
}

func decodeJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func normalizeRole(role string) string {
	switch role {
	case "producer":
		return "producer"
	case "director":
		return "director"
	case "actor":
		return "actor"
	case "cast":
		return "cast"
	case "crew":
		return "crew"
	case "audience":
		return "audience"
	default:
		return role
	}
}

func actionTargetElementSlug(a Action) string {
	if slug, ok := a.Target["element_slug"].(string); ok {
		return slug
	}
	if slug, ok := a.Target["slug"].(string); ok {
		return slug
	}
	return ""
}

func actionTargetLayer(a Action) string {
	if layer, ok := a.Target["layer"].(string); ok {
		return layer
	}
	if layer, ok := a.Payload["layer"].(string); ok {
		return layer
	}
	return ""
}

func actionTargetElementID(a Action) string {
	if id, ok := a.Target["element_id"].(string); ok {
		return id
	}
	return ""
}

func deriveElementLayerVisibility(actions []Action) map[string]bool {
	out := map[string]bool{}

	for _, a := range actions {
		layer := actionTargetLayer(a)
		if layer == "" {
			continue
		}

		elementID := actionTargetElementID(a)
		elementSlug := actionTargetElementSlug(a)

		targetKey := ""
		if elementID != "" {
			targetKey = elementID + ":" + layer
		} else if elementSlug != "" {
			targetKey = "slug:" + elementSlug + ":" + layer
		} else {
			continue
		}

		if _, exists := out[targetKey]; exists {
			continue
		}

		switch a.Type {
		case "act/reveal_element":
			out[targetKey] = true
		case "act/hide_element":
			out[targetKey] = false
		}
	}

	return out
}

func elementVisibilityForLayer(visibility map[string]bool, el PlacedElement, layer string, defaultVisible bool) bool {
	if el.ElementID != "" {
		if v, ok := visibility[el.ElementID+":"+layer]; ok {
			return v
		}
	}
	if el.Slug != "" {
		if v, ok := visibility["slug:"+el.Slug+":"+layer]; ok {
			return v
		}
	}
	return defaultVisible
}