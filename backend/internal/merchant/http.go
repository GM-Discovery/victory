package merchant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/network"
)

type response struct {
	Ok   bool `json:"ok"`
	Data any  `json:"data,omitempty"`
}

// requireAuthenticatedUser mirrors cues/http.go's helper exactly -- same
// session-cookie-to-user-id resolution every HTTP-facing package in this
// codebase uses.
func requireAuthenticatedUser(ctx context.Context, pool *pgxpool.Pool, r *http.Request) (string, error) {
	sessionCookie := ""
	if c, err := r.Cookie("victory_session"); err == nil {
		sessionCookie = c.Value
	}
	userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
	if err != nil || strings.TrimSpace(userID) == "" {
		return "", errors.New("not_authenticated")
	}
	return userID, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, response{Ok: true, Data: data})
}

// writeError maps every typed error string this package returns to an HTTP
// status. Spec S12's exact proof list drives this table: anonymous->401,
// audience/non-roster/non-owner->403, forged IDs->404-shaped "not_found".
func writeError(w http.ResponseWriter, err error) {
	code := err.Error()
	status := http.StatusBadRequest
	switch {
	case code == "not_authenticated":
		status = http.StatusUnauthorized
	case code == "not_authorized",
		code == "forbidden",
		code == "not_a_roster_member",
		code == "insufficient_role",
		code == "character_not_owned",
		code == "interaction_disabled",
		code == "scene_not_current",
		// Kernel 74: the reveal gate refusing an invocation. 403 rather than
		// 404 because the interaction genuinely exists -- this caller simply
		// has not earned it yet.
		code == "milestone_required",
		code == "topic_locked",
		code == "required_topics_unseen",
		code == "interaction_type_mismatch":
		status = http.StatusForbidden
	case strings.HasSuffix(code, "_not_found"),
		code == "no_active_session",
		code == "no_character_selected",
		code == "unknown_target":
		status = http.StatusNotFound
	}
	writeJSON(w, status, response{Ok: false, Data: map[string]any{"error": code}})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
}

// --- Director authoring: participant_interactions CRUD ---------------------

// HandlePlacementInteractionsCollection handles GET (backstage list) and
// POST (create) for /api/shows/{show_id}/scenes/{placement_id}/participant-interactions.
func HandlePlacementInteractionsCollection(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		placementID := strings.TrimSpace(r.PathValue("placement_id"))

		switch r.Method {
		case http.MethodGet:
			_, _, locationID, err := placementShowShowRunLocation(ctx, pool, placementID)
			if err != nil {
				writeError(w, err)
				return
			}
			// Reuses the same backstage-visibility gate the Cue authoring
			// list uses (showruns.CanViewBackstage) -- viewing the authoring
			// list is a backstage concern, distinct from ResolveEligibleContext's
			// Player-only trigger gate used by the player-facing listing below.
			canView, err := canManageEquipment(ctx, pool, userID, locationID)
			if err != nil {
				writeError(w, err)
				return
			}
			if !canView {
				writeError(w, errors.New("not_authorized"))
				return
			}
			list, err := ListInteractionsForPlacement(ctx, pool, placementID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"interactions": list})

		case http.MethodPost:
			var body struct {
				InternalName     string         `json:"internal_name"`
				StageButtonLabel string         `json:"stage_button_label"`
				InteractionType  string         `json:"interaction_type"`
				Configuration    map[string]any `json:"configuration_json"`
				SortOrder        int            `json:"sort_order"`
				Enabled          *bool          `json:"enabled"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			it, err := CreateInteraction(ctx, pool, userID, placementID, CreateInteractionInput{
				InternalName: body.InternalName, StageButtonLabel: body.StageButtonLabel,
				InteractionType: body.InteractionType, Configuration: body.Configuration,
				SortOrder: body.SortOrder, Enabled: body.Enabled,
			})
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"interaction": it})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandlePlacementPlayerInteractions handles GET
// /api/shows/{show_id}/scenes/{placement_id}/player-interactions -- the
// curated player-facing stage button listing. Mirrors HandlePlacementPlayerCues:
// no backstage authority required, ListTriggerableInteractionsForViewer's
// own per-interaction ResolveEligibleContext determines what comes back.
func HandlePlacementPlayerInteractions(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		placementID := strings.TrimSpace(r.PathValue("placement_id"))
		list, err := ListTriggerableInteractionsForViewer(ctx, pool, userID, placementID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"interactions": list})
	}
}

// HandleInteractionByID handles PATCH /api/participant-interactions/{interaction_id}.
func HandleInteractionByID(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		interactionID := strings.TrimSpace(r.PathValue("interaction_id"))

		var body struct {
			StageButtonLabel *string         `json:"stage_button_label"`
			Configuration    *map[string]any `json:"configuration_json"`
			Enabled          *bool           `json:"enabled"`
			SortOrder        *int            `json:"sort_order"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		it, err := UpdateInteraction(ctx, pool, userID, interactionID, UpdateInteractionPatch{
			StageButtonLabel: body.StageButtonLabel, Configuration: body.Configuration,
			Enabled: body.Enabled, SortOrder: body.SortOrder,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"interaction": it})
	}
}

