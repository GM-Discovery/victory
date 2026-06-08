package showings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
)

type ReviewCounts struct {
	Total      int `json:"total"`
	Chat       int `json:"chat"`
	Speech     int `json:"speech"`
	Reactions  int `json:"reactions"`
	RevealHide int `json:"reveal_hide"`
	Overlay    int `json:"overlay"`
	Cards      int `json:"cards"`
	Persona    int `json:"persona"`
}

type ReviewShowing struct {
	ID             string `json:"id"`
	SessionID      string `json:"session_id"`
	ProductionID   string `json:"production_id"`
	ProductionName string `json:"production_name"`
	ProductionSlug string `json:"production_slug"`
	VenueID        string `json:"venue_id"`
	VenueName      string `json:"venue_name"`
	VenueSlug      string `json:"venue_slug"`
	LocationID     string `json:"location_id"`
	LocationName   string `json:"location_name"`
	RunID          string `json:"run_id,omitempty"`
	Status         string `json:"status"`
	AudienceView   bool   `json:"audience_view_enabled"`
	StartedAt      string `json:"started_at"`
	EndedAt        string `json:"ended_at,omitempty"`
	CreatedBy      string `json:"created_by"`
}

type ReviewListItem struct {
	Showing ReviewShowing `json:"showing"`
	Counts  ReviewCounts  `json:"counts"`
}

type ReviewPersona struct {
	CharacterCardID string `json:"character_card_id,omitempty"`
	Name            string `json:"name,omitempty"`
	Pronouns        string `json:"pronouns,omitempty"`
	PortraitURL     string `json:"portrait_url,omitempty"`
	Color           string `json:"color,omitempty"`
	Tagline         string `json:"tagline,omitempty"`
}

type ReviewEvent struct {
	ID               string `json:"id"`
	MomentID         int64  `json:"moment_id"`
	Timestamp        string `json:"timestamp"`
	Category         string `json:"category"`
	ActionType       string `json:"action_type"`
	ActorID          string `json:"actor_id"`
	ActorDisplayName string `json:"actor_display_name"`
	ActorHandle      string `json:"actor_handle,omitempty"`
	ActorRole        string `json:"actor_role,omitempty"`
	Persona          any    `json:"persona,omitempty"`
	PersonaLabel     string `json:"persona_label,omitempty"`
	Headline         string `json:"headline"`
	Detail           string `json:"detail"`
	ElementName      string `json:"element_name,omitempty"`
	ElementSlug      string `json:"element_slug,omitempty"`
	TargetLayer      string `json:"target_layer,omitempty"`
	OverlayType      string `json:"overlay_type,omitempty"`
	ReactionKind     string `json:"reaction_kind,omitempty"`
	CardFront        string `json:"card_front,omitempty"`
	CardBack         string `json:"card_back,omitempty"`
	CardColor        string `json:"card_color,omitempty"`
}

type ReviewDetail struct {
	Showing ReviewShowing `json:"showing"`
	Counts  ReviewCounts  `json:"counts"`
	Events  []ReviewEvent `json:"events"`
}

type reviewScopeMode string

const (
	reviewScopeAll         reviewScopeMode = "all"
	reviewScopeLocation    reviewScopeMode = "location"
	reviewScopeProductions reviewScopeMode = "productions"
)

type reviewScope struct {
	Mode          reviewScopeMode
	LocationID    string
	ProductionIDs []string
}

type reviewShowingRow struct {
	Showing ReviewShowing
	Counts  ReviewCounts
}

type reviewActionRow struct {
	ID           string
	MomentID     int64
	ActorID      string
	ActorDisplay string
	ActorHandle  string
	ActorRole    string
	ActionType   string
	ShowingID    string
	TargetRaw    []byte
	PayloadRaw   []byte
	Timestamp    time.Time
}

