package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"victory/backend/internal/actions"
	"victory/backend/internal/identity"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const discordGatewayVersion = 10

type discordGatewayHello struct {
	HeartbeatInterval int `json:"heartbeat_interval"`
}

type discordGatewayEnvelope struct {
	Op int             `json:"op"`
	S  *int64          `json:"s,omitempty"`
	T  string          `json:"t,omitempty"`
	D  json.RawMessage `json:"d,omitempty"`
}

type discordGatewayIdentify struct {
	Token      string            `json:"token"`
	Properties map[string]string `json:"properties"`
	Intents    int64             `json:"intents"`
	Compress   bool              `json:"compress,omitempty"`
	Presence   map[string]any    `json:"presence,omitempty"`
	Shard      []int             `json:"shard,omitempty"`
	Large      bool              `json:"large,omitempty"`
	ResumeURL  string            `json:"resume_gateway_url,omitempty"`
}

type discordGatewayResume struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Seq       *int64 `json:"seq"`
}

type discordGatewayHeartbeat struct {
	Op   int    `json:"op"`
	Data *int64 `json:"d"`
}

type discordGatewayUser struct {
	ID         string `json:"id"`
	Username   string `json:"username,omitempty"`
	GlobalName string `json:"global_name,omitempty"`
	Bot        bool   `json:"bot,omitempty"`
}

type discordGatewayMember struct {
	Nick string              `json:"nick,omitempty"`
	User *discordGatewayUser `json:"user,omitempty"`
}

type discordGatewayAttachment struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type discordGatewayMessage struct {
	ID              string                     `json:"id"`
	ChannelID       string                     `json:"channel_id"`
	GuildID         string                     `json:"guild_id,omitempty"`
	Content         string                     `json:"content"`
	Type            int                        `json:"type"`
	Timestamp       string                     `json:"timestamp,omitempty"`
	EditedTimestamp string                     `json:"edited_timestamp,omitempty"`
	Author          *discordGatewayUser        `json:"author,omitempty"`
	Member          *discordGatewayMember      `json:"member,omitempty"`
	Attachments     []discordGatewayAttachment `json:"attachments,omitempty"`
}

type discordGatewayMessageFetchResult struct {
	ID              string                     `json:"id"`
	ChannelID       string                     `json:"channel_id"`
	GuildID         string                     `json:"guild_id,omitempty"`
	Content         string                     `json:"content"`
	Type            int                        `json:"type"`
	Timestamp       string                     `json:"timestamp,omitempty"`
	EditedTimestamp string                     `json:"edited_timestamp,omitempty"`
	Author          *discordGatewayUser        `json:"author,omitempty"`
	Member          *discordGatewayMember      `json:"member,omitempty"`
	Attachments     []discordGatewayAttachment `json:"attachments,omitempty"`
}

type discordGatewayLinkRecord struct {
	LocationID       string
	DiscordGuildID   string
	DiscordGuildName string
	Active           bool
}

type discordGatewayThreadRow struct {
	LocationID      string
	VenueSlug       string
	SessionID       string
	ShowingID       string
	DiscordServerID string
	ThreadID        string
	Status          string
}

type discordGatewayConn struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}

func RunDiscordGatewayWorker(ctx context.Context, pool *pgxpool.Pool, hub *Hub, cfg identity.DiscordServerLinkConfig, gatewayCfg identity.DiscordGatewayConfig) {
	location, err := loadDiscordGatewayLocation(ctx, pool)
	if err != nil {
		log.Printf("discord gateway location lookup failed: %v", err)
		return
	}
	log.Printf("discord gateway worker starting for location=%s", location.ID)

	backoff := time.Second
	for ctx.Err() == nil {
		runtimeLinkCfg, err := identity.ResolveDiscordServerLinkRuntimeConfig(ctx, pool, cfg)
		if err != nil {
			log.Printf("discord gateway runtime config lookup failed: %v", err)
			updateDiscordGatewayState(ctx, pool, location.ID, gatewayStateFromConfig(gatewayCfg, false, false, "", "", 0, err.Error(), nil, nil))
			sleepWithContext(ctx, backoff)
			backoff = nextBackoff(backoff)
			continue
		}

		effectiveGatewayCfg := gatewayCfg
		effectiveGatewayCfg.Enabled = runtimeLinkCfg.Enabled
		if strings.TrimSpace(runtimeLinkCfg.BotToken) != "" {
			effectiveGatewayCfg.BotToken = runtimeLinkCfg.BotToken
		}
		log.Printf("discord gateway runtime config enabled=%t token_set=%t intents=%d", effectiveGatewayCfg.Enabled, strings.TrimSpace(effectiveGatewayCfg.BotToken) != "", effectiveGatewayCfg.Intents)

		if !identity.DiscordGatewayConfigured(effectiveGatewayCfg) {
			log.Printf("discord gateway disabled after config resolution")
			updateDiscordGatewayState(ctx, pool, location.ID, gatewayStateFromConfig(effectiveGatewayCfg, false, false, "", "", 0, "discord gateway disabled", nil, nil))
			sleepWithContext(ctx, 10*time.Second)
			continue
		}

		linkRecord, err := loadDiscordGatewayLinkRecord(ctx, pool, location.ID)
		if err != nil {
			log.Printf("discord gateway link lookup failed: %v", err)
			updateDiscordGatewayState(ctx, pool, location.ID, gatewayStateFromConfig(effectiveGatewayCfg, false, false, "", "", 0, err.Error(), nil, nil))
			sleepWithContext(ctx, backoff)
			backoff = nextBackoff(backoff)
			continue
		}
		log.Printf("discord gateway link active=%t guild=%s", linkRecord.Active, strings.TrimSpace(linkRecord.DiscordGuildID))
		if !linkRecord.Active || strings.TrimSpace(linkRecord.DiscordGuildID) == "" {
			updateDiscordGatewayState(ctx, pool, location.ID, gatewayStateFromConfig(effectiveGatewayCfg, false, false, "", "", 0, "discord server not linked", nil, nil))
			sleepWithContext(ctx, 10*time.Second)
			continue
		}

		if err := runDiscordGatewayConnection(ctx, pool, hub, runtimeLinkCfg, effectiveGatewayCfg, location.ID, linkRecord); err != nil {
			log.Printf("discord gateway connection ended: %v", err)
			updateDiscordGatewayState(ctx, pool, location.ID, gatewayStateFromConfig(effectiveGatewayCfg, false, false, "", "", 0, err.Error(), nil, nil))
			sleepWithContext(ctx, backoff+time.Duration(rand.Int63n(int64(backoff/2)+1)))
			backoff = nextBackoff(backoff)
			continue
		}

		backoff = time.Second
	}
}

