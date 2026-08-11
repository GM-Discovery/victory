package world

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/rollaudience"
	"victory/backend/internal/showings"
	"victory/backend/internal/tutorial"
)

// resolveStoredRollAudienceMode defaults an absent/malformed stored
// audienceMode to Show -- deliberately not rollaudience.NormalizeMode,
// whose "" case means "apply the Cohort default" for a *new* roll request.
// A roll/dice Action already in the actions table either predates Kernel
// 86 (no audienceMode key at all) or has a value written by
// actions.StoreDiceRoll; either way "" here must fail open to the old
// unrestricted (Show) behavior, matching network/ws.go's identical
// resolvedStoredMode for the live-delivery side of the same contract.
func resolveStoredRollAudienceMode(raw string) string {
	switch strings.TrimSpace(raw) {
	case rollaudience.ModeCohort, rollaudience.ModeDirector, rollaudience.ModePrivate, rollaudience.ModeShow:
		return strings.TrimSpace(raw)
	default:
		return rollaudience.ModeShow
	}
}

type Snapshot struct {
	Location       string          `json:"location"`
	Lot            string          `json:"lot"`
	Venue          Venue           `json:"venue"`
	Session        Session         `json:"session"`
	Showing        Showing         `json:"showing"`
	Elements       []PlacedElement `json:"elements"`
	Overlay        *Overlay        `json:"overlay,omitempty"`
	Actions        []Action        `json:"actions"`
	TheaterContext TheaterContext  `json:"theater_context"`
}

// TheaterContext is the Kernel 70A backend-computed answer to "what should
// this specific viewer see right now" -- role/registration facts a client
// must never re-derive itself. Kind is one of "participant" (a registered
// Show Run player, active session linked to a Show), "audience" (anyone
// else watching an active Show), "venue_open" (no active Show context, a
// non-backstage viewer), or "backstage" (Director/Producer/Operator/Crew,
// either idle or watching without playing).
type TheaterContext struct {
	Kind    string `json:"kind"`
	Message string `json:"message,omitempty"`
	// SelectedCharacterID is the Kernel 71 Show-Run-scoped Character the
	// viewer is presenting as, when Kind is "participant" and they've
	// chosen one. Empty otherwise.
	SelectedCharacterID string `json:"selected_character_id,omitempty"`
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
	// ShowID and CurrentShowScenePlacementID are the minimum Kernel 70
	// linkage a client needs to look up its own eligible Cue stage
	// buttons (GET /api/shows/{show_id}/scenes/{placement_id}/player-
	// cues) -- deliberately NOT Show variables or Cue definitions
	// themselves, which stay backstage-only. Empty when this session
	// isn't linked to a Show, or the Show has no current placement.
	ShowID                      string `json:"show_id,omitempty"`
	CurrentShowScenePlacementID string `json:"current_show_scene_placement_id,omitempty"`
	// LocalProjection is Kernel 74's participant-local tutorial-handoff
	// presentation, present only for the one viewer who entered it. When
	// set, this viewer's Elements and backdrop come from the projection's
	// Scene instead of the current placement's -- but
	// CurrentShowScenePlacementID above still reports the UNCHANGED shared
	// Scene, which is what makes "the shared current Scene did not move"
	// directly assertable from a Player's own snapshot.
	LocalProjection *LocalProjection `json:"local_projection,omitempty"`
	// TutorialState is Kernel 75's viewer-scoped tutorial position, present
	// only for a participant who has recorded progress. It exists so the
	// completion Program's reopen affordance and the waiting copy render
	// straight from the snapshot instead of polling.
	//
	// Viewer-scoped in the strict sense: it reports THIS viewer's own
	// milestones and nothing about anyone else's. It deliberately carries no
	// denominator, no participant count, and no "N of M ready" field --
	// S1.11 forbids that framing, and a type with no such field cannot grow
	// one by accident.
	TutorialState *TutorialState `json:"tutorial_state,omitempty"`
}

// TutorialState is the curated, viewer-facing tutorial position.
type TutorialState struct {
	Milestones []string `json:"milestones"`
	// CompletionAvailable is true once the gate is open, i.e. once this
	// Player could press Continue or reopen the completion Program.
	CompletionAvailable bool `json:"completion_available"`
	// Completed is true once they actually pressed it.
	Completed bool `json:"completed"`
	// CompletionInteractionID is the Ra interaction the Program hangs off,
	// so the client can call /tutorial/completion without rediscovering it.
	CompletionInteractionID string `json:"completion_interaction_id,omitempty"`
}

