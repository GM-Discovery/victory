package identity

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"victory/backend/internal/access"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	discordInteractionTypePing                                     = 1
	discordInteractionTypeApplicationCommand                       = 2
	discordInteractionResponseTypePong                             = 1
	discordInteractionResponseTypeChannelReply                     = 4
	discordInteractionResponseTypeDeferredChannelMessageWithSource = 5
	discordChannelTypePrivateThread                                = 12

	discordMicCommandName = "mic"
	discordMicCommandDesc = "Control the Victory mic thread for this venue."
)

var discordMicVenues = map[string]string{
	"the-cave":            "The Cave",
	"first-theater":       "First Theater",
	"middle-school-stage": "Middle School Stage",
}

type discordInteractionEnvelope struct {
	Type        int                       `json:"type"`
	ID          string                    `json:"id"`
	GuildID     string                    `json:"guild_id"`
	ChannelID   string                    `json:"channel_id"`
	Token       string                    `json:"token"`
	Application string                    `json:"application_id"`
	Data        *discordInteractionData   `json:"data,omitempty"`
	Member      *discordInteractionMember `json:"member,omitempty"`
	User        *discordInteractionUser   `json:"user,omitempty"`
}

type discordInteractionData struct {
	Name    string                     `json:"name"`
	Options []discordInteractionOption `json:"options,omitempty"`
}

type discordInteractionOption struct {
	Name    string                     `json:"name"`
	Type    int                        `json:"type"`
	Value   any                        `json:"value,omitempty"`
	Options []discordInteractionOption `json:"options,omitempty"`
}

type discordInteractionMember struct {
	User *discordInteractionUser `json:"user,omitempty"`
}

type discordInteractionUser struct {
	ID       string `json:"id"`
	Username string `json:"username,omitempty"`
}

type discordInteractionResponse struct {
	Type int                             `json:"type"`
	Data *discordInteractionResponseData `json:"data,omitempty"`
}

type discordInteractionResponseData struct {
	Content string `json:"content,omitempty"`
	Flags   int    `json:"flags,omitempty"`
}

type discordMicCommand struct {
	Name                     string                    `json:"name"`
	Description              string                    `json:"description"`
	Type                     int                       `json:"type,omitempty"`
	DefaultMemberPermissions string                    `json:"default_member_permissions,omitempty"`
	DMPermission             *bool                     `json:"dm_permission,omitempty"`
	Options                  []discordMicCommandOption `json:"options,omitempty"`
}

type discordMicCommandOption struct {
	Type        int                       `json:"type"`
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Required    bool                      `json:"required,omitempty"`
	Options     []discordMicCommandOption `json:"options,omitempty"`
}

type discordMicThreadRow struct {
	ID                     string
	LocationID             string
	VenueID                string
	VenueSlug              string
	VenueName              string
	SessionID              string
	ShowingID              string
	DiscordServerID        string
	ParentChannelID        string
	ThreadID               string
	ThreadName             string
	StartedByUserID        string
	StartedByDiscordUserID string
	StartedAt              time.Time
	ShowtimeAt             time.Time
	EndedAt                *time.Time
	Status                 string
}

type discordMicStatusResponse struct {
	VenueSlug                      string                         `json:"venue_slug"`
	VenueName                      string                         `json:"venue_name,omitempty"`
	Linked                         bool                           `json:"linked"`
	PublicKeyConfigured            bool                           `json:"public_key_configured"`
	InteractionsEndpointConfigured bool                           `json:"interactions_endpoint_configured"`
	CommandRegistered              bool                           `json:"command_registered"`
	CommandName                    string                         `json:"command_name,omitempty"`
	CanControl                     bool                           `json:"can_control"`
	CanManage                      bool                           `json:"can_manage"`
	ParentChannel                  *DiscordChannelMappingItem     `json:"parent_channel,omitempty"`
	Mic                            *discordMicThreadStateResponse `json:"mic,omitempty"`
	CommandStatus                  string                         `json:"command_status,omitempty"`
	CommandError                   string                         `json:"command_error,omitempty"`
}

type discordMicControlRequest struct {
	VenueSlug string `json:"venue_slug"`
	Command   string `json:"command"`
}

type discordMicControlResponse struct {
	VenueSlug string `json:"venue_slug"`
	Command   string `json:"command"`
	Message   string `json:"message"`
}

type discordMicThreadStateResponse struct {
	Active     bool   `json:"active"`
	ThreadID   string `json:"thread_id,omitempty"`
	ThreadName string `json:"thread_name,omitempty"`
	ThreadURL  string `json:"thread_url,omitempty"`
	StartedAt  string `json:"started_at,omitempty"`
	ShowtimeAt string `json:"showtime_at,omitempty"`
	ParentID   string `json:"parent_channel_id,omitempty"`
}