func loadDiscordGatewayLocation(ctx context.Context, pool *pgxpool.Pool) (struct {
	ID   string
	Slug string
	Name string
}, error) {
	var location struct {
		ID   string
		Slug string
		Name string
	}
	err := pool.QueryRow(ctx, `
		SELECT l.id::text, l.slug, l.name
		FROM venues v
		JOIN lots lo ON lo.id = v.lot_id
		JOIN locations l ON l.id = lo.location_id
		WHERE v.slug = 'producers-office'
		LIMIT 1
	`).Scan(&location.ID, &location.Slug, &location.Name)
	if err == nil {
		return location, nil
	}
	err = pool.QueryRow(ctx, `
		SELECT id::text, slug, name
		FROM locations
		WHERE slug = 'amurray-family'
		LIMIT 1
	`).Scan(&location.ID, &location.Slug, &location.Name)
	return location, err
}

func loadDiscordGatewayLinkRecord(ctx context.Context, pool *pgxpool.Pool, locationID string) (discordGatewayLinkRecord, error) {
	var row discordGatewayLinkRecord
	err := pool.QueryRow(ctx, `
		SELECT
			location_id::text,
			COALESCE(NULLIF(discord_guild_id, ''), ''),
			COALESCE(NULLIF(discord_guild_name, ''), ''),
			COALESCE(active, FALSE)
		FROM auth.discord_server_links
		WHERE location_id = $1::uuid
		LIMIT 1
	`, locationID).Scan(&row.LocationID, &row.DiscordGuildID, &row.DiscordGuildName, &row.Active)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return discordGatewayLinkRecord{}, nil
		}
		return discordGatewayLinkRecord{}, err
	}
	return row, nil
}

