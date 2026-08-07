package storyboards

// HTTP handlers for /api/storyboards/{board_id}/reference-fields/* (Kernel
// 82). Same envelope/auth conventions as http.go; every Reference Panel
// change broadcasts EventReferencePanelChanged, and the frontend's
// existing "any storyboard/* event -> reload the snapshot" behavior
// (unchanged from Kernel 80/81) means no incremental-patch client code is
// needed for any of these.

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/network"
)

// HandleReferenceFields serves POST (create) at
// /api/storyboards/{board_id}/reference-fields.
func HandleReferenceFields(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}
		var body struct {
			Label     string `json:"label"`
			FieldType string `json:"field_type"`
			SublabelA string `json:"sublabel_a"`
			SublabelB string `json:"sublabel_b"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		field, err := AddReferenceField(ctx, pool, userID, boardID, body.Label, body.FieldType, body.SublabelA, body.SublabelB)
		if err != nil {
			writeError(w, err)
			return
		}
		broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventReferencePanelChanged, map[string]any{"field": field})
		writeOK(w, map[string]any{"field": field})
	}
}

// HandleReferenceFieldItem serves PATCH (rename) and DELETE (remove,
// `?confirm=true` required for a nonempty field) at
// /api/storyboards/{board_id}/reference-fields/{field_id}.
func HandleReferenceFieldItem(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		fieldID := strings.TrimSpace(r.PathValue("field_id"))
		switch r.Method {
		case http.MethodPatch:
			var body struct {
				Label     string `json:"label"`
				SublabelA string `json:"sublabel_a"`
				SublabelB string `json:"sublabel_b"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_json"))
				return
			}
			field, err := RenameReferenceField(ctx, pool, userID, boardID, fieldID, body.Label, body.SublabelA, body.SublabelB)
			if err != nil {
				writeError(w, err)
				return
			}
			broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventReferencePanelChanged, map[string]any{"field": field})
			writeOK(w, map[string]any{"field": field})
		case http.MethodDelete:
			confirmed := r.URL.Query().Get("confirm") == "true"
			if err := RemoveReferenceField(ctx, pool, userID, boardID, fieldID, confirmed); err != nil {
				writeError(w, err)
				return
			}
			broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventReferencePanelChanged, map[string]any{"field_id": fieldID, "removed": true})
			writeOK(w, map[string]any{"deleted": true})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
		}
	}
}

// HandleReferenceFieldType serves POST
// /api/storyboards/{board_id}/reference-fields/{field_id}/type with body
// {"field_type": "..."}.
func HandleReferenceFieldType(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		fieldID := strings.TrimSpace(r.PathValue("field_id"))
		var body struct {
			FieldType string `json:"field_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		field, err := ChangeReferenceFieldType(ctx, pool, userID, boardID, fieldID, body.FieldType)
		if err != nil {
			writeError(w, err)
			return
		}
		broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventReferencePanelChanged, map[string]any{"field": field})
		writeOK(w, map[string]any{"field": field})
	}
}

// HandleReferenceFieldContent serves POST
// /api/storyboards/{board_id}/reference-fields/{field_id}/content with
// body {"text": "..."} -- Crew+ (spec 3.4).
func HandleReferenceFieldContent(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		fieldID := strings.TrimSpace(r.PathValue("field_id"))
		var body struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		field, err := SetReferenceFieldTextContent(ctx, pool, userID, boardID, fieldID, body.Text)
		if err != nil {
			writeError(w, err)
			return
		}
		broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventReferencePanelChanged, map[string]any{"field": field})
		writeOK(w, map[string]any{"field": field})
	}
}

// HandleReferenceFieldsReorder serves POST
// /api/storyboards/{board_id}/reference-fields/reorder with body
// {"ordered_field_ids": [...]}.
func HandleReferenceFieldsReorder(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		var body struct {
			OrderedFieldIDs []string `json:"ordered_field_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		if err := ReorderReferenceFields(ctx, pool, userID, boardID, body.OrderedFieldIDs); err != nil {
			writeError(w, err)
			return
		}
		broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventReferencePanelChanged, map[string]any{"reordered": true})
		writeOK(w, map[string]any{"reordered": true})
	}
}

// HandleReferenceItems serves POST (create) at
// /api/storyboards/{board_id}/reference-fields/{field_id}/items.
func HandleReferenceItems(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		fieldID := strings.TrimSpace(r.PathValue("field_id"))
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}
		var body struct {
			Side    string `json:"side"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		item, err := AddReferenceItem(ctx, pool, userID, boardID, fieldID, body.Side, body.Content)
		if err != nil {
			writeError(w, err)
			return
		}
		broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventReferencePanelChanged, map[string]any{"item": item})
		writeOK(w, map[string]any{"item": item})
	}
}

// HandleReferenceItemItem serves PATCH (content) and DELETE (remove) at
// /api/storyboards/{board_id}/reference-fields/{field_id}/items/{item_id}.
func HandleReferenceItemItem(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		fieldID := strings.TrimSpace(r.PathValue("field_id"))
		itemID := strings.TrimSpace(r.PathValue("item_id"))
		switch r.Method {
		case http.MethodPatch:
			var body struct {
				Content string `json:"content"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_json"))
				return
			}
			item, err := UpdateReferenceItemContent(ctx, pool, userID, boardID, fieldID, itemID, body.Content)
			if err != nil {
				writeError(w, err)
				return
			}
			broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventReferencePanelChanged, map[string]any{"item": item})
			writeOK(w, map[string]any{"item": item})
		case http.MethodDelete:
			if err := RemoveReferenceItem(ctx, pool, userID, boardID, fieldID, itemID); err != nil {
				writeError(w, err)
				return
			}
			broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventReferencePanelChanged, map[string]any{"item_id": itemID, "removed": true})
			writeOK(w, map[string]any{"deleted": true})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
		}
	}
}

// HandleReferenceItemsReorder serves POST
// /api/storyboards/{board_id}/reference-fields/{field_id}/items/reorder
// with body {"side": "...", "ordered_item_ids": [...]}.
func HandleReferenceItemsReorder(pool *pgxpool.Pool, hub *network.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context10(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		boardID := strings.TrimSpace(r.PathValue("board_id"))
		fieldID := strings.TrimSpace(r.PathValue("field_id"))
		var body struct {
			Side           string   `json:"side"`
			OrderedItemIDs []string `json:"ordered_item_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		if err := ReorderReferenceItems(ctx, pool, userID, boardID, fieldID, body.Side, body.OrderedItemIDs); err != nil {
			writeError(w, err)
			return
		}
		broadcastGenericBoardEvent(ctx, pool, hub, boardID, EventReferencePanelChanged, map[string]any{"reordered": true})
		writeOK(w, map[string]any{"reordered": true})
	}
}
