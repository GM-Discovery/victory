package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/identity"
	"victory/backend/internal/showings"
	"victory/backend/internal/world"
)

type directorConsoleCurrentShowing struct {
	ID                  string `json:"id"`
	SessionID           string `json:"session_id"`
	ProductionID        string `json:"production_id"`
	ProductionName      string `json:"production_name"`
	ProductionSlug      string `json:"production_slug"`
	VenueID             string `json:"venue_id"`
	VenueName           string `json:"venue_name"`
	VenueSlug           string `json:"venue_slug"`
	LocationID          string `json:"location_id"`
	LocationName        string `json:"location_name"`
	RunID               string `json:"run_id,omitempty"`
	Status              string `json:"status"`
	AudienceViewEnabled bool   `json:"audience_view_enabled"`
	StartedAt           string `json:"started_at"`
	EndedAt             string `json:"ended_at,omitempty"`
	CreatedBy           string `json:"created_by"`
}

type directorConsoleChatPolicy struct {
	ChatEnabled    bool `json:"chat_enabled"`
	TalkingEnabled bool `json:"talking_enabled"`
}

type directorConsoleControls struct {
	CanStartShowing  bool   `json:"can_start_showing"`
	StartShowingNote string `json:"start_showing_note"`
}

type directorConsoleState struct {
	CurrentRole    string                         `json:"current_role"`
	CurrentShowing *directorConsoleCurrentShowing `json:"current_showing,omitempty"`
	Snapshot       *world.Snapshot                `json:"snapshot,omitempty"`
	Presence       []PresenceUser                 `json:"presence"`
	ChatPolicy     directorConsoleChatPolicy      `json:"chat_policy"`
	Controls       directorConsoleControls        `json:"controls"`
}

type directorConsoleChatPolicyRequest struct {
	ChatEnabled    bool `json:"chat_enabled"`
	TalkingEnabled bool `json:"talking_enabled"`
}

type showingAudienceViewRequest struct {
	Enabled bool `json:"enabled"`
}

func HandleDirectorConsoleCurrent(hub *Hub, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := currentUserIDFromRequest(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		allowed, err := canAccessDirectorConsole(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed"})
			return
		}
		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		state, err := loadDirectorConsoleState(ctx, hub, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "director_console_lookup_failed"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"data": state,
		})
	}
}

func HandleDirectorConsoleAudienceView(hub *Hub, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := currentUserIDFromRequest(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		allowed, err := canAccessDirectorConsole(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed"})
			return
		}
		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		sessionID, err := identity.ResolveActiveCaveSessionID(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "no_active_showing"})
			return
		}

		showingID := strings.TrimSpace(r.PathValue("id"))
		if showingID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "showing_id_required"})
			return
		}

		currentShowing, err := showings.LoadBySession(ctx, pool, sessionID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "showing_not_found"})
			return
		}
		if currentShowing.ID != showingID {
			writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "showing_not_found"})
			return
		}

		var req showingAudienceViewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_json"})
			return
		}

		updated, err := showings.UpdateAudienceViewByID(ctx, pool, showingID, req.Enabled)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
			return
		}

		msgOut, _ := json.Marshal(map[string]any{
			"type":    "showing/update",
			"showing": updated,
		})
		hub.Broadcast(msgOut)

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"data": updated,
		})
	}
}

func HandleDirectorConsoleCloseShowing(hub *Hub, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := currentUserIDFromRequest(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		allowed, err := canAccessDirectorConsole(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed"})
			return
		}
		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		sessionID, err := identity.ResolveActiveCaveSessionID(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "no_active_showing"})
			return
		}

		showingID := strings.TrimSpace(r.PathValue("id"))
		if showingID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "showing_id_required"})
			return
		}

		currentShowing, err := showings.LoadBySession(ctx, pool, sessionID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "showing_not_found"})
			return
		}
		if currentShowing.ID != showingID {
			writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "showing_not_found"})
			return
		}

		closed, err := showings.CloseBySession(ctx, pool, sessionID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
			return
		}

		msgOut, _ := json.Marshal(map[string]any{
			"type":    "showing/update",
			"showing": closed,
		})
		hub.Broadcast(msgOut)

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"data": closed,
		})
	}
}

func HandleDirectorConsoleStartShowing(_ *Hub, _ *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		writeJSON(w, http.StatusNotImplemented, map[string]any{
			"ok":    false,
			"error": "not_implemented",
		})
	}
}