func runDiscordGatewayConnection(ctx context.Context, pool *pgxpool.Pool, hub *Hub, cfg identity.DiscordServerLinkConfig, gatewayCfg identity.DiscordGatewayConfig, locationID string, linkRecord discordGatewayLinkRecord) error {
	wsURL := strings.TrimSpace(gatewayCfg.GatewayURL)
	if wsURL == "" {
		wsURL = fmt.Sprintf("wss://gateway.discord.gg/?v=%d&encoding=json", discordGatewayVersion)
	}
	log.Printf("discord gateway dialing %s", wsURL)

	dialer := *websocket.DefaultDialer
	dialer.HandshakeTimeout = 15 * time.Second
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		log.Printf("discord gateway dial failed: %v", err)
		return err
	}
	log.Printf("discord gateway dial connected")
	defer conn.Close()

	client := &discordGatewayConn{conn: conn}
	connectedAt := time.Now().UTC()
	lastEventAt := connectedAt
	var seq *int64
	var sessionID string
	var botUserID string
	heartbeatInterval := 0 * time.Millisecond

	setState := func(connected bool, running bool, lastError string) {
		activeCount, _ := countActiveDiscordThreads(ctx, pool, locationID)
		_ = updateDiscordGatewayState(ctx, pool, locationID, gatewayStateFromConfig(gatewayCfg, connected, running, sessionID, botUserID, activeCount, lastError, &connectedAt, &lastEventAt))
	}

	setState(true, true, "")

	defer func() {
		setState(false, true, "")
	}()

	for {
		if err := conn.SetReadDeadline(time.Now().Add(90 * time.Second)); err != nil {
			return err
		}

		_, payload, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var envelope discordGatewayEnvelope
		if err := json.Unmarshal(payload, &envelope); err != nil {
			continue
		}
		if envelope.S != nil {
			seq = envelope.S
		}
		lastEventAt = time.Now().UTC()

		switch envelope.Op {
		case 10:
			var hello discordGatewayHello
			if err := json.Unmarshal(envelope.D, &hello); err != nil {
				return err
			}
			if hello.HeartbeatInterval <= 0 {
				return errors.New("discord gateway hello missing heartbeat interval")
			}
			log.Printf("discord gateway hello heartbeat_interval=%d", hello.HeartbeatInterval)
			heartbeatInterval = time.Duration(hello.HeartbeatInterval) * time.Millisecond
			if sessionID != "" && seq != nil {
				log.Printf("discord gateway resuming session=%s seq=%d", sessionID, *seq)
				if err := client.sendGatewayOp(6, discordGatewayResume{
					Token:     gatewayCfg.BotToken,
					SessionID: sessionID,
					Seq:       seq,
				}); err != nil {
					return err
				}
			} else {
				log.Printf("discord gateway identifying intents=%d token_set=%t", gatewayCfg.Intents, strings.TrimSpace(gatewayCfg.BotToken) != "")
				if err := client.sendGatewayOp(2, discordGatewayIdentify{
					Token:   gatewayCfg.BotToken,
					Intents: gatewayCfg.Intents,
					Properties: map[string]string{
						"os":      runtime.GOOS,
						"browser": "victory",
						"device":  "victory",
					},
				}); err != nil {
					return err
				}
			}

			stopHeartbeat := make(chan struct{})
			heartbeatErr := make(chan error, 1)
			go func(interval time.Duration) {
				ticker := time.NewTicker(interval)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						if err := client.sendJSON(discordGatewayHeartbeat{Op: 1, Data: seq}, nil); err != nil {
							select {
							case heartbeatErr <- err:
							default:
							}
							return
						}
					case <-stopHeartbeat:
						return
					case <-ctx.Done():
						return
					}
				}
			}(heartbeatInterval)

			defer close(stopHeartbeat)

			go func() {
				select {
				case err := <-heartbeatErr:
					_ = conn.Close()
					log.Printf("discord heartbeat failed: %v", err)
				case <-ctx.Done():
				}
			}()

		case 11:
			setState(true, true, "")

		case 7:
			return errors.New("reconnect requested")

		case 9:
			return errors.New("invalid session")

		case 0:
			switch envelope.T {
			case "READY":
				var ready struct {
					SessionID string `json:"session_id"`
					User      struct {
						ID string `json:"id"`
					} `json:"user"`
				}
				if err := json.Unmarshal(envelope.D, &ready); err != nil {
					return err
				}
				sessionID = strings.TrimSpace(ready.SessionID)
				botUserID = strings.TrimSpace(ready.User.ID)
				connectedAt = time.Now().UTC()
				setState(true, true, "")

			case "RESUMED":
				connectedAt = time.Now().UTC()
				setState(true, true, "")

			case "MESSAGE_CREATE":
				var msg discordGatewayMessage
				if err := json.Unmarshal(envelope.D, &msg); err != nil {
					log.Printf("discord gateway message decode failed: %v", err)
					continue
				}
				debugEnabled, _ := identity.LoadDiscordGatewayDebugEnabled(ctx, pool, locationID)
				if debugEnabled {
					log.Printf("discord gateway message create id=%s channel=%s guild=%s author=%t content_len=%d", strings.TrimSpace(msg.ID), strings.TrimSpace(msg.ChannelID), strings.TrimSpace(msg.GuildID), msg.Author != nil, len(strings.TrimSpace(msg.Content)))
				}
				lastEventAt = time.Now().UTC()
				if err := handleDiscordGatewayMessageCreate(ctx, pool, hub, cfg, locationID, linkRecord, gatewayCfg, botUserID, msg); err != nil {
					log.Printf("discord gateway import failed: %v", err)
				}
				setState(true, true, "")

			case "MESSAGE_UPDATE":
				var msg discordGatewayMessage
				if err := json.Unmarshal(envelope.D, &msg); err != nil {
					log.Printf("discord gateway update decode failed: %v", err)
					continue
				}
				debugEnabled, _ := identity.LoadDiscordGatewayDebugEnabled(ctx, pool, locationID)
				if debugEnabled {
					log.Printf("discord gateway message update id=%s channel=%s guild=%s author=%t content_len=%d", strings.TrimSpace(msg.ID), strings.TrimSpace(msg.ChannelID), strings.TrimSpace(msg.GuildID), msg.Author != nil, len(strings.TrimSpace(msg.Content)))
				}
				if strings.TrimSpace(msg.ID) != "" {
					if err := handleDiscordGatewayMessageUpdate(ctx, pool, hub, cfg, locationID, linkRecord, gatewayCfg, msg); err != nil {
						log.Printf("discord gateway edit handling failed: %v", err)
					}
				}
				setState(true, true, "")
			}
		}

		if heartbeatInterval > 0 && time.Since(lastEventAt) > 3*heartbeatInterval {
			return errors.New("heartbeat timeout")
		}
	}
}