func HandleDiscordMicStatus(pool *pgxpool.Pool, cfg DiscordServerLinkConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		venueSlug := normalizeMicVenueSlug(r.URL.Query().Get("venue_slug"))
		if venueSlug == "" {
			venueSlug = "the-cave"
		}
		venueName, ok := discordMicVenues[venueSlug]
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "venue_not_mic_enabled"})
			return
		}

		location, err := resolveProducerOfficeLocation(ctx, pool)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "location_lookup_failed"})
			return
		}

		runtimeCfg, err := resolveDiscordServerLinkRuntimeConfig(ctx, pool, cfg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "config_lookup_failed"})
			return
		}

		linkRecord, err := loadDiscordServerLinkRecord(ctx, pool, location.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "link_lookup_failed"})
			return
		}

		canControl, err := discordMicCanControl(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed"})
			return
		}
		canManage, err := access.IsOperatorUser(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed"})
			return
		}

		commandRegistered, commandStatus, commandError := false, "unavailable", ""
		if linkRecord.Active && strings.TrimSpace(linkRecord.DiscordGuildID) != "" && DiscordMicCommandConfigured(runtimeCfg) {
			commandRegistered, commandStatus, commandError = discordMicCommandStatus(ctx, runtimeCfg, linkRecord.DiscordGuildID)
		}

		parentItem, err := discordMicParentChannelItem(ctx, pool, location.ID, venueSlug)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "mapping_lookup_failed"})
			return
		}

		row, err := loadDiscordMicThread(ctx, pool, location.ID, venueSlug)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "thread_lookup_failed"})
			return
		}

		status := discordMicStatusResponse{
			VenueSlug:                      venueSlug,
			VenueName:                      venueName,
			Linked:                         linkRecord.Active && strings.TrimSpace(linkRecord.DiscordGuildID) != "",
			PublicKeyConfigured:            strings.TrimSpace(runtimeCfg.PublicKey) != "",
			InteractionsEndpointConfigured: strings.TrimSpace(runtimeCfg.PublicKey) != "",
			CommandRegistered:              commandRegistered,
			CommandName:                    discordMicCommandName,
			CanControl:                     canControl,
			CanManage:                      canManage,
			CommandStatus:                  commandStatus,
			CommandError:                   commandError,
		}
		if parentItem != nil {
			status.ParentChannel = parentItem
		}
		if row != nil {
			status.Mic = &discordMicThreadStateResponse{
				Active:     strings.EqualFold(strings.TrimSpace(row.Status), "active"),
				ThreadID:   row.ThreadID,
				ThreadName: row.ThreadName,
				ThreadURL:  discordMicThreadURL(row.DiscordServerID, row.ThreadID),
				StartedAt:  row.StartedAt.UTC().Format(time.RFC3339),
				ShowtimeAt: row.ShowtimeAt.UTC().Format(time.RFC3339),
				ParentID:   row.ParentChannelID,
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": status})
	}
}

func HandleDiscordMicRegister(pool *pgxpool.Pool, cfg DiscordServerLinkConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		allowed, err := access.IsOperatorUser(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed"})
			return
		}
		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		location, err := resolveProducerOfficeLocation(ctx, pool)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "location_lookup_failed"})
			return
		}

		runtimeCfg, err := resolveDiscordServerLinkRuntimeConfig(ctx, pool, cfg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "config_lookup_failed"})
			return
		}
		if !DiscordMicCommandConfigured(runtimeCfg) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "discord_mic_unavailable"})
			return
		}

		linkRecord, err := loadDiscordServerLinkRecord(ctx, pool, location.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "link_lookup_failed"})
			return
		}
		if !linkRecord.Active || strings.TrimSpace(linkRecord.DiscordGuildID) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "discord_server_not_linked"})
			return
		}

		registered, commandID, err := ensureDiscordMicCommand(ctx, runtimeCfg, linkRecord.DiscordGuildID)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": "mic_command_register_failed", "detail": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"data": map[string]any{
				"command_name":     discordMicCommandName,
				"command_id":       commandID,
				"registered":       registered,
				"registered_at":    time.Now().UTC().Format(time.RFC3339),
				"guild_id":         linkRecord.DiscordGuildID,
				"interactions_url": "/api/discord/interactions",
			},
		})
	}
}