// --- Player-side Equip Mode actions -----------------------------------------

// HandleInteractionOpen handles POST /api/participant-interactions/{interaction_id}/open.
// Opening the shop never changes the shared Show Scene -- this is a pure
// read of eligibility + packet + inventory (spec S2.1).
func HandleInteractionOpen(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		interactionID := strings.TrimSpace(r.PathValue("interaction_id"))

		// Kernel 74: /open is the one "show me this Program" verb for all
		// three interaction types, dispatched on the interaction's own stored
		// type rather than on anything the client sends. A client that lies
		// about the type gets the Program its interaction actually is.
		//
		// All three are pure reads that never change the shared Show Scene.
		// Only guided_dialogue records anything (ra_intro_started), because
		// "the Player has begun hearing Ra" is a real milestone; looking at
		// the door or the shop is not.
		it, err := LoadInteractionByID(ctx, pool, interactionID)
		if err != nil {
			writeError(w, err)
			return
		}
		switch it.InteractionType {
		case InteractionTypeFreeformSubmission:
			out, err := OpenFreeform(ctx, pool, userID, interactionID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, out)
		case InteractionTypeGuidedDialogue:
			state, err := OpenDialogue(ctx, pool, userID, interactionID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"dialogue": state})
		default:
			out, err := OpenEquipMode(ctx, pool, userID, interactionID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, out)
		}
	}
}

// HandleInteractionStance handles POST /api/participant-interactions/{interaction_id}/stance.
func HandleInteractionStance(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		interactionID := strings.TrimSpace(r.PathValue("interaction_id"))

		var body struct {
			Stance string `json:"stance"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		result, err := AttemptStance(ctx, pool, userID, interactionID, body.Stance)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"result": result})
	}
}

// HandleInteractionHaggle handles GET (preview, no roll) and POST (attempt,
// actual server-authoritative roll) /api/participant-interactions/{interaction_id}/haggle.
func HandleInteractionHaggle(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		interactionID := strings.TrimSpace(r.PathValue("interaction_id"))

		switch r.Method {
		case http.MethodGet:
			preview, err := PreviewHaggle(ctx, pool, userID, interactionID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"preview": preview})
		case http.MethodPost:
			result, err := AttemptHaggle(ctx, pool, userID, interactionID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"result": result})
		default:
			methodNotAllowed(w)
		}
	}
}

// HandleInteractionPurchase handles POST /api/participant-interactions/{interaction_id}/purchase.
// Body: {"equipment_item_id": "...", "idempotency_key": "..."}. On success,
// pushes a targeted invalidation (network.Hub.BroadcastToSessionUser) so a
// second open tab of the SAME Player -- never any other Player or Audience,
// spec S6.2/S12 -- refetches its inventory view. hub may be nil (e.g. in
// tests); the push is best-effort and never blocks the HTTP response on it.
func HandleInteractionPurchase(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		interactionID := strings.TrimSpace(r.PathValue("interaction_id"))

		var body struct {
			EquipmentItemID string `json:"equipment_item_id"`
			IdempotencyKey  string `json:"idempotency_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		sessionID, entry, err := attemptPurchaseWithSession(ctx, pool, userID, interactionID, body.EquipmentItemID, body.IdempotencyKey)
		if err != nil {
			writeError(w, err)
			return
		}
		if hub != nil && sessionID != "" {
			msg, _ := json.Marshal(map[string]any{
				"type":              "merchant/inventory_updated",
				"character_card_id": entry.CharacterCardID,
			})
			hub.BroadcastToSessionUser(sessionID, userID, msg)
		}
		writeOK(w, map[string]any{"inventory_item": entry})
	}
}