func handleDiscordGatewayMessageCreate(ctx context.Context, pool *pgxpool.Pool, hub *Hub, cfg identity.DiscordServerLinkConfig, locationID string, linkRecord discordGatewayLinkRecord, gatewayCfg identity.DiscordGatewayConfig, botUserID string, msg discordGatewayMessage) error {
	debugEnabled, _ := identity.LoadDiscordGatewayDebugEnabled(ctx, pool, locationID)
	if strings.TrimSpace(msg.ID) == "" || strings.TrimSpace(msg.ChannelID) == "" {
		if debugEnabled {
			log.Printf("discord gateway message dropped reason=missing_ids id=%s channel=%s", strings.TrimSpace(msg.ID), strings.TrimSpace(msg.ChannelID))
		}
		return nil
	}
	if needsDiscordGatewayMessageBackfill(msg) {
		if fetched, err := fetchDiscordGatewayMessage(ctx, cfg, msg.ChannelID, msg.ID); err == nil {
			msg = mergeDiscordGatewayMessages(msg, fetched)
			if debugEnabled {
				log.Printf("discord gateway message backfilled id=%s channel=%s guild=%s author=%t content_len=%d", strings.TrimSpace(msg.ID), strings.TrimSpace(msg.ChannelID), strings.TrimSpace(msg.GuildID), msg.Author != nil, len(strings.TrimSpace(msg.Content)))
			}
		} else {
			if debugEnabled {
				log.Printf("discord gateway message backfill failed id=%s channel=%s err=%v", strings.TrimSpace(msg.ID), strings.TrimSpace(msg.ChannelID), err)
			}
		}
	}
	if msg.Author != nil && (msg.Author.Bot || strings.EqualFold(strings.TrimSpace(msg.Author.ID), strings.TrimSpace(botUserID))) {
		if debugEnabled {
			log.Printf("discord gateway message dropped reason=bot_or_self id=%s author=%s bot=%t bot_user=%s", strings.TrimSpace(msg.ID), strings.TrimSpace(msg.Author.ID), msg.Author.Bot, strings.TrimSpace(botUserID))
		}
		return nil
	}
	if strings.TrimSpace(msg.Content) == "" && len(msg.Attachments) == 0 {
		if debugEnabled {
			log.Printf("discord gateway message dropped reason=empty_content id=%s channel=%s", strings.TrimSpace(msg.ID), strings.TrimSpace(msg.ChannelID))
		}
		return nil
	}
	if !strings.EqualFold(strings.TrimSpace(msg.GuildID), strings.TrimSpace(linkRecord.DiscordGuildID)) {
		if debugEnabled {
			log.Printf("discord gateway message dropped reason=guild_mismatch id=%s guild=%s expected=%s", strings.TrimSpace(msg.ID), strings.TrimSpace(msg.GuildID), strings.TrimSpace(linkRecord.DiscordGuildID))
		}
		return nil
	}

	threadRow, err := loadActiveDiscordThreadByID(ctx, pool, locationID, msg.ChannelID)
	if err != nil {
		return err
	}
	if threadRow == nil {
		if debugEnabled {
			log.Printf("discord gateway message dropped reason=thread_not_found id=%s channel=%s location=%s", strings.TrimSpace(msg.ID), strings.TrimSpace(msg.ChannelID), strings.TrimSpace(locationID))
		}
		return nil
	}

	duplicate, err := discordGatewayMessageAlreadyHandled(ctx, pool, msg.ID)
	if err != nil {
		return err
	}
	if duplicate {
		return nil
	}

	authorID := ""
	authorUsername := ""
	authorGlobalName := ""
	if msg.Author != nil {
		authorID = strings.TrimSpace(msg.Author.ID)
		authorUsername = strings.TrimSpace(msg.Author.Username)
		authorGlobalName = strings.TrimSpace(msg.Author.GlobalName)
	}
	if authorID == "" && msg.Member != nil && msg.Member.User != nil {
		authorID = strings.TrimSpace(msg.Member.User.ID)
		if authorUsername == "" {
			authorUsername = strings.TrimSpace(msg.Member.User.Username)
		}
		if authorGlobalName == "" {
			authorGlobalName = strings.TrimSpace(msg.Member.Nick)
		}
	}
	if authorID == "" {
		if debugEnabled {
			log.Printf("discord gateway message dropped reason=missing_author id=%s channel=%s", strings.TrimSpace(msg.ID), strings.TrimSpace(msg.ChannelID))
		}
		return nil
	}

	claimed, err := claimDiscordGatewayImport(ctx, pool, discordGatewayImportRow{
		LocationID:              threadRow.LocationID,
		VenueSlug:               threadRow.VenueSlug,
		SessionID:               threadRow.SessionID,
		ShowingID:               threadRow.ShowingID,
		DiscordServerID:         threadRow.DiscordServerID,
		DiscordThreadID:         threadRow.ThreadID,
		DiscordChannelID:        msg.ChannelID,
		DiscordMessageID:        msg.ID,
		DiscordAuthorID:         authorID,
		DiscordAuthorUsername:   authorUsername,
		DiscordAuthorGlobalName: authorGlobalName,
		MessageCreatedAt:        parseDiscordTimestamp(msg.Timestamp),
	})
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}

	actorID, actorDisplayName, actorHandle, actorRole, actorPersona, linkedUserID, linkedUserSessionMissing, err := resolveDiscordGatewayActor(ctx, pool, threadRow.SessionID, authorID, authorUsername, authorGlobalName, debugEnabled)
	if err != nil {
		_ = updateDiscordGatewayImportStatus(ctx, pool, msg.ID, "failed", "", "")
		return err
	}

	text := strings.TrimSpace(msg.Content)
	if len(msg.Attachments) > 0 && text == "" {
		_ = updateDiscordGatewayImportStatus(ctx, pool, msg.ID, "ignored", "", "")
		return nil
	}
	if strings.TrimSpace(text) == "" {
		_ = updateDiscordGatewayImportStatus(ctx, pool, msg.ID, "ignored", "", "")
		return nil
	}

	if linkedUserID == "" && !linkedUserSessionMissing {
		allowed, err := canImportUnlinkedDiscordMessage(ctx, pool, threadRow.SessionID)
		if err != nil {
			_ = updateDiscordGatewayImportStatus(ctx, pool, msg.ID, "failed", "", "")
			return err
		}
		if !allowed {
			_ = updateDiscordGatewayImportStatus(ctx, pool, msg.ID, "ignored", "", "")
			return nil
		}
	}

	storedAction, err := actions.StoreDiscordChatMessage(ctx, pool, actions.DiscordChatMessageRequest{
		SessionID:        threadRow.SessionID,
		ActorID:          actorID,
		ActorDisplayName: actorDisplayName,
		ActorHandle:      actorHandle,
		ActorRole:        actorRole,
		ActorPersona:     actorPersona,
		Text:             text,
		Discord: actions.DiscordChatMetadata{
			MessageID:        msg.ID,
			ThreadID:         msg.ChannelID,
			ChannelID:        msg.ChannelID,
			ServerID:         threadRow.DiscordServerID,
			AuthorID:         authorID,
			AuthorUsername:   authorUsername,
			AuthorGlobalName: authorGlobalName,
			LinkedUserID:     linkedUserID,
		},
	})
	if err != nil {
		_ = updateDiscordGatewayImportStatus(ctx, pool, msg.ID, "failed", "", "")
		return err
	}

	if err := updateDiscordGatewayImportStatus(ctx, pool, msg.ID, "imported", storedAction.ID, linkedUserID); err != nil {
		return err
	}

	msgOut, _ := json.Marshal(map[string]any{
		"type": "action",
		"data": storedAction,
	})
	hub.Broadcast(msgOut)

	return nil
}