// LocalProjection is the curated, viewer-facing shape of an active
// participant-local projection -- deliberately not the whole row (no
// cleared_at, no origin placement, no other Player's identity), mirroring
// how cues.PlayerVisibleCue curates a Cue.
type LocalProjection struct {
	SceneID    string `json:"scene_id"`
	SceneSlug  string `json:"scene_slug"`
	SceneTitle string `json:"scene_title"`
	// BackdropURL is the projection Scene's map_backdrop content URL, if it
	// authored one. Supplied here so the stage renderer can swap backdrops
	// through its existing texture lifecycle without a second fetch.
	BackdropURL string `json:"backdrop_url,omitempty"`
	GridEnabled bool   `json:"grid_enabled"`
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

func LoadCaveSnapshot(ctx context.Context, pool *pgxpool.Pool, viewerRole, viewerUserID string) (*Snapshot, error) {
	return LoadVenueSnapshot(ctx, pool, viewerRole, viewerUserID, "the-cave")
}

func LoadVenueSnapshot(ctx context.Context, pool *pgxpool.Pool, viewerRole, viewerUserID, venueSlug string) (*Snapshot, error) {
	venueSlug = strings.ToLower(strings.TrimSpace(venueSlug))
	if venueSlug == "" {
		return nil, errors.New("venue slug is required")
	}

	var snap Snapshot
	snap.Elements = []PlacedElement{}
	snap.Actions = []Action{}

	var venueConfig []byte
	var startedAt *time.Time
	var showID string

	// s.id/s.status/s.started_at come from a LEFT JOIN and must tolerate a
	// venue with zero active sessions -- COALESCE the text columns (same
	// pattern already used for sh.* two lines below) and scan started_at
	// into a nullable pointer, since COALESCE alone can't produce a
	// non-null time.Time from SQL NULL the way it can for text.
	err := pool.QueryRow(ctx, `
		SELECT
			l.name AS location_name,
			lo.name AS lot_name,
			v.id,
			v.name,
			v.slug,
			v.kind,
			v.config,
			COALESCE(s.id::text, ''),
			COALESCE(s.status::text, ''),
			s.started_at,
			COALESCE(s.show_id::text, ''),
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
		WHERE v.slug = $1
		ORDER BY s.started_at DESC NULLS LAST
		LIMIT 1
	`, venueSlug).Scan(
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
		&showID,
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

	if startedAt != nil {
		snap.Session.StartedAt = startedAt.UTC().Format(time.RFC3339)
	}
	snap.Session.ShowID = showID
	if showID != "" {
		var currentPlacementID *string
		if err := pool.QueryRow(ctx, `
			SELECT current_show_scene_placement_id::text FROM shows WHERE id = $1
		`, showID).Scan(&currentPlacementID); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		if currentPlacementID != nil {
			snap.Session.CurrentShowScenePlacementID = *currentPlacementID
		}
	}

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
		WHERE v.slug = $1
		ORDER BY e.name ASC
	`, venueSlug)
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

	placementIndex := make(map[string]int, len(snap.Elements))
	for i, el := range snap.Elements {
		placementIndex[placedElementKey(el)] = i
	}

	// showID (the active session's linked Show, if any -- Kernel 70 SS4.3)
	// folds persistent show_id-scoped actions in alongside this session's
	// own session_id-scoped actions, so a Show's state set by a Cue in an
	// earlier, now-ended session is still replayed here. When showID is
	// "" (no linked Show, the pre-Kernel-70 case), the query below reduces
	// to exactly the old `WHERE a.session_id = $1` filter -- byte-identical
	// behavior for every session not linked to a Show.
	if err := reconcilePlacedElementsFromActions(ctx, pool, snap.Session.ID, showID, &snap.Elements, placementIndex); err != nil {
		return nil, err
	}
	compactElements := snap.Elements[:0]
	for _, el := range snap.Elements {
		if strings.TrimSpace(el.ElementID) == "" {
			continue
		}
		compactElements = append(compactElements, el)
	}
	snap.Elements = compactElements

	// Kernel 70A: a venue with no active session is a normal, expected
	// "idle theater" state, not an error -- resolveTheaterContext below is
	// what tells a caller whether to render "No Show is currently on stage
	// here." (venue_open) or a backstage idle view, instead of an error
	// page. snap.Session.ID/showID stay "" here since there's no active
	// session row to derive them from.

	// The WHERE clause folds in show_id-scoped actions (persistent, may
	// have been recorded in a different, now-ended session) alongside
	// this session's own session_id-scoped actions. When showID is ""
	// this reduces to exactly `a.session_id = $1`, matching pre-Kernel-70
	// behavior byte-for-byte. Ordering switches from moment_id (only
	// unique/comparable within one session) to ts, which is comparable
	// across sessions and Show-scoped rows alike. Both sides are guarded
	// against an empty-string comparison against a NOT NULL uuid column
	// (reachable now that a sessionless snapshot no longer returns early).
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
		WHERE (($1 <> '' AND a.session_id::text = $1) OR ($2 <> '' AND a.show_id IS NOT NULL AND a.show_id::text = $2))
		ORDER BY a.ts DESC, a.moment_id DESC
	`, snap.Session.ID, showID)
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
		if source, _ := a.Payload["source"].(string); strings.EqualFold(strings.TrimSpace(source), "discord") {
			if discordMeta, ok := a.Payload["discord"].(map[string]any); ok {
				if linkedUserID := strings.TrimSpace(stringValueMap(discordMeta, "linked_user_id")); linkedUserID == "" {
					if value := strings.TrimSpace(stringValueMap(discordMeta, "author_global_name")); value != "" {
						a.ActorDisplayName = value
					} else if value := strings.TrimSpace(stringValueMap(discordMeta, "author_username")); value != "" {
						a.ActorDisplayName = value
					}
					if value := strings.TrimSpace(stringValueMap(discordMeta, "author_username")); value != "" {
						a.ActorHandle = value
					}
					a.ActorRole = "audience"
				}
			}
		}
		a.Actor = map[string]any{
			"user_id":      a.ActorID,
			"handle":       a.ActorHandle,
			"display_name": a.ActorDisplayName,
			"role":         a.ActorRole,
			"persona":      a.Persona,
		}
		a.Timestamp = ts.UTC().Format(time.RFC3339)

		if a.Type == "roll/dice" {
			audienceMode, _ := a.Visibility["audienceMode"].(string)
			cohortID, _ := a.Visibility["cohortId"].(string)
			decision := rollaudience.Decision{Mode: resolveStoredRollAudienceMode(audienceMode), CohortID: cohortID}
			visible, verr := rollaudience.VisibleToViewer(ctx, pool, decision, a.ActorID, showID, viewerUserID, viewerRole)
			if verr != nil {
				return nil, verr
			}
			if !visible {
				continue
			}
		}

		snap.Actions = append(snap.Actions, a)
	}

	if err := actionRows.Err(); err != nil {
		return nil, err
	}

	// Kernel 73A: when this session's Show has a current Scene Placement,
	// fold that Scene's own authored composition (backend/internal/scenes'
	// scene_stage_elements, Base layer + this placement's Show layer) into
	// the same Elements array warehouse-asset tokens/index cards already
	// live in, so the existing role-based visibility filtering below and
	// the frontend's existing normalizeSnapshot/applyAction rendering path
	// pick them up with zero further client-side special-casing. This is
	// the seam that replaces Kernel 73A pass one's manual "Load Composition
	// Into Live Session" button and its stopgap 'scene_composition_loaded'
	// action type -- composition is now simply part of what a snapshot
	// *is* for the Scene that's current, computed fresh on every read
	// (same "not persisted, computed on read" model this function already
	// uses for the actions-log replay above), not a client-interpreted event.
	//
	// Kernel 74 moved resolveTheaterContext above this block (it used to run
	// after the visibility filter) because composition loading now needs the
	// viewer's own participation identity -- specifically the selected
	// Character -- to decide whether a milestone-gated hotspot is part of
	// this viewer's stage at all. Nothing about the context computation
	// itself changed.
	theaterContext, err := resolveTheaterContext(ctx, pool, viewerUserID, viewerRole, snap.Session.ID, showID)
	if err != nil {
		return nil, err
	}
	snap.TheaterContext = theaterContext

	// Kernel 74: a viewer with an active participant-local projection
	// resolves a DIFFERENT Scene's composition -- their own tutorial-handoff
	// map -- while every other viewer, the Audience, and
	// snap.Session.CurrentShowScenePlacementID above are all untouched.
	//
	// Note what is substituted and what is not: only WHICH Scene's elements
	// are loaded. The elements themselves then fall through the identical
	// role-based visibility filter below. There is still exactly one object
	// model, which is the constraint cues/types.go:43-48 recorded.
	// Scoped to the viewer's CURRENTLY SELECTED Character, not just their
	// account: switching Character must return them to the shared stage
	// rather than stranding a Character who never played the tutorial on the
	// handoff map (S5.2).
	localProjection, err := loadLocalProjectionForViewer(ctx, pool, viewerUserID, theaterContext.SelectedCharacterID, showID)
	if err != nil {
		return nil, err
	}
	snap.Session.LocalProjection = localProjection

	// Kernel 75: this viewer's own tutorial position, on the same
	// already-resolved (viewer, Character, Show) triple the projection
	// lookup above just used. One extra query on a path that is already
	// querying, and it is what lets the completion Program's reopen
	// affordance and the waiting copy render without polling.
	tutorialState, err := loadTutorialStateForViewer(ctx, pool, viewerUserID, theaterContext.SelectedCharacterID, showID)
	if err != nil {
		return nil, err
	}
	snap.Session.TutorialState = tutorialState

	participation := tutorial.Participation{
		UserID:          viewerUserID,
		CharacterCardID: theaterContext.SelectedCharacterID,
		ShowID:          showID,
	}
	// Backstage viewers see milestone-gated elements unconditionally so
	// Directors can author and preview the door without first playing the
	// tutorial as a Player (S6.2).
	gate := milestoneGate{pool: pool, participation: participation, bypass: isBackstageRole(viewerRole)}

	// Kernel 85: a viewer assigned to a Show Cohort with its own current
	// Scene resolves THAT placement's composition instead of the Show's
	// shared one -- the group-scale sibling of Kernel 74's individual
	// localProjection immediately above. Checked only when there is no
	// active local projection (an individual tutorial-handoff override
	// always wins) and only by cohort ASSIGNMENT, never by role or tutorial
	// state -- which is what keeps an Ungrouped participant on the
	// unchanged Show-global path with no separate boundary check needed
	// (kernel-85 S5.6): Ungrouped participants have no assignment row, so
	// this lookup always returns "", and nothing here can ever move them
	// past the tutorial-finish Scene. snap.Session.CurrentShowScenePlacementID
	// still reports the Show's own unchanged shared Scene either way, the
	// same "what moved is only which Scene's elements load" guarantee
	// documented on localProjection above.
	cohortPlacementID := ""
	if localProjection == nil && showID != "" && viewerUserID != "" {
		cohortPlacementID, err = resolveCohortPlacementForViewer(ctx, pool, showID, viewerUserID)
		if err != nil {
			return nil, err
		}
	}

	switch {
	case localProjection != nil:
		compositionElements, err := loadSceneBaseCompositionAsPlacedElements(ctx, pool, localProjection.SceneID, gate)
		if err != nil {
			return nil, err
		}
		snap.Elements = append(snap.Elements, compositionElements...)
	case cohortPlacementID != "":
		compositionElements, err := loadSceneCompositionAsPlacedElements(ctx, pool, cohortPlacementID, gate)
		if err != nil {
			return nil, err
		}
		snap.Elements = append(snap.Elements, compositionElements...)
	case snap.Session.CurrentShowScenePlacementID != "":
		compositionElements, err := loadSceneCompositionAsPlacedElements(ctx, pool, snap.Session.CurrentShowScenePlacementID, gate)
		if err != nil {
			return nil, err
		}
		snap.Elements = append(snap.Elements, compositionElements...)
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

// loadSceneCompositionAsPlacedElements resolves a Show Scene Placement's
// Kernel 73A composition (scene_stage_elements' Base layer + this
// placement's own Show layer, plus each element's stage_element_bindings
// row if any) and converts each row into a synthetic PlacedElement, so it
// flows through this file's existing overlay/visibility/role-filtering
// pipeline identically to a warehouse-asset-backed token or index card.
//
// This package deliberately does NOT import backend/internal/scenes to get
// at scenes.LoadResolvedComposition -- world is imported by
// backend/internal/network (director_console.go), and shows (which scenes
// itself imports) imports network back (sessions.go), so
// world -> scenes -> shows -> network -> world would be a real import
// cycle. A narrow duplicated SQL query against tables this package doesn't
// "own" is the established smaller, safer fix in this codebase for exactly
// this shape of problem (e.g. shows/stage.go's own placementOwnership
// query against show_scene_placements rather than importing scenes back).
//
// ElementID is prefixed "scene:" (never a bare UUID) so a client can tell a
// Kernel 73A composition element apart from a legacy warehouse element
// sharing this same array -- composition elements have no warehouse asset,
// no elements-table row, and are never targets of create/token,
// act/place_element, or the lock/nameplate/duplicate/remove action family
// those rows support; they are edited exclusively through
// backend/internal/scenes' own stage-element endpoints and simply re-read
// fresh on every snapshot call (never mutated by an action replay).
func loadSceneCompositionAsPlacedElements(ctx context.Context, pool *pgxpool.Pool, placementID string, gate milestoneGate) ([]PlacedElement, error) {
	var sceneID string
	if err := pool.QueryRow(ctx, `
		SELECT scene_id::text FROM show_scene_placements WHERE id = $1
	`, placementID).Scan(&sceneID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// A stale current-scene pointer (placement archived/removed
			// out from under it) should not fail the whole snapshot --
			// render the rest of the stage with no composition layer.
			return nil, nil
		}
		return nil, err
	}
	return loadCompositionRows(ctx, pool, sceneID, placementID, gate)
}

// loadSceneBaseCompositionAsPlacedElements is the Kernel 74 sibling used by
// a participant-local projection: it renders a Scene's Base layer with no
// Show-layer overrides, because a local projection is not a Show Scene
// Placement and therefore has no placement-scoped layer to resolve. Passing
// an empty placementID makes the Show-layer predicate match nothing, which
// is exactly the intended semantics rather than an accident of SQL.
func loadSceneBaseCompositionAsPlacedElements(ctx context.Context, pool *pgxpool.Pool, sceneID string, gate milestoneGate) ([]PlacedElement, error) {
	return loadCompositionRows(ctx, pool, sceneID, "", gate)
}

func loadCompositionRows(ctx context.Context, pool *pgxpool.Pool, sceneID, placementID string, gate milestoneGate) ([]PlacedElement, error) {
	var placementArg any
	if strings.TrimSpace(placementID) != "" {
		placementArg = placementID
	}

	rows, err := pool.Query(ctx, `
		SELECT
			sse.id::text, sse.kind, COALESCE(sse.label, ''),
			CASE WHEN sse.show_scene_placement_id IS NULL THEN 'base' ELSE 'show' END,
			sse.data::text, sse.position::text, sse.visibility::text,
			sse.width, sse.height,
			COALESCE(seb.binding_type, ''), COALESCE(seb.participant_interaction_id::text, ''),
			COALESCE(pi.stage_button_label, ''), COALESCE(pi.enabled, FALSE),
			COALESCE(pi.interaction_type, ''), COALESCE(seb.requires_milestone, '')
		FROM scene_stage_elements sse
		LEFT JOIN stage_element_bindings seb ON seb.scene_stage_element_id = sse.id AND seb.binding_type = 'participant_interaction'
		LEFT JOIN participant_interactions pi ON pi.id = seb.participant_interaction_id
		WHERE sse.scene_id = $1 AND (sse.show_scene_placement_id IS NULL OR sse.show_scene_placement_id = $2::uuid)
		ORDER BY (sse.show_scene_placement_id IS NULL) DESC, sse.sort_order ASC, sse.created_at ASC
	`, sceneID, placementArg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PlacedElement
	for rows.Next() {
		var id, kind, label, layer, dataText, positionText, visibilityText, bindingType, bindingInteractionID, bindingLabel, bindingInteractionType, requiresMilestone string
		var bindingEnabled bool
		var width, height *float64
		if err := rows.Scan(&id, &kind, &label, &layer, &dataText, &positionText, &visibilityText,
			&width, &height, &bindingType, &bindingInteractionID, &bindingLabel, &bindingEnabled,
			&bindingInteractionType, &requiresMilestone); err != nil {
			return nil, err
		}

		// Kernel 74 S6.2: the reveal gate is applied HERE, by omitting the
		// element from the snapshot entirely, rather than by shipping it
		// with a "hidden" flag the client is trusted to honor. A Player who
		// has not completed Kessa has no door element in their payload at
		// all, so forging client state cannot reveal it -- and the submit
		// endpoint refuses independently even if they synthesize one.
		if requiresMilestone != "" {
			allowed, err := gate.allows(ctx, requiresMilestone)
			if err != nil {
				return nil, err
			}
			if !allowed {
				continue
			}
		}

		data := decodeJSONMap([]byte(dataText))
		if data == nil {
			data = map[string]any{}
		}
		data["scene_stage_element_id"] = id
		data["kind"] = kind
		data["layer"] = layer
		if width != nil {
			data["width"] = *width
		}
		if height != nil {
			data["height"] = *height
		}
		if bindingInteractionID != "" {
			data["binding"] = map[string]any{
				"binding_type":               bindingType,
				"participant_interaction_id": bindingInteractionID,
				"stage_button_label":         bindingLabel,
				"enabled":                    bindingEnabled,
				// Kernel 74: the client needs the type to know WHICH Program
				// to open -- Equip Mode, the freeform door prompt, or Ra's
				// guided dialogue. It is a curated authoring label, not
				// configuration: no packet slug, prompt text, or milestone key
				// crosses to the Player here.
				"interaction_type": bindingInteractionType,
			}
		}

		position := decodeJSONMap([]byte(positionText))
		if position == nil {
			position = map[string]any{}
		}
		visibility := decodeJSONMap([]byte(visibilityText))
		if visibility == nil {
			visibility = map[string]any{"toRoles": []any{"audience", "cast", "crew", "director", "producer"}, "privateTo": []any{}}
		}
		// Bridge Kernel 73A's {toRoles, privateTo} visibility model onto the
		// older {visible: bool} model effectiveAudienceVisible/
		// elementVisibilityForLayer actually read (inherited from warehouse
		// tokens) -- without this, a composition element with no explicit
		// "visible" key reads as invisible by default for every audience-
		// tier viewer, silently, regardless of toRoles ever including
		// "audience". This was a real bug: Kessa's own base-layer token
		// vanished for any viewer whose session role resolved to audience.
		if _, hasVisibleKey := visibility["visible"]; !hasVisibleKey {
			visibleToAudience := true
			if toRoles, ok := visibility["toRoles"].([]any); ok {
				visibleToAudience = false
				for _, r := range toRoles {
					if s, ok := r.(string); ok && strings.EqualFold(s, "audience") {
						visibleToAudience = true
						break
					}
				}
			}
			visibility["visible"] = visibleToAudience
		}

		out = append(out, PlacedElement{
			ElementID:    "scene:" + id,
			Name:         label,
			Slug:         "scene-" + id,
			ElementType:  "scene_" + kind,
			ContextClass: "scene_composition",
			Surface:      "stage",
			Position:     position,
			Visibility:   visibility,
			State: map[string]any{
				"locked":            false,
				"nameplate_visible": true,
				"visible":           true,
			},
			Data: data,
		})
	}
	return out, rows.Err()
}

// isBackstageRole reports whether role is one of the Kernel 70A
// "Director/Producer/Operator/Crew" backstage tier -- the set of viewers
// who get a "backstage" TheaterContext instead of "venue_open"/"audience"
// when they're not a registered Show Run player themselves.
func isBackstageRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "producer", "director", "operator", "crew":
		return true
	default:
		return false
	}
}

