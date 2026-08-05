package merchant

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dialogue"
	"victory/backend/internal/projection"
	"victory/backend/internal/scenes"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
	"victory/backend/internal/tutorial"
)

const kessaTokenAssetURL = "/assets/Kessa.png"

// PrepareCourtyardOpeningResult is both the outcome of the idempotent
// "Prepare Locked Courtyard Opening" action and a Kessa reachability
// diagnostics report (Kernel 73A S8, S9) -- every field that would make
// Kessa unreachable from a fresh Show is checked and reported explicitly,
// rather than only reporting success/failure.
type PrepareCourtyardOpeningResult struct {
	LocationID                          string   `json:"location_id"`
	SceneID                             string   `json:"scene_id,omitempty"`
	SceneFound                          bool     `json:"scene_found"`
	PlacementID                         string   `json:"placement_id,omitempty"`
	PlacementCreated                    bool     `json:"placement_created"`
	MerchantPacketFound                 bool     `json:"merchant_packet_found"`
	MerchantPacketStockCount            int      `json:"merchant_packet_stock_count"`
	InteractionID                       string   `json:"interaction_id,omitempty"`
	InteractionCreated                  bool     `json:"interaction_created"`
	InteractionEnabled                  bool     `json:"interaction_enabled"`
	StageElementID                      string   `json:"stage_element_id,omitempty"`
	StageElementCreated                 bool     `json:"stage_element_created"`
	BindingID                           string   `json:"binding_id,omitempty"`
	BindingCreated                      bool     `json:"binding_created"`
	VenueSlug                           string   `json:"venue_slug,omitempty"`
	VenueParticipantInteractionsEnabled bool     `json:"venue_participant_interactions_enabled"`
	VenueSceneComposerEnabled           bool     `json:"venue_scene_composer_enabled"`
	Ready                               bool     `json:"ready"`
	Gaps                                []string `json:"gaps,omitempty"`

	// Kernel 74 additions: the rest of the tutorial tail. Reported with the
	// same explicit found/created granularity as the Kessa fields above, so
	// a Director can see exactly which link is missing rather than only that
	// the path is not Ready.
	DoorInteractionID         string `json:"door_interaction_id,omitempty"`
	DoorInteractionCreated    bool   `json:"door_interaction_created"`
	DoorElementID             string `json:"door_element_id,omitempty"`
	DoorElementCreated        bool   `json:"door_element_created"`
	DoorBindingID             string `json:"door_binding_id,omitempty"`
	DoorBindingCreated        bool   `json:"door_binding_created"`
	RaInteractionID           string `json:"ra_interaction_id,omitempty"`
	RaInteractionCreated      bool   `json:"ra_interaction_created"`
	RaDialoguePacketFound     bool   `json:"ra_dialogue_packet_found"`
	TutorialHandoffSceneFound bool   `json:"tutorial_handoff_scene_found"`
}