func HandleDirectorConsoleChatPolicy(hub *Hub, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := currentUserIDFromRequest(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		allowed, err := canAccessDirectorConsole(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed"})
			return
		}
		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		venueSlug := strings.TrimSpace(r.PathValue("slug"))
		if venueSlug == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "venue_slug_required"})
			return
		}
		if venueSlug != "the-cave" {
			writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "venue_not_found"})
			return
		}

		if _, err := identity.ResolveActiveCaveSessionID(ctx, pool, userID); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "no_active_showing"})
			return
		}

		var req directorConsoleChatPolicyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_json"})
			return
		}

		config, err := updateVenueChatPolicy(ctx, pool, venueSlug, req.ChatEnabled, req.TalkingEnabled)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "policy_update_failed"})
			return
		}

		msgOut, _ := json.Marshal(map[string]any{
			"type":       "venue/update",
			"venue_slug": venueSlug,
			"config":     config,
		})
		hub.Broadcast(msgOut)

		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"data": map[string]any{
				"venue_slug": venueSlug,
				"config":     config,
			},
		})
	}
}

func loadDirectorConsoleState(ctx context.Context, hub *Hub, pool *pgxpool.Pool, userID string) (directorConsoleState, error) {
	role, err := access.CurrentLocationRole(ctx, pool, userID)
	if err != nil {
		return directorConsoleState{}, err
	}

	allowed, err := canAccessDirectorConsole(ctx, pool, userID)
	if err != nil {
		return directorConsoleState{}, err
	}
	if !allowed {
		return directorConsoleState{}, pgx.ErrNoRows
	}

	state := directorConsoleState{
		CurrentRole: strings.ToLower(strings.TrimSpace(role)),
		Presence:    []PresenceUser{},
		ChatPolicy: directorConsoleChatPolicy{
			ChatEnabled:    true,
			TalkingEnabled: true,
		},
		Controls: directorConsoleControls{
			CanStartShowing:  false,
			StartShowingNote: "Kernel 28 defers new-showing startup.",
		},
	}

	venueConfig, err := loadVenueConfig(ctx, pool, "the-cave")
	if err == nil {
		state.ChatPolicy = chatPolicyFromConfig(venueConfig)
	}

	sessionID, sessionErr := identity.ResolveActiveCaveSessionID(ctx, pool, userID)
	if sessionErr != nil {
		if !errors.Is(sessionErr, pgx.ErrNoRows) {
			return directorConsoleState{}, sessionErr
		}
		if latestShowing, err := loadLatestShowingForVenue(ctx, pool, "the-cave"); err == nil {
			state.CurrentShowing = &latestShowing
		}
		return state, nil
	}

	viewerRole := "director"
	if role == "producer" {
		viewerRole = "producer"
	}

	snapshot, err := world.LoadCaveSnapshot(ctx, pool, viewerRole, userID)
	if err != nil {
		return directorConsoleState{}, err
	}
	state.Snapshot = snapshot
	state.Presence = sortPresenceForDirector(hub.Presence().Snapshot(sessionID))

	if currentShowing, err := loadCurrentShowing(ctx, pool, sessionID); err == nil {
		state.CurrentShowing = &currentShowing
	}

	if snapshot != nil {
		state.ChatPolicy = chatPolicyFromConfig(snapshot.Venue.Config)
	}

	return state, nil
}

func loadCurrentShowing(ctx context.Context, pool *pgxpool.Pool, sessionID string) (directorConsoleCurrentShowing, error) {
	var showing directorConsoleCurrentShowing
	if err := pool.QueryRow(ctx, `
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
			sh.created_by::text
		FROM showings sh
		JOIN venues v ON v.id = sh.venue_id
		JOIN lots lo ON lo.id = v.lot_id
		JOIN locations l ON l.id = lo.location_id
		LEFT JOIN productions p ON p.id = sh.production_id
		WHERE sh.session_id = $1::uuid
		LIMIT 1
	`, sessionID).Scan(
		&showing.ID,
		&showing.SessionID,
		&showing.ProductionID,
		&showing.ProductionName,
		&showing.ProductionSlug,
		&showing.VenueID,
		&showing.VenueName,
		&showing.VenueSlug,
		&showing.LocationID,
		&showing.LocationName,
		&showing.RunID,
		&showing.Status,
		&showing.AudienceViewEnabled,
		&showing.StartedAt,
		&showing.EndedAt,
		&showing.CreatedBy,
	); err != nil {
		return directorConsoleCurrentShowing{}, err
	}

	return showing, nil
}

func loadLatestShowingForVenue(ctx context.Context, pool *pgxpool.Pool, venueSlug string) (directorConsoleCurrentShowing, error) {
	var showing directorConsoleCurrentShowing
	if err := pool.QueryRow(ctx, `
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
			sh.created_by::text
		FROM showings sh
		JOIN venues v ON v.id = sh.venue_id
		JOIN lots lo ON lo.id = v.lot_id
		JOIN locations l ON l.id = lo.location_id
		LEFT JOIN productions p ON p.id = sh.production_id
		WHERE v.slug = $1
		ORDER BY COALESCE(sh.ended_at, sh.started_at) DESC, sh.started_at DESC
		LIMIT 1
	`, venueSlug).Scan(
		&showing.ID,
		&showing.SessionID,
		&showing.ProductionID,
		&showing.ProductionName,
		&showing.ProductionSlug,
		&showing.VenueID,
		&showing.VenueName,
		&showing.VenueSlug,
		&showing.LocationID,
		&showing.LocationName,
		&showing.RunID,
		&showing.Status,
		&showing.AudienceViewEnabled,
		&showing.StartedAt,
		&showing.EndedAt,
		&showing.CreatedBy,
	); err != nil {
		return directorConsoleCurrentShowing{}, err
	}

	return showing, nil
}

