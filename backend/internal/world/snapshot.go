package world

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showings"
)

type Snapshot struct {
	Location string          `json:"location"`
	Lot      string          `json:"lot"`
	Venue    Venue           `json:"venue"`
	Session  Session         `json:"session"`
	Showing  Showing         `json:"showing"`
	Elements []PlacedElement `json:"elements"`
	Overlay  *Overlay        `json:"overlay,omitempty"`
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

type Showing = showings.Showing

type PlacedElement struct {
	ElementID    string         `json:"element_id"`
	Name         string         `json:"name"`
	Slug         string         `json:"slug"`
	ElementType  string         `json:"element_type"`
	ContextClass string         `json:"context_class"`
	Surface      string         `json:"surface"`
	Position     map[string]any `json:"position"`
	Visibility   map[string]any `json:"visibility"`
	State        map[string]any `json:"state,omitempty"`
	Data         map[string]any `json:"data"`
}

type Overlay struct {
	ElementID    string         `json:"element_id"`
	ElementSlug  string         `json:"element_slug"`
	OverlayType  string         `json:"overlay_type"`
	ContextClass string         `json:"context_class"`
	Title        string         `json:"title"`
	Text         string         `json:"text,omitempty"`
	ImageURL     string         `json:"image_url,omitempty"`
	Surface      string         `json:"surface"`
	Position     map[string]any `json:"position,omitempty"`
	Data         map[string]any `json:"data,omitempty"`
	Visibility   map[string]any `json:"visibility,omitempty"`
}

type Action struct {
	ID               string         `json:"id"`
	MomentID         int64          `json:"moment_id"`
	ActorID          string         `json:"actor_id"`
	ShowingID        string         `json:"showing_id,omitempty"`
	ActorDisplayName string         `json:"actor_display_name,omitempty"`
	ActorHandle      string         `json:"actor_handle,omitempty"`
	ActorRole        string         `json:"actor_role,omitempty"`
	Actor            map[string]any `json:"actor,omitempty"`
	Persona          any            `json:"persona"`
	Type             string         `json:"type"`
	Target           map[string]any `json:"target"`
	Payload          map[string]any `json:"payload"`
	Scope            map[string]any `json:"scope"`
	Visibility       map[string]any `json:"visibility"`
	Timestamp        string         `json:"ts"`
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
			s.started_at,
			COALESCE(sh.id::text, ''),
			COALESCE(sh.production_id::text, ''),
			COALESCE(sh.venue_id::text, ''),
			COALESCE(sh.run_id::text, ''),
			COALESCE(sh.status::text, ''),
			COALESCE(sh.audience_view_enabled, FALSE),
			COALESCE(sh.started_at::text, ''),
			COALESCE(sh.ended_at::text, ''),
			COALESCE(sh.created_by::text, ''),
			COALESCE(sh.session_id::text, '')
		FROM venues v
		JOIN lots lo ON lo.id = v.lot_id
		JOIN locations l ON l.id = lo.location_id
		LEFT JOIN sessions s ON s.venue_id = v.id AND s.status IN ('rehearsal', 'live')
		LEFT JOIN showings sh ON sh.session_id = s.id
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
		&snap.Showing.ID,
		&snap.Showing.ProductionID,
		&snap.Showing.VenueID,
		&snap.Showing.RunID,
		&snap.Showing.Status,
		&snap.Showing.AudienceViewEnabled,
		&snap.Showing.StartedAt,
		&snap.Showing.EndedAt,
		&snap.Showing.CreatedBy,
		&snap.Showing.SessionID,
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
			COALESCE(NULLIF(e.context_class, ''), '') AS context_class,
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
			&item.ContextClass,
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
		item.State = map[string]any{
			"locked":            visibilityBool(item.Visibility, "locked", false),
			"nameplate_visible": visibilityBool(item.Visibility, "nameplate_visible", true),
			"visible":           visibilityBool(item.Visibility, "visible", true),
		}
		item.Data = decodeJSONMap(dataRaw)
		if value := strings.TrimSpace(item.ContextClass); value != "" {
			item.ContextClass = strings.ToLower(value)
		} else {
			item.ContextClass = deriveElementContextClass(item.ElementType, item.Slug, item.Data)
		}
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
			a.id,
			a.moment_id,
			a.actor_id,
			COALESCE(NULLIF(u.display_name, ''), NULLIF(u.handle, ''), LEFT(u.id::text, 8), 'Unknown Participant'),
			COALESCE(NULLIF(u.handle, ''), LEFT(u.id::text, 8), 'Unknown Participant'),
			COALESCE(sp.role::text, 'audience'),
			a.type,
			COALESCE(a.showing_id::text, ''),
			a.target,
			a.payload,
			a.scope,
			a.visibility,
			a.ts
		FROM actions a
		LEFT JOIN users u ON u.id = a.actor_id
		LEFT JOIN session_participants sp ON sp.session_id = a.session_id AND sp.user_id = a.actor_id
		WHERE a.session_id = $1
		ORDER BY a.moment_id DESC
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
			&a.ActorDisplayName,
			&a.ActorHandle,
			&a.ActorRole,
			&a.Type,
			&a.ShowingID,
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
		a.Persona = a.Payload["actor_persona"]
		if a.Persona == nil && a.Type == "persona/equip" {
			a.Persona = a.Payload["persona"]
		}
		a.Actor = map[string]any{
			"user_id":      a.ActorID,
			"handle":       a.ActorHandle,
			"display_name": a.ActorDisplayName,
			"role":         a.ActorRole,
			"persona":      a.Persona,
		}
		a.Timestamp = ts.UTC().Format(time.RFC3339)

		snap.Actions = append(snap.Actions, a)
	}

	if err := actionRows.Err(); err != nil {
		return nil, err
	}

	snap.Overlay = deriveActiveOverlay(snap.Actions, snap.Elements)

	layerVisibility := deriveElementLayerVisibility(snap.Actions)

	filtered := make([]PlacedElement, 0, len(snap.Elements))
	for _, el := range snap.Elements {
		audienceVisible := effectiveAudienceVisible(layerVisibility, el)
		if strings.TrimSpace(strings.ToLower(el.Surface)) == "stage" {
			el.Visibility = cloneMap(el.Visibility)
			el.Visibility["visible"] = audienceVisible
			el.State = cloneMap(el.State)
			el.State["visible"] = audienceVisible
		}

		switch normalizeRole(viewerRole) {
		case "producer", "director":
			filtered = append(filtered, el)

		case "actor", "crew", "cast":
			if strings.TrimSpace(strings.ToLower(el.Surface)) == "stage" {
				filtered = append(filtered, el)
				continue
			}
			if elementVisibilityForLayer(layerVisibility, el, "actor", false) {
				filtered = append(filtered, el)
			}

		case "audience":
			if audienceVisible {
				filtered = append(filtered, el)
			}

		default:
			if audienceVisible {
				filtered = append(filtered, el)
			}
		}
	}
	snap.Elements = filtered
	return &snap, nil
}

