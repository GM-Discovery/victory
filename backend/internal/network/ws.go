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
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/actions"
	"victory/backend/internal/announcements"
	"victory/backend/internal/identity"
	"victory/backend/internal/ratelimit"
	"victory/backend/internal/rollaudience"
	"victory/backend/internal/showings"
	"victory/backend/internal/stageeffects"
	"victory/backend/internal/world"
)

// Kernel 77 K77-06: WebSocket connections and messages had no rate limiting
// at all (K76's own websocket-matrix recorded this as an open item).
// wsConnectionLimiter bounds new-connection attempts per client IP;
// wsMessageLimiter bounds inbound messages per connected user, shared
// across every venue/cave/profile socket that calls checkMessageRate.
var wsConnectionLimiter = ratelimit.New(10, 20)
var wsMessageLimiter = ratelimit.New(20, 120)

// checkMessageRate reports whether userID may process another inbound
// WebSocket message right now. Over-limit messages are dropped silently
// (not a connection-closing offense -- a burst during normal play, e.g.
// rapid clicking, should not disconnect a legitimate user), matching the
// same token-bucket approach already used for HTTP routes.
func checkMessageRate(userID string) bool {
	if strings.TrimSpace(userID) == "" {
		return true
	}
	return wsMessageLimiter.Allow(userID)
}

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
var storeICChatMessageFunc = actions.StoreICChatMessage
var storeSpeakFunc = actions.StoreSpeak
var storeRevealFunc = actions.StoreReveal
var storeDiceRollFunc = actions.StoreDiceRoll
var storePlayerMechanicRollFunc = actions.StorePlayerMechanicRoll
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
var storeCreateTokenFunc = actions.StoreCreateToken
var storeUpdateTokenFunc = actions.StoreUpdateToken
var storePersonaEquipFunc = actions.StorePersonaEquip
var storePersonaUnequipFunc = actions.StorePersonaUnequip
var discordBridgeConfig identity.DiscordServerLinkConfig

// stageEffectRegistry is a single process-wide in-memory store, matching
// hub's own singleton lifecycle (one Hub, created once in cmd/victory/
// main.go). It carries no DB/pool dependency, so unlike venuecoordination
// (threaded explicitly into storyboards' handlers for its DI-friendly
// signature) it can live as a package var here the same way
// rollDiceLimiter and wsMessageLimiter already do -- avoiding a signature
// change to ServeVenueWS/ServeCaveWS/handleVenuePayload/handleCavePayload
// and every existing caller/test of them.
var stageEffectRegistry = stageeffects.NewRegistry()

// stageEffectDefaultDurationMs / stageEffectMaxDurationMs bound the
// transient hold Kernel 86 §1.4 asks for, while still letting a caller
// request a longer hold within reason -- never unbounded, per §16's "queue
// must not enable unbounded client memory growth." Raised from the
// original ~3.5s per live-testing feedback (2026-08-28): dice cleared too
// fast to actually read before fading. A viewer can still Pin a settled
// roll from dice-projection.js's announcement node to hold it indefinitely
// past this.
const stageEffectDefaultDurationMs = 8500
const stageEffectMaxDurationMs = 15000

// announcementDefaultDurationMs holds a Kernel 89 announcement its own
// bounded duration, independent of stageEffectDefaultDurationMs -- an
// announcement IS the whole message and needs long enough for a table
// mid-conversation to look up and take it in. Still bounded by
// stageEffectMaxDurationMs like everything else on the registry.
const announcementDefaultDurationMs = 5000

// resolvedStoredMode defaults an already-stored (or absent/malformed)
// audience mode to Show -- deliberately NOT rollaudience.NormalizeMode,
// whose "" case means "client didn't request a mode, apply the Cohort
// default" for a *new* roll request. Here "" means "this stored/mocked
// effect never got an audienceMode written," which must fail open to the
// old unrestricted behavior (Show), not silently reinterpret it as a fresh
// Cohort request.
func resolvedStoredMode(raw string) string {
	switch strings.TrimSpace(raw) {
	case rollaudience.ModeCohort, rollaudience.ModeDirector, rollaudience.ModePrivate, rollaudience.ModeShow:
		return strings.TrimSpace(raw)
	default:
		return rollaudience.ModeShow
	}
}