func ListReviewShowings(ctx context.Context, pool *pgxpool.Pool, userID string) ([]ReviewListItem, error) {
	scope, err := resolveReviewScope(ctx, pool, userID)
	if err != nil {
		return nil, err
	}

	rows, err := queryReviewShowings(ctx, pool, scope, "")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ReviewListItem
	for rows.Next() {
		row, err := scanReviewShowingRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ReviewListItem{
			Showing: row.Showing,
			Counts:  row.Counts,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func LoadReviewDetail(ctx context.Context, pool *pgxpool.Pool, userID, showingID string) (ReviewDetail, error) {
	scope, err := resolveReviewScope(ctx, pool, userID)
	if err != nil {
		return ReviewDetail{}, err
	}

	rows, err := queryReviewShowings(ctx, pool, scope, showingID)
	if err != nil {
		return ReviewDetail{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return ReviewDetail{}, err
		}
		return ReviewDetail{}, pgx.ErrNoRows
	}

	row, err := scanReviewShowingRow(rows)
	if err != nil {
		return ReviewDetail{}, err
	}
	if rows.Next() {
		return ReviewDetail{}, errors.New("unexpected duplicate showing rows")
	}

	events, err := loadReviewEvents(ctx, pool, row.Showing.ID)
	if err != nil {
		return ReviewDetail{}, err
	}

	return ReviewDetail{
		Showing: row.Showing,
		Counts:  row.Counts,
		Events:  events,
	}, nil
}

func resolveReviewScope(ctx context.Context, pool *pgxpool.Pool, userID string) (reviewScope, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return reviewScope{}, pgx.ErrNoRows
	}

	if ok, err := access.IsOperatorUser(ctx, pool, userID); err != nil {
		return reviewScope{}, err
	} else if ok {
		return reviewScope{Mode: reviewScopeAll}, nil
	}

	locationRole, locationID, err := primaryLocationRole(ctx, pool, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return reviewScope{}, pgx.ErrNoRows
		}
		return reviewScope{}, err
	}

	switch locationRole {
	case "producer":
		return reviewScope{Mode: reviewScopeLocation, LocationID: locationID}, nil
	case "director":
		productionIDs, err := directorProductionIDs(ctx, pool, userID)
		if err != nil {
			return reviewScope{}, err
		}
		if len(productionIDs) > 0 {
			sort.Strings(productionIDs)
			return reviewScope{Mode: reviewScopeProductions, LocationID: locationID, ProductionIDs: productionIDs}, nil
		}
		return reviewScope{Mode: reviewScopeLocation, LocationID: locationID}, nil
	default:
		return reviewScope{}, pgx.ErrNoRows
	}
}

func primaryLocationRole(ctx context.Context, pool *pgxpool.Pool, userID string) (string, string, error) {
	var locationID, role string
	err := pool.QueryRow(ctx, `
		SELECT location_id::text, role::text
		FROM (
			SELECT location_id, role, created_at
			FROM location_memberships
			WHERE user_id = $1
			  AND active = TRUE
			  AND role IN ('producer', 'director')

			UNION ALL

			SELECT location_id, role, created_at
			FROM memberships
			WHERE user_id = $1
			  AND active = TRUE
			  AND role IN ('producer', 'director')
		) roles
		ORDER BY
		  CASE role
			WHEN 'producer' THEN 1
			WHEN 'director' THEN 2
			ELSE 99
		  END,
		  created_at ASC
		LIMIT 1
	`, userID).Scan(&locationID, &role)
	if err != nil {
		return "", "", err
	}
	return strings.ToLower(strings.TrimSpace(role)), locationID, nil
}

func directorProductionIDs(ctx context.Context, pool *pgxpool.Pool, userID string) ([]string, error) {
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT production_id::text
		FROM memberships
		WHERE user_id = $1
		  AND active = TRUE
		  AND role = 'director'
		  AND production_id IS NOT NULL
		ORDER BY production_id::text ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var productionID string
		if err := rows.Scan(&productionID); err != nil {
			return nil, err
		}
		if strings.TrimSpace(productionID) != "" {
			out = append(out, productionID)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func queryReviewShowings(ctx context.Context, pool *pgxpool.Pool, scope reviewScope, showingID string) (pgx.Rows, error) {
	sql := `
		SELECT
			sh.id::text,
			sh.session_id::text,
			COALESCE(sh.production_id::text, ''),
			COALESCE(p.name, ''),
			COALESCE(p.slug, ''),
			sh.venue_id::text,
			v.name,
			v.slug,
			l.id::text,
			l.name,
			COALESCE(sh.run_id::text, ''),
			sh.status::text,
			sh.audience_view_enabled,
			sh.started_at::text,
			COALESCE(sh.ended_at::text, ''),
			sh.created_by::text,
			COALESCE(ac.total_count, 0)::int,
			COALESCE(ac.chat_count, 0)::int,
			COALESCE(ac.speech_count, 0)::int,
			COALESCE(ac.reaction_count, 0)::int,
			COALESCE(ac.reveal_hide_count, 0)::int,
			COALESCE(ac.overlay_count, 0)::int,
			COALESCE(ac.card_count, 0)::int,
			COALESCE(ac.persona_count, 0)::int
		FROM showings sh
		JOIN venues v ON v.id = sh.venue_id
		JOIN lots lo ON lo.id = v.lot_id
		JOIN locations l ON l.id = lo.location_id
		LEFT JOIN productions p ON p.id = sh.production_id
		LEFT JOIN LATERAL (
			SELECT
				COUNT(*)::int AS total_count,
				COUNT(*) FILTER (WHERE a.type = 'chat/message')::int AS chat_count,
				COUNT(*) FILTER (WHERE a.type = 'perform/speak')::int AS speech_count,
				COUNT(*) FILTER (WHERE a.type = 'react/emote')::int AS reaction_count,
				COUNT(*) FILTER (WHERE a.type IN ('act/reveal_element', 'act/hide_element'))::int AS reveal_hide_count,
				COUNT(*) FILTER (WHERE a.type IN ('act/show_overlay', 'act/hide_overlay'))::int AS overlay_count,
				COUNT(*) FILTER (WHERE a.type IN ('create/index_card', 'update/index_card', 'delete/index_card', 'act/place_element'))::int AS card_count,
				COUNT(*) FILTER (WHERE a.type IN ('persona/equip', 'persona/unequip'))::int AS persona_count
			FROM actions a
			WHERE a.showing_id = sh.id
			  AND a.recorded = TRUE
		) ac ON TRUE
		WHERE sh.status = 'closed'
	`

	args := []any{}
	switch scope.Mode {
	case reviewScopeLocation:
		sql += " AND l.id::text = $1"
		args = append(args, scope.LocationID)
	case reviewScopeProductions:
		if len(scope.ProductionIDs) == 0 {
			sql += " AND FALSE"
		} else {
			placeholders := make([]string, len(scope.ProductionIDs))
			for i, productionID := range scope.ProductionIDs {
				placeholders[i] = fmt.Sprintf("$%d", i+1)
				args = append(args, productionID)
			}
			sql += " AND sh.production_id::text IN (" + strings.Join(placeholders, ",") + ")"
		}
	}

	if showingID != "" {
		paramIndex := len(args) + 1
		sql += " AND sh.id::text = $" + strconv.Itoa(paramIndex)
		args = append(args, showingID)
	}

	sql += `
		ORDER BY COALESCE(sh.ended_at, sh.started_at) DESC, sh.started_at DESC
	`

	return pool.Query(ctx, sql, args...)
}

func scanReviewShowingRow(rows pgx.Rows) (reviewShowingRow, error) {
	var row reviewShowingRow
	if err := rows.Scan(
		&row.Showing.ID,
		&row.Showing.SessionID,
		&row.Showing.ProductionID,
		&row.Showing.ProductionName,
		&row.Showing.ProductionSlug,
		&row.Showing.VenueID,
		&row.Showing.VenueName,
		&row.Showing.VenueSlug,
		&row.Showing.LocationID,
		&row.Showing.LocationName,
		&row.Showing.RunID,
		&row.Showing.Status,
		&row.Showing.AudienceView,
		&row.Showing.StartedAt,
		&row.Showing.EndedAt,
		&row.Showing.CreatedBy,
		&row.Counts.Total,
		&row.Counts.Chat,
		&row.Counts.Speech,
		&row.Counts.Reactions,
		&row.Counts.RevealHide,
		&row.Counts.Overlay,
		&row.Counts.Cards,
		&row.Counts.Persona,
	); err != nil {
		return reviewShowingRow{}, err
	}
	return row, nil
}

func loadReviewEvents(ctx context.Context, pool *pgxpool.Pool, showingID string) ([]ReviewEvent, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			a.id::text,
			a.moment_id,
			a.actor_id::text,
			COALESCE(NULLIF(u.display_name, ''), NULLIF(u.handle, ''), LEFT(u.id::text, 8), 'Unknown Participant'),
			COALESCE(NULLIF(u.handle, ''), LEFT(u.id::text, 8), 'Unknown Participant'),
			COALESCE(sp.role::text, 'audience'),
			a.type,
			COALESCE(a.showing_id::text, ''),
			a.target,
			a.payload,
			a.ts
		FROM actions a
		LEFT JOIN users u ON u.id = a.actor_id
		LEFT JOIN session_participants sp ON sp.session_id = a.session_id AND sp.user_id = a.actor_id
		WHERE a.showing_id = $1::uuid
		  AND a.recorded = TRUE
		ORDER BY a.moment_id ASC
	`, showingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	elementCache := newReviewElementCache(ctx, pool)
	cardCache := newReviewCardCache(ctx, pool)

	var out []ReviewEvent
	for rows.Next() {
		var row reviewActionRow
		if err := rows.Scan(
			&row.ID,
			&row.MomentID,
			&row.ActorID,
			&row.ActorDisplay,
			&row.ActorHandle,
			&row.ActorRole,
			&row.ActionType,
			&row.ShowingID,
			&row.TargetRaw,
			&row.PayloadRaw,
			&row.Timestamp,
		); err != nil {
			return nil, err
		}
		payload := decodeJSONMap(row.PayloadRaw)
		target := decodeJSONMap(row.TargetRaw)
		if source, _ := payload["source"].(string); strings.EqualFold(strings.TrimSpace(source), "discord") {
			if discordMeta, ok := payload["discord"].(map[string]any); ok {
				if linkedUserID := strings.TrimSpace(stringValueMap(discordMeta, "linked_user_id")); linkedUserID == "" {
					if value := strings.TrimSpace(stringValueMap(discordMeta, "author_global_name")); value != "" {
						row.ActorDisplay = value
					} else if value := strings.TrimSpace(stringValueMap(discordMeta, "author_username")); value != "" {
						row.ActorDisplay = value
					}
					if value := strings.TrimSpace(stringValueMap(discordMeta, "author_username")); value != "" {
						row.ActorHandle = value
					}
					row.ActorRole = "audience"
				}
			}
		}
		event := normalizeReviewEvent(row, target, payload, elementCache, cardCache)
		out = append(out, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func normalizeReviewEvent(row reviewActionRow, target, payload map[string]any, elements *reviewElementCache, cards *reviewCardCache) ReviewEvent {
	category := reviewCategoryForAction(row.ActionType)
	headline := reviewHeadlineForAction(row.ActionType)
	detail := reviewDetailForAction(row.ActionType, target, payload, elements, cards)
	persona := reviewPersonaFromPayload(payload, cards)
	var personaLabel string
	if persona != nil {
		personaLabel = strings.TrimSpace(persona.Name)
		if personaLabel == "" {
			personaLabel = strings.TrimSpace(persona.CharacterCardID)
		}
	}

	event := ReviewEvent{
		ID:               row.ID,
		MomentID:         row.MomentID,
		Timestamp:        row.Timestamp.UTC().Format(time.RFC3339),
		Category:         category,
		ActionType:       row.ActionType,
		ActorID:          row.ActorID,
		ActorDisplayName: row.ActorDisplay,
		ActorHandle:      row.ActorHandle,
		ActorRole:        row.ActorRole,
		Persona:          persona,
		PersonaLabel:     personaLabel,
		Headline:         headline,
		Detail:           detail,
	}

	if elementName := reviewTargetElementLabel(target, elements); elementName != "" {
		event.ElementName = elementName
	}
	if slug, _ := target["element_slug"].(string); strings.TrimSpace(slug) != "" {
		event.ElementSlug = slug
	}
	if layer, _ := target["layer"].(string); strings.TrimSpace(layer) != "" {
		event.TargetLayer = layer
	}
	if overlayType, _ := payload["overlay_type"].(string); strings.TrimSpace(overlayType) != "" {
		event.OverlayType = overlayType
	}
	if reactionKind, _ := payload["kind"].(string); strings.TrimSpace(reactionKind) != "" {
		event.ReactionKind = reactionKind
	}
	if frontText, _ := payload["front_text"].(string); strings.TrimSpace(frontText) != "" {
		event.CardFront = previewText(frontText, 120)
	}
	if backText, _ := payload["back_text"].(string); strings.TrimSpace(backText) != "" {
		event.CardBack = previewText(backText, 120)
	}
	if color, _ := payload["color"].(string); strings.TrimSpace(color) != "" {
		event.CardColor = strings.TrimSpace(color)
	}
	return event
}

func stringValueMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	value, ok := m[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func reviewCategoryForAction(actionType string) string {
	switch actionType {
	case "chat/message":
		return "chat"
	case "perform/speak":
		return "speech"
	case "react/emote":
		return "reactions"
	case "act/reveal_element", "act/hide_element":
		return "reveal_hide"
	case "act/show_overlay", "act/hide_overlay":
		return "overlay"
	case "create/index_card", "update/index_card", "delete/index_card", "act/place_element":
		return "cards"
	case "persona/equip", "persona/unequip":
		return "persona"
	default:
		return "other"
	}
}

func reviewHeadlineForAction(actionType string) string {
	switch actionType {
	case "chat/message":
		return "Chat"
	case "perform/speak":
		return "Stage Speech"
	case "react/emote":
		return "Reaction"
	case "act/reveal_element", "act/hide_element":
		return "Reveal / Hide"
	case "act/show_overlay", "act/hide_overlay":
		return "Overlay"
	case "create/index_card", "update/index_card", "delete/index_card", "act/place_element":
		return "Index Card"
	case "persona/equip", "persona/unequip":
		return "Persona"
	default:
		return "Action"
	}
}

func reviewDetailForAction(actionType string, target, payload map[string]any, elements *reviewElementCache, cards *reviewCardCache) string {
	switch actionType {
	case "chat/message", "perform/speak":
		return strings.TrimSpace(stringFromMap(payload, "text"))
	case "react/emote":
		kind := strings.TrimSpace(stringFromMap(payload, "kind"))
		if kind == "" {
			return "Reaction"
		}
		return humanizeActionWord(kind)
	case "act/reveal_element":
		name := reviewTargetElementLabel(target, elements)
		if name == "" {
			name = "element"
		}
		layer := strings.TrimSpace(stringFromMap(target, "layer"))
		if layer == "" {
			layer = strings.TrimSpace(stringFromMap(payload, "layer"))
		}
		if layer != "" {
			return "Revealed " + name + " on " + layer
		}
		return "Revealed " + name
	case "act/hide_element":
		name := reviewTargetElementLabel(target, elements)
		if name == "" {
			name = "element"
		}
		layer := strings.TrimSpace(stringFromMap(target, "layer"))
		if layer == "" {
			layer = strings.TrimSpace(stringFromMap(payload, "layer"))
		}
		if layer != "" {
			return "Hid " + name + " on " + layer
		}
		return "Hid " + name
	case "act/show_overlay":
		name := reviewTargetElementLabel(target, elements)
		overlayType := strings.TrimSpace(stringFromMap(payload, "overlay_type"))
		switch {
		case name != "" && overlayType != "":
			return "Showed " + overlayType + " overlay for " + name
		case name != "":
			return "Showed overlay for " + name
		case overlayType != "":
			return "Showed " + overlayType + " overlay"
		default:
			return "Showed overlay"
		}
	case "act/hide_overlay":
		name := reviewTargetElementLabel(target, elements)
		overlayType := strings.TrimSpace(stringFromMap(payload, "overlay_type"))
		switch {
		case name != "" && overlayType != "":
			return "Hid " + overlayType + " overlay for " + name
		case name != "":
			return "Hid overlay for " + name
		case overlayType != "":
			return "Hid " + overlayType + " overlay"
		default:
			return "Hid overlay"
		}
	case "create/index_card":
		front := previewText(strings.TrimSpace(stringFromMap(payload, "front_text")), 90)
		back := previewText(strings.TrimSpace(stringFromMap(payload, "back_text")), 90)
		switch {
		case front != "" && back != "":
			return front + " / " + back
		case front != "":
			return front
		case back != "":
			return back
		default:
			return "Created an index card"
		}
	case "update/index_card":
		front := previewText(strings.TrimSpace(stringFromMap(payload, "front_text")), 90)
		back := previewText(strings.TrimSpace(stringFromMap(payload, "back_text")), 90)
		switch {
		case front != "" && back != "":
			return front + " / " + back
		case front != "":
			return front
		case back != "":
			return back
		default:
			return "Updated an index card"
		}
	case "delete/index_card":
		name := reviewTargetElementLabel(target, elements)
		if name == "" {
			name = "index card"
		}
		return "Deleted " + name
	case "act/place_element":
		name := reviewTargetElementLabel(target, elements)
		if name == "" {
			name = "index card"
		}
		layer := strings.TrimSpace(stringFromMap(target, "layer"))
		if layer == "" {
			layer = strings.TrimSpace(stringFromMap(payload, "layer"))
		}
		x := strings.TrimSpace(stringFromMapInt(payload, "x"))
		y := strings.TrimSpace(stringFromMapInt(payload, "y"))
		if layer != "" && x != "" && y != "" {
			return "Placed " + name + " on " + layer + " at " + x + ", " + y
		}
		if layer != "" {
			return "Placed " + name + " on " + layer
		}
		return "Placed " + name
	case "persona/equip":
		label := reviewPersonaLabel(reviewPersonaFromPayload(payload, cards))
		if label == "" {
			label = "character"
		}
		return "Put on " + label
	case "persona/unequip":
		label := reviewPersonaLabel(reviewPersonaFromPayload(payload, cards))
		if label == "" {
			label = "character"
		}
		return "Took off " + label
	default:
		return "Recorded action"
	}
}

func reviewPersonaFromPayload(payload map[string]any, cards *reviewCardCache) *ReviewPersona {
	if payload == nil {
		return nil
	}
	raw := payload["actor_persona"]
	if raw == nil {
		raw = payload["persona"]
	}
	persona := reviewPersonaFromAny(raw)
	if persona != nil {
		return persona
	}
	if cardID := strings.TrimSpace(stringFromMap(payload, "character_card_id")); cardID != "" {
		if card, ok := cards.get(cardID); ok {
			return card
		}
	}
	return nil
}

func reviewPersonaLabel(persona *ReviewPersona) string {
	if persona == nil {
		return ""
	}
	if value := strings.TrimSpace(persona.Name); value != "" {
		return value
	}
	if value := strings.TrimSpace(persona.CharacterCardID); value != "" {
		return value
	}
	return ""
}

func reviewPersonaFromAny(raw any) *ReviewPersona {
	switch v := raw.(type) {
	case map[string]any:
		persona := &ReviewPersona{
			CharacterCardID: stringFromAny(v["character_card_id"]),
			Name:            stringFromAny(v["name"]),
			Pronouns:        stringFromAny(v["pronouns"]),
			PortraitURL:     stringFromAny(v["portrait_url"]),
			Color:           stringFromAny(v["color"]),
			Tagline:         stringFromAny(v["tagline"]),
		}
		if persona.Name == "" && persona.CharacterCardID == "" && persona.Pronouns == "" && persona.PortraitURL == "" && persona.Color == "" && persona.Tagline == "" {
			return nil
		}
		return persona
	case map[string]string:
		persona := &ReviewPersona{
			CharacterCardID: v["character_card_id"],
			Name:            v["name"],
			Pronouns:        v["pronouns"],
			PortraitURL:     v["portrait_url"],
			Color:           v["color"],
			Tagline:         v["tagline"],
		}
		if persona.Name == "" && persona.CharacterCardID == "" && persona.Pronouns == "" && persona.PortraitURL == "" && persona.Color == "" && persona.Tagline == "" {
			return nil
		}
		return persona
	default:
		return nil
	}
}

func reviewTargetElementLabel(target map[string]any, elements *reviewElementCache) string {
	if target == nil {
		return ""
	}
	elementID := strings.TrimSpace(stringFromMap(target, "element_id"))
	elementSlug := strings.TrimSpace(stringFromMap(target, "element_slug"))
	if elementID == "" && elementSlug == "" {
		return ""
	}
	if label, ok := elements.get(elementID, elementSlug); ok {
		return label
	}
	if elementSlug != "" {
		return elementSlug
	}
	return elementID
}

func reviewCountsFromEvents(events []ReviewEvent) ReviewCounts {
	var counts ReviewCounts
	for _, event := range events {
		counts.Total++
		switch event.Category {
		case "chat":
			counts.Chat++
		case "speech":
			counts.Speech++
		case "reactions":
			counts.Reactions++
		case "reveal_hide":
			counts.RevealHide++
		case "overlay":
			counts.Overlay++
		case "cards":
			counts.Cards++
		case "persona":
			counts.Persona++
		}
	}
	return counts
}

func stringFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	return stringFromAny(m[key])
}

func stringFromMapInt(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	switch v := m[key].(type) {
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case json.Number:
		return v.String()
	default:
		return strings.TrimSpace(stringFromAny(v))
	}
}

func stringFromAny(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case fmt.Stringer:
		return t.String()
	case json.Number:
		return t.String()
	default:
		return ""
	}
}

func previewText(value string, limit int) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\n", " "))
	if value == "" {
		return ""
	}
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit]) + "…"
}

func humanizeActionWord(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "_", " ")
	parts := strings.Fields(value)
	for i, part := range parts {
		if len(part) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}

type reviewElementCache struct {
	ctx   context.Context
	pool  *pgxpool.Pool
	cache map[string]string
}

func newReviewElementCache(ctx context.Context, pool *pgxpool.Pool) *reviewElementCache {
	return &reviewElementCache{
		ctx:   ctx,
		pool:  pool,
		cache: map[string]string{},
	}
}

func (c *reviewElementCache) get(elementID, elementSlug string) (string, bool) {
	key := strings.TrimSpace(elementID) + "|" + strings.TrimSpace(elementSlug)
	if value, ok := c.cache[key]; ok {
		return value, value != ""
	}

	var name, slug string
	switch {
	case strings.TrimSpace(elementID) != "":
		_ = c.pool.QueryRow(c.ctx, `
			SELECT COALESCE(NULLIF(name, ''), slug), COALESCE(NULLIF(slug, ''), '')
			FROM elements
			WHERE id = $1::uuid
			LIMIT 1
		`, strings.TrimSpace(elementID)).Scan(&name, &slug)
	case strings.TrimSpace(elementSlug) != "":
		_ = c.pool.QueryRow(c.ctx, `
			SELECT COALESCE(NULLIF(name, ''), slug), COALESCE(NULLIF(slug, ''), '')
			FROM elements
			WHERE slug = $1
			LIMIT 1
		`, strings.TrimSpace(elementSlug)).Scan(&name, &slug)
	}

	if strings.TrimSpace(name) == "" {
		name = strings.TrimSpace(slug)
	}
	c.cache[key] = name
	return name, name != ""
}

type reviewCardCache struct {
	ctx   context.Context
	pool  *pgxpool.Pool
	cache map[string]*ReviewPersona
}

func newReviewCardCache(ctx context.Context, pool *pgxpool.Pool) *reviewCardCache {
	return &reviewCardCache{
		ctx:   ctx,
		pool:  pool,
		cache: map[string]*ReviewPersona{},
	}
}

func (c *reviewCardCache) get(cardID string) (*ReviewPersona, bool) {
	cardID = strings.TrimSpace(cardID)
	if cardID == "" {
		return nil, false
	}
	if value, ok := c.cache[cardID]; ok {
		return value, value != nil
	}

	var persona ReviewPersona
	err := c.pool.QueryRow(c.ctx, `
		SELECT
			id::text,
			COALESCE(NULLIF(name, ''), ''),
			COALESCE(NULLIF(pronouns, ''), ''),
			COALESCE(NULLIF(portrait_url, ''), ''),
			COALESCE(NULLIF(color, ''), ''),
			COALESCE(NULLIF(tagline, ''), '')
		FROM character_cards
		WHERE id = $1::uuid
		  AND is_deleted = FALSE
		LIMIT 1
	`, cardID).Scan(&persona.CharacterCardID, &persona.Name, &persona.Pronouns, &persona.PortraitURL, &persona.Color, &persona.Tagline)
	if err != nil {
		c.cache[cardID] = nil
		return nil, false
	}

	c.cache[cardID] = &persona
	return &persona, true
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