func HandleDiscordMicControl(pool *pgxpool.Pool, cfg DiscordServerLinkConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		var req discordMicControlRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_json"})
			return
		}

		venueSlug := normalizeMicVenueSlug(req.VenueSlug)
		if venueSlug == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "venue_not_mic_enabled"})
			return
		}
		venueName, ok := discordMicVenues[venueSlug]
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "venue_not_mic_enabled"})
			return
		}

		canControl, err := discordMicCanControl(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed"})
			return
		}
		if !canControl {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		location, err := resolveProducerOfficeLocation(ctx, pool)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "location_lookup_failed"})
			return
		}

		command := normalizeMicCommand(req.Command)
		if command == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "mic_command_required"})
			return
		}

		row, err := loadDiscordMicThread(ctx, pool, location.ID, venueSlug)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "thread_lookup_failed"})
			return
		}

		runtimeCfg, runtimeErr := resolveDiscordServerLinkRuntimeConfig(ctx, pool, cfg)
		linkRecord, linkErr := loadDiscordServerLinkRecord(ctx, pool, location.ID)
		hasDiscordConfig := runtimeErr == nil && DiscordMicCommandConfigured(runtimeCfg) && linkErr == nil && linkRecord.Active && strings.TrimSpace(linkRecord.DiscordGuildID) != ""

		var message string
		switch command {
		case "hot", "on", "start":
			if !hasDiscordConfig {
				writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "discord_mic_unavailable"})
				return
			}
			parentRow, err := discordMicParentChannelItem(ctx, pool, location.ID, venueSlug)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "parent_channel_missing"})
				return
			}
			message, err = discordMicTurnOn(ctx, pool, runtimeCfg, location.ID, linkRecord.DiscordGuildID, venueSlug, venueName, parentRow, userID, "")
		case "off":
			if hasDiscordConfig {
				message, err = discordMicTurnOff(ctx, pool, venueSlug, location.ID, linkRecord.DiscordGuildID, userID)
				break
			}
			if row == nil || strings.TrimSpace(row.ThreadID) == "" || !strings.EqualFold(strings.TrimSpace(row.Status), "active") {
				message = "Mic: Off"
				break
			}
			now := time.Now().UTC()
			if err := saveDiscordMicThread(ctx, pool, discordMicThreadRow{
				ID:                     row.ID,
				LocationID:             location.ID,
				VenueSlug:              venueSlug,
				VenueName:              venueName,
				SessionID:              row.SessionID,
				ShowingID:              row.ShowingID,
				DiscordServerID:        row.DiscordServerID,
				ParentChannelID:        row.ParentChannelID,
				ThreadID:               row.ThreadID,
				ThreadName:             row.ThreadName,
				StartedByUserID:        row.StartedByUserID,
				StartedByDiscordUserID: row.StartedByDiscordUserID,
				StartedAt:              row.StartedAt,
				ShowtimeAt:             row.ShowtimeAt,
				EndedAt:                &now,
				Status:                 "inactive",
			}); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "mic_update_failed"})
				return
			}
			message = "Mic: Off"
		case "status":
			if hasDiscordConfig {
				message, err = discordMicStatusText(ctx, pool, location.ID, linkRecord.DiscordGuildID, venueSlug, venueName)
				break
			}
			if row == nil || strings.TrimSpace(row.ThreadID) == "" || !strings.EqualFold(strings.TrimSpace(row.Status), "active") {
				message = "Mic: Off"
				break
			}
			threadName := row.ThreadName
			if threadName == "" {
				threadName = discordMicThreadName(venueName, row.ShowtimeAt)
			}
			message = fmt.Sprintf("Mic: On\nThread: %s", threadName)
		default:
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "unsupported_mic_command"})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": "mic_command_failed", "detail": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"data": discordMicControlResponse{
				VenueSlug: venueSlug,
				Command:   command,
				Message:   message,
			},
		})
	}
}

func HandleDiscordInteractions(pool *pgxpool.Pool, cfg DiscordServerLinkConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"type": discordInteractionResponseTypeChannelReply, "data": map[string]any{"content": "method_not_allowed"}})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()

		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"type": discordInteractionResponseTypeChannelReply, "data": map[string]any{"content": "invalid_request"}})
			return
		}

		runtimeCfg, err := resolveDiscordServerLinkRuntimeConfig(ctx, pool, cfg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "config_lookup_failed"})
			return
		}

		if strings.TrimSpace(runtimeCfg.PublicKey) == "" {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "discord_interactions_unavailable"})
			return
		}

		if !verifyDiscordInteractionSignature(runtimeCfg.PublicKey, r.Header.Get("X-Signature-Timestamp"), body, r.Header.Get("X-Signature-Ed25519")) {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid_signature"})
			return
		}

		var interaction discordInteractionEnvelope
		if err := json.Unmarshal(body, &interaction); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json"})
			return
		}

		switch interaction.Type {
		case discordInteractionTypePing:
			writeJSON(w, http.StatusOK, discordInteractionResponse{Type: discordInteractionResponseTypePong})
			return
		case discordInteractionTypeApplicationCommand:
			resp := handleDiscordMicCommandDeferred(pool, cfg, interaction)
			writeJSON(w, http.StatusOK, resp)
			return
		default:
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unsupported_interaction"})
			return
		}
	}
}