func handleDiscordGatewayMessageUpdate(ctx context.Context, pool *pgxpool.Pool, hub *Hub, cfg identity.DiscordServerLinkConfig, locationID string, linkRecord discordGatewayLinkRecord, gatewayCfg identity.DiscordGatewayConfig, msg discordGatewayMessage) error {
	debugEnabled, _ := identity.LoadDiscordGatewayDebugEnabled(ctx, pool, locationID)
	if strings.TrimSpace(msg.ID) == "" || strings.TrimSpace(msg.ChannelID) == "" {
		return nil
	}
	if needsDiscordGatewayMessageBackfill(msg) {
		if fetched, err := fetchDiscordGatewayMessage(ctx, cfg, msg.ChannelID, msg.ID); err == nil {
			msg = mergeDiscordGatewayMessages(msg, fetched)
			if debugEnabled {
				log.Printf("discord gateway update backfilled id=%s channel=%s guild=%s author=%t content_len=%d", strings.TrimSpace(msg.ID), strings.TrimSpace(msg.ChannelID), strings.TrimSpace(msg.GuildID), msg.Author != nil, len(strings.TrimSpace(msg.Content)))
			}
		} else {
			if debugEnabled {
				log.Printf("discord gateway update backfill failed id=%s channel=%s err=%v", strings.TrimSpace(msg.ID), strings.TrimSpace(msg.ChannelID), err)
			}
		}
	}
	if !strings.EqualFold(strings.TrimSpace(msg.GuildID), strings.TrimSpace(linkRecord.DiscordGuildID)) {
		return nil
	}

	threadRow, err := loadActiveDiscordThreadByID(ctx, pool, locationID, msg.ChannelID)
	if err != nil {
		return err
	}
	if threadRow == nil {
		return nil
	}

	importRow, err := loadDiscordGatewayImportByMessageID(ctx, pool, msg.ID)
	if err != nil {
		return err
	}
	if importRow == nil || strings.TrimSpace(importRow.ActionID) == "" {
		return nil
	}

	editedAt := parseDiscordTimestamp(msg.EditedTimestamp)
	if err := updateDiscordGatewayImportEditedAt(ctx, pool, msg.ID, editedAt); err != nil {
		return err
	}

	text := strings.TrimSpace(msg.Content)
	if text == "" {
		return nil
	}

	originalText, err := loadDiscordGatewayActionText(ctx, pool, importRow.ActionID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(originalText) == text {
		return nil
	}

	actorID, actorDisplayName, actorHandle, actorRole, actorPersona, linkedUserID, err := resolveDiscordGatewayActorFromImport(ctx, pool, threadRow.SessionID, importRow)
	if err != nil {
		return err
	}

	storedAction, err := actions.StoreDiscordChatMessage(ctx, pool, actions.DiscordChatMessageRequest{
		SessionID:        threadRow.SessionID,
		ActorID:          actorID,
		ActorDisplayName: actorDisplayName,
		ActorHandle:      actorHandle,
		ActorRole:        actorRole,
		ActorPersona:     actorPersona,
		Text:             text,
		Discord: actions.DiscordChatMetadata{
			MessageID:        msg.ID,
			ThreadID:         msg.ChannelID,
			ChannelID:        msg.ChannelID,
			ServerID:         threadRow.DiscordServerID,
			AuthorID:         importRow.DiscordAuthorID,
			AuthorUsername:   importRow.DiscordAuthorUsername,
			AuthorGlobalName: importRow.DiscordAuthorGlobalName,
			LinkedUserID:     linkedUserID,
			Edited:           true,
			EditOfMessageID:  msg.ID,
		},
		EditedAt: editedAt,
	})
	if err != nil {
		return err
	}

	if err := updateDiscordGatewayImportEditAction(ctx, pool, msg.ID, storedAction.ID, linkedUserID, editedAt); err != nil {
		return err
	}

	msgOut, _ := json.Marshal(map[string]any{
		"type": "action",
		"data": storedAction,
	})
	hub.Broadcast(msgOut)
	return nil
}

func resolveDiscordGatewayActor(ctx context.Context, pool *pgxpool.Pool, sessionID, discordAuthorID, authorUsername, authorGlobalName string, debugEnabled bool) (actorID, actorDisplayName, actorHandle, actorRole string, actorPersona any, linkedUserID string, linkedUserSessionMissing bool, err error) {
	linkedUser, err := identity.ResolveBootstrapUserByDiscordID(ctx, pool, discordAuthorID)
	if err == nil {
		identityRow, err := identity.ResolveSessionIdentity(ctx, pool, sessionID, linkedUser.ID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				if debugEnabled {
					log.Printf("discord gateway linked user not in session; falling back to bridge actor user=%s session=%s discord=%s", linkedUser.ID, strings.TrimSpace(sessionID), strings.TrimSpace(discordAuthorID))
				}
				linkedUserSessionMissing = true
				goto bridgeFallback
			}
			return "", "", "", "", nil, linkedUser.ID, false, err
		}
		return linkedUser.ID, identityRow.DisplayName, identityRow.Handle, identityRow.Role, identityRow.Persona, linkedUser.ID, false, nil
	}
	if !errors.Is(err, identity.ErrBootstrapUserNotFound) {
		return "", "", "", "", nil, "", false, err
	}

bridgeFallback:
	actorID, err = ensureDiscordBridgeUser(ctx, pool)
	if err != nil {
		return "", "", "", "", nil, "", linkedUserSessionMissing, err
	}
	if strings.TrimSpace(authorGlobalName) != "" {
		actorDisplayName = authorGlobalName
	} else if strings.TrimSpace(authorUsername) != "" {
		actorDisplayName = authorUsername
	} else {
		actorDisplayName = "Discord"
	}
	if strings.TrimSpace(authorUsername) != "" {
		actorHandle = authorUsername
	} else {
		actorHandle = "discord"
	}
	actorRole = "audience"
	return actorID, actorDisplayName, actorHandle, actorRole, nil, "", linkedUserSessionMissing, nil
}

