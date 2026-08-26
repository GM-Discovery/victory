package scenes

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/actions"
	"victory/backend/internal/showruns"
	"victory/backend/internal/tutorial"
)

// SceneComposerEnabled checks the venues.config scene_composer_enabled
// boolean flag, following the Kernel 72A/73 venue-capability-flag pattern
// (actions.stageElementsEnabled, merchant.VenueParticipantInteractionsEnabled):
// fail-closed for unknown venues and missing flags.
func SceneComposerEnabled(ctx context.Context, pool *pgxpool.Pool, venueSlug string) (bool, error) {
	venueSlug = strings.ToLower(strings.TrimSpace(venueSlug))
	if venueSlug == "" {
		return false, nil
	}
	var enabled bool
	err := pool.QueryRow(ctx, `
		SELECT COALESCE((config ->> 'scene_composer_enabled')::boolean, FALSE)
		FROM venues
		WHERE slug = $1
		LIMIT 1
	`, venueSlug).Scan(&enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return enabled, nil
}

// composerVenueSlugForScene resolves the venue a Scene's Base layer is
// authored against -- its own default_venue_id. A Show-layer element's
// venue is resolved via composerVenueSlugForPlacement instead, since a
// placement may override the venue.
func composerVenueSlugForScene(ctx context.Context, pool *pgxpool.Pool, sceneID string) (string, error) {
	var slug string
	err := pool.QueryRow(ctx, `
		SELECT v.slug FROM scenes s
		JOIN venues v ON v.id = s.default_venue_id
		WHERE s.id = $1
	`, sceneID).Scan(&slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return slug, nil
}

func composerVenueSlugForPlacement(ctx context.Context, pool *pgxpool.Pool, placementID string) (string, error) {
	var slug string
	err := pool.QueryRow(ctx, `
		SELECT v.slug
		FROM show_scene_placements p
		JOIN scenes s ON s.id = p.scene_id
		JOIN venues v ON v.id = COALESCE(p.venue_id, s.default_venue_id)
		WHERE p.id = $1
	`, placementID).Scan(&slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return slug, nil
}

const stageElementColumns = `
	id::text, scene_id::text, show_scene_placement_id::text, kind, COALESCE(label, ''),
	data::text, position::text, visibility::text, sort_order, width, height,
	created_by_user_id::text, created_at, updated_at
`

func scanStageElement(row pgx.Row) (StageElement, error) {
	var e StageElement
	var placementID, createdByUserID *string
	var dataText, positionText, visibilityText string
	if err := row.Scan(
		&e.ID, &e.SceneID, &placementID, &e.Kind, &e.Label,
		&dataText, &positionText, &visibilityText, &e.SortOrder, &e.Width, &e.Height,
		&createdByUserID, &e.CreatedAt, &e.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StageElement{}, errors.New("stage_element_not_found")
		}
		return StageElement{}, err
	}
	e.ShowScenePlacementID = placementID
	e.CreatedByUserID = createdByUserID
	if dataText != "" {
		e.Data = json.RawMessage(dataText)
	}
	if positionText != "" {
		e.Position = json.RawMessage(positionText)
	}
	if visibilityText != "" {
		e.Visibility = json.RawMessage(visibilityText)
	}
	if placementID != nil {
		e.Layer = "show"
	} else {
		e.Layer = "base"
	}
	return e, nil
}

// canManageComposer is Producer/Director/Operator manage authority for the
// Scene's location, or Crew's non-destructive-edit right -- the same two-
// tier authority Kernel 73's participant_interactions authoring already
// uses (showruns.CanCrewPerformNonDestructiveEdit), since a Scene's
// composition is an authoring surface with the same blast radius as a
// participant interaction, not a destructive Show-lifecycle action.
func canManageComposer(ctx context.Context, pool *pgxpool.Pool, actorUserID, locationID string) (bool, error) {
	ok, err := showruns.CanManageShowRun(ctx, pool, actorUserID, locationID)
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}
	return showruns.CanCrewPerformNonDestructiveEdit(ctx, pool, actorUserID, locationID)
}

// CreateSceneStageElement adds a Base-layer element to a Scene's own
// composition. Requires the scene_composer_enabled capability flag on the
// Scene's default venue.
func CreateSceneStageElement(ctx context.Context, pool *pgxpool.Pool, actorUserID, sceneID string, in CreateStageElementInput) (StageElement, error) {
	return createStageElement(ctx, pool, actorUserID, sceneID, nil, in)
}

// CreatePlacementStageElement adds a Show-layer element scoped to one
// placement, never mutating the Scene's Base layer.
func CreatePlacementStageElement(ctx context.Context, pool *pgxpool.Pool, actorUserID, placementID string, in CreateStageElementInput) (StageElement, error) {
	p, err := LoadPlacementByID(ctx, pool, placementID)
	if err != nil {
		return StageElement{}, err
	}
	return createStageElement(ctx, pool, actorUserID, p.SceneID, &placementID, in)
}

// resolveTokenAssetData fills in the authoritative asset_name/shape/
// content_url/thumbnail_url/etc fields from the Warehouse asset row,
// mirroring what the live create/token and update/token actions compute
// server-side (internal/actions.ResolveTokenAssetFields). The Configurator
// draft path builds a token element straight from client JSON with no
// action-stream round trip, so this is its only chance to do that lookup --
// without it, a client that only ever sends asset_id renders with no
// content URL at all. A no-op when the caller didn't set asset_id (a
// not-yet-assigned token slot).
func resolveTokenAssetData(ctx context.Context, pool *pgxpool.Pool, data map[string]any) error {
	assetID, _ := data["asset_id"].(string)
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return nil
	}
	fields, err := actions.ResolveTokenAssetFields(ctx, pool, assetID)
	if err != nil {
		return err
	}
	for k, v := range fields {
		data[k] = v
	}
	return nil
}