func handleDiscordMicCommandDeferred(pool *pgxpool.Pool, cfg DiscordServerLinkConfig, interaction discordInteractionEnvelope) discordInteractionResponse {
	interactionCopy := interaction
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		content := handleDiscordMicCommand(ctx, pool, cfg, interactionCopy)
		if err := editDiscordInteractionOriginalResponse(ctx, cfg, interactionCopy, content); err != nil {
			_ = editDiscordInteractionOriginalResponse(context.Background(), cfg, interactionCopy, "Victory could not finish the mic request.")
		}
	}()

	return discordInteractionResponse{
		Type: discordInteractionResponseTypeDeferredChannelMessageWithSource,
		Data: &discordInteractionResponseData{
			Flags: 1 << 6,
		},
	}
}

func handleDiscordMicCommand(ctx context.Context, pool *pgxpool.Pool, cfg DiscordServerLinkConfig, interaction discordInteractionEnvelope) string {
	subcommand := discordMicSubcommand(interaction.Data)
	if subcommand == "" {
		return "Use /mic on, /mic off, or /mic status."
	}

	discordUserID := discordMicInteractionUserID(interaction)
	if strings.TrimSpace(discordUserID) == "" {
		return "Please log into Victory with Discord first. https://victory.amurray.family/auth/discord/start"
	}

	userID, err := resolveVictoryUserForDiscordUser(ctx, pool, discordUserID)
	if err != nil {
		return "Please log into Victory with Discord first. https://victory.amurray.family/auth/discord/start"
	}

	canControl, err := discordMicCanControl(ctx, pool, userID)
	if err != nil {
		return "Victory could not verify your mic authority right now."
	}
	if !canControl {
		return "Producer or director access is required for /mic."
	}

	venueSlug, venueName, parentRow, err := resolveDiscordMicVenueFromChannel(ctx, pool, interaction.ChannelID)
	if err != nil {
		return "Use /mic in a Victory venue chat channel."
	}

	location, err := resolveProducerOfficeLocation(ctx, pool)
	if err != nil {
		return "Victory could not resolve the venue location."
	}

	runtimeCfg, err := resolveDiscordServerLinkRuntimeConfig(ctx, pool, cfg)
	if err != nil || !DiscordMicCommandConfigured(runtimeCfg) {
		return "Discord mic control is not configured yet."
	}

	switch subcommand {
	case "on", "hot", "start":
		text, err := discordMicTurnOn(ctx, pool, runtimeCfg, location.ID, interaction.GuildID, venueSlug, venueName, parentRow, userID, discordUserID)
		if err != nil {
			return "Victory could not start the mic thread."
		}
		return text
	case "off":
		text, err := discordMicTurnOff(ctx, pool, venueSlug, location.ID, interaction.GuildID, userID)
		if err != nil {
			return "Victory could not end the mic thread."
		}
		return text
	case "status":
		text, err := discordMicStatusText(ctx, pool, location.ID, interaction.GuildID, venueSlug, venueName)
		if err != nil {
			return "Victory could not load the mic status."
		}
		return text
	default:
		return "Use /mic on, /mic off, or /mic status."
	}
}

func discordMicChannelReply(content string, ephemeral bool) discordInteractionResponse {
	data := &discordInteractionResponseData{Content: content}
	if ephemeral {
		data.Flags = 1 << 6
	}
	return discordInteractionResponse{Type: discordInteractionResponseTypeChannelReply, Data: data}
}

func discordInteractionWebhookBaseURL(cfg DiscordServerLinkConfig, interaction discordInteractionEnvelope) string {
	baseURL := strings.TrimSpace(cfg.APIBaseURL)
	if baseURL == "" {
		baseURL = discordServerLinkAPIBaseURL
	}
	applicationID := strings.TrimSpace(cfg.ApplicationID)
	if applicationID == "" {
		applicationID = strings.TrimSpace(interaction.Application)
	}
	if applicationID == "" || strings.TrimSpace(interaction.Token) == "" {
		return ""
	}
	return strings.TrimRight(baseURL, "/") + "/webhooks/" + url.PathEscape(applicationID) + "/" + url.PathEscape(strings.TrimSpace(interaction.Token))
}

func editDiscordInteractionOriginalResponse(ctx context.Context, cfg DiscordServerLinkConfig, interaction discordInteractionEnvelope, content string) error {
	baseURL := discordInteractionWebhookBaseURL(cfg, interaction)
	if baseURL == "" {
		return errors.New("discord_interaction_webhook_unavailable")
	}

	payload := map[string]any{
		"content": content,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, baseURL+"/messages/@original", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := discordServerLinkHTTPClient(cfg).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		if message, ok := errBody["message"].(string); ok && strings.TrimSpace(message) != "" {
			return fmt.Errorf("discord interaction followup http %d: %s", resp.StatusCode, message)
		}
		return fmt.Errorf("discord interaction followup http %d", resp.StatusCode)
	}
	return nil
}

func discordMicSubcommand(data *discordInteractionData) string {
	if data == nil {
		return ""
	}
	if strings.TrimSpace(data.Name) != discordMicCommandName {
		return ""
	}
	if len(data.Options) == 0 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(data.Options[0].Name))
}