// audienceDiceRollsHidden reports whether sessionID's Showing has the
// Kernel 93 Audience Dice Rolls toggle turned off. A local, deliberate
// duplicate of audienceprojection.DiceRollsHiddenFromAudience's single-row
// read (raw SQL against audience_projection_configs, not a package import)
// -- network cannot import audienceprojection, since audienceprojection
// imports shows, and shows already imports network (shows/http.go,
// shows/sessions.go), which would be a cycle. A session with no Showing
// (legacy/tutorial venue) is unaffected -- dice stays visible, matching
// current public rollaudience.ModeShow behavior (spec §5).
func audienceDiceRollsHidden(ctx context.Context, pool *pgxpool.Pool, sessionID string) bool {
	showing, err := showings.LoadBySession(ctx, pool, sessionID)
	if err != nil {
		return false
	}
	var showDiceRolls bool
	err = pool.QueryRow(ctx, `
		SELECT show_dice_rolls FROM audience_projection_configs WHERE showing_id = $1
	`, showing.ID).Scan(&showDiceRolls)
	if err != nil {
		// No row yet means the documented default (spec §5: dice ON,
		// matching current public behavior) -- not hidden.
		return false
	}
	return !showDiceRolls
}

// nonAudienceSessionRecipients returns every session participant whose role
// is not "audience", plus actorID unconditionally (the roller always sees
// their own roll -- rollaudience.VisibleToViewer's same rule). Used only
// when Show-mode delivery would otherwise broadcast to the whole session but
// the Showing's Audience Dice Rolls toggle is off.
func nonAudienceSessionRecipients(ctx context.Context, pool *pgxpool.Pool, sessionID, actorID string) []string {
	set := map[string]bool{actorID: true}
	rows, err := pool.Query(ctx, `
		SELECT user_id::text FROM session_participants
		WHERE session_id = $1 AND role::text != 'audience'
	`, sessionID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id string
			if scanErr := rows.Scan(&id); scanErr == nil && id != "" {
				set[id] = true
			}
		}
	}
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	return out
}

// deliverStageMessage sends msg to exactly the audience Kernel 86 resolved
// for a roll (backend/internal/rollaudience), reusing the same Decision
// shape StoreDiceRoll already computed and stored on the Action's
// visibility. Show mode is delivered via Hub.BroadcastSession (every
// socket on this session, not the entire server -- the audit's global-leak
// fix) rather than enumerating a recipient set.
//
// effectType distinguishes what's actually being delivered ("dice_roll",
// "announcement", or a stage_effect/pin-dismiss's own effect.Type) --
// deliverStageMessage is the one shared dispatch point for every Kernel 86
// Stage Effect, not just dice rolls, so the Director's "Audience Dice
// Rolls" toggle must only narrow delivery when a roll is what's actually
// being sent. Live testing (2026-08-28) found announcements silently
// stopped reaching Audience too whenever that toggle was off, because the
// gate below used to apply unconditionally to every Show-mode message
// this function ever sends.
func deliverStageMessage(ctx context.Context, hub *Hub, pool *pgxpool.Pool, sessionID, actorID, audienceMode, cohortID, effectType string, msg []byte) {
	decision := rollaudience.Decision{Mode: resolvedStoredMode(audienceMode), CohortID: strings.TrimSpace(cohortID)}

	recipients, useSessionBroadcast, err := rollaudience.LiveRecipients(ctx, pool, decision, actorID, sessionID)
	if err != nil {
		log.Printf("stage effect recipient resolution failed: session=%s err=%v", sessionID, err)
		return
	}
	if useSessionBroadcast {
		// Kernel 93 §18: Show-mode delivery is normally a full session
		// broadcast (everyone already able to view this session). When the
		// Showing's Director has turned Audience Dice Rolls off, narrow that
		// to everyone except Audience-role sockets instead, and only for an
		// actual dice roll -- Director+/Cast/Crew are unaffected, and the
		// roller always sees their own roll (nonAudienceSessionRecipients
		// always includes actorID).
		if decision.Mode == rollaudience.ModeShow && effectType == "dice_roll" && audienceDiceRollsHidden(ctx, pool, sessionID) {
			hub.SendToUsers(sessionID, nonAudienceSessionRecipients(ctx, pool, sessionID, actorID), msg)
			return
		}
		hub.BroadcastSession(sessionID, msg)
		return
	}
	hub.SendToUsers(sessionID, recipients, msg)
}

// stageEffectAuthorized reports whether userID may pin/dismiss e: the
// roller always may; otherwise Director+ may, for any audience except
// Private (kernel §10: "unrelated users cannot dismiss another cohort's
// private/static projection" -- Private has no authorized viewer besides
// the roller in the first place, so no one else should be able to touch it
// either).
func stageEffectAuthorized(ctx context.Context, pool *pgxpool.Pool, e stageeffects.Effect, userID string) (bool, error) {
	userID = strings.TrimSpace(userID)
	if userID != "" && userID == e.ActorID {
		return true, nil
	}
	if e.Audience == rollaudience.ModePrivate {
		return false, nil
	}
	return rollaudience.IsDirectorPlus(ctx, pool, e.SessionID, userID)
}

func ServeCaveWS(hub *Hub, pool *pgxpool.Pool, discordLinkCfg identity.DiscordServerLinkConfig) http.HandlerFunc {
	return ServeVenueWS(hub, pool, discordLinkCfg, "the-cave")
}