func effectiveAudienceVisible(layerVisibility map[string]bool, el PlacedElement) bool {
	defaultVisible := strings.TrimSpace(strings.ToLower(el.Surface)) == "stage" &&
		visibilityBool(el.Visibility, "visible", false)
	return elementVisibilityForLayer(layerVisibility, el, "audience", defaultVisible)
}

func cloneMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
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

func deriveActiveOverlay(actions []Action, elements []PlacedElement) *Overlay {
	if len(actions) == 0 {
		return nil
	}

	elementByID := map[string]PlacedElement{}
	elementBySlug := map[string]PlacedElement{}
	for _, el := range elements {
		if strings.TrimSpace(el.ElementID) != "" {
			elementByID[strings.ToLower(strings.TrimSpace(el.ElementID))] = el
		}
		if strings.TrimSpace(el.Slug) != "" {
			elementBySlug[strings.ToLower(strings.TrimSpace(el.Slug))] = el
		}
	}

	for _, a := range actions {
		switch a.Type {
		case "act/show_overlay":
			el, ok := lookupOverlayElement(a, elementByID, elementBySlug)
			if !ok {
				continue
			}
			return buildOverlayState(el, overlayTypeFromActionOrElement(a, el))
		case "act/hide_overlay":
			return nil
		}
	}

	return nil
}

func lookupOverlayElement(a Action, elementByID, elementBySlug map[string]PlacedElement) (PlacedElement, bool) {
	if id := strings.ToLower(strings.TrimSpace(actionTargetElementID(a))); id != "" {
		if el, ok := elementByID[id]; ok {
			return el, true
		}
	}
	if slug := strings.ToLower(strings.TrimSpace(actionTargetElementSlug(a))); slug != "" {
		if el, ok := elementBySlug[slug]; ok {
			return el, true
		}
	}
	return PlacedElement{}, false
}

func overlayTypeFromActionOrElement(a Action, el PlacedElement) string {
	if raw, ok := a.Payload["overlay_type"].(string); ok {
		if value := strings.TrimSpace(strings.ToLower(raw)); value != "" {
			return value
		}
	}

	if raw, ok := el.Data["overlay_type"].(string); ok {
		if value := strings.TrimSpace(strings.ToLower(raw)); value != "" {
			return value
		}
	}

	switch strings.ToLower(strings.TrimSpace(el.ContextClass)) {
	case "card":
		return "text"
	case "media", "prop", "scenery", "surface":
		return "image"
	default:
		return "text"
	}
}

func buildOverlayState(el PlacedElement, overlayType string) *Overlay {
	title := strings.TrimSpace(el.Name)
	if title == "" {
		title = strings.TrimSpace(el.Slug)
	}
	if title == "" {
		title = "Overlay"
	}

	text := ""
	imageURL := ""
	if overlayType == "image" {
		imageURL = overlayImageURL(el)
	}
	if overlayType != "image" || imageURL == "" {
		text = overlayTextContent(el)
	}

	return &Overlay{
		ElementID:    el.ElementID,
		ElementSlug:  el.Slug,
		OverlayType:  overlayType,
		ContextClass: el.ContextClass,
		Title:        title,
		Text:         text,
		ImageURL:     imageURL,
		Surface:      el.Surface,
		Position:     el.Position,
		Data:         el.Data,
		Visibility:   el.Visibility,
	}
}

func overlayTextContent(el PlacedElement) string {
	if text, ok := el.Data["front_text"].(string); ok && strings.TrimSpace(text) != "" {
		return strings.TrimSpace(text)
	}
	if text, ok := el.Data["text"].(string); ok && strings.TrimSpace(text) != "" {
		return strings.TrimSpace(text)
	}
	if text, ok := el.Data["title"].(string); ok && strings.TrimSpace(text) != "" {
		return strings.TrimSpace(text)
	}
	return strings.TrimSpace(el.Name)
}

func overlayImageURL(el PlacedElement) string {
	for _, key := range []string{"image_url", "icon_url", "overlay_image_url", "src"} {
		if raw, ok := el.Data[key].(string); ok && strings.TrimSpace(raw) != "" {
			return strings.TrimSpace(raw)
		}
	}
	return ""
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