func createStageElement(ctx context.Context, pool *pgxpool.Pool, actorUserID, sceneID string, placementID *string, in CreateStageElementInput) (StageElement, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return StageElement{}, errors.New("not_authenticated")
	}
	kind := strings.TrimSpace(in.Kind)
	if !ValidStageElementKinds[kind] {
		return StageElement{}, errors.New("invalid_kind")
	}

	scene, err := LoadSceneByID(ctx, pool, sceneID)
	if err != nil {
		return StageElement{}, err
	}
	ok, err := canManageComposer(ctx, pool, actorUserID, scene.LocationID)
	if err != nil {
		return StageElement{}, err
	}
	if !ok {
		return StageElement{}, errors.New("not_authorized")
	}

	var venueSlug string
	if placementID != nil {
		venueSlug, err = composerVenueSlugForPlacement(ctx, pool, *placementID)
	} else {
		venueSlug, err = composerVenueSlugForScene(ctx, pool, sceneID)
	}
	if err != nil {
		return StageElement{}, err
	}
	enabled, err := SceneComposerEnabled(ctx, pool, venueSlug)
	if err != nil {
		return StageElement{}, err
	}
	if !enabled {
		return StageElement{}, errors.New("scene_composer_disabled")
	}

	data := in.Data
	if data == nil {
		data = map[string]any{}
	}
	if kind == "token" {
		if err := resolveTokenAssetData(ctx, pool, data); err != nil {
			return StageElement{}, err
		}
	}
	position := in.Position
	if position == nil {
		position = map[string]any{}
	}
	dataJSON, _ := json.Marshal(data)
	positionJSON, _ := json.Marshal(position)

	var row pgx.Row
	if in.Visibility != nil {
		visJSON, _ := json.Marshal(in.Visibility)
		row = pool.QueryRow(ctx, `
			INSERT INTO scene_stage_elements (
				scene_id, show_scene_placement_id, kind, label, data, position, visibility, sort_order, width, height, created_by_user_id
			)
			VALUES ($1, $2, $3, NULLIF($4, ''), $5::jsonb, $6::jsonb, $7::jsonb, $8, $9, $10, $11::uuid)
			RETURNING `+stageElementColumns,
			sceneID, placementID, kind, in.Label, dataJSON, positionJSON, visJSON, in.SortOrder,
			clampNormalized(in.Width), clampNormalized(in.Height), actorUserID)
	} else {
		row = pool.QueryRow(ctx, `
			INSERT INTO scene_stage_elements (
				scene_id, show_scene_placement_id, kind, label, data, position, sort_order, width, height, created_by_user_id
			)
			VALUES ($1, $2, $3, NULLIF($4, ''), $5::jsonb, $6::jsonb, $7, $8, $9, $10::uuid)
			RETURNING `+stageElementColumns,
			sceneID, placementID, kind, in.Label, dataJSON, positionJSON, in.SortOrder,
			clampNormalized(in.Width), clampNormalized(in.Height), actorUserID)
	}
	return scanStageElement(row)
}

// LoadStageElementByID returns the raw row with no authority check.
func LoadStageElementByID(ctx context.Context, pool *pgxpool.Pool, id string) (StageElement, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return StageElement{}, errors.New("stage_element_not_found")
	}
	row := pool.QueryRow(ctx, `SELECT `+stageElementColumns+` FROM scene_stage_elements WHERE id = $1`, id)
	return scanStageElement(row)
}