func ServeVenueWS(hub *Hub, pool *pgxpool.Pool, discordLinkCfg identity.DiscordServerLinkConfig, venueSlug string) http.HandlerFunc {
	discordBridgeConfig = discordLinkCfg
	venueSlug = strings.ToLower(strings.TrimSpace(venueSlug))
	if venueSlug == "" {
		venueSlug = "the-cave"
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// Kernel 77 K77-06: bound connection *attempts* per IP, before doing
		// any authentication work at all.
		if !wsConnectionLimiter.Allow(ratelimit.ClientIP(r)) {
			http.Error(w, "rate_limited", http.StatusTooManyRequests)
			return
		}

		// Kernel 77 K77-08 (K76-L01): authenticate and authorize before
		// completing the WebSocket handshake, not after. An anonymous or
		// unauthorized caller previously drove a full upgrade -- cheap flood
		// capacity -- and only then learned it would be rejected. Every
		// check below runs as a plain HTTP request/response; the connection
		// is only ever upgraded once it's already known to be allowed.
		sessionCookie := ""
		if c, err := r.Cookie("victory_session"); err == nil {
			sessionCookie = c.Value
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
		if err != nil {
			http.Error(w, "not_authenticated", http.StatusUnauthorized)
			return
		}

		allowed, err := access.UserCanAccessVenueSlug(ctx, pool, userID, venueSlug)
		if err != nil {
			http.Error(w, "access_check_failed", http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		sessionID, err := identity.ResolveActiveVenueSessionID(ctx, pool, userID, venueSlug)
		if err != nil {
			http.Error(w, "not_session_participant", http.StatusForbidden)
			return
		}

		sessionIdentity, err := identity.ResolveSessionIdentityForVenue(ctx, pool, sessionID, userID, venueSlug)
		if err != nil {
			http.Error(w, "not_session_participant", http.StatusForbidden)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("ws upgrade failed: %v", err)
			return
		}

		client := &Client{
			Conn:      conn,
			// Kernel 88B raised this from 16: per-client replies (pong,
			// error frames, command acks) used to bypass this buffer via a
			// direct connection write, so it was sized for broadcasts alone.
			// Now every write shares it, and overflow means a dropped reply,
			// not just a skipped broadcast frame.
			Send:      make(chan []byte, 64),
			UserID:    sessionIdentity.UserID,
			SessionID: sessionIdentity.SessionID,
			VenueSlug: venueSlug,
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

		// The handshake writes below deliberately stay on conn.WriteJSON
		// rather than Client.SendJSON. They run before `go writePump(client)`
		// starts, so this goroutine is provably the connection's only writer
		// -- the race SendJSON exists to prevent cannot occur here -- and each
		// one needs its error synchronously in order to abandon the
		// connection. Queuing them would hand the failure to a goroutine that
		// has not been started yet. Everything after writePump starts must use
		// SendJSON.
		snapshot, err := world.LoadVenueSnapshot(ctx, pool, sessionIdentity.Role, sessionIdentity.UserID, venueSlug)
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

		// Kernel 86 §10: a pinned/static roll projection survives ordinary
		// reconnect. Each pinned effect is re-checked against this viewer's
		// current audience authority (never trusted from what it was
		// created with) before being sent -- the same rollaudience.
		// VisibleToViewer test snapshot filtering uses, so a reconnecting
		// viewer never receives a pinned effect their role/cohort no longer
		// authorizes.
		visiblePinned := make([]stageeffects.Effect, 0)
		for _, e := range stageEffectRegistry.ListPinned(sessionIdentity.SessionID) {
			decision := rollaudience.Decision{Mode: resolvedStoredMode(e.Audience), CohortID: e.CohortID}
			visible, err := rollaudience.VisibleToViewer(ctx, pool, decision, e.ActorID, snapshot.Session.ShowID, sessionIdentity.UserID, sessionIdentity.Role)
			if err != nil {
				log.Printf("ws pinned stage effect visibility check failed: %v", err)
				continue
			}
			// Kernel 93 §18: same narrowing as deliverStageMessage's live
			// path -- a reconnecting Audience viewer never receives a
			// Show-mode pinned roll the Showing's Director has hidden from
			// Audience, except their own.
			if visible && decision.Mode == rollaudience.ModeShow && strings.EqualFold(strings.TrimSpace(sessionIdentity.Role), "audience") && e.ActorID != sessionIdentity.UserID {
				if audienceDiceRollsHidden(ctx, pool, sessionIdentity.SessionID) {
					visible = false
				}
			}
			if visible {
				visiblePinned = append(visiblePinned, e)
			}
		}
		if err := conn.WriteJSON(map[string]any{
			"type": "stage_effects/pinned",
			"data": visiblePinned,
		}); err != nil {
			log.Printf("ws pinned stage effects write failed: %v", err)
			hub.Presence().Disconnect(sessionIdentity.SessionID, client.UserID)
			_ = conn.Close()
			hub.Remove(client)
			return
		}

		go writePump(client)

		if joined {
			broadcastPresenceEvent(hub, sessionIdentity.SessionID, "presence/join", client.Presence)
		}

		readPump(hub, pool, client, venueSlug)
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

// maxVenueMessageBytes caps a single inbound venue frame. Victory's client
// messages are commands, chat lines, and index cards; none approach this.
// Kernel 76 (K76-M03): there was no limit at all, so one authorized socket
// could stream an arbitrarily large frame straight into the server's memory.
const maxVenueMessageBytes = 256 << 10

func readPump(hub *Hub, pool *pgxpool.Pool, c *Client, venueSlug string) {
	c.Conn.SetReadLimit(maxVenueMessageBytes)

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

		if !checkMessageRate(c.UserID) {
			continue
		}

		var payload map[string]any
		if err := json.Unmarshal(msg, &payload); err != nil {
			continue
		}

		handleVenuePayload(hub, pool, c, payload, venueSlug)
	}
}

func handleCavePayload(hub *Hub, pool *pgxpool.Pool, c *Client, payload map[string]any) {
	handleVenuePayload(hub, pool, c, payload, "the-cave")
}

func handleVenuePayload(hub *Hub, pool *pgxpool.Pool, c *Client, payload map[string]any, venueSlug string) {
	switch payload["type"] {
	case "ping":
		_ = c.SendJSON(map[string]any{
			"type": "pong",
			"ts":   time.Now().UTC().Format(time.RFC3339),
		})

	case "character/projection_updated":
		_ = c.SendJSON(map[string]any{
			"type":  "error",
			"error": "server_authored_event_only",
		})

	case "venue/focus_ping":
		{
			sessionID := c.SessionID
			if strings.TrimSpace(sessionID) == "" {
				_ = c.SendJSON(map[string]any{
					"type":  "error",
					"error": "not_session_participant",
				})
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			allowed, err := canAccessDirectorConsole(ctx, pool, c.UserID)
			cancel()
			if err != nil {
				_ = c.SendJSON(map[string]any{
					"type":  "error",
					"error": "access_check_failed",
				})
				return
			}
			if !allowed {
				_ = c.SendJSON(map[string]any{
					"type":  "error",
					"error": "forbidden",
				})
				return
			}

			venueSlug = strings.ToLower(strings.TrimSpace(venueSlug))
			if venueSlug == "" {
				venueSlug = "the-cave"
			}

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
				_ = c.SendJSON(map[string]any{
					"type":  "error",
					"error": clientSafeError(err),
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
					_ = c.SendJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("chat denied: user=%s session=%s action=%s reason=%s", actorID, sessionID, payload["type"], denied.Reason)
					return
				}

				_ = c.SendJSON(map[string]any{
					"type":  "error",
					"error": clientSafeError(err),
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

	case "chat/ic_message":
		{
			// Kernel 87 §10: the client sends only session_id/text. The
			// speaker Character is resolved server-side inside
			// storeICChatMessageFunc -- there is no character_id field
			// read from payload here, by design, so there is nothing a
			// forged request could populate to impersonate a Character.
			sessionID, _ := payload["session_id"].(string)
			actorID := c.UserID
			text, _ := payload["text"].(string)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			storedAction, err := storeICChatMessageFunc(ctx, pool, actions.ICChatMessageRequest{
				SessionID: sessionID,
				ActorID:   actorID,
				Text:      text,
			})
			cancel()

			if err != nil {
				reason := clientSafeError(err)
				var denied *actions.ActionDeniedError
				if errors.As(err, &denied) {
					reason = denied.Reason
				}
				_ = c.SendJSON(map[string]any{
					"type":  "error",
					"error": reason,
				})
				log.Printf("ic chat store failed: user=%s session=%s err=%v", actorID, sessionID, err)
				return
			}

			msgOut, _ := json.Marshal(map[string]any{
				"type": "action",
				"data": storedAction,
			})
			hub.Broadcast(msgOut)
		}

	case "roll/dice":
		{
			sessionID, _ := payload["session_id"].(string)
			actorID := c.UserID
			requestID, _ := payload["request_id"].(string)
			expression, _ := payload["expression"].(string)
			visibility, _ := payload["visibility"].(string)
			label, _ := payload["label"].(string)
			skillID, _ := payload["skill_id"].(string)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			storedAction, err := storeDiceRollFunc(ctx, pool, actions.DiceRollRequest{
				SessionID:  sessionID,
				ActorID:    actorID,
				RequestID:  requestID,
				Expression: expression,
				Visibility: visibility,
				Label:      label,
				SkillID:    skillID,
			})
			cancel()

			if err != nil {
				var denied *actions.ActionDeniedError
				if errors.As(err, &denied) {
					errorPayload := map[string]any{
						"type":  "error",
						"error": denied.Reason,
					}
					if strings.TrimSpace(requestID) != "" {
						errorPayload["request_id"] = requestID
					}
					_ = c.SendJSON(errorPayload)
					log.Printf("dice denied: user=%s session=%s action=%s reason=%s", actorID, sessionID, payload["type"], denied.Reason)
					return
				}

				errorPayload := map[string]any{
					"type":  "error",
					"error": clientSafeError(err),
				}
				if strings.TrimSpace(requestID) != "" {
					errorPayload["request_id"] = requestID
				}
				_ = c.SendJSON(errorPayload)
				log.Printf("dice store failed: user=%s session=%s action=%s err=%v", actorID, sessionID, payload["type"], err)
				return
			}

			audienceMode, _ := storedAction.Visibility["audienceMode"].(string)
			cohortID, _ := storedAction.Visibility["cohortId"].(string)

			msgOut, _ := json.Marshal(map[string]any{
				"type": "action",
				"data": storedAction,
			})
			deliverCtx, deliverCancel := context.WithTimeout(context.Background(), 5*time.Second)
			deliverStageMessage(deliverCtx, hub, pool, sessionID, actorID, audienceMode, cohortID, "dice_roll", msgOut)

			durationMs := stageEffectDefaultDurationMs
			if raw, ok := payload["duration_ms"].(float64); ok && raw > 0 {
				durationMs = int(raw)
				if durationMs > stageEffectMaxDurationMs {
					durationMs = stageEffectMaxDurationMs
				}
			}
			effect := stageEffectRegistry.Create(sessionID, stageeffects.Effect{
				Type:           "dice_roll",
				SourceActionID: storedAction.ID,
				CohortID:       cohortID,
				Audience:       audienceMode,
				ActorID:        actorID,
				Label:          label,
				DurationMs:     durationMs,
				Payload: map[string]any{
					"actor":           storedAction.Actor,
					"label":           storedAction.Payload["label"],
					"expression":      storedAction.Payload["expression"],
					"dice":            storedAction.Payload["dice"],
					"modifier":        storedAction.Payload["modifier"],
					"total":           storedAction.Payload["total"],
					"explosion_count": storedAction.Payload["explosion_count"],
					"skill_id":        storedAction.Payload["skill_id"],
				},
			})
			effectMsg, _ := json.Marshal(map[string]any{
				"type": "stage_effect",
				"data": effect,
			})
			deliverStageMessage(deliverCtx, hub, pool, sessionID, actorID, audienceMode, cohortID, "dice_roll", effectMsg)
			deliverCancel()
		}

	case "roll/dice_own_mechanic":
		{
			// Kernel 88 §11: the client sends no expression and no
			// character_id -- both are resolved/computed server-side inside
			// storePlayerMechanicRollFunc from the caller's own Show-Run
			// roster selection and their own Character's own compiler-backed
			// skill, mirroring chat/ic_message's identity-resolution pattern
			// above. Delivery mirrors "roll/dice" exactly (same action
			// broadcast, same theatrical stage_effect) so a Player's own
			// roll plays out on stage for the Audience the same way a
			// Director-triggered roll does.
			sessionID, _ := payload["session_id"].(string)
			actorID := c.UserID
			requestID, _ := payload["request_id"].(string)
			skillID, _ := payload["skill_id"].(string)
			primaryActionID, _ := payload["primary_action_id"].(string)
			visibility, _ := payload["visibility"].(string)
			label, _ := payload["label"].(string)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			storedAction, err := storePlayerMechanicRollFunc(ctx, pool, actions.PlayerMechanicRollRequest{
				SessionID:       sessionID,
				ActorID:         actorID,
				RequestID:       requestID,
				SkillID:         skillID,
				PrimaryActionID: primaryActionID,
				Visibility:      visibility,
				Label:           label,
			})
			cancel()

			if err != nil {
				var denied *actions.ActionDeniedError
				if errors.As(err, &denied) {
					errorPayload := map[string]any{
						"type":  "error",
						"error": denied.Reason,
					}
					if strings.TrimSpace(requestID) != "" {
						errorPayload["request_id"] = requestID
					}
					_ = c.SendJSON(errorPayload)
					log.Printf("player mechanic roll denied: user=%s session=%s reason=%s", actorID, sessionID, denied.Reason)
					return
				}

				errorPayload := map[string]any{
					"type":  "error",
					"error": clientSafeError(err),
				}
				if strings.TrimSpace(requestID) != "" {
					errorPayload["request_id"] = requestID
				}
				_ = c.SendJSON(errorPayload)
				log.Printf("player mechanic roll store failed: user=%s session=%s err=%v", actorID, sessionID, err)
				return
			}

			audienceMode, _ := storedAction.Visibility["audienceMode"].(string)
			cohortID, _ := storedAction.Visibility["cohortId"].(string)

			msgOut, _ := json.Marshal(map[string]any{
				"type": "action",
				"data": storedAction,
			})
			deliverCtx, deliverCancel := context.WithTimeout(context.Background(), 5*time.Second)
			deliverStageMessage(deliverCtx, hub, pool, sessionID, actorID, audienceMode, cohortID, "dice_roll", msgOut)

			durationMs := stageEffectDefaultDurationMs
			if raw, ok := payload["duration_ms"].(float64); ok && raw > 0 {
				durationMs = int(raw)
				if durationMs > stageEffectMaxDurationMs {
					durationMs = stageEffectMaxDurationMs
				}
			}
			effect := stageEffectRegistry.Create(sessionID, stageeffects.Effect{
				Type:           "dice_roll",
				SourceActionID: storedAction.ID,
				CohortID:       cohortID,
				Audience:       audienceMode,
				ActorID:        actorID,
				Label:          label,
				DurationMs:     durationMs,
				Payload: map[string]any{
					"actor":           storedAction.Actor,
					"label":           storedAction.Payload["label"],
					"expression":      storedAction.Payload["expression"],
					"dice":            storedAction.Payload["dice"],
					"modifier":        storedAction.Payload["modifier"],
					"total":           storedAction.Payload["total"],
					"explosion_count": storedAction.Payload["explosion_count"],
					"skill_id":        storedAction.Payload["skill_id"],
				},
			})
			effectMsg, _ := json.Marshal(map[string]any{
				"type": "stage_effect",
				"data": effect,
			})
			deliverStageMessage(deliverCtx, hub, pool, sessionID, actorID, audienceMode, cohortID, "dice_roll", effectMsg)
			deliverCancel()
		}

	case "announce/push":
		{
			// Kernel 89 §10: a theatrical Director announcement. The
			// Director chooses BOTH the words and what the moment means --
			// nothing here reads a die, a total, or a target complexity,
			// and §10.4 forbids adding such a mapping later without an
			// explicit canonical rule requiring it.
			//
			// Deliberately a Stage Effect and not an Action: an
			// announcement is presentation derived from a decision the
			// Director already made out loud, not a new canonical fact
			// about the world. That keeps it on Kernel 86's existing
			// ephemeral registry (no table, no migration, no second dice-
			// history universe) and gives it pin/dismiss for free.
			sessionID, _ := payload["session_id"].(string)
			if strings.TrimSpace(sessionID) == "" {
				sessionID = c.SessionID
			}
			actorID := c.UserID
			requestID, _ := payload["request_id"].(string)
			styleKey, _ := payload["style"].(string)
			text, _ := payload["text"].(string)
			visibility, _ := payload["visibility"].(string)

			fail := func(reason string) {
				errorPayload := map[string]any{"type": "error", "error": reason}
				if strings.TrimSpace(requestID) != "" {
					errorPayload["request_id"] = requestID
				}
				_ = c.SendJSON(errorPayload)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Authority first, before the palette is even consulted: a
			// non-Director must not be able to probe which style keys exist
			// by watching which error comes back.
			isDirector, err := rollaudience.IsDirectorPlus(ctx, pool, sessionID, actorID)
			if err != nil {
				fail("access_check_failed")
				return
			}
			if !isDirector {
				log.Printf("announcement denied: user=%s session=%s", actorID, sessionID)
				fail("not_authorized")
				return
			}

			style, resolvedText, err := announcements.Compose(styleKey, text)
			if err != nil {
				log.Printf("announcement compose failed: user=%s session=%s err=%v", actorID, sessionID, err)
				fail(clientSafeError(err))
				return
			}

			mode := rollaudience.NormalizeMode(visibility)
			if mode == "" {
				fail("unsupported_visibility_mode")
				return
			}
			decision, err := rollaudience.Resolve(ctx, pool, sessionID, actorID, mode)
			if err != nil {
				log.Printf("announcement audience resolve failed: user=%s session=%s err=%v", actorID, sessionID, err)
				fail(clientSafeError(err))
				return
			}

			durationMs := announcementDefaultDurationMs
			if raw, ok := payload["duration_ms"].(float64); ok && raw > 0 {
				durationMs = int(raw)
				if durationMs > stageEffectMaxDurationMs {
					durationMs = stageEffectMaxDurationMs
				}
			}

			effect := stageEffectRegistry.Create(sessionID, stageeffects.Effect{
				Type:       "announcement",
				SessionID:  sessionID,
				ShowID:     decision.ShowID,
				CohortID:   decision.CohortID,
				Audience:   decision.Mode,
				ActorID:    actorID,
				Label:      style.Label,
				DurationMs: durationMs,
				Payload: map[string]any{
					"text":  resolvedText,
					"style": style,
				},
			})
			effectMsg, _ := json.Marshal(map[string]any{
				"type": "stage_effect",
				"data": effect,
			})
			deliverStageMessage(ctx, hub, pool, sessionID, actorID, decision.Mode, decision.CohortID, "announcement", effectMsg)
		}

	case "stage_effect/pin", "stage_effect/dismiss":
		{
			effectID, _ := payload["effect_id"].(string)
			sessionID := c.SessionID
			if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(effectID) == "" {
				_ = c.SendJSON(map[string]any{"type": "error", "error": "effect_id_required"})
				return
			}

			effect, ok := stageEffectRegistry.Get(sessionID, effectID)
			if !ok {
				_ = c.SendJSON(map[string]any{"type": "error", "error": "stage_effect_not_found"})
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			authorized, err := stageEffectAuthorized(ctx, pool, effect, c.UserID)
			if err != nil {
				cancel()
				_ = c.SendJSON(map[string]any{"type": "error", "error": "access_check_failed"})
				return
			}
			if !authorized {
				cancel()
				_ = c.SendJSON(map[string]any{"type": "error", "error": "forbidden"})
				return
			}

			if payload["type"] == "stage_effect/pin" {
				pinned, ok := stageEffectRegistry.Pin(sessionID, effectID)
				if !ok {
					cancel()
					_ = c.SendJSON(map[string]any{"type": "error", "error": "stage_effect_not_found"})
					return
				}
				msgOut, _ := json.Marshal(map[string]any{"type": "stage_effect_pinned", "data": pinned})
				deliverStageMessage(ctx, hub, pool, sessionID, effect.ActorID, effect.Audience, effect.CohortID, effect.Type, msgOut)
			} else {
				stageEffectRegistry.Dismiss(sessionID, effectID)
				msgOut, _ := json.Marshal(map[string]any{"type": "stage_effect_dismissed", "effect_id": effectID})
				deliverStageMessage(ctx, hub, pool, sessionID, effect.ActorID, effect.Audience, effect.CohortID, effect.Type, msgOut)
			}
			cancel()
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
				_ = c.SendJSON(map[string]any{
					"type":  "error",
					"error": clientSafeError(err),
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
					_ = c.SendJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("action denied: user=%s session=%s action=%s target=%s reason=%s", actorID, sessionID, payload["type"], elementSlug, denied.Reason)
					return
				}

				_ = c.SendJSON(map[string]any{
					"type":  "error",
					"error": clientSafeError(err),
				})
				log.Printf("action store failed: user=%s session=%s action=%s target=%s err=%v", actorID, sessionID, payload["type"], elementSlug, err)
				return
			}

			// Kernel 90: the action row above is now evidence, not authority.
			// world/snapshot.go no longer replays act/reveal_element to decide
			// what anyone sees, so this control has to write canonical state
			// or it would appear to work and change nothing.
			//
			// Both this legacy control and the Kernel 90 Director menu now
			// reach the same stageobjects.ApplyMutation, so they are two doors
			// to one room rather than the two competing mechanisms §0 asked us
			// to converge. Kept working rather than deleted because the Cave
			// venue still drives it.
			//
			// A session with no Show has no canonical state to write (state is
			// Show-scoped per §26) -- pre-Show legacy sessions therefore lose
			// reveal/hide entirely rather than silently hiding things nobody
			// can un-hide. Recorded in the reportback as a known consequence
			// of the no-backfill decision.
			applyLegacyRevealToCanonicalState(pool, hub, sessionID, c.UserID, storedAction, visible)

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
					_ = c.SendJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("overlay denied: user=%s session=%s action=%s target=%s reason=%s", actorID, sessionID, payload["type"], elementSlug, denied.Reason)
					return
				}

				_ = c.SendJSON(map[string]any{
					"type":  "error",
					"error": clientSafeError(err),
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
					_ = c.SendJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("place element denied: user=%s session=%s venue=%s target=%s reason=%s", actorID, sessionID, venueSlug, elementSlug, denied.Reason)
					return
				}

				_ = c.SendJSON(map[string]any{
					"type":  "error",
					"error": clientSafeError(err),
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

	case "create/token":
		{
			sessionID, _ := payload["session_id"].(string)
			assetID, _ := payload["asset_id"].(string)
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
			scale := 100.0
			if raw, ok := payload["scale"].(float64); ok {
				scale = raw
			}
			snapMode, _ := payload["snap_mode"].(string)
			tokenLayer, _ := payload["token_layer"].(string)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			storedAction, err := storeCreateTokenFunc(ctx, pool, actions.TokenPlacementRequest{
				SessionID:  sessionID,
				ActorID:    c.UserID,
				AssetID:    assetID,
				VenueSlug:  venueSlug,
				Layer:      layer,
				X:          x,
				Y:          y,
				Order:      order,
				SnapMode:   snapMode,
				TokenLayer: tokenLayer,
				Scale:      scale,
			})
			cancel()

			if err != nil {
				var denied *actions.ActionDeniedError
				if errors.As(err, &denied) {
					_ = c.SendJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("create token denied: user=%s session=%s asset=%s reason=%s", c.UserID, sessionID, assetID, denied.Reason)
					return
				}

				_ = c.SendJSON(map[string]any{
					"type":  "error",
					"error": clientSafeError(err),
				})
				log.Printf("create token failed: user=%s session=%s asset=%s err=%v", c.UserID, sessionID, assetID, err)
				return
			}

			msgOut, _ := json.Marshal(map[string]any{
				"type": "action",
				"data": storedAction,
			})
			hub.Broadcast(msgOut)
		}

	case "update/token":
		{
			sessionID, _ := payload["session_id"].(string)
			elementID, _ := payload["element_id"].(string)
			elementSlug, _ := payload["element_slug"].(string)
			assetID, _ := payload["asset_id"].(string)
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
			snapMode, _ := payload["snap_mode"].(string)
			tokenLayer, _ := payload["token_layer"].(string)

			scale := 0.0
			if raw, ok := payload["scale"].(float64); ok {
				scale = raw
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			storedAction, err := storeUpdateTokenFunc(ctx, pool, actions.TokenUpdateRequest{
				SessionID:   sessionID,
				ActorID:     c.UserID,
				ElementID:   elementID,
				ElementSlug: elementSlug,
				AssetID:     assetID,
				VenueSlug:   venueSlug,
				Layer:       layer,
				X:           x,
				Y:           y,
				Order:       order,
				SnapMode:    snapMode,
				TokenLayer:  tokenLayer,
				Scale:       scale,
			})
			cancel()

			if err != nil {
				var denied *actions.ActionDeniedError
				if errors.As(err, &denied) {
					_ = c.SendJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("update token denied: user=%s session=%s target=%s reason=%s", c.UserID, sessionID, elementSlug, denied.Reason)
					return
				}

				_ = c.SendJSON(map[string]any{
					"type":  "error",
					"error": clientSafeError(err),
				})
				log.Printf("update token failed: user=%s session=%s target=%s err=%v", c.UserID, sessionID, elementSlug, err)
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
					_ = c.SendJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("duplicate denied: user=%s session=%s venue=%s target=%s reason=%s", c.UserID, sessionID, venueSlug, elementSlug, denied.Reason)
					return
				}

				_ = c.SendJSON(map[string]any{
					"type":  "error",
					"error": clientSafeError(err),
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
					_ = c.SendJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("lock denied: user=%s session=%s reason=%s", c.UserID, sessionID, denied.Reason)
					return
				}

				_ = c.SendJSON(map[string]any{
					"type":  "error",
					"error": clientSafeError(err),
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
					_ = c.SendJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("nameplate denied: user=%s session=%s reason=%s", c.UserID, sessionID, denied.Reason)
					return
				}

				_ = c.SendJSON(map[string]any{
					"type":  "error",
					"error": clientSafeError(err),
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
					_ = c.SendJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("index card denied: user=%s session=%s action=%s reason=%s", c.UserID, sessionID, payload["type"], denied.Reason)
					return
				}

				_ = c.SendJSON(map[string]any{
					"type":  "error",
					"error": clientSafeError(err),
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
			face, _ := payload["face"].(string)
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
				Face:        face,
				WorldX:      worldX,
				WorldY:      worldY,
				ScreenX:     screenX,
				ScreenY:     screenY,
			})
			cancel()

			if err != nil {
				var denied *actions.ActionDeniedError
				if errors.As(err, &denied) {
					_ = c.SendJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("index card denied: user=%s session=%s action=%s reason=%s", c.UserID, sessionID, payload["type"], denied.Reason)
					return
				}

				_ = c.SendJSON(map[string]any{
					"type":  "error",
					"error": clientSafeError(err),
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
					_ = c.SendJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("index card denied: user=%s session=%s action=%s reason=%s", c.UserID, sessionID, payload["type"], denied.Reason)
					return
				}

				_ = c.SendJSON(map[string]any{
					"type":  "error",
					"error": clientSafeError(err),
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
					_ = c.SendJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("remove element denied: user=%s session=%s action=%s reason=%s", c.UserID, sessionID, payload["type"], denied.Reason)
					return
				}

				_ = c.SendJSON(map[string]any{
					"type":  "error",
					"error": clientSafeError(err),
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
					_ = c.SendJSON(map[string]any{
						"type":  "error",
						"error": denied.Reason,
					})
					log.Printf("persona denied: user=%s session=%s action=%s reason=%s", c.UserID, sessionID, payload["type"], denied.Reason)
					return
				}

				_ = c.SendJSON(map[string]any{
					"type":  "error",
					"error": clientSafeError(err),
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