func resolveDiscordGatewayActorFromImport(ctx context.Context, pool *pgxpool.Pool, sessionID string, row *discordGatewayImportRecord) (actorID, actorDisplayName, actorHandle, actorRole string, actorPersona any, linkedUserID string, err error) {
	linkedUserID = strings.TrimSpace(row.LinkedUserID)
	if linkedUserID != "" {
		identityRow, err := identity.ResolveSessionIdentity(ctx, pool, sessionID, linkedUserID)
		if err != nil {
			return "", "", "", "", nil, "", err
		}
		return linkedUserID, identityRow.DisplayName, identityRow.Handle, identityRow.Role, identityRow.Persona, linkedUserID, nil
	}

	actorID, err = ensureDiscordBridgeUser(ctx, pool)
	if err != nil {
		return "", "", "", "", nil, "", err
	}
	if strings.TrimSpace(row.DiscordAuthorGlobalName) != "" {
		actorDisplayName = row.DiscordAuthorGlobalName
	} else if strings.TrimSpace(row.DiscordAuthorUsername) != "" {
		actorDisplayName = row.DiscordAuthorUsername
	} else {
		actorDisplayName = "Discord"
	}
	if strings.TrimSpace(row.DiscordAuthorUsername) != "" {
		actorHandle = row.DiscordAuthorUsername
	} else {
		actorHandle = "discord"
	}
	actorRole = "audience"
	return actorID, actorDisplayName, actorHandle, actorRole, nil, "", nil
}

type discordGatewayImportRow struct {
	LocationID              string
	VenueSlug               string
	SessionID               string
	ShowingID               string
	DiscordServerID         string
	DiscordThreadID         string
	DiscordChannelID        string
	DiscordMessageID        string
	DiscordAuthorID         string
	DiscordAuthorUsername   string
	DiscordAuthorGlobalName string
	MessageCreatedAt        *time.Time
}

type discordGatewayImportRecord struct {
	ID                      string
	ActionID                string
	EditActionID            string
	SessionID               string
	ShowingID               string
	VenueSlug               string
	DiscordServerID         string
	DiscordThreadID         string
	DiscordChannelID        string
	DiscordMessageID        string
	DiscordAuthorID         string
	DiscordAuthorUsername   string
	DiscordAuthorGlobalName string
	LinkedUserID            string
	ImportStatus            string
	MessageCreatedAt        *time.Time
	MessageEditedAt         *time.Time
}