// --- Character Inventory read path -----------------------------------------

// HandleCharacterInventory handles GET /api/characters/{character_card_id}/inventory
// -- the new dedicated Inventory page's read path (spec S5.2).
func HandleCharacterInventory(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		characterCardID := strings.TrimSpace(r.PathValue("character_card_id"))
		list, err := ListInventoryForCharacter(ctx, pool, userID, characterCardID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"inventory": list})
	}
}

// --- Minimal equipment editor ------------------------------------------------

// HandleVenueEquipmentCollection handles GET (list) and POST (create) for
// /api/venues/{venue_slug}/equipment -- the minimal reusable equipment
// editor (spec S5.4).
func HandleVenueEquipmentCollection(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		venueSlug := strings.TrimSpace(r.PathValue("venue_slug"))
		locationID, err := LocationIDForVenueSlug(ctx, pool, venueSlug)
		if err != nil {
			writeError(w, err)
			return
		}

		switch r.Method {
		case http.MethodGet:
			allowed, err := canManageEquipment(ctx, pool, userID, locationID)
			if err != nil {
				writeError(w, err)
				return
			}
			if !allowed {
				writeError(w, errors.New("not_authorized"))
				return
			}
			list, err := ListEquipmentItems(ctx, pool, locationID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"equipment_items": list})

		case http.MethodPost:
			var body struct {
				Name             string         `json:"name"`
				Slug             string         `json:"slug"`
				ImageAssetID     string         `json:"image_asset_id"`
				ShortDescription string         `json:"short_description"`
				Descriptors      []string       `json:"descriptors"`
				QuantityMode     string         `json:"quantity_mode"`
				Active           *bool          `json:"active"`
				Category         string         `json:"category"`
				CostCredits      *float64       `json:"cost_credits"`
				StatsJSON        map[string]any `json:"stats"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_request_body"))
				return
			}
			item, err := CreateEquipmentItem(ctx, pool, userID, locationID, EquipmentItemInput{
				Name: body.Name, Slug: body.Slug, ImageAssetID: body.ImageAssetID,
				ShortDescription: body.ShortDescription, Descriptors: body.Descriptors,
				QuantityMode: body.QuantityMode, Active: body.Active,
				Category: body.Category, CostCredits: body.CostCredits, StatsJSON: body.StatsJSON,
			})
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"equipment_item": item})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandleEquipmentItemByID handles PATCH /api/equipment/{equipment_item_id}.
func HandleEquipmentItemByID(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		itemID := strings.TrimSpace(r.PathValue("equipment_item_id"))

		var body struct {
			Name             *string         `json:"name"`
			ImageAssetID     *string         `json:"image_asset_id"`
			ShortDescription *string         `json:"short_description"`
			Descriptors      *[]string       `json:"descriptors"`
			Active           *bool           `json:"active"`
			Category         *string         `json:"category"`
			CostCredits      *float64        `json:"cost_credits"`
			StatsJSON        *map[string]any `json:"stats"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_request_body"))
			return
		}
		item, err := UpdateEquipmentItem(ctx, pool, userID, itemID, EquipmentItemPatch{
			Name: body.Name, ImageAssetID: body.ImageAssetID, ShortDescription: body.ShortDescription,
			Descriptors: body.Descriptors, Active: body.Active,
			Category: body.Category, CostCredits: body.CostCredits, StatsJSON: body.StatsJSON,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"equipment_item": item})
	}
}

// HandleInteractionPreview handles GET /api/participant-interactions/{interaction_id}/preview
// -- the Kernel 73A "Preview as Player" endpoint (spec item 7). Backstage-
// authority gated (the opposite gate from OpenEquipMode's participant
// eligibility), read-only, and structurally unable to reach any
// purchase/roll/participation code path -- see PreviewInteraction's own
// doc comment for exactly why.
func HandleInteractionPreview(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		interactionID := strings.TrimSpace(r.PathValue("interaction_id"))
		out, err := PreviewInteraction(ctx, pool, userID, interactionID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, out)
	}
}