// PrepareLockedCourtyardOpening is the idempotent Director action that
// makes the Locked Courtyard / Kessa proof (Kernel 73A S12.4) actually
// reachable end-to-end for one Show, calling only the same building
// blocks a Director could otherwise click through by hand in the Scene
// Setup / Add-Scene-to-Show / participant-interaction-authoring flows:
//  1. Find the reusable "courtyard" Scene at the Show's location (seeded
//     by migration 057/merchant.EnsureCourtyardScene -- never created
//     here; if missing, reported as a gap, not silently invented).
//  2. Ensure a show_scene_placements row exists for (show, courtyard
//     Scene) -- creating one via scenes.CreatePlacement if not.
//  3. Ensure a participant_interactions row of type open_equip_mode
//     exists on that placement, configured for the Kessa merchant packet
//     -- creating one if not.
//  4. Ensure a Base-layer scene_stage_elements token exists for the
//     Scene representing Kessa's stall -- creating one if not.
//  5. Ensure a stage_element_bindings row links that token to the
//     interaction from step 3.
//
// Every step is check-then-create, so calling this twice is a no-op the
// second time (idempotent per Kernel 73A S8). Authority: Director/
// Producer/Operator only (showruns.CanManageShowRun) -- this seeds
// authoring state, not a normal play action.
func PrepareLockedCourtyardOpening(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID string) (PrepareCourtyardOpeningResult, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return PrepareCourtyardOpeningResult{}, errors.New("not_authenticated")
	}
	showID = strings.TrimSpace(showID)
	if showID == "" {
		return PrepareCourtyardOpeningResult{}, errors.New("show_id_required")
	}

	s, err := shows.LoadShowByID(ctx, pool, showID)
	if err != nil {
		return PrepareCourtyardOpeningResult{}, err
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return PrepareCourtyardOpeningResult{}, err
	}
	ok, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return PrepareCourtyardOpeningResult{}, err
	}
	if !ok {
		return PrepareCourtyardOpeningResult{}, errors.New("not_authorized")
	}

	result := PrepareCourtyardOpeningResult{LocationID: sr.LocationID}

	// Step 1: find the Courtyard Scene (never created here).
	var sceneID string
	err = pool.QueryRow(ctx, `
		SELECT id::text FROM scenes WHERE location_id = $1 AND slug = 'courtyard'
	`, sr.LocationID).Scan(&sceneID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return PrepareCourtyardOpeningResult{}, err
	}
	if sceneID == "" {
		result.Gaps = append(result.Gaps, "courtyard_scene_not_found")
		return result, nil
	}
	result.SceneID = sceneID
	result.SceneFound = true

	// Step 2: ensure a placement.
	var placementID string
	err = pool.QueryRow(ctx, `
		SELECT id::text FROM show_scene_placements WHERE show_id = $1 AND scene_id = $2
	`, showID, sceneID).Scan(&placementID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return PrepareCourtyardOpeningResult{}, err
	}
	if placementID == "" {
		p, err := scenes.CreatePlacement(ctx, pool, actorUserID, showID, scenes.CreatePlacementInput{SceneID: sceneID})
		if err != nil {
			return PrepareCourtyardOpeningResult{}, err
		}
		placementID = p.ID
		result.PlacementCreated = true
	}
	result.PlacementID = placementID

	venueSlug, err := resolvePlacementVenueSlug(ctx, pool, placementID)
	if err != nil {
		return PrepareCourtyardOpeningResult{}, err
	}
	result.VenueSlug = venueSlug
	result.VenueParticipantInteractionsEnabled, err = VenueParticipantInteractionsEnabled(ctx, pool, venueSlug)
	if err != nil {
		return PrepareCourtyardOpeningResult{}, err
	}
	if !result.VenueParticipantInteractionsEnabled {
		result.Gaps = append(result.Gaps, "participant_interactions_disabled_on_venue")
	}
	result.VenueSceneComposerEnabled, err = scenes.SceneComposerEnabled(ctx, pool, venueSlug)
	if err != nil {
		return PrepareCourtyardOpeningResult{}, err
	}
	if !result.VenueSceneComposerEnabled {
		result.Gaps = append(result.Gaps, "scene_composer_disabled_on_venue")
	}

	// Kessa's merchant packet -- seeded by migration 061, never created
	// here.
	packet, err := LoadPacketBySlug(ctx, pool, sr.LocationID, "kessa")
	if err != nil {
		result.Gaps = append(result.Gaps, "kessa_merchant_packet_not_found")
	} else {
		result.MerchantPacketFound = true
		result.MerchantPacketStockCount = len(packet.Stock)
		if len(packet.Stock) == 0 {
			result.Gaps = append(result.Gaps, "kessa_merchant_packet_has_no_stock")
		}
	}

	// Step 3: ensure the open_equip_mode participant interaction.
	var interactionID string
	var interactionEnabled bool
	err = pool.QueryRow(ctx, `
		SELECT id::text, enabled FROM participant_interactions
		WHERE show_scene_placement_id = $1 AND interaction_type = 'open_equip_mode'
		ORDER BY created_at ASC LIMIT 1
	`, placementID).Scan(&interactionID, &interactionEnabled)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return PrepareCourtyardOpeningResult{}, err
	}
	if interactionID == "" {
		it, err := CreateInteraction(ctx, pool, actorUserID, placementID, CreateInteractionInput{
			InternalName:     "kessa_stall",
			StageButtonLabel: "Speak with Kessa",
			InteractionType:  InteractionTypeOpenEquipMode,
			Configuration:    map[string]any{"packet_slug": "kessa"},
		})
		if err != nil {
			return PrepareCourtyardOpeningResult{}, err
		}
		interactionID = it.ID
		interactionEnabled = it.Enabled
		result.InteractionCreated = true
	}
	result.InteractionID = interactionID
	result.InteractionEnabled = interactionEnabled
	if !interactionEnabled {
		result.Gaps = append(result.Gaps, "kessa_interaction_disabled")
	}

	// Step 4: ensure a Base-layer token element for Kessa's stall.
	var elementID string
	err = pool.QueryRow(ctx, `
		SELECT id::text FROM scene_stage_elements
		WHERE scene_id = $1 AND show_scene_placement_id IS NULL AND kind = 'token' AND label = 'Kessa'
		LIMIT 1
	`, sceneID).Scan(&elementID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return PrepareCourtyardOpeningResult{}, err
	}
	if elementID == "" {
		if result.VenueSceneComposerEnabled {
			e, err := scenes.CreateSceneStageElement(ctx, pool, actorUserID, sceneID, scenes.CreateStageElementInput{
				Kind:  scenes.StageElementKindToken,
				Label: "Kessa",
				Data: map[string]any{
					"appearance":          "merchant_stall",
					"asset_content_url":   kessaTokenAssetURL,
					"asset_thumbnail_url": kessaTokenAssetURL,
				},
				Position: map[string]any{"x": 0.5, "y": 0.55},
			})
			if err != nil {
				return PrepareCourtyardOpeningResult{}, err
			}
			elementID = e.ID
			result.StageElementCreated = true
		} else {
			result.Gaps = append(result.Gaps, "kessa_stage_element_not_created_composer_disabled")
		}
	}
	result.StageElementID = elementID
	if elementID != "" {
		// Kessa is shipped with the application, not owned by a disposable
		// warehouse upload. Repair older scene rows that predate her packaged
		// artwork so container rebuilds and warehouse cleanup cannot send her
		// back through the generic construction fallback.
		if _, err := pool.Exec(ctx, `
			UPDATE scene_stage_elements
			SET data = COALESCE(data, '{}'::jsonb) || jsonb_build_object(
				'asset_content_url', $2::text,
				'asset_thumbnail_url', $2::text
			), updated_at = NOW()
			WHERE id = $1
		`, elementID, kessaTokenAssetURL); err != nil {
			return PrepareCourtyardOpeningResult{}, err
		}
	}

	// Step 5: ensure the binding.
	if elementID != "" && interactionID != "" {
		var bindingID string
		err = pool.QueryRow(ctx, `
			SELECT id::text FROM stage_element_bindings
			WHERE scene_stage_element_id = $1 AND binding_type = 'participant_interaction'
		`, elementID).Scan(&bindingID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return PrepareCourtyardOpeningResult{}, err
		}
		if bindingID == "" {
			b, err := scenes.CreateStageElementBinding(ctx, pool, actorUserID, elementID, interactionID, "")
			if err != nil {
				return PrepareCourtyardOpeningResult{}, err
			}
			bindingID = b.ID
			result.BindingCreated = true
		}
		result.BindingID = bindingID
	}

	// Kernel 74 steps 6-8: the rest of the tutorial tail, prepared by the
	// same check-then-create discipline so the whole Kessa -> door -> Ra ->
	// handoff path is reachable from one idempotent Director action rather
	// than three hand-authored records.
	if err := prepareKernel74TutorialTail(ctx, pool, actorUserID, sr.LocationID, sceneID, placementID, &result); err != nil {
		return PrepareCourtyardOpeningResult{}, err
	}

	result.Ready = result.SceneFound && result.PlacementID != "" && result.MerchantPacketFound &&
		result.InteractionID != "" && result.InteractionEnabled && result.StageElementID != "" &&
		result.BindingID != "" && result.VenueParticipantInteractionsEnabled && result.VenueSceneComposerEnabled &&
		result.DoorInteractionID != "" && result.DoorBindingID != "" && result.RaInteractionID != ""
	return result, nil
}