// UpdateStageElement authority-checks against the owning Scene's location,
// then applies only the fields present in the patch. Never moves an
// element between the Base and Show layers -- that requires delete +
// recreate, keeping layer membership an explicit, deliberate act.
func UpdateStageElement(ctx context.Context, pool *pgxpool.Pool, actorUserID, elementID string, patch UpdateStageElementPatch) (StageElement, error) {
	e, err := LoadStageElementByID(ctx, pool, elementID)
	if err != nil {
		return StageElement{}, err
	}
	scene, err := LoadSceneByID(ctx, pool, e.SceneID)
	if err != nil {
		return StageElement{}, err
	}
	ok, err := canManageComposer(ctx, pool, actorUserID, scene.LocationID)
	if err != nil {
		return StageElement{}, err
	}
	if !ok {
		return StageElement{}, errors.New("not_authorized")
	}

	label := e.Label
	if patch.Label != nil {
		label = strings.TrimSpace(*patch.Label)
	}
	dataJSON := []byte(e.Data)
	if len(dataJSON) == 0 {
		dataJSON = []byte("{}")
	}
	if patch.Data != nil {
		// Merge onto the existing stored data rather than replacing it
		// wholesale -- this function's own contract above is "applies only
		// the fields present in the patch," but every real caller
		// (dispatchConfiguratorAction in runtime.js) sends a genuinely
		// partial fragment: a drag-move's patch is only
		// {venue_slug, layer, snap_mode, token_layer, scale}, a card flip's
		// is only {front_text, back_text, color, face}. A plain replace
		// silently dropped every other stored field on the very next edit --
		// a token lost its own asset_id/asset_content_url/asset_name the
		// first time it was dragged, immediately falling back to the
		// generic placeholder art with its raw element id as a label (this
		// was a real, very confusing bug: it looked like dragging spawned a
		// brand new token with a "random string" name and a broken asset).
		existing := map[string]any{}
		if len(e.Data) > 0 {
			_ = json.Unmarshal(e.Data, &existing)
		}
		for k, v := range *patch.Data {
			existing[k] = v
		}
		if e.Kind == "token" {
			if err := resolveTokenAssetData(ctx, pool, existing); err != nil {
				return StageElement{}, err
			}
		}
		dataJSON, _ = json.Marshal(existing)
	}
	positionJSON := []byte(e.Position)
	if len(positionJSON) == 0 {
		positionJSON = []byte("{}")
	}
	if patch.Position != nil {
		positionJSON, _ = json.Marshal(*patch.Position)
	}
	visibilityJSON := []byte(e.Visibility)
	if len(visibilityJSON) == 0 {
		visibilityJSON = []byte(`{"toRoles":["audience","cast","crew","director","producer"],"privateTo":[]}`)
	}
	if patch.Visibility != nil {
		visibilityJSON, _ = json.Marshal(*patch.Visibility)
	}
	sortOrder := e.SortOrder
	if patch.SortOrder != nil {
		sortOrder = *patch.SortOrder
	}
	// Size follows the same present-means-change rule as every other patch
	// field: a nil Width/Height leaves the stored value alone rather than
	// clearing it, so a drag that only moves an element cannot silently
	// reset its size.
	width := e.Width
	if patch.Width != nil {
		width = clampNormalized(patch.Width)
	}
	height := e.Height
	if patch.Height != nil {
		height = clampNormalized(patch.Height)
	}

	row := pool.QueryRow(ctx, `
		UPDATE scene_stage_elements
		SET label = NULLIF($2, ''), data = $3::jsonb, position = $4::jsonb, visibility = $5::jsonb,
		    sort_order = $6, width = $7, height = $8, updated_at = NOW()
		WHERE id = $1
		RETURNING `+stageElementColumns,
		elementID, label, dataJSON, positionJSON, visibilityJSON, sortOrder, width, height)
	return scanStageElement(row)
}

// clampNormalized keeps a stored size inside the same 0-1 normalized space
// the composer's coordinates use. A hotspot sized outside that range would
// either vanish or swallow the whole stage, and the composer's drag handles
// cannot produce it -- so an out-of-range value only ever arrives from a
// hand-crafted request, and clamping is the quiet correct answer.
func clampNormalized(v *float64) *float64 {
	if v == nil {
		return nil
	}
	out := *v
	if out < 0 {
		out = 0
	}
	if out > 1 {
		out = 1
	}
	return &out
}