func discordMicInteractionUserID(interaction discordInteractionEnvelope) string {
	if interaction.Member != nil && interaction.Member.User != nil {
		return strings.TrimSpace(interaction.Member.User.ID)
	}
	if interaction.User != nil {
		return strings.TrimSpace(interaction.User.ID)
	}
	return ""
}

func resolveVictoryUserForDiscordUser(ctx context.Context, pool *pgxpool.Pool, discordUserID string) (string, error) {
	var userID string
	err := pool.QueryRow(ctx, `
		SELECT user_id::text
		FROM auth.discord_identities
		WHERE discord_user_id = $1
		LIMIT 1
	`, strings.TrimSpace(discordUserID)).Scan(&userID)
	if err != nil {
		return "", err
	}
	return userID, nil
}

func discordMicCanControl(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, error) {
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err == nil && ok {
		return true, nil
	} else if err != nil {
		return false, err
	}

	role, err := access.CurrentLocationRole(ctx, pool, userID)
	if err != nil {
		return false, err
	}
	switch role {
	case "producer", "director":
		return true, nil
	default:
		return false, nil
	}
}

func resolveDiscordMicVenueFromChannel(ctx context.Context, pool *pgxpool.Pool, channelID string) (string, string, *DiscordChannelMappingItem, error) {
	location, err := resolveProducerOfficeLocation(ctx, pool)
	if err != nil {
		return "", "", nil, err
	}

	rows, err := loadDiscordChannelMappings(ctx, pool, location.ID)
	if err != nil {
		return "", "", nil, err
	}

	for _, spec := range venueChatChannelSpecs() {
		row, ok := rows[mappingKey(spec.MappingKind, spec.VictoryScopeKind, spec.VictoryScopeSlug)]
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(row.DiscordChannelID), strings.TrimSpace(channelID)) {
			item := mappingItemFromRow(row)
			venueName := venueChatDisplayName(spec.VictoryScopeSlug)
			return spec.VictoryScopeSlug, venueName, &item, nil
		}
	}

	return "", "", nil, pgx.ErrNoRows
}

func discordMicParentChannelItem(ctx context.Context, pool *pgxpool.Pool, locationID, venueSlug string) (*DiscordChannelMappingItem, error) {
	rows, err := loadDiscordChannelMappings(ctx, pool, locationID)
	if err != nil {
		return nil, err
	}
	spec := venueChatSpecForSlug(venueSlug)
	row, ok := rows[mappingKey(spec.MappingKind, spec.VictoryScopeKind, spec.VictoryScopeSlug)]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	item := mappingItemFromRow(row)
	return &item, nil
}

func venueChatSpecForSlug(slug string) discordChannelMappingSpec {
	for _, spec := range venueChatChannelSpecs() {
		if strings.EqualFold(strings.TrimSpace(spec.VictoryScopeSlug), strings.TrimSpace(slug)) {
			return spec
		}
	}
	return discordChannelMappingSpec{}
}

func venueChatDisplayName(slug string) string {
	if name, ok := discordMicVenues[strings.TrimSpace(slug)]; ok {
		return name
	}
	return strings.TrimSpace(slug)
}

func discordMicStatusText(ctx context.Context, pool *pgxpool.Pool, locationID, guildID, venueSlug, venueName string) (string, error) {
	row, err := loadDiscordMicThread(ctx, pool, locationID, venueSlug)
	if err != nil {
		return "", err
	}
	if row == nil || strings.TrimSpace(row.Status) != "active" {
		return "Mic: Off", nil
	}

	threadName := row.ThreadName
	if threadName == "" {
		threadName = discordMicThreadName(venueName, row.ShowtimeAt)
	}

	return fmt.Sprintf("Mic: On\nThread: %s\nLink: %s", threadName, discordMicThreadURL(guildID, row.ThreadID)), nil
}

