package identity

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// This file exposes the "/mic" chat-bridge mechanism (discordMicTurnOn/Off,
// unchanged) to callers outside this package -- specifically Kernel 92's
// Showtime composition (backend/internal/showtime), which needs to connect/
// disconnect the same Discord thread bridge as one step of BeginShowtime/
// EndShowtime, without going through HandleDiscordMicControl's HTTP body
// (kernel doc §17: "Normal Showtime should activate the bridge
// automatically"). The vocabulary here is "Chat Bridge" rather than "mic"
// per kernel doc §18 -- this is a text/forum bridge, not audio.
//
// Chat-bridge failure is always a soft/warning-level result for Showtime
// (kernel doc §14 vs §15: it is never listed as a true blocker), so unlike
// HandleDiscordMicControl's "hot" case (which 503s if Discord isn't
// configured), TryTurnOnChatBridgeForVenue returns ready=false with no
// error in that situation -- many venues simply have no Discord link, and
// Showtime must still succeed.

// TryTurnOnChatBridgeForVenue attempts to connect the chat bridge for a
// venue as part of Showtime composition. ready=false (no error) means the
// bridge is not available for this venue right now -- most commonly no
// Discord server link is configured -- and callers should surface this as
// a non-blocking warning, not fail Showtime.
func TryTurnOnChatBridgeForVenue(ctx context.Context, pool *pgxpool.Pool, cfg DiscordServerLinkConfig, venueSlug, userID string) (message string, ready bool, err error) {
	venueSlug = normalizeMicVenueSlug(venueSlug)
	venueName, ok := discordMicVenues[venueSlug]
	if !ok {
		return "Chat Bridge: not available for this venue", false, nil
	}

	location, err := resolveProducerOfficeLocation(ctx, pool)
	if err != nil {
		return "", false, err
	}

	runtimeCfg, runtimeErr := resolveDiscordServerLinkRuntimeConfig(ctx, pool, cfg)
	linkRecord, linkErr := loadDiscordServerLinkRecord(ctx, pool, location.ID)
	hasDiscordConfig := runtimeErr == nil && DiscordMicCommandConfigured(runtimeCfg) &&
		linkErr == nil && linkRecord.Active && strings.TrimSpace(linkRecord.DiscordGuildID) != ""
	if !hasDiscordConfig {
		return "Chat Bridge: not configured for this venue", false, nil
	}

	parentRow, err := discordMicParentChannelItem(ctx, pool, location.ID, venueSlug)
	if err != nil {
		return "Chat Bridge: parent channel not configured", false, nil
	}

	message, err = discordMicTurnOn(ctx, pool, runtimeCfg, location.ID, linkRecord.DiscordGuildID, venueSlug, venueName, parentRow, userID, "")
	if err != nil {
		return "Chat Bridge: connection failed", false, nil
	}
	return message, true, nil
}

// TurnOffChatBridge disconnects the chat bridge for a venue as part of End
// Showtime. Errors are non-fatal to the caller by convention -- End
// Showtime must not fail because Discord is unreachable -- so callers
// should log a non-nil error rather than surfacing it to the Director.
func TurnOffChatBridge(ctx context.Context, pool *pgxpool.Pool, cfg DiscordServerLinkConfig, venueSlug, userID string) (message string, err error) {
	venueSlug = normalizeMicVenueSlug(venueSlug)
	if venueSlug == "" {
		return "", nil
	}

	location, err := resolveProducerOfficeLocation(ctx, pool)
	if err != nil {
		return "", err
	}

	runtimeCfg, runtimeErr := resolveDiscordServerLinkRuntimeConfig(ctx, pool, cfg)
	linkRecord, linkErr := loadDiscordServerLinkRecord(ctx, pool, location.ID)
	hasDiscordConfig := runtimeErr == nil && DiscordMicCommandConfigured(runtimeCfg) &&
		linkErr == nil && linkRecord.Active && strings.TrimSpace(linkRecord.DiscordGuildID) != ""

	if hasDiscordConfig {
		return discordMicTurnOff(ctx, pool, venueSlug, location.ID, linkRecord.DiscordGuildID, userID)
	}

	// Local-only fallback, matching HandleDiscordMicControl's "off" case:
	// flip any locally-tracked thread row to inactive even without Discord.
	row, err := loadDiscordMicThread(ctx, pool, location.ID, venueSlug)
	if err != nil {
		return "", err
	}
	if row == nil || strings.TrimSpace(row.ThreadID) == "" || !strings.EqualFold(strings.TrimSpace(row.Status), "active") {
		return "Chat Bridge: Off", nil
	}
	now := time.Now().UTC()
	if err := saveDiscordMicThread(ctx, pool, discordMicThreadRow{
		ID:                     row.ID,
		LocationID:             location.ID,
		VenueSlug:              venueSlug,
		VenueName:              row.VenueName,
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
		return "", err
	}
	return "Chat Bridge: Off", nil
}

// ChatBridgeReady reports whether the chat bridge is currently active for a
// venue, for Showtime preflight -- read-only, no state change.
func ChatBridgeReady(ctx context.Context, pool *pgxpool.Pool, locationSlugVenue string) (bool, error) {
	venueSlug := normalizeMicVenueSlug(locationSlugVenue)
	if venueSlug == "" {
		return false, nil
	}
	location, err := resolveProducerOfficeLocation(ctx, pool)
	if err != nil {
		return false, err
	}
	row, err := loadDiscordMicThread(ctx, pool, location.ID, venueSlug)
	if err != nil {
		return false, err
	}
	return row != nil && strings.EqualFold(strings.TrimSpace(row.Status), "active"), nil
}