func chatPolicyFromConfig(config map[string]any) directorConsoleChatPolicy {
	return directorConsoleChatPolicy{
		ChatEnabled:    configBoolDefaultTrue(config, "chat_enabled"),
		TalkingEnabled: configBoolDefaultTrue(config, "talking_enabled"),
	}
}

func configBoolDefaultTrue(config map[string]any, key string) bool {
	if config == nil {
		return true
	}
	if raw, ok := config[key]; ok {
		switch value := raw.(type) {
		case bool:
			return value
		case string:
			switch strings.ToLower(strings.TrimSpace(value)) {
			case "false", "0", "off", "no":
				return false
			case "true", "1", "on", "yes":
				return true
			}
		}
	}
	return true
}

func sortPresenceForDirector(users []PresenceUser) []PresenceUser {
	out := append([]PresenceUser{}, users...)
	roleOrder := map[string]int{
		"producer": 0,
		"director": 1,
		"cast":     2,
		"crew":     3,
		"audience": 4,
	}
	sort.SliceStable(out, func(i, j int) bool {
		roleI := strings.ToLower(strings.TrimSpace(out[i].Role))
		roleJ := strings.ToLower(strings.TrimSpace(out[j].Role))
		orderI := roleOrder[roleI]
		orderJ := roleOrder[roleJ]
		if orderI != orderJ {
			return orderI < orderJ
		}
		if out[i].DisplayName != out[j].DisplayName {
			return out[i].DisplayName < out[j].DisplayName
		}
		return out[i].Handle < out[j].Handle
	})
	return out
}

func currentUserIDFromRequest(ctx context.Context, pool *pgxpool.Pool, r *http.Request) (string, error) {
	sessionCookie := ""
	if c, err := r.Cookie("victory_session"); err == nil {
		sessionCookie = c.Value
	}
	return access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
}

func canAccessDirectorConsole(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, error) {
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}

	role, err := access.CurrentLocationRole(ctx, pool, userID)
	if err != nil {
		return false, err
	}

	switch strings.ToLower(strings.TrimSpace(role)) {
	case "producer", "director":
		return true, nil
	default:
		return false, nil
	}
}

func loadVenueConfig(ctx context.Context, pool *pgxpool.Pool, slug string) (map[string]any, error) {
	var raw []byte
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(config, '{}'::jsonb)
		FROM venues
		WHERE slug = $1
		LIMIT 1
	`, slug).Scan(&raw); err != nil {
		return nil, err
	}

	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func updateVenueChatPolicy(ctx context.Context, pool *pgxpool.Pool, slug string, chatEnabled, talkingEnabled bool) (map[string]any, error) {
	var raw []byte
	if err := pool.QueryRow(ctx, `
		UPDATE venues
		SET config = jsonb_set(
			jsonb_set(COALESCE(config, '{}'::jsonb), '{chat_enabled}', to_jsonb($2::boolean), true),
			'{talking_enabled}',
			to_jsonb($3::boolean),
			true
		)
		WHERE slug = $1
		RETURNING config
	`, slug, chatEnabled, talkingEnabled).Scan(&raw); err != nil {
		return nil, err
	}

	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func directorConsoleElementVisibility(el world.PlacedElement) bool {
	if v, ok := boolFromVisibilityMap(el.Visibility, "audience"); ok {
		return v
	}
	if roles, ok := el.Visibility["toRoles"].([]any); ok {
		for _, role := range roles {
			if strings.EqualFold(strings.TrimSpace(fmt.Sprint(role)), "audience") {
				return true
			}
		}
	}
	if roles, ok := el.Visibility["toRoles"].([]string); ok {
		for _, role := range roles {
			if strings.EqualFold(strings.TrimSpace(role), "audience") {
				return true
			}
		}
	}
	return false
}

func boolFromVisibilityMap(visibility map[string]any, key string) (bool, bool) {
	if visibility == nil {
		return false, false
	}
	raw, ok := visibility[key]
	if !ok {
		return false, false
	}
	switch value := raw.(type) {
	case bool:
		return value, true
	case string:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "true", "1", "yes", "on":
			return true, true
		case "false", "0", "no", "off":
			return false, true
		}
	}
	return false, false
}