func discordMicTurnOn(ctx context.Context, pool *pgxpool.Pool, cfg DiscordServerLinkConfig, locationID, guildID, venueSlug, venueName string, parentRow *DiscordChannelMappingItem, userID, discordUserID string) (string, error) {
	parentChannelID := ""
	if parentRow != nil {
		parentChannelID = strings.TrimSpace(parentRow.DiscordChannelID)
	}
	if parentChannelID == "" {
		return "", errors.New("parent_channel_missing")
	}

	sessionID, showingID, showtimeAt, err := resolveMicSessionContext(ctx, pool, venueSlug)
	if err != nil {
		return "", err
	}
	threadName := discordMicThreadName(venueName, showtimeAt)

	row, err := loadDiscordMicThread(ctx, pool, locationID, venueSlug)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	if row != nil && strings.TrimSpace(row.ThreadID) != "" {
		startedAt := now
		startedByUserID := userID
		startedByDiscordID := discordUserID
		if strings.EqualFold(strings.TrimSpace(row.Status), "active") {
			startedAt = row.StartedAt
			startedByUserID = row.StartedByUserID
			startedByDiscordID = row.StartedByDiscordUserID
		}
		if err := saveDiscordMicThread(ctx, pool, discordMicThreadRow{
			LocationID:             locationID,
			VenueSlug:              venueSlug,
			VenueName:              venueName,
			SessionID:              sessionID,
			ShowingID:              showingID,
			DiscordServerID:        guildID,
			ParentChannelID:        parentChannelID,
			ThreadID:               row.ThreadID,
			ThreadName:             threadName,
			StartedByUserID:        startedByUserID,
			StartedByDiscordUserID: startedByDiscordID,
			StartedAt:              startedAt,
			ShowtimeAt:             showtimeAt,
			Status:                 "active",
		}); err != nil {
			return "", err
		}
		if !strings.EqualFold(strings.TrimSpace(row.Status), "active") {
			_ = sendDiscordMicThreadStarter(ctx, cfg, row.ThreadID, discordMicStartMessage(venueName, showtimeAt))
		}
		return fmt.Sprintf("Victory Theater — Session Start — %s — %s\nThread: %s", venueName, discordMicFormatShowtime(showtimeAt), threadName), nil
	}

	created, err := createDiscordMicThread(ctx, cfg, parentChannelID, threadName)
	if err != nil {
		return "", err
	}
	if err := saveDiscordMicThread(ctx, pool, discordMicThreadRow{
		LocationID:             locationID,
		VenueSlug:              venueSlug,
		VenueName:              venueName,
		SessionID:              sessionID,
		ShowingID:              showingID,
		DiscordServerID:        guildID,
		ParentChannelID:        parentChannelID,
		ThreadID:               created.ID,
		ThreadName:             created.Name,
		StartedByUserID:        userID,
		StartedByDiscordUserID: discordUserID,
		StartedAt:              now,
		ShowtimeAt:             showtimeAt,
		Status:                 "active",
	}); err != nil {
		return "", err
	}
	_ = sendDiscordMicThreadStarter(ctx, cfg, created.ID, discordMicStartMessage(venueName, showtimeAt))

	return fmt.Sprintf("Victory Theater — Session Start — %s — %s\nThread: %s", venueName, discordMicFormatShowtime(showtimeAt), threadName), nil
}

func discordMicTurnOff(ctx context.Context, pool *pgxpool.Pool, venueSlug, locationID, guildID, userID string) (string, error) {
	row, err := loadDiscordMicThread(ctx, pool, locationID, venueSlug)
	if err != nil {
		return "", err
	}
	if row == nil || strings.TrimSpace(row.ThreadID) == "" {
		return fmt.Sprintf("Mic: Off"), nil
	}
	now := time.Now().UTC()
	if err := saveDiscordMicThread(ctx, pool, discordMicThreadRow{
		ID:                     row.ID,
		LocationID:             locationID,
		VenueSlug:              venueSlug,
		VenueName:              venueChatDisplayName(venueSlug),
		SessionID:              row.SessionID,
		ShowingID:              row.ShowingID,
		DiscordServerID:        guildID,
		ParentChannelID:        row.ParentChannelID,
		ThreadID:               row.ThreadID,
		ThreadName:             row.ThreadName,
		StartedByUserID:        row.StartedByUserID,
		StartedByDiscordUserID: row.StartedByDiscordUserID,
		StartedAt:              row.StartedAt,
		ShowtimeAt:             row.ShowtimeAt,
		EndedAt:                &now,
		Status:                 "inactive",
	}); err != nil {
		return "", err
	}

	return fmt.Sprintf("Victory Theater — Session End — %s — %s\nThread: %s", venueChatDisplayName(venueSlug), discordMicFormatShowtime(row.ShowtimeAt), row.ThreadName), nil
}

func discordMicCommandStatus(ctx context.Context, cfg DiscordServerLinkConfig, guildID string) (bool, string, string) {
	if !DiscordMicCommandConfigured(cfg) {
		return false, "unavailable", ""
	}

	commands, err := discordMicListCommands(ctx, cfg, guildID)
	if err != nil {
		return false, "error", err.Error()
	}
	for _, cmd := range commands {
		if strings.EqualFold(strings.TrimSpace(cmd.Name), discordMicCommandName) {
			return true, "registered", ""
		}
	}
	return false, "missing", ""
}

