package network

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/actions"
	"victory/backend/internal/identity"
	"victory/backend/internal/world"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return false
		}

		expectedHTTP := "http://" + r.Host
		expectedHTTPS := "https://" + r.Host

		return origin == expectedHTTP || origin == expectedHTTPS
	},
}

var storeReactionFunc = actions.StoreReaction
var storeChatMessageFunc = actions.StoreChatMessage
var storeSpeakFunc = actions.StoreSpeak
var storeRevealFunc = actions.StoreReveal
var storeOverlayShowFunc = actions.StoreOverlayShow
var storeOverlayHideFunc = actions.StoreOverlayHide
var storeIndexCardCreateFunc = actions.StoreIndexCardCreate
var storeIndexCardUpdateFunc = actions.StoreIndexCardUpdate
var storeIndexCardDeleteFunc = actions.StoreIndexCardDelete
var storeRemoveElementFunc = actions.StoreRemoveElement
var storePlaceElementFunc = actions.StorePlaceElement
var storeDuplicateElementFunc = actions.StoreDuplicateElement
var storeSetElementLockFunc = actions.StoreSetElementLock
var storeSetNameplateVisibilityFunc = actions.StoreSetNameplateVisibility
var storePersonaEquipFunc = actions.StorePersonaEquip
var storePersonaUnequipFunc = actions.StorePersonaUnequip
var discordBridgeConfig identity.DiscordServerLinkConfig

func ServeCaveWS(hub *Hub, pool *pgxpool.Pool, discordLinkCfg identity.DiscordServerLinkConfig) http.HandlerFunc {
	discordBridgeConfig = discordLinkCfg
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("ws upgrade failed: %v", err)
			return
		}

		sessionCookie := ""
		if c, err := r.Cookie("victory_session"); err == nil {
			sessionCookie = c.Value
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
		if err != nil {
			log.Printf("ws current user failed: %v", err)
			_ = conn.WriteJSON(map[string]any{
				"type":  "error",
				"error": "not_authenticated",
			})
			_ = conn.Close()
			return
		}

		allowed, err := access.UserCanAccessVenueSlug(ctx, pool, userID, "the-cave")
		if err != nil {
			log.Printf("ws venue access check failed: %v", err)
			_ = conn.WriteJSON(map[string]any{
				"type":  "error",
				"error": "access_check_failed",
			})
			_ = conn.Close()
			return
		}

		if !allowed {
			_ = conn.WriteJSON(map[string]any{
				"type":  "error",
				"error": "forbidden",
			})
			_ = conn.Close()
			return
		}

		sessionID, err := identity.ResolveActiveCaveSessionID(ctx, pool, userID)
		if err != nil {
			log.Printf("ws active session lookup failed: %v", err)
			_ = conn.WriteJSON(map[string]any{
				"type":  "error",
				"error": "not_session_participant",
			})
			_ = conn.Close()
			return
		}

		sessionIdentity, err := identity.ResolveSessionIdentity(ctx, pool, sessionID, userID)
		if err != nil {
			log.Printf("ws identity lookup failed: %v", err)
			_ = conn.WriteJSON(map[string]any{
				"type":  "error",
				"error": "not_session_participant",
			})
			_ = conn.Close()
			return
		}

		client := &Client{
			Conn:      conn,
			Send:      make(chan []byte, 16),
			UserID:    sessionIdentity.UserID,
			SessionID: sessionIdentity.SessionID,
			Presence: PresenceUser{
				UserID:      sessionIdentity.UserID,
				Handle:      sessionIdentity.Handle,
				DisplayName: sessionIdentity.DisplayName,
				Role:        sessionIdentity.Role,
				Persona:     sessionIdentity.Persona,
				ConnectedAt: time.Now().UTC().Format(time.RFC3339),
			},
		}
		hub.Add(client)

		presenceSnapshot, joined := hub.Presence().Connect(sessionIdentity.SessionID, client.Presence)

		snapshot, err := world.LoadCaveSnapshot(ctx, pool, sessionIdentity.Role)
		if err != nil {
			log.Printf("ws snapshot failed: %v", err)
			hub.Presence().Disconnect(sessionIdentity.SessionID, client.UserID)
			_ = conn.WriteJSON(map[string]any{
				"type":  "error",
				"error": "failed_to_load_world",
			})
			_ = conn.Close()
			hub.Remove(client)
			return
		}

		if err := conn.WriteJSON(map[string]any{
			"type": "snapshot",
			"data": snapshot,
		}); err != nil {
			log.Printf("ws initial write failed: %v", err)
			hub.Presence().Disconnect(sessionIdentity.SessionID, client.UserID)
			_ = conn.Close()
			hub.Remove(client)
			return
		}

		if err := conn.WriteJSON(map[string]any{
			"type":  "presence/snapshot",
			"users": presenceSnapshot,
		}); err != nil {
			log.Printf("ws presence snapshot failed: %v", err)
			hub.Presence().Disconnect(sessionIdentity.SessionID, client.UserID)
			_ = conn.Close()
			hub.Remove(client)
			return
		}

		go writePump(client)

		if joined {
			broadcastPresenceEvent(hub, sessionIdentity.SessionID, "presence/join", client.Presence)
		}

		readPump(hub, pool, client)
	}
}