func claimDiscordGatewayImport(ctx context.Context, pool *pgxpool.Pool, row discordGatewayImportRow) (bool, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO auth.discord_chat_imports (
			location_id,
			venue_slug,
			session_id,
			showing_id,
			discord_server_id,
			discord_thread_id,
			discord_channel_id,
			discord_message_id,
			discord_author_id,
			discord_author_username,
			discord_author_global_name,
			message_created_at,
			import_status,
			imported_at,
			created_at,
			updated_at
		)
		VALUES (
			$1::uuid,
			$2,
			NULLIF($3, '')::uuid,
			NULLIF($4, '')::uuid,
			$5,
			$6,
			NULLIF($7, ''),
			$8,
			$9,
			NULLIF($10, ''),
			NULLIF($11, ''),
			$12,
			'pending',
			NOW(),
			NOW(),
			NOW()
		)
		ON CONFLICT (discord_message_id) DO NOTHING
		RETURNING id::text
	`, row.LocationID, row.VenueSlug, row.SessionID, row.ShowingID, row.DiscordServerID, row.DiscordThreadID, row.DiscordChannelID, row.DiscordMessageID, row.DiscordAuthorID, row.DiscordAuthorUsername, row.DiscordAuthorGlobalName, row.MessageCreatedAt).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return strings.TrimSpace(id) != "", nil
}

func loadDiscordGatewayImportByMessageID(ctx context.Context, pool *pgxpool.Pool, messageID string) (*discordGatewayImportRecord, error) {
	var row discordGatewayImportRecord
	err := pool.QueryRow(ctx, `
		SELECT
			id::text,
			COALESCE(action_id::text, ''),
			COALESCE(edit_action_id::text, ''),
			COALESCE(session_id::text, ''),
			COALESCE(showing_id::text, ''),
			venue_slug,
			discord_server_id,
			discord_thread_id,
			COALESCE(NULLIF(discord_channel_id, ''), ''),
			discord_message_id,
			discord_author_id,
			COALESCE(NULLIF(discord_author_username, ''), ''),
			COALESCE(NULLIF(discord_author_global_name, ''), ''),
			COALESCE(linked_user_id::text, ''),
			import_status,
			message_created_at,
			message_edited_at
		FROM auth.discord_chat_imports
		WHERE discord_message_id = $1
		LIMIT 1
	`, messageID).Scan(
		&row.ID,
		&row.ActionID,
		&row.EditActionID,
		&row.SessionID,
		&row.ShowingID,
		&row.VenueSlug,
		&row.DiscordServerID,
		&row.DiscordThreadID,
		&row.DiscordChannelID,
		&row.DiscordMessageID,
		&row.DiscordAuthorID,
		&row.DiscordAuthorUsername,
		&row.DiscordAuthorGlobalName,
		&row.LinkedUserID,
		&row.ImportStatus,
		&row.MessageCreatedAt,
		&row.MessageEditedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func loadDiscordGatewayActionText(ctx context.Context, pool *pgxpool.Pool, actionID string) (string, error) {
	var text string
	err := pool.QueryRow(ctx, `
		SELECT COALESCE(payload ->> 'text', '')
		FROM actions
		WHERE id = $1::uuid
		LIMIT 1
	`, actionID).Scan(&text)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func updateDiscordGatewayImportStatus(ctx context.Context, pool *pgxpool.Pool, messageID, status, actionID, linkedUserID string) error {
	_, err := pool.Exec(ctx, `
		UPDATE auth.discord_chat_imports
		SET import_status = $2,
			action_id = NULLIF($3, '')::uuid,
			linked_user_id = NULLIF($4, '')::uuid,
			updated_at = NOW()
		WHERE discord_message_id = $1
	`, messageID, status, actionID, linkedUserID)
	return err
}

func updateDiscordGatewayImportEditedAt(ctx context.Context, pool *pgxpool.Pool, messageID string, editedAt *time.Time) error {
	_, err := pool.Exec(ctx, `
		UPDATE auth.discord_chat_imports
		SET message_edited_at = COALESCE($2, message_edited_at),
			updated_at = NOW()
		WHERE discord_message_id = $1
	`, messageID, editedAt)
	return err
}

func updateDiscordGatewayImportEditAction(ctx context.Context, pool *pgxpool.Pool, messageID, editActionID, linkedUserID string, editedAt *time.Time) error {
	_, err := pool.Exec(ctx, `
		UPDATE auth.discord_chat_imports
		SET import_status = 'edited',
			edit_action_id = NULLIF($2, '')::uuid,
			linked_user_id = NULLIF($3, '')::uuid,
			message_edited_at = COALESCE($4, message_edited_at),
			updated_at = NOW()
		WHERE discord_message_id = $1
	`, messageID, editActionID, linkedUserID, editedAt)
	return err
}

func discordGatewayMessageAlreadyHandled(ctx context.Context, pool *pgxpool.Pool, messageID string) (bool, error) {
	var handled bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM auth.discord_chat_imports WHERE discord_message_id = $1
		) OR EXISTS (
			SELECT 1 FROM auth.discord_chat_message_bridges WHERE discord_message_id = $1
		)
	`, messageID).Scan(&handled)
	if err != nil {
		return false, err
	}
	return handled, nil
}