// resolveTheaterContext computes the Kernel 70A theater-context priority
// order (kernel §4): a registered Show Run player in the current Show
// outranks a plain Audience viewer, which outranks the generic "venue is
// open, nothing on stage" state, which outranks the backstage-staff idle
// view. sessionID/showID being "" means there's no active session at this
// venue right now -- the only two reachable kinds are then "venue_open" or
// "backstage", split purely on viewerRole.
func resolveTheaterContext(ctx context.Context, pool *pgxpool.Pool, viewerUserID, viewerRole, sessionID, showID string) (TheaterContext, error) {
	backstageTier := isBackstageRole(viewerRole)

	if sessionID == "" || showID == "" {
		if backstageTier {
			return TheaterContext{Kind: "backstage"}, nil
		}
		return TheaterContext{Kind: "venue_open", Message: "No Show is currently on stage here."}, nil
	}

	viewerUserID = strings.TrimSpace(viewerUserID)
	isRegisteredPlayer := false
	selectedCharacterID := ""
	if viewerUserID != "" {
		var showRunID string
		err := pool.QueryRow(ctx, `SELECT show_run_id::text FROM shows WHERE id = $1`, showID).Scan(&showRunID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return TheaterContext{}, err
		}
		if showRunID != "" {
			// Kernel 71: the Show-Run-scoped character_card_id is the
			// canonical "has this Player chosen a Character" signal,
			// replacing the Session-scoped current_session_personas check
			// -- a Player's selection now belongs to their participation
			// in the Show Run, not to any one live Session. An archived
			// Character (is_deleted) is treated as unselected, per the
			// required fallback (spec §7.2).
			var rosterRole string
			var characterCardID *string
			var characterIsDeleted *bool
			err := pool.QueryRow(ctx, `
				SELECT rm.role, rm.character_card_id::text, cc.is_deleted
				FROM show_run_roster_members rm
				LEFT JOIN character_cards cc ON cc.id = rm.character_card_id
				WHERE rm.show_run_id = $1 AND rm.user_id = $2 AND rm.removed_at IS NULL
				LIMIT 1
			`, showRunID, viewerUserID).Scan(&rosterRole, &characterCardID, &characterIsDeleted)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return TheaterContext{}, err
			}
			isRegisteredPlayer = rosterRole == "player"
			if characterCardID != nil && (characterIsDeleted == nil || !*characterIsDeleted) {
				selectedCharacterID = *characterCardID
			}
		}
	}

	if isRegisteredPlayer {
		if selectedCharacterID == "" {
			return TheaterContext{Kind: "participant", Message: "You are registered for this Show, but you have not chosen a Character yet."}, nil
		}
		return TheaterContext{Kind: "participant", SelectedCharacterID: selectedCharacterID}, nil
	}

	if backstageTier {
		return TheaterContext{Kind: "backstage"}, nil
	}

	return TheaterContext{Kind: "audience", Message: "You are watching this Show. Player controls are not active."}, nil
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