// prepareKernel74TutorialTail ensures the door hotspot, its freeform
// interaction, the milestone-gated binding, and Ra's guided-dialogue
// interaction.
//
// The door hotspot is created over the EXISTING door in the Courtyard map
// art (S1.1) -- it renders no art of its own, which is why it needs an
// explicit width/height while Kessa's token does not. The default box is a
// starting position a Director drags and resizes in Scene Setup to match
// whatever Courtyard art is current; nothing here assumes final artwork.
func prepareKernel74TutorialTail(ctx context.Context, pool *pgxpool.Pool, actorUserID, locationID, sceneID, placementID string, result *PrepareCourtyardOpeningResult) error {
	// Ra's authored packet -- seeded by migration 066, never created here,
	// for the same reason Kessa's merchant packet is not created here.
	if _, err := dialogue.LoadPacketBySlug(ctx, pool, locationID, "ra"); err != nil {
		result.Gaps = append(result.Gaps, "ra_dialogue_packet_not_found")
	} else {
		result.RaDialoguePacketFound = true
	}
	if _, err := projection.ResolveDestinationScene(ctx, pool, locationID, "tutorial-handoff"); err != nil {
		result.Gaps = append(result.Gaps, "tutorial_handoff_scene_not_found")
	} else {
		result.TutorialHandoffSceneFound = true
	}

	// Step 6: Ra's guided_dialogue interaction. Created before the door so
	// the door's next_interaction_slug always resolves to a real row.
	raID, created, err := ensureInteraction(ctx, pool, actorUserID, placementID, InteractionTypeGuidedDialogue,
		"ra_gate_dialogue", "Speak with Ra", map[string]any{"packet_slug": "ra"})
	if err != nil {
		return err
	}
	result.RaInteractionID = raID
	result.RaInteractionCreated = created

	// Step 7: the door's freeform_submission interaction.
	doorID, created, err := ensureInteraction(ctx, pool, actorUserID, placementID, InteractionTypeFreeformSubmission,
		"locked_courtyard_door", "Try the Door", map[string]any{
			"title": "The Locked Door",
			// S1.3's authored description, deliberately establishing that no
			// ordinary lock is visible from this side -- the state Ra's
			// closing narration later reveals a concealed mechanism behind.
			"description":  "The oak door is heavy, reinforced with black iron bands. There is no obvious lock or keyhole on this side.",
			"prompt":       "What does your Character try?",
			"submit_label": "Make the Attempt",
			"max_length":   defaultFreeformMaxLength,
			// Chained by internal_name, resolved server-side -- the Player
			// never names what comes next.
			"next_interaction_slug": "ra_gate_dialogue",
			"note_subject":          "Unresolved Door Intention",
			"note_suffix":           "Interrupted by Ra before completion.",
		})
	if err != nil {
		return err
	}
	result.DoorInteractionID = doorID
	result.DoorInteractionCreated = created

	// Step 8: the hotspot element and its milestone-gated binding.
	var doorElementID string
	err = pool.QueryRow(ctx, `
		SELECT id::text FROM scene_stage_elements
		WHERE scene_id = $1 AND show_scene_placement_id IS NULL
		  AND kind = 'interaction_hotspot' AND label = 'Locked Courtyard Door'
		LIMIT 1
	`, sceneID).Scan(&doorElementID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if doorElementID == "" {
		if !result.VenueSceneComposerEnabled {
			result.Gaps = append(result.Gaps, "door_hotspot_not_created_composer_disabled")
			return nil
		}
		// Measured off the current Courtyard art (frontend/assets/
		// courtyard.png, 1448x1086): the ironbound oak double door in the
		// top gatehouse sits at roughly x 728/1448, y 135/1086. The courtyard
		// has a second gate at the bottom; the top one is the exit the
		// tutorial means.
		//
		// A starting position, not a constant. The Director drags and resizes
		// this in Scene Setup, which is the whole reason the hotspot is a
		// placeable element rather than hardcoded geometry (S1.1) -- and the
		// only way to get it right against art this code cannot see.
		w, h := 0.06, 0.075
		e, err := scenes.CreateSceneStageElement(ctx, pool, actorUserID, sceneID, scenes.CreateStageElementInput{
			Kind:     scenes.StageElementKindInteractionHotspot,
			Label:    "Locked Courtyard Door",
			Data:     map[string]any{"nameplate_visible": true, "highlight_on_focus": true},
			Position: map[string]any{"x": 0.503, "y": 0.124},
			Width:    &w,
			Height:   &h,
		})
		if err != nil {
			return err
		}
		doorElementID = e.ID
		result.DoorElementCreated = true
	}
	result.DoorElementID = doorElementID

	if doorElementID != "" && doorID != "" {
		var bindingID string
		var storedMilestone string
		err = pool.QueryRow(ctx, `
			SELECT id::text, COALESCE(requires_milestone, '') FROM stage_element_bindings
			WHERE scene_stage_element_id = $1 AND binding_type = 'participant_interaction'
		`, doorElementID).Scan(&bindingID, &storedMilestone)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		// Re-create when the gate is missing as well as when the binding is:
		// a binding without requires_milestone would show the door to every
		// Player from the start, which is precisely the bug this gate exists
		// to prevent, and a pre-Kernel-74 row would look "already prepared".
		if bindingID == "" || storedMilestone != tutorial.MilestoneKessaIntroCompleted {
			b, err := scenes.CreateStageElementBinding(ctx, pool, actorUserID, doorElementID, doorID,
				tutorial.MilestoneKessaIntroCompleted)
			if err != nil {
				return err
			}
			bindingID = b.ID
			result.DoorBindingCreated = true
		}
		result.DoorBindingID = bindingID
	}
	return nil
}

// ensureInteraction is the check-then-create the Kernel 73A steps do inline,
// factored out once the third and fourth callers appeared.
func ensureInteraction(ctx context.Context, pool *pgxpool.Pool, actorUserID, placementID, interactionType, internalName, buttonLabel string, config map[string]any) (string, bool, error) {
	var id string
	err := pool.QueryRow(ctx, `
		SELECT id::text FROM participant_interactions
		WHERE show_scene_placement_id = $1 AND interaction_type = $2 AND internal_name = $3
		ORDER BY created_at ASC LIMIT 1
	`, placementID, interactionType, internalName).Scan(&id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", false, err
	}
	if id != "" {
		return id, false, nil
	}
	it, err := CreateInteraction(ctx, pool, actorUserID, placementID, CreateInteractionInput{
		InternalName:     internalName,
		StageButtonLabel: buttonLabel,
		InteractionType:  interactionType,
		Configuration:    config,
	})
	if err != nil {
		return "", false, err
	}
	return it.ID, true, nil
}

// KessaReachabilityDiagnostics is a read-only variant of the same checks
// PrepareLockedCourtyardOpening performs, for a Director to inspect
// without mutating anything (Kernel 73A S9).
func KessaReachabilityDiagnostics(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID string) (PrepareCourtyardOpeningResult, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return PrepareCourtyardOpeningResult{}, errors.New("not_authenticated")
	}
	s, err := shows.LoadShowByID(ctx, pool, showID)
	if err != nil {
		return PrepareCourtyardOpeningResult{}, err
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return PrepareCourtyardOpeningResult{}, err
	}
	canView, err := showruns.CanViewBackstage(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return PrepareCourtyardOpeningResult{}, err
	}
	if !canView {
		return PrepareCourtyardOpeningResult{}, errors.New("not_authorized")
	}

	result := PrepareCourtyardOpeningResult{LocationID: sr.LocationID}
	var sceneID string
	_ = pool.QueryRow(ctx, `SELECT id::text FROM scenes WHERE location_id = $1 AND slug = 'courtyard'`, sr.LocationID).Scan(&sceneID)
	if sceneID == "" {
		result.Gaps = append(result.Gaps, "courtyard_scene_not_found")
		return result, nil
	}
	result.SceneID = sceneID
	result.SceneFound = true

	var placementID string
	_ = pool.QueryRow(ctx, `SELECT id::text FROM show_scene_placements WHERE show_id = $1 AND scene_id = $2`, showID, sceneID).Scan(&placementID)
	if placementID == "" {
		result.Gaps = append(result.Gaps, "placement_not_found")
		return result, nil
	}
	result.PlacementID = placementID

	venueSlug, err := resolvePlacementVenueSlug(ctx, pool, placementID)
	if err == nil {
		result.VenueSlug = venueSlug
		result.VenueParticipantInteractionsEnabled, _ = VenueParticipantInteractionsEnabled(ctx, pool, venueSlug)
		result.VenueSceneComposerEnabled, _ = scenes.SceneComposerEnabled(ctx, pool, venueSlug)
	}

	packet, err := LoadPacketBySlug(ctx, pool, sr.LocationID, "kessa")
	if err == nil {
		result.MerchantPacketFound = true
		result.MerchantPacketStockCount = len(packet.Stock)
	} else {
		result.Gaps = append(result.Gaps, "kessa_merchant_packet_not_found")
	}

	var interactionID string
	var interactionEnabled bool
	_ = pool.QueryRow(ctx, `
		SELECT id::text, enabled FROM participant_interactions
		WHERE show_scene_placement_id = $1 AND interaction_type = 'open_equip_mode'
		ORDER BY created_at ASC LIMIT 1
	`, placementID).Scan(&interactionID, &interactionEnabled)
	result.InteractionID = interactionID
	result.InteractionEnabled = interactionEnabled
	if interactionID == "" {
		result.Gaps = append(result.Gaps, "kessa_interaction_not_found")
	} else if !interactionEnabled {
		result.Gaps = append(result.Gaps, "kessa_interaction_disabled")
	}

	var elementID string
	_ = pool.QueryRow(ctx, `
		SELECT id::text FROM scene_stage_elements
		WHERE scene_id = $1 AND show_scene_placement_id IS NULL AND kind = 'token' AND label = 'Kessa' LIMIT 1
	`, sceneID).Scan(&elementID)
	result.StageElementID = elementID
	if elementID == "" {
		result.Gaps = append(result.Gaps, "kessa_stage_element_not_found")
	} else if interactionID != "" {
		var bindingID string
		_ = pool.QueryRow(ctx, `
			SELECT id::text FROM stage_element_bindings WHERE scene_stage_element_id = $1 AND binding_type = 'participant_interaction'
		`, elementID).Scan(&bindingID)
		result.BindingID = bindingID
		if bindingID == "" {
			result.Gaps = append(result.Gaps, "kessa_binding_not_found")
		}
	}

	result.Ready = len(result.Gaps) == 0
	return result, nil
}