func broadcastPresenceEvent(hub *Hub, sessionID, event string, user PresenceUser) {
	_ = sessionID
	hub.Broadcast(mustJSON(map[string]any{
		"type": event,
		"user": user,
	}))
}

func mustJSON(payload any) []byte {
	msg, err := json.Marshal(payload)
	if err != nil {
		log.Printf("presence JSON marshal failed: %v", err)
		return []byte(`{"type":"error","error":"presence_serialization_failed"}`)
	}

	return msg
}

func writePump(c *Client) {
	defer c.Conn.Close()

	for msg := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func readPump(hub *Hub, pool *pgxpool.Pool, c *Client) {
	defer func() {
		hub.Remove(c)

		if strings.TrimSpace(c.SessionID) != "" && strings.TrimSpace(c.UserID) != "" {
			_, left := hub.Presence().Disconnect(c.SessionID, c.UserID)
			if left {
				broadcastPresenceEvent(hub, c.SessionID, "presence/leave", c.Presence)
			}
			_ = c.Conn.Close()
			return
		}

		_ = c.Conn.Close()
	}()

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}

		var payload map[string]any
		if err := json.Unmarshal(msg, &payload); err != nil {
			continue
		}

		handleCavePayload(hub, pool, c, payload)
	}
}

func handleCavePayload(hub *Hub, pool *pgxpool.Pool, c *Client, payload map[string]any) {
	switch payload["type"] {
	case "ping":
		_ = c.Conn.WriteJSON(map[string]any{
			"type": "pong",
			"ts":   time.Now().UTC().Format(time.RFC3339),
		})

	case "venue/focus_ping":
		{
			sessionID := c.SessionID
			if strings.TrimSpace(sessionID) == "" {
				_ = c.Conn.WriteJSON(map[string]any{
					"type":  "error",
					"error": "not_session_participant",
				})
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			allowed, err := canAccessDirectorConsole(ctx, pool, c.UserID)
			cancel()
			if err != nil {
				_ = c.Conn.WriteJSON(map[string]any{
					"type":  "error",
					"error": "access_check_failed",
				})
				return
			}
			if !allowed {
				_ = c.Conn.WriteJSON(map[string]any{
					"type":  "error",
					"error": "forbidden",
				})
				return
			}

			venueSlug := "the-cave"

			focusX := 0.0
			if raw, ok := payload["focus_x"].(float64); ok {
				focusX = raw
			}
			focusY := 0.0
			if raw, ok := payload["focus_y"].(float64); ok {
				focusY = raw
			}
			cameraCenterX := 0.0
			if raw, ok := payload["camera_center_x"].(float64); ok {
				cameraCenterX = raw
			}
			cameraCenterY := 0.0
			if raw, ok := payload["camera_center_y"].(float64); ok {
				cameraCenterY = raw
			}
			cameraPanX := 0.0
			if raw, ok := payload["camera_pan_x"].(float64); ok {
				cameraPanX = raw
			}
			cameraPanY := 0.0
			if raw, ok := payload["camera_pan_y"].(float64); ok {
				cameraPanY = raw
			}
			cameraZoom := 1.0
			if raw, ok := payload["camera_zoom_relative_to_fit"].(float64); ok {
				cameraZoom = raw
			}
			eventID, _ := payload["event_id"].(string)
			if strings.TrimSpace(eventID) == "" {
				eventID = time.Now().UTC().Format(time.RFC3339Nano)
			}
			ts := time.Now().UTC().Format(time.RFC3339)

			msgOut, _ := json.Marshal(map[string]any{
				"type": "venue/focus_ping",
				"data": map[string]any{
					"venue_slug":                  venueSlug,
					"session_id":                  sessionID,
					"sender_user_id":              c.UserID,
					"sender_display_name":         c.Presence.DisplayName,
					"sender_handle":               c.Presence.Handle,
					"sender_role":                 c.Presence.Role,
					"focus_x":                     focusX,
					"focus_y":                     focusY,
					"camera_center_x":             cameraCenterX,
					"camera_center_y":             cameraCenterY,
					"camera_pan_x":                cameraPanX,
					"camera_pan_y":                cameraPanY,
					"camera_zoom_relative_to_fit": cameraZoom,
					"event_id":                    eventID,
					"ts":                          ts,
				},
			})
			hub.BroadcastSession(sessionID, msgOut)
		}

	case "react/emote":
		{
			sessionID, _ := payload["session_id"].(string)
			actorID := c.UserID
			kind, _ := payload["kind"].(string)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			storedAction, err := storeReactionFunc(ctx, pool, actions.ReactRequest{
				SessionID: sessionID,
				ActorID:   actorID,
				Kind:      kind,
			})
			cancel()

			if err != nil {
				_ = c.Conn.WriteJSON(map[string]any{
					"type":  "error",
					"error": err.Error(),
				})
				return
			}

			msgOut, _ := json.Marshal(map[string]any{
				"type": "action",
				"data": storedAction,
			})
			hub.Broadcast(msgOut)
		}

	case "chat/message":
		{
			sessionID, _ := payload["session_id"].(string)
			actorID := c.UserID
			text, _ := payload["text"].(string)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			storedAction, err := storeChatMessageFunc(ctx, pool, actions.ChatMessageRequest{
				SessionID: sessionID,
				ActorID:   actorID,
				Text:      text,
			})
			cancel()

			if err != nil {
				var denied *actions.ActionDeniedError
				if errors.As(err, &denied) {
					_ = c.Conn.WriteJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("chat denied: user=%s session=%s action=%s reason=%s", actorID, sessionID, payload["type"], denied.Reason)
					return
				}

				_ = c.Conn.WriteJSON(map[string]any{
					"type":  "error",
					"error": err.Error(),
				})
				log.Printf("chat store failed: user=%s session=%s action=%s err=%v", actorID, sessionID, payload["type"], err)
				return
			}

			msgOut, _ := json.Marshal(map[string]any{
				"type": "action",
				"data": storedAction,
			})
			hub.Broadcast(msgOut)
			go mirrorVictoryChatToDiscord(context.Background(), pool, discordBridgeConfig, storedAction)
		}

	case "perform/speak":
		{
			sessionID, _ := payload["session_id"].(string)
			actorID := c.UserID
			text, _ := payload["text"].(string)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			storedAction, err := storeSpeakFunc(ctx, pool, sessionID, actorID, text)
			cancel()

			if err != nil {
				_ = c.Conn.WriteJSON(map[string]any{
					"type":  "error",
					"error": err.Error(),
				})
				return
			}

			msgOut, _ := json.Marshal(map[string]any{
				"type": "action",
				"data": storedAction,
			})
			hub.Broadcast(msgOut)
		}

	case "act/reveal_element", "act/hide_element":
		{
			sessionID, _ := payload["session_id"].(string)
			actorID := c.UserID
			elementID, _ := payload["element_id"].(string)
			elementSlug, _ := payload["element_slug"].(string)
			layer, _ := payload["layer"].(string)

			visible := payload["type"] == "act/reveal_element"

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			storedAction, err := storeRevealFunc(ctx, pool, actions.RevealRequest{
				SessionID:   sessionID,
				ActorID:     actorID,
				ElementID:   elementID,
				ElementSlug: elementSlug,
				Layer:       layer,
				Visible:     visible,
			})
			cancel()

			if err != nil {
				var denied *actions.ActionDeniedError
				if errors.As(err, &denied) {
					_ = c.Conn.WriteJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("action denied: user=%s session=%s action=%s target=%s reason=%s", actorID, sessionID, payload["type"], elementSlug, denied.Reason)
					return
				}

				_ = c.Conn.WriteJSON(map[string]any{
					"type":  "error",
					"error": err.Error(),
				})
				log.Printf("action store failed: user=%s session=%s action=%s target=%s err=%v", actorID, sessionID, payload["type"], elementSlug, err)
				return
			}

			msgOut, _ := json.Marshal(map[string]any{
				"type": "action",
				"data": storedAction,
			})
			hub.Broadcast(msgOut)
		}

	case "act/show_overlay", "act/hide_overlay":
		{
			sessionID, _ := payload["session_id"].(string)
			actorID := c.UserID
			elementID, _ := payload["element_id"].(string)
			elementSlug, _ := payload["element_slug"].(string)
			overlayType, _ := payload["overlay_type"].(string)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			var storedAction *actions.StoredAction
			var err error
			if payload["type"] == "act/show_overlay" {
				storedAction, err = storeOverlayShowFunc(ctx, pool, actions.OverlayRequest{
					SessionID:   sessionID,
					ActorID:     actorID,
					ElementID:   elementID,
					ElementSlug: elementSlug,
					OverlayType: overlayType,
				})
			} else {
				storedAction, err = storeOverlayHideFunc(ctx, pool, actions.OverlayRequest{
					SessionID:   sessionID,
					ActorID:     actorID,
					ElementID:   elementID,
					ElementSlug: elementSlug,
					OverlayType: overlayType,
				})
			}
			cancel()

			if err != nil {
				var denied *actions.ActionDeniedError
				if errors.As(err, &denied) {
					_ = c.Conn.WriteJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("overlay denied: user=%s session=%s action=%s target=%s reason=%s", actorID, sessionID, payload["type"], elementSlug, denied.Reason)
					return
				}

				_ = c.Conn.WriteJSON(map[string]any{
					"type":  "error",
					"error": err.Error(),
				})
				log.Printf("overlay store failed: user=%s session=%s action=%s target=%s err=%v", actorID, sessionID, payload["type"], elementSlug, err)
				return
			}

			msgOut, _ := json.Marshal(map[string]any{
				"type": "action",
				"data": storedAction,
			})
			hub.Broadcast(msgOut)
		}

	case "act/place_element":
		{
			sessionID, _ := payload["session_id"].(string)
			actorID := c.UserID
			elementID, _ := payload["element_id"].(string)
			elementSlug, _ := payload["element_slug"].(string)
			venueSlug, _ := payload["venue_slug"].(string)
			layer, _ := payload["layer"].(string)

			x := 0.0
			if raw, ok := payload["x"].(float64); ok {
				x = raw
			}
			y := 0.0
			if raw, ok := payload["y"].(float64); ok {
				y = raw
			}
			order := 0
			if raw, ok := payload["order"].(float64); ok {
				order = int(raw)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			storedAction, err := storePlaceElementFunc(ctx, pool, actions.PlaceElementRequest{
				SessionID:   sessionID,
				ActorID:     actorID,
				ElementID:   elementID,
				ElementSlug: elementSlug,
				VenueSlug:   venueSlug,
				Layer:       layer,
				X:           x,
				Y:           y,
				Order:       order,
			})
			cancel()

			if err != nil {
				var denied *actions.ActionDeniedError
				if errors.As(err, &denied) {
					_ = c.Conn.WriteJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("place element denied: user=%s session=%s venue=%s target=%s reason=%s", actorID, sessionID, venueSlug, elementSlug, denied.Reason)
					return
				}

				_ = c.Conn.WriteJSON(map[string]any{
					"type":  "error",
					"error": err.Error(),
				})
				log.Printf("place element failed: user=%s session=%s venue=%s target=%s err=%v", actorID, sessionID, venueSlug, elementSlug, err)
				return
			}

			msgOut, _ := json.Marshal(map[string]any{
				"type": "action",
				"data": storedAction,
			})
			hub.Broadcast(msgOut)
		}

	case "act/duplicate_element":
		{
			sessionID, _ := payload["session_id"].(string)
			elementID, _ := payload["element_id"].(string)
			elementSlug, _ := payload["element_slug"].(string)
			venueSlug, _ := payload["venue_slug"].(string)
			layer, _ := payload["layer"].(string)

			x := 0.0
			if raw, ok := payload["x"].(float64); ok {
				x = raw
			}
			y := 0.0
			if raw, ok := payload["y"].(float64); ok {
				y = raw
			}
			order := 0
			if raw, ok := payload["order"].(float64); ok {
				order = int(raw)
			}
			pinMode, _ := payload["pin_mode"].(string)
			worldX := 0.0
			if raw, ok := payload["world_x"].(float64); ok {
				worldX = raw
			}
			worldY := 0.0
			if raw, ok := payload["world_y"].(float64); ok {
				worldY = raw
			}
			screenX := 0.0
			if raw, ok := payload["screen_x"].(float64); ok {
				screenX = raw
			}
			screenY := 0.0
			if raw, ok := payload["screen_y"].(float64); ok {
				screenY = raw
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			storedAction, err := storeDuplicateElementFunc(ctx, pool, actions.DuplicateElementRequest{
				SessionID:   sessionID,
				ActorID:     c.UserID,
				ElementID:   elementID,
				ElementSlug: elementSlug,
				VenueSlug:   venueSlug,
				Layer:       layer,
				X:           x,
				Y:           y,
				Order:       order,
				PinMode:     pinMode,
				WorldX:      worldX,
				WorldY:      worldY,
				ScreenX:     screenX,
				ScreenY:     screenY,
			})
			cancel()

			if err != nil {
				var denied *actions.ActionDeniedError
				if errors.As(err, &denied) {
					_ = c.Conn.WriteJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("duplicate denied: user=%s session=%s venue=%s target=%s reason=%s", c.UserID, sessionID, venueSlug, elementSlug, denied.Reason)
					return
				}

				_ = c.Conn.WriteJSON(map[string]any{
					"type":  "error",
					"error": err.Error(),
				})
				log.Printf("duplicate failed: user=%s session=%s venue=%s target=%s err=%v", c.UserID, sessionID, venueSlug, elementSlug, err)
				return
			}

			msgOut, _ := json.Marshal(map[string]any{
				"type": "action",
				"data": storedAction,
			})
			hub.Broadcast(msgOut)
		}

	case "act/set_element_lock":
		{
			sessionID, _ := payload["session_id"].(string)
			elementID, _ := payload["element_id"].(string)
			elementSlug, _ := payload["element_slug"].(string)
			locked := false
			if v, ok := payload["locked"].(bool); ok {
				locked = v
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			storedAction, err := storeSetElementLockFunc(ctx, pool, actions.ElementLockRequest{
				SessionID:   sessionID,
				ActorID:     c.UserID,
				ElementID:   elementID,
				ElementSlug: elementSlug,
				Locked:      locked,
			})
			cancel()

			if err != nil {
				var denied *actions.ActionDeniedError
				if errors.As(err, &denied) {
					_ = c.Conn.WriteJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("lock denied: user=%s session=%s reason=%s", c.UserID, sessionID, denied.Reason)
					return
				}

				_ = c.Conn.WriteJSON(map[string]any{
					"type":  "error",
					"error": err.Error(),
				})
				log.Printf("lock store failed: user=%s session=%s err=%v", c.UserID, sessionID, err)
				return
			}

			msgOut, _ := json.Marshal(map[string]any{
				"type": "action",
				"data": storedAction,
			})
			hub.Broadcast(msgOut)
		}

	case "act/set_nameplate_visibility":
		{
			sessionID, _ := payload["session_id"].(string)
			elementID, _ := payload["element_id"].(string)
			elementSlug, _ := payload["element_slug"].(string)
			layer, _ := payload["layer"].(string)
			visible := true
			if v, ok := payload["visible"].(bool); ok {
				visible = v
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			storedAction, err := storeSetNameplateVisibilityFunc(ctx, pool, actions.NameplateVisibilityRequest{
				SessionID:   sessionID,
				ActorID:     c.UserID,
				ElementID:   elementID,
				ElementSlug: elementSlug,
				Visible:     visible,
				Layer:       layer,
			})
			cancel()

			if err != nil {
				var denied *actions.ActionDeniedError
				if errors.As(err, &denied) {
					_ = c.Conn.WriteJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("nameplate denied: user=%s session=%s reason=%s", c.UserID, sessionID, denied.Reason)
					return
				}

				_ = c.Conn.WriteJSON(map[string]any{
					"type":  "error",
					"error": err.Error(),
				})
				log.Printf("nameplate store failed: user=%s session=%s err=%v", c.UserID, sessionID, err)
				return
			}

			msgOut, _ := json.Marshal(map[string]any{
				"type": "action",
				"data": storedAction,
			})
			hub.Broadcast(msgOut)
		}

	case "create/index_card":
		{
			sessionID, _ := payload["session_id"].(string)
			frontText, _ := payload["front_text"].(string)
			backText, _ := payload["back_text"].(string)
			color, _ := payload["color"].(string)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			storedAction, err := storeIndexCardCreateFunc(ctx, pool, actions.IndexCardRequest{
				SessionID: sessionID,
				ActorID:   c.UserID,
				FrontText: frontText,
				BackText:  backText,
				Color:     color,
			})
			cancel()

			if err != nil {
				var denied *actions.ActionDeniedError
				if errors.As(err, &denied) {
					_ = c.Conn.WriteJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("index card denied: user=%s session=%s action=%s reason=%s", c.UserID, sessionID, payload["type"], denied.Reason)
					return
				}

				_ = c.Conn.WriteJSON(map[string]any{
					"type":  "error",
					"error": err.Error(),
				})
				log.Printf("index card create failed: user=%s session=%s err=%v", c.UserID, sessionID, err)
				return
			}

			msgOut, _ := json.Marshal(map[string]any{
				"type": "action",
				"data": storedAction,
			})
			hub.Broadcast(msgOut)
		}

	case "update/index_card":
		{
			sessionID, _ := payload["session_id"].(string)
			elementID, _ := payload["element_id"].(string)
			elementSlug, _ := payload["element_slug"].(string)
			frontText, _ := payload["front_text"].(string)
			backText, _ := payload["back_text"].(string)
			color, _ := payload["color"].(string)
			pinMode, _ := payload["pin_mode"].(string)
			worldX := 0.0
			if raw, ok := payload["world_x"].(float64); ok {
				worldX = raw
			}
			worldY := 0.0
			if raw, ok := payload["world_y"].(float64); ok {
				worldY = raw
			}
			screenX := 0.0
			if raw, ok := payload["screen_x"].(float64); ok {
				screenX = raw
			}
			screenY := 0.0
			if raw, ok := payload["screen_y"].(float64); ok {
				screenY = raw
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			storedAction, err := storeIndexCardUpdateFunc(ctx, pool, actions.IndexCardRequest{
				SessionID:   sessionID,
				ActorID:     c.UserID,
				ElementID:   elementID,
				ElementSlug: elementSlug,
				FrontText:   frontText,
				BackText:    backText,
				Color:       color,
				PinMode:     pinMode,
				WorldX:      worldX,
				WorldY:      worldY,
				ScreenX:     screenX,
				ScreenY:     screenY,
			})
			cancel()

			if err != nil {
				var denied *actions.ActionDeniedError
				if errors.As(err, &denied) {
					_ = c.Conn.WriteJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("index card denied: user=%s session=%s action=%s reason=%s", c.UserID, sessionID, payload["type"], denied.Reason)
					return
				}

				_ = c.Conn.WriteJSON(map[string]any{
					"type":  "error",
					"error": err.Error(),
				})
				log.Printf("index card update failed: user=%s session=%s err=%v", c.UserID, sessionID, err)
				return
			}

			msgOut, _ := json.Marshal(map[string]any{
				"type": "action",
				"data": storedAction,
			})
			hub.Broadcast(msgOut)
		}

	case "delete/index_card":
		{
			sessionID, _ := payload["session_id"].(string)
			elementID, _ := payload["element_id"].(string)
			elementSlug, _ := payload["element_slug"].(string)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			storedAction, err := storeIndexCardDeleteFunc(ctx, pool, actions.IndexCardRequest{
				SessionID:   sessionID,
				ActorID:     c.UserID,
				ElementID:   elementID,
				ElementSlug: elementSlug,
			})
			cancel()

			if err != nil {
				var denied *actions.ActionDeniedError
				if errors.As(err, &denied) {
					_ = c.Conn.WriteJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("index card denied: user=%s session=%s action=%s reason=%s", c.UserID, sessionID, payload["type"], denied.Reason)
					return
				}

				_ = c.Conn.WriteJSON(map[string]any{
					"type":  "error",
					"error": err.Error(),
				})
				log.Printf("index card delete failed: user=%s session=%s err=%v", c.UserID, sessionID, err)
				return
			}

			msgOut, _ := json.Marshal(map[string]any{
				"type": "action",
				"data": storedAction,
			})
			hub.Broadcast(msgOut)
		}

	case "act/remove_element":
		{
			sessionID, _ := payload["session_id"].(string)
			elementID, _ := payload["element_id"].(string)
			elementSlug, _ := payload["element_slug"].(string)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			storedAction, err := storeRemoveElementFunc(ctx, pool, actions.RemoveElementRequest{
				SessionID:   sessionID,
				ActorID:     c.UserID,
				ElementID:   elementID,
				ElementSlug: elementSlug,
			})
			cancel()

			if err != nil {
				var denied *actions.ActionDeniedError
				if errors.As(err, &denied) {
					_ = c.Conn.WriteJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("remove element denied: user=%s session=%s action=%s reason=%s", c.UserID, sessionID, payload["type"], denied.Reason)
					return
				}

				_ = c.Conn.WriteJSON(map[string]any{
					"type":  "error",
					"error": err.Error(),
				})
				log.Printf("remove element failed: user=%s session=%s err=%v", c.UserID, sessionID, err)
				return
			}

			msgOut, _ := json.Marshal(map[string]any{
				"type": "action",
				"data": storedAction,
			})
			hub.Broadcast(msgOut)
		}

	case "persona/equip", "persona/unequip":
		{
			sessionID, _ := payload["session_id"].(string)
			characterCardID, _ := payload["character_card_id"].(string)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			var storedAction *actions.StoredAction
			var err error
			if payload["type"] == "persona/equip" {
				storedAction, err = storePersonaEquipFunc(ctx, pool, actions.PersonaRequest{
					SessionID:       sessionID,
					ActorID:         c.UserID,
					CharacterCardID: characterCardID,
				})
			} else {
				storedAction, err = storePersonaUnequipFunc(ctx, pool, actions.PersonaRequest{
					SessionID: sessionID,
					ActorID:   c.UserID,
				})
			}
			cancel()

			if err != nil {
				var denied *actions.ActionDeniedError
				if errors.As(err, &denied) {
					_ = c.Conn.WriteJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("persona denied: user=%s session=%s action=%s reason=%s", c.UserID, sessionID, payload["type"], denied.Reason)
					return
				}

				_ = c.Conn.WriteJSON(map[string]any{
					"type":  "error",
					"error": err.Error(),
				})
				log.Printf("persona action failed: user=%s session=%s action=%s err=%v", c.UserID, sessionID, payload["type"], err)
				return
			}

			msgOut, _ := json.Marshal(map[string]any{
				"type": "action",
				"data": storedAction,
			})
			hub.Broadcast(msgOut)

			hub.UpdateClientPresence(sessionID, c.UserID, storedAction.Persona)
			if updated, ok := hub.Presence().Update(sessionID, PresenceUser{
				UserID:      c.UserID,
				Handle:      c.Presence.Handle,
				DisplayName: c.Presence.DisplayName,
				Role:        c.Presence.Role,
				Persona:     storedAction.Persona,
			}); ok {
				broadcastPresenceEvent(hub, sessionID, "presence/update", updated)
			}
		}
	}
}

func lookupVenueRole(ctx context.Context, pool *pgxpool.Pool, userID string, venueSlug string) (string, error) {
	var role string

	err := pool.QueryRow(ctx, `
		SELECT lower(role_text) FROM (
			-- exact venue membership first
			SELECT m.role::text AS role_text, 1 AS priority
			FROM memberships m
			JOIN venues v ON v.id = m.venue_id
			WHERE m.user_id = $1
			  AND v.slug = $2

			UNION ALL

			-- fallback: global producer membership
			SELECT m.role::text AS role_text, 2 AS priority
			FROM memberships m
			WHERE m.user_id = $1
			  AND m.venue_id IS NULL
			  AND m.role::text = 'producer'
		) ranked
		ORDER BY priority
		LIMIT 1
	`, userID, venueSlug).Scan(&role)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "none", nil
		}
		return "", err
	}

	return role, nil
}