// reconcilePlacedElementsFromActions replays stage-object actions onto the
// persisted default layout. showID (Kernel 70 SS4.3) folds in persistent
// show_id-scoped actions -- recorded under any session ever linked to this
// Show, not just the current one -- alongside this session's own
// session_id-scoped actions, replayed in a single chronological (ts ASC)
// pass so a later action always wins regardless of which session recorded
// it. When showID is "" (no linked Show), the query reduces to exactly
// `a.session_id = $1`, matching pre-Kernel-70 behavior byte-for-byte.
func reconcilePlacedElementsFromActions(ctx context.Context, pool *pgxpool.Pool, sessionID, showID string, elements *[]PlacedElement, placementIndex map[string]int) error {
	// sessionID may be "" when a venue has no active session (Kernel 70A:
	// LoadVenueSnapshot no longer errors out before reaching this call in
	// that case) -- guard the session_id comparison the same way showID
	// already is, since a.session_id is a NOT NULL uuid column and
	// comparing it against a bare empty string fails with a Postgres
	// invalid-uuid-syntax error rather than simply matching nothing.
	rows, err := pool.Query(ctx, `
		SELECT
			a.type,
			a.target,
			a.payload
		FROM actions a
		WHERE (($1 <> '' AND a.session_id::text = $1) OR ($2 <> '' AND a.show_id IS NOT NULL AND a.show_id::text = $2))
		  AND a.type IN ('act/place_element', 'act/remove_element', 'create/token', 'update/token', 'act/duplicate_element', 'delete/index_card')
		ORDER BY a.ts ASC, a.moment_id ASC
	`, sessionID, showID)
	if err != nil {
		return err
	}
	defer rows.Close()

	deletedIndexCards := map[string]struct{}{}

	for rows.Next() {
		var actionType string
		var targetRaw, payloadRaw []byte
		if err := rows.Scan(&actionType, &targetRaw, &payloadRaw); err != nil {
			return err
		}

		action := Action{
			Type:    actionType,
			Target:  decodeJSONMap(targetRaw),
			Payload: decodeJSONMap(payloadRaw),
		}

		switch actionType {
		case "delete/index_card":
			elementID := strings.ToLower(strings.TrimSpace(actionTargetElementID(action)))
			if elementID == "" {
				continue
			}
			deletedIndexCards[elementID] = struct{}{}
			removePlacedElementByAction(elements, placementIndex, Action{
				Type:   "delete/index_card",
				Target: action.Target,
			})
		case "act/remove_element":
			removePlacedElementByAction(elements, placementIndex, action)
		case "act/place_element":
			if elementID := strings.ToLower(strings.TrimSpace(actionTargetElementID(action))); elementID != "" {
				if _, deleted := deletedIndexCards[elementID]; deleted {
					continue
				}
			}
			if err := upsertPlacedElementFromAction(ctx, pool, elements, placementIndex, action, false); err != nil {
				return err
			}
		case "create/token":
			if err := upsertPlacedElementFromAction(ctx, pool, elements, placementIndex, action, true); err != nil {
				return err
			}
		case "update/token":
			if err := upsertPlacedElementFromAction(ctx, pool, elements, placementIndex, action, true); err != nil {
				return err
			}
		case "act/duplicate_element":
			if err := upsertDuplicatePlacedElementFromAction(ctx, pool, elements, placementIndex, action); err != nil {
				return err
			}
		}
	}

	return rows.Err()
}