func ensureDiscordMicCommand(ctx context.Context, cfg DiscordServerLinkConfig, guildID string) (bool, string, error) {
	if !DiscordMicCommandConfigured(cfg) {
		return false, "", errors.New("discord_mic_unavailable")
	}

	commands, err := discordMicListCommands(ctx, cfg, guildID)
	if err != nil {
		return false, "", err
	}

	payload := discordMicCommand{
		Name:        discordMicCommandName,
		Description: discordMicCommandDesc,
		Type:        1,
		Options: []discordMicCommandOption{
			{
				Type:        1,
				Name:        "on",
				Description: "Start or reuse the active session thread.",
			},
			{
				Type:        1,
				Name:        "off",
				Description: "End the active mic state.",
			},
			{
				Type:        1,
				Name:        "status",
				Description: "Report the mic state.",
			},
			{
				Type:        1,
				Name:        "hot",
				Description: "Alias for on.",
			},
			{
				Type:        1,
				Name:        "start",
				Description: "Alias for on.",
			},
		},
	}

	for _, cmd := range commands {
		if strings.EqualFold(strings.TrimSpace(cmd.Name), discordMicCommandName) {
			var updated map[string]any
			if err := discordServerLinkRequest(ctx, cfg, http.MethodPatch, "/applications/"+url.PathEscape(strings.TrimSpace(cfg.ApplicationID))+"/guilds/"+url.PathEscape(strings.TrimSpace(guildID))+"/commands/"+url.PathEscape(strings.TrimSpace(cmd.ID)), payload, &updated); err != nil {
				return false, "", err
			}
			if id, ok := updated["id"].(string); ok && strings.TrimSpace(id) != "" {
				return true, id, nil
			}
			return true, cmd.ID, nil
		}
	}

	var created map[string]any
	if err := discordServerLinkRequest(ctx, cfg, http.MethodPost, "/applications/"+url.PathEscape(strings.TrimSpace(cfg.ApplicationID))+"/guilds/"+url.PathEscape(strings.TrimSpace(guildID))+"/commands", payload, &created); err != nil {
		return false, "", err
	}
	if id, ok := created["id"].(string); ok && strings.TrimSpace(id) != "" {
		return true, id, nil
	}
	return true, "", nil
}

type discordApplicationCommand struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func discordMicListCommands(ctx context.Context, cfg DiscordServerLinkConfig, guildID string) ([]discordApplicationCommand, error) {
	var commands []discordApplicationCommand
	if err := discordServerLinkRequest(ctx, cfg, http.MethodGet, "/applications/"+url.PathEscape(strings.TrimSpace(cfg.ApplicationID))+"/guilds/"+url.PathEscape(strings.TrimSpace(guildID))+"/commands", nil, &commands); err != nil {
		return nil, err
	}
	return commands, nil
}

func resolveMicSessionContext(ctx context.Context, pool *pgxpool.Pool, venueSlug string) (string, string, time.Time, error) {
	var sessionID, showingID string
	var startedAt time.Time
	err := pool.QueryRow(ctx, `
		SELECT
			COALESCE(s.id::text, ''),
			COALESCE(sh.id::text, ''),
			COALESCE(s.started_at, NOW())
		FROM sessions s
		JOIN venues v ON v.id = s.venue_id
		LEFT JOIN showings sh ON sh.session_id = s.id
		WHERE v.slug = $1
		  AND s.status IN ('rehearsal', 'live')
		ORDER BY s.started_at DESC
		LIMIT 1
	`, strings.TrimSpace(venueSlug)).Scan(&sessionID, &showingID, &startedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", time.Now().UTC().Round(15 * time.Minute), nil
		}
		return "", "", time.Time{}, err
	}
	return sessionID, showingID, startedAt.UTC().Round(15 * time.Minute), nil
}