func loadActiveDiscordThreadByID(ctx context.Context, pool *pgxpool.Pool, locationID, threadID string) (*discordGatewayThreadRow, error) {
	var row discordGatewayThreadRow
	err := pool.QueryRow(ctx, `
		SELECT
			location_id::text,
			venue_slug,
			COALESCE(session_id::text, ''),
			COALESCE(showing_id::text, ''),
			discord_server_id,
			thread_id,
			status
		FROM auth.discord_session_threads
		WHERE thread_id = $1::text
		  AND status = 'active'
		LIMIT 1
	`, threadID).Scan(&row.LocationID, &row.VenueSlug, &row.SessionID, &row.ShowingID, &row.DiscordServerID, &row.ThreadID, &row.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func canImportUnlinkedDiscordMessage(ctx context.Context, pool *pgxpool.Pool, sessionID string) (bool, error) {
	var chatEnabled bool
	var showingStatus string
	err := pool.QueryRow(ctx, `
		SELECT
			COALESCE((v.config ->> 'chat_enabled')::boolean, FALSE),
			COALESCE(sh.status::text, '')
		FROM sessions s
		JOIN venues v ON v.id = s.venue_id
		LEFT JOIN showings sh ON sh.session_id = s.id
		WHERE s.id = $1::uuid
		LIMIT 1
	`, sessionID).Scan(&chatEnabled, &showingStatus)
	if err != nil {
		return false, err
	}
	if !chatEnabled {
		return false, nil
	}
	if strings.EqualFold(strings.TrimSpace(showingStatus), "closed") {
		return false, nil
	}
	return true, nil
}

func ensureDiscordBridgeUser(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO users (handle, display_name)
			SELECT 'discord_bridge', 'Discord Bridge'
			WHERE NOT EXISTS (
				SELECT 1 FROM users WHERE handle = 'discord_bridge'
			)
			RETURNING id::text
		)
		SELECT id::text
		FROM users
		WHERE handle = 'discord_bridge'
		LIMIT 1
	`).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func parseDiscordTimestamp(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	ts, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		ts, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil
		}
	}
	utc := ts.UTC()
	return &utc
}

func needsDiscordGatewayMessageBackfill(msg discordGatewayMessage) bool {
	if strings.TrimSpace(msg.Content) == "" {
		return true
	}
	if strings.TrimSpace(msg.GuildID) == "" {
		return true
	}
	if msg.Author == nil {
		return true
	}
	return false
}

func mergeDiscordGatewayMessages(current discordGatewayMessage, fetched discordGatewayMessageFetchResult) discordGatewayMessage {
	if strings.TrimSpace(current.Content) == "" {
		current.Content = fetched.Content
	}
	if strings.TrimSpace(current.GuildID) == "" {
		current.GuildID = fetched.GuildID
	}
	if current.Author == nil {
		current.Author = fetched.Author
	}
	if current.Member == nil {
		current.Member = fetched.Member
	}
	if len(current.Attachments) == 0 && len(fetched.Attachments) > 0 {
		current.Attachments = fetched.Attachments
	}
	if current.Timestamp == "" {
		current.Timestamp = fetched.Timestamp
	}
	if current.EditedTimestamp == "" {
		current.EditedTimestamp = fetched.EditedTimestamp
	}
	if current.Type == 0 {
		current.Type = fetched.Type
	}
	if strings.TrimSpace(current.ChannelID) == "" {
		current.ChannelID = fetched.ChannelID
	}
	if strings.TrimSpace(current.ID) == "" {
		current.ID = fetched.ID
	}
	return current
}

func fetchDiscordGatewayMessage(ctx context.Context, cfg identity.DiscordServerLinkConfig, channelID, messageID string) (discordGatewayMessageFetchResult, error) {
	var out discordGatewayMessageFetchResult
	if err := identity.DiscordServerLinkRequest(ctx, cfg, http.MethodGet, "/channels/"+urlPathEscape(strings.TrimSpace(channelID))+"/messages/"+urlPathEscape(strings.TrimSpace(messageID)), nil, &out); err != nil {
		return discordGatewayMessageFetchResult{}, err
	}
	return out, nil
}

func gatewayStateFromConfig(cfg identity.DiscordGatewayConfig, connected, running bool, sessionID, botUserID string, activeThreadCount int, lastError string, lastConnectedAt, lastEventAt *time.Time) identity.DiscordGatewayStateRow {
	return identity.DiscordGatewayStateRow{
		Enabled:              cfg.Enabled,
		Configured:           identity.DiscordGatewayConfigured(cfg),
		Running:              running,
		Connected:            connected,
		SessionID:            sessionID,
		BotUserID:            botUserID,
		Intents:              cfg.Intents,
		MessageContentIntent: cfg.Intents&(1<<15) != 0,
		ActiveThreadCount:    activeThreadCount,
		LastConnectedAt:      lastConnectedAt,
		LastEventAt:          lastEventAt,
		LastError:            lastError,
	}
}

func updateDiscordGatewayState(ctx context.Context, pool *pgxpool.Pool, locationID string, row identity.DiscordGatewayStateRow) error {
	row.LocationID = locationID
	return identity.UpsertDiscordGatewayState(ctx, pool, row)
}

func nextBackoff(current time.Duration) time.Duration {
	if current <= 0 {
		return time.Second
	}
	next := current * 2
	if next > 60*time.Second {
		return 60 * time.Second
	}
	return next
}

func sleepWithContext(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
	case <-ctx.Done():
	}
}

func (c *discordGatewayConn) sendJSON(v any, _ *time.Time) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteJSON(v)
}

func (c *discordGatewayConn) sendGatewayOp(op int, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.sendJSON(discordGatewayEnvelope{Op: op, D: raw}, nil)
}

func countActiveDiscordThreads(ctx context.Context, pool *pgxpool.Pool, locationID string) (int, error) {
	var count int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM auth.discord_session_threads
		WHERE location_id = $1::uuid
		  AND status = 'active'
	`, locationID).Scan(&count)
	return count, err
}