func upsertPlacedElementFromAction(ctx context.Context, pool *pgxpool.Pool, elements *[]PlacedElement, placementIndex map[string]int, action Action, tokenOnly bool) error {
	elementID := strings.TrimSpace(actionTargetElementID(action))
	if elementID == "" {
		return nil
	}
	venueSlug := strings.ToLower(strings.TrimSpace(stringValueMap(action.Target, "venue_slug")))
	if venueSlug == "" {
		venueSlug = "the-cave"
	}
	surface := strings.ToLower(strings.TrimSpace(actionTargetLayer(action)))
	if surface == "" {
		surface = strings.ToLower(strings.TrimSpace(stringValueMap(action.Payload, "surface")))
	}
	if surface == "" {
		surface = "stage"
	}

	var row struct {
		ElementID    string
		Name         string
		Slug         string
		ElementType  string
		ContextClass string
		Data         map[string]any
	}
	var positionRaw []byte
	var dataRaw []byte
	if err := pool.QueryRow(ctx, `
		SELECT
			e.id::text,
			e.name,
			e.slug,
			e.element_type,
			COALESCE(NULLIF(e.context_class, ''), '') AS context_class,
			e.data,
			COALESCE(vle.position, '{}'::jsonb) AS position
		FROM elements e
		JOIN venues v ON v.slug = $1
		LEFT JOIN venue_layout_elements vle ON vle.venue_id = v.id AND vle.element_id = e.id AND vle.surface = $2
		WHERE e.id = $3
		  AND COALESCE(e.state::text, '') <> 'deleted'
		LIMIT 1
	`, venueSlug, surface, elementID).Scan(&row.ElementID, &row.Name, &row.Slug, &row.ElementType, &row.ContextClass, &dataRaw, &positionRaw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	row.Data = decodeJSONMap(dataRaw)
	positionData := decodeJSONMap(positionRaw)
	if assetID := strings.TrimSpace(stringValueMap(row.Data, "asset_id")); assetID != "" {
		if assetDeleted, err := tokenAssetDeleted(ctx, pool, assetID); err == nil && assetDeleted {
			row.Data["asset_content_url"] = "/assets/construction.png"
			row.Data["asset_thumbnail_url"] = "/assets/construction.png"
			row.Data["asset_status"] = "deleted"
		}
	}

	position := map[string]any{
		"anchor": surface,
		"x":      numberValueMap(action.Payload, "x", numberValueMap(positionData, "x", 0)),
		"y":      numberValueMap(action.Payload, "y", numberValueMap(positionData, "y", 0)),
		"z":      0,
		"order":  int(numberValueMap(action.Payload, "order", numberValueMap(positionData, "order", 0))),
	}
	if tokenOnly {
		if frame := strings.TrimSpace(stringValueMap(positionData, "frame")); frame != "" {
			position["frame"] = frame
		} else {
			position["frame"] = "center"
		}
	} else {
		if frame := strings.TrimSpace(stringValueMap(positionData, "frame")); frame != "" {
			position["frame"] = frame
		} else {
			position["frame"] = "top-left"
		}
	}

	visibility := map[string]any{
		"toRoles":           []string{"director", "producer"},
		"privateTo":         []string{},
		"visible":           true,
		"nameplate_visible": true,
		"locked":            false,
	}
	if existingIndex, ok := placementIndex[placedElementKey(PlacedElement{ElementID: row.ElementID, Surface: surface})]; ok {
		existing := (*elements)[existingIndex]
		for k, v := range existing.Visibility {
			visibility[k] = v
		}
		if nameplateVisible, ok := existing.State["nameplate_visible"]; ok {
			visibility["nameplate_visible"] = nameplateVisible
		}
		if locked, ok := existing.State["locked"]; ok {
			visibility["locked"] = locked
		}
		if visible, ok := existing.State["visible"]; ok {
			visibility["visible"] = visible
		}
	}
	if layer, _ := action.Payload["token_layer"].(string); strings.EqualFold(strings.TrimSpace(layer), "director") {
		visibility["visible"] = false
	}

	placement := PlacedElement{
		ElementID:    row.ElementID,
		Name:         row.Name,
		Slug:         row.Slug,
		ElementType:  row.ElementType,
		ContextClass: strings.ToLower(strings.TrimSpace(row.ContextClass)),
		Surface:      surface,
		Position:     position,
		Visibility:   visibility,
		State: map[string]any{
			"locked":            false,
			"nameplate_visible": visibilityBool(visibility, "nameplate_visible", true),
			"visible":           visibilityBool(visibility, "visible", true),
		},
		Data: row.Data,
	}
	if placement.ContextClass == "" {
		placement.ContextClass = deriveElementContextClass(row.ElementType, row.Slug, row.Data)
	}

	key := placedElementKey(placement)
	if existingIndex, ok := placementIndex[key]; ok {
		(*elements)[existingIndex] = placement
		return nil
	}

	*elements = append(*elements, placement)
	placementIndex[key] = len(*elements) - 1
	return nil
}

func upsertDuplicatePlacedElementFromAction(ctx context.Context, pool *pgxpool.Pool, elements *[]PlacedElement, placementIndex map[string]int, action Action) error {
	newElementID := strings.TrimSpace(stringValueMap(action.Target, "new_element_id"))
	if newElementID == "" {
		newElementID = strings.TrimSpace(actionTargetElementID(action))
	}
	if newElementID == "" {
		return nil
	}
	venueSlug := strings.ToLower(strings.TrimSpace(stringValueMap(action.Target, "venue_slug")))
	if venueSlug == "" {
		venueSlug = "the-cave"
	}

	surface := strings.ToLower(strings.TrimSpace(actionTargetLayer(action)))
	if surface == "" {
		surface = "stage"
	}

	var row struct {
		ElementID    string
		Name         string
		Slug         string
		ElementType  string
		ContextClass string
		Data         map[string]any
	}
	var positionRaw []byte
	var dataRaw []byte
	if err := pool.QueryRow(ctx, `
		SELECT
			e.id::text,
			e.name,
			e.slug,
			e.element_type,
			COALESCE(NULLIF(e.context_class, ''), '') AS context_class,
			e.data,
			COALESCE(vle.position, '{}'::jsonb) AS position
		FROM elements e
		JOIN venues v ON v.slug = $1
		LEFT JOIN venue_layout_elements vle ON vle.venue_id = v.id AND vle.element_id = e.id AND vle.surface = $2
		WHERE e.id = $3
		  AND COALESCE(e.state::text, '') <> 'deleted'
		LIMIT 1
	`, venueSlug, surface, newElementID).Scan(&row.ElementID, &row.Name, &row.Slug, &row.ElementType, &row.ContextClass, &dataRaw, &positionRaw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	row.Data = decodeJSONMap(dataRaw)
	positionData := decodeJSONMap(positionRaw)
	if assetID := strings.TrimSpace(stringValueMap(row.Data, "asset_id")); assetID != "" {
		if assetDeleted, err := tokenAssetDeleted(ctx, pool, assetID); err == nil && assetDeleted {
			row.Data["asset_content_url"] = "/assets/construction.png"
			row.Data["asset_thumbnail_url"] = "/assets/construction.png"
			row.Data["asset_status"] = "deleted"
		}
	}

	position := map[string]any{
		"anchor": surface,
		"x":      numberValueMap(action.Payload, "x", numberValueMap(positionData, "x", 0)),
		"y":      numberValueMap(action.Payload, "y", numberValueMap(positionData, "y", 0)),
		"z":      0,
		"order":  int(numberValueMap(action.Payload, "order", numberValueMap(positionData, "order", 0))),
		"frame":  "top-left",
	}
	visibility := map[string]any{
		"toRoles":           []string{"director", "producer"},
		"privateTo":         []string{},
		"visible":           true,
		"nameplate_visible": true,
		"locked":            false,
	}
	if existingIndex, ok := placementIndex[placedElementKey(PlacedElement{ElementID: row.ElementID, Surface: surface})]; ok {
		existing := (*elements)[existingIndex]
		for k, v := range existing.Visibility {
			visibility[k] = v
		}
		if nameplateVisible, ok := existing.State["nameplate_visible"]; ok {
			visibility["nameplate_visible"] = nameplateVisible
		}
		if locked, ok := existing.State["locked"]; ok {
			visibility["locked"] = locked
		}
		if visible, ok := existing.State["visible"]; ok {
			visibility["visible"] = visible
		}
	}
	placement := PlacedElement{
		ElementID:    row.ElementID,
		Name:         row.Name,
		Slug:         row.Slug,
		ElementType:  row.ElementType,
		ContextClass: strings.ToLower(strings.TrimSpace(row.ContextClass)),
		Surface:      surface,
		Position:     position,
		Visibility:   visibility,
		State: map[string]any{
			"locked":            false,
			"nameplate_visible": visibilityBool(visibility, "nameplate_visible", true),
			"visible":           true,
		},
		Data: row.Data,
	}
	if placement.ContextClass == "" {
		placement.ContextClass = deriveElementContextClass(row.ElementType, row.Slug, row.Data)
	}

	key := placedElementKey(placement)
	if existingIndex, ok := placementIndex[key]; ok {
		(*elements)[existingIndex] = placement
		return nil
	}

	*elements = append(*elements, placement)
	placementIndex[key] = len(*elements) - 1
	return nil
}

func removePlacedElementByAction(elements *[]PlacedElement, placementIndex map[string]int, action Action) {
	elementID := strings.ToLower(strings.TrimSpace(actionTargetElementID(action)))
	if elementID == "" {
		return
	}

	filtered := (*elements)[:0]
	for _, el := range *elements {
		if strings.ToLower(strings.TrimSpace(el.ElementID)) == elementID {
			continue
		}
		filtered = append(filtered, el)
	}
	*elements = filtered
	for key := range placementIndex {
		delete(placementIndex, key)
	}
	for idx, el := range *elements {
		placementIndex[placedElementKey(el)] = idx
	}
}

func placedElementKey(el PlacedElement) string {
	return strings.ToLower(strings.TrimSpace(el.ElementID)) + ":" + strings.ToLower(strings.TrimSpace(el.Surface))
}

func numberValueMap(m map[string]any, key string, fallback float64) float64 {
	if m == nil {
		return fallback
	}
	switch raw := m[key].(type) {
	case float64:
		return raw
	case float32:
		return float64(raw)
	case int:
		return float64(raw)
	case int32:
		return float64(raw)
	case int64:
		return float64(raw)
	case json.Number:
		if v, err := raw.Float64(); err == nil {
			return v
		}
	}
	return fallback
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

func tokenAssetDeleted(ctx context.Context, pool *pgxpool.Pool, assetID string) (bool, error) {
	var deletedAt *time.Time
	var status string
	err := pool.QueryRow(ctx, `
		SELECT deleted_at, COALESCE(status::text, '')
		FROM assets
		WHERE id = $1
		LIMIT 1
	`, assetID).Scan(&deletedAt, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return true, nil
		}
		return false, err
	}
	if deletedAt != nil {
		return true, nil
	}
	return strings.EqualFold(strings.TrimSpace(status), "deleted"), nil
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