// DeleteStageElement removes a composition element outright (no archive
// state -- composition rows are pure authoring state, not an audit log).
// Cascades to any stage_element_bindings row on this element.
func DeleteStageElement(ctx context.Context, pool *pgxpool.Pool, actorUserID, elementID string) error {
	e, err := LoadStageElementByID(ctx, pool, elementID)
	if err != nil {
		return err
	}
	scene, err := LoadSceneByID(ctx, pool, e.SceneID)
	if err != nil {
		return err
	}
	ok, err := canManageComposer(ctx, pool, actorUserID, scene.LocationID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("not_authorized")
	}
	_, err = pool.Exec(ctx, `DELETE FROM scene_stage_elements WHERE id = $1`, elementID)
	return err
}

// ListSceneStageElements returns only the Base-layer elements for a Scene
// (the Scene Library / Scene Setup default composition view).
func ListSceneStageElements(ctx context.Context, pool *pgxpool.Pool, sceneID string) ([]StageElement, error) {
	rows, err := pool.Query(ctx, `
		SELECT `+stageElementColumns+`
		FROM scene_stage_elements
		WHERE scene_id = $1 AND show_scene_placement_id IS NULL
		ORDER BY sort_order ASC, created_at ASC
	`, sceneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectStageElements(rows)
}

func collectStageElements(rows pgx.Rows) ([]StageElement, error) {
	var out []StageElement
	for rows.Next() {
		e, err := scanStageElement(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// LoadResolvedComposition merges a placement's composition: every Base-
// layer element from its Scene, plus every Show-layer element scoped to
// this exact placement, layered on top (Kernel 73A S4's Base/Show model).
// Base and Show layers are simply concatenated (Show-layer elements are
// additions, e.g. Kessa's stall placed only for this Show's staging of the
// Courtyard) rather than field-merged by matching id -- a Director wanting
// to change a Base element's position for one Show only should add a new
// Show-layer element and hide/remove the base one, keeping resolution
// logic simple and predictable rather than a partial-override merge.
// Every returned element's Binding is resolved from stage_element_bindings
// when present.
func LoadResolvedComposition(ctx context.Context, pool *pgxpool.Pool, placementID string) (ResolvedComposition, error) {
	p, err := LoadPlacementByID(ctx, pool, placementID)
	if err != nil {
		return ResolvedComposition{}, err
	}
	rows, err := pool.Query(ctx, `
		SELECT `+stageElementColumns+`
		FROM scene_stage_elements
		WHERE scene_id = $1 AND (show_scene_placement_id IS NULL OR show_scene_placement_id = $2)
		ORDER BY (show_scene_placement_id IS NULL) DESC, sort_order ASC, created_at ASC
	`, p.SceneID, placementID)
	if err != nil {
		return ResolvedComposition{}, err
	}
	defer rows.Close()
	elements, err := collectStageElements(rows)
	if err != nil {
		return ResolvedComposition{}, err
	}

	if len(elements) > 0 {
		ids := make([]string, len(elements))
		byID := make(map[string]*StageElement, len(elements))
		for i := range elements {
			ids[i] = elements[i].ID
			byID[elements[i].ID] = &elements[i]
		}
		bindRows, err := pool.Query(ctx, `
			SELECT id::text, scene_stage_element_id::text, binding_type, participant_interaction_id::text,
			       COALESCE(requires_milestone, ''), created_by_user_id::text, created_at, updated_at
			FROM stage_element_bindings
			WHERE scene_stage_element_id = ANY($1::uuid[])
		`, ids)
		if err != nil {
			return ResolvedComposition{}, err
		}
		defer bindRows.Close()
		for bindRows.Next() {
			var b StageElementBinding
			var createdBy *string
			if err := bindRows.Scan(&b.ID, &b.SceneStageElementID, &b.BindingType, &b.ParticipantInteractionID,
				&b.RequiresMilestone, &createdBy, &b.CreatedAt, &b.UpdatedAt); err != nil {
				return ResolvedComposition{}, err
			}
			b.CreatedByUserID = createdBy
			if el, ok := byID[b.SceneStageElementID]; ok {
				bCopy := b
				el.Binding = &bCopy
			}
		}
		if err := bindRows.Err(); err != nil {
			return ResolvedComposition{}, err
		}
	}

	return ResolvedComposition{SceneID: p.SceneID, PlacementID: placementID, Elements: elements}, nil
}

// --- Element-interaction bindings (Kernel 73A S6) -----------------------

// CreateStageElementBinding binds a composition element to an existing
// participant_interactions row -- clicking/tapping the element in a live
// session or Preview as Player opens that interaction's Program. Authority
// mirrors the composer's own (Producer/Director/Operator manage, or
// Crew's non-destructive-edit right). The FK on participant_interaction_id
// is the source of truth for "does this interaction exist" -- a bad id
// surfaces as a foreign-key violation, translated to a stable error code.
func CreateStageElementBinding(ctx context.Context, pool *pgxpool.Pool, actorUserID, elementID, participantInteractionID, requiresMilestone string) (StageElementBinding, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return StageElementBinding{}, errors.New("not_authenticated")
	}
	participantInteractionID = strings.TrimSpace(participantInteractionID)
	if participantInteractionID == "" {
		return StageElementBinding{}, errors.New("participant_interaction_id_required")
	}
	// Kernel 74: an unrecognized milestone key would silently hide the
	// element from every Player forever, since the gate is fail-closed.
	// Reject it at authoring time instead.
	requiresMilestone = strings.TrimSpace(requiresMilestone)
	if requiresMilestone != "" && !tutorial.IsMilestone(requiresMilestone) {
		return StageElementBinding{}, errors.New("invalid_milestone")
	}
	e, err := LoadStageElementByID(ctx, pool, elementID)
	if err != nil {
		return StageElementBinding{}, err
	}
	scene, err := LoadSceneByID(ctx, pool, e.SceneID)
	if err != nil {
		return StageElementBinding{}, err
	}
	ok, err := canManageComposer(ctx, pool, actorUserID, scene.LocationID)
	if err != nil {
		return StageElementBinding{}, err
	}
	if !ok {
		return StageElementBinding{}, errors.New("not_authorized")
	}

	row := pool.QueryRow(ctx, `
		INSERT INTO stage_element_bindings (scene_stage_element_id, binding_type, participant_interaction_id, requires_milestone, created_by_user_id)
		VALUES ($1, 'participant_interaction', $2, $3, $4::uuid)
		ON CONFLICT (scene_stage_element_id, binding_type)
		DO UPDATE SET participant_interaction_id = EXCLUDED.participant_interaction_id,
		              requires_milestone = EXCLUDED.requires_milestone, updated_at = NOW()
		RETURNING id::text, scene_stage_element_id::text, binding_type, participant_interaction_id::text,
		          COALESCE(requires_milestone, ''), created_by_user_id::text, created_at, updated_at
	`, elementID, participantInteractionID, requiresMilestone, actorUserID)

	var b StageElementBinding
	var createdBy *string
	if err := row.Scan(&b.ID, &b.SceneStageElementID, &b.BindingType, &b.ParticipantInteractionID,
		&b.RequiresMilestone, &createdBy, &b.CreatedAt, &b.UpdatedAt); err != nil {
		if strings.Contains(err.Error(), "stage_element_bindings_participant_interaction_id_fkey") {
			return StageElementBinding{}, errors.New("participant_interaction_not_found")
		}
		return StageElementBinding{}, err
	}
	b.CreatedByUserID = createdBy
	return b, nil
}

// DeleteStageElementBinding removes an element's participant_interaction
// binding, if any.
func DeleteStageElementBinding(ctx context.Context, pool *pgxpool.Pool, actorUserID, elementID string) error {
	e, err := LoadStageElementByID(ctx, pool, elementID)
	if err != nil {
		return err
	}
	scene, err := LoadSceneByID(ctx, pool, e.SceneID)
	if err != nil {
		return err
	}
	ok, err := canManageComposer(ctx, pool, actorUserID, scene.LocationID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("not_authorized")
	}
	_, err = pool.Exec(ctx, `
		DELETE FROM stage_element_bindings WHERE scene_stage_element_id = $1 AND binding_type = 'participant_interaction'
	`, elementID)
	return err
}

// Note: an earlier pass of this file had a ProjectCompositionIntoSession
// function here that appended a one-off 'scene_composition_loaded' action
// into a session's actions log, requiring a frontend client to specially
// interpret that action type and a Director to press a separate "Load
// Composition Into Live Session" button. That was a stopgap. The real
// integration point is backend/internal/world/snapshot.go's
// LoadVenueSnapshot, which now resolves the current placement's Base+Show
// composition fresh on every snapshot read (see its
// loadSceneCompositionAsPlacedElements) -- composition is simply part of
// what a snapshot *is*, the same way the actions-log replay above already
// works, so there is nothing left to "project" or load as a separate step.
// cmd/victory/scene_live_bridge.go's handleShowCurrentSceneWithLiveBridge
// (formerly shows.HandleShowCurrentScene) and cues.HandleCueGo both already
// broadcast network.BroadcastShowStageInvalidation on a successful
// current-Scene change, which is what every connected client already
// refetches its snapshot on.