func loadDiscordMicThread(ctx context.Context, pool *pgxpool.Pool, locationID, venueSlug string) (*discordMicThreadRow, error) {
	var row discordMicThreadRow
	var endedAt sql.NullTime
	err := pool.QueryRow(ctx, `
		SELECT
			id::text,
			location_id::text,
			COALESCE(venue_id::text, ''),
			venue_slug,
			COALESCE(session_id::text, ''),
			COALESCE(showing_id::text, ''),
			discord_server_id,
			parent_channel_id,
			thread_id,
			thread_name,
			COALESCE(started_by_user_id::text, ''),
			COALESCE(started_by_discord_user_id, ''),
			started_at,
			showtime_at,
			ended_at,
			status
		FROM auth.discord_session_threads
		WHERE location_id = $1::uuid
		  AND venue_slug = $2
		LIMIT 1
	`, locationID, venueSlug).Scan(&row.ID, &row.LocationID, &row.VenueID, &row.VenueSlug, &row.SessionID, &row.ShowingID, &row.DiscordServerID, &row.ParentChannelID, &row.ThreadID, &row.ThreadName, &row.StartedByUserID, &row.StartedByDiscordUserID, &row.StartedAt, &row.ShowtimeAt, &endedAt, &row.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if endedAt.Valid {
		ts := endedAt.Time
		row.EndedAt = &ts
	}
	return &row, nil
}

func saveDiscordMicThread(ctx context.Context, pool *pgxpool.Pool, row discordMicThreadRow) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO auth.discord_session_threads (
			location_id,
			venue_id,
			venue_slug,
			session_id,
			showing_id,
			discord_server_id,
			parent_channel_id,
			thread_id,
			thread_name,
			started_by_user_id,
			started_by_discord_user_id,
			started_at,
			showtime_at,
			ended_at,
			status,
			updated_at
		)
		VALUES (
			$1::uuid,
			NULLIF($2, '')::uuid,
			$3,
			NULLIF($4, '')::uuid,
			NULLIF($5, '')::uuid,
			$6,
			$7,
			$8,
			$9,
			NULLIF($10, '')::uuid,
			NULLIF($11, ''),
			$12,
			$13,
			$14,
			$15,
			NOW()
		)
		ON CONFLICT (location_id, venue_slug) DO UPDATE
		SET venue_id = EXCLUDED.venue_id,
			session_id = EXCLUDED.session_id,
			showing_id = EXCLUDED.showing_id,
			discord_server_id = EXCLUDED.discord_server_id,
			parent_channel_id = EXCLUDED.parent_channel_id,
			thread_id = EXCLUDED.thread_id,
			thread_name = EXCLUDED.thread_name,
			started_by_user_id = EXCLUDED.started_by_user_id,
			started_by_discord_user_id = EXCLUDED.started_by_discord_user_id,
			started_at = EXCLUDED.started_at,
			showtime_at = EXCLUDED.showtime_at,
			ended_at = EXCLUDED.ended_at,
			status = EXCLUDED.status,
			updated_at = NOW()
	`, row.LocationID, row.VenueID, row.VenueSlug, row.SessionID, row.ShowingID, row.DiscordServerID, row.ParentChannelID, row.ThreadID, row.ThreadName, row.StartedByUserID, row.StartedByDiscordUserID, row.StartedAt, row.ShowtimeAt, row.EndedAt, row.Status)
	return err
}

func createDiscordMicThread(ctx context.Context, cfg DiscordServerLinkConfig, parentChannelID, threadName string) (discordChannel, error) {
	payload := map[string]any{
		"name":                  threadName,
		"type":                  discordChannelTypePrivateThread,
		"auto_archive_duration": 10080,
		"invitable":             true,
	}
	var created discordChannel
	if err := discordServerLinkRequest(ctx, cfg, http.MethodPost, "/channels/"+url.PathEscape(strings.TrimSpace(parentChannelID))+"/threads", payload, &created); err != nil {
		return discordChannel{}, err
	}
	return created, nil
}

func sendDiscordMicThreadStarter(ctx context.Context, cfg DiscordServerLinkConfig, threadID, content string) error {
	payload := map[string]any{"content": content}
	return discordServerLinkRequest(ctx, cfg, http.MethodPost, "/channels/"+url.PathEscape(strings.TrimSpace(threadID))+"/messages", payload, nil)
}

func discordMicStartMessage(venueName string, showtimeAt time.Time) string {
	return fmt.Sprintf("Victory Theater — Session Start — %s — %s", venueName, discordMicFormatShowtime(showtimeAt))
}

func discordMicThreadName(venueName string, showtimeAt time.Time) string {
	return fmt.Sprintf("%s — Showtime — %s", venueName, discordMicFormatShowtime(showtimeAt))
}

func discordMicFormatShowtime(t time.Time) string {
	return t.UTC().Format("January 2, 2006 3:04 PM")
}

func discordMicThreadURL(guildID, threadID string) string {
	guildID = strings.TrimSpace(guildID)
	threadID = strings.TrimSpace(threadID)
	if guildID == "" || threadID == "" {
		return ""
	}
	return fmt.Sprintf("https://discord.com/channels/%s/%s", guildID, threadID)
}

func normalizeMicVenueSlug(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func normalizeMicCommand(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = strings.TrimPrefix(raw, "/")
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "mic ") {
		raw = strings.TrimSpace(strings.TrimPrefix(raw, "mic "))
	}
	switch raw {
	case "hot", "on", "start", "off", "status":
		return raw
	default:
		return ""
	}
}

func verifyDiscordInteractionSignature(publicKeyHex, timestamp string, body []byte, signatureHex string) bool {
	return verifyDiscordInteractionSignatureString(publicKeyHex, timestamp, body, signatureHex)
}

func verifyDiscordInteractionSignatureString(publicKeyHex, timestamp string, body []byte, signatureHex string) bool {
	publicKeyBytes, err := hex.DecodeString(strings.TrimSpace(publicKeyHex))
	if err != nil || len(publicKeyBytes) != ed25519.PublicKeySize {
		return false
	}
	signatureBytes, err := hex.DecodeString(strings.TrimSpace(signatureHex))
	if err != nil || len(signatureBytes) != ed25519.SignatureSize {
		return false
	}
	message := append([]byte(strings.TrimSpace(timestamp)), body...)
	return ed25519.Verify(ed25519.PublicKey(publicKeyBytes), message, signatureBytes)
}
