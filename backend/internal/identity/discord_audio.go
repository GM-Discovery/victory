package identity

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DiscordAudioStatusResponse struct {
	VenueSlug           string                     `json:"venue_slug"`
	VenueName           string                     `json:"venue_name,omitempty"`
	Configured          bool                       `json:"configured"`
	DiscordServerLinked bool                       `json:"discord_server_linked"`
	AudioChannel        *DiscordAudioChannelStatus `json:"audio_channel,omitempty"`
	CanOpen             bool                       `json:"can_open"`
	Participants        []DiscordAudioParticipant  `json:"participants"`
	Speaking            []DiscordAudioParticipant  `json:"speaking"`
	VoiceStateTracking  DiscordAudioFeatureStatus  `json:"voice_state_tracking,omitempty"`
	SpeakerIndicator    DiscordAudioFeatureStatus  `json:"speaker_indicator,omitempty"`
	VolumeControls      DiscordAudioFeatureStatus  `json:"volume_controls,omitempty"`
	Status              string                     `json:"status"`
	Message             string                     `json:"message,omitempty"`
	RepairHint          string                     `json:"repair_hint,omitempty"`
	MappingKind         string                     `json:"mapping_kind,omitempty"`
}

type DiscordAudioChannelStatus struct {
	ID             string `json:"id,omitempty"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	OpenURL        string `json:"open_url,omitempty"`
	Status         string `json:"status"`
	ExpectedName   string `json:"expected_name,omitempty"`
	LastVerifiedAt string `json:"last_verified_at,omitempty"`
}

type DiscordAudioParticipant struct {
	DiscordUserID      string `json:"discord_user_id,omitempty"`
	DiscordDisplayName string `json:"discord_display_name,omitempty"`
	DisplayName        string `json:"display_name,omitempty"`
	AvatarURL          string `json:"avatar_url,omitempty"`
	LinkedUserID       string `json:"linked_user_id,omitempty"`
	VictoryDisplayName string `json:"victory_display_name,omitempty"`
	CharacterName      string `json:"character_name,omitempty"`
	SourceLabel        string `json:"source_label,omitempty"`
	Speaking           bool   `json:"speaking,omitempty"`
	Status             string `json:"status,omitempty"`
}

type DiscordAudioFeatureStatus struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

type discordAudioVenueContext struct {
	VenueID      string
	VenueSlug    string
	VenueName    string
	LocationID   string
	LocationSlug string
	LocationName string
}

func HandleDiscordAudioStatus(pool *pgxpool.Pool, cfg DiscordServerLinkConfig, presenceStore *DiscordAudioPresenceStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		_, err := currentUserID(ctx, pool, r)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		venueSlug := strings.TrimSpace(r.URL.Query().Get("venue_slug"))
		if venueSlug == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "venue_slug_required"})
			return
		}

		venue, err := loadDiscordAudioVenueContext(ctx, pool, venueSlug)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "venue_not_found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "venue_lookup_failed"})
			return
		}

		spec := venueAudioSpecForSlug(venueSlug)
		if strings.TrimSpace(spec.MappingKind) == "" || strings.TrimSpace(spec.ExpectedName) == "" {
			writeJSON(w, http.StatusOK, map[string]any{
				"ok": true,
				"data": DiscordAudioStatusResponse{
					VenueSlug:           venueSlug,
					VenueName:           venue.VenueName,
					Configured:          false,
					DiscordServerLinked: false,
					CanOpen:             false,
					Participants:        []DiscordAudioParticipant{},
					Speaking:            []DiscordAudioParticipant{},
					Status:              "not_enabled",
					Message:             "Audio is not enabled for this venue.",
				},
			})
			return
		}

		linkRecord, err := loadDiscordServerLinkRecord(ctx, pool, venue.LocationID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "link_lookup_failed"})
			return
		}

		runtimeCfg, err := resolveDiscordServerLinkRuntimeConfig(ctx, pool, cfg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "config_lookup_failed"})
			return
		}

		gatewayState, err := loadDiscordGatewayState(ctx, pool, venue.LocationID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "gateway_state_lookup_failed"})
			return
		}

		status := DiscordAudioStatusResponse{
			VenueSlug:           venueSlug,
			VenueName:           venue.VenueName,
			DiscordServerLinked: linkRecord.Active && strings.TrimSpace(linkRecord.DiscordGuildID) != "",
			Participants:        []DiscordAudioParticipant{},
			Speaking:            []DiscordAudioParticipant{},
			MappingKind:         spec.MappingKind,
			VolumeControls: DiscordAudioFeatureStatus{
				Available: false,
				Reason:    "Discord bot/Gateway cannot control the local Discord client volume.",
			},
			SpeakerIndicator: DiscordAudioFeatureStatus{
				Available: false,
				Reason:    "Discord Gateway voice-state events show membership, not active audio levels. A Discord voice websocket or Social SDK client context would be required.",
			},
		}
		if gatewayState != nil {
			status.VoiceStateTracking = DiscordAudioFeatureStatus{
				Available: gatewayState.Intents&(1<<7) != 0,
			}
			if !status.VoiceStateTracking.Available {
				status.VoiceStateTracking.Reason = "Enable the GUILD_VOICE_STATES gateway intent to track voice-channel participants."
			}
		} else {
			status.VoiceStateTracking = DiscordAudioFeatureStatus{
				Available: false,
				Reason:    "Discord gateway state is unavailable.",
			}
		}

		if !status.DiscordServerLinked {
			status.Status = "server_not_linked"
			status.Message = "Discord server is not linked."
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": status})
			return
		}

		rows, err := loadDiscordChannelMappings(ctx, pool, venue.LocationID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "mapping_lookup_failed"})
			return
		}

		row, ok := rows[mappingKey(spec.MappingKind, spec.VictoryScopeKind, spec.VictoryScopeSlug)]
		if !ok {
			status.Status = "needs_repair"
			status.Message = "Needs repair in Producer's Office."
			status.RepairHint = "Open Producer's Office and run Create / Repair Discord Channels."
			status.AudioChannel = &DiscordAudioChannelStatus{
				Name:         spec.ExpectedName,
				Type:         discordChannelTypeVoice,
				Status:       "missing",
				ExpectedName: spec.ExpectedName,
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": status})
			return
		}

		if !DiscordServerLinkConfigured(runtimeCfg) {
			status.Status = "needs_repair"
			status.Message = "Discord bot settings are unavailable."
			status.RepairHint = "Open Producer's Office and verify Discord server bootstrap settings."
			status.AudioChannel = &DiscordAudioChannelStatus{
				ID:             row.DiscordChannelID,
				Name:           fallbackString(row.DiscordChannelName, spec.ExpectedName),
				Type:           row.DiscordChannelType,
				Status:         "found",
				ExpectedName:   spec.ExpectedName,
				LastVerifiedAt: formatTimePtr(row.LastVerifiedAt),
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": status})
			return
		}

		channels, err := fetchDiscordServerChannels(ctx, runtimeCfg, linkRecord.DiscordGuildID)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": "discord_audio_channel_list_failed", "detail": err.Error()})
			return
		}

		item := mappingStatusForSpec(rows, channels, spec)
		status.AudioChannel = &DiscordAudioChannelStatus{
			ID:             item.DiscordChannelID,
			Name:           fallbackString(item.ActualName, item.ExpectedName),
			Type:           fallbackString(item.DiscordChannelType, discordChannelTypeVoice),
			Status:         item.Status,
			ExpectedName:   item.ExpectedName,
			LastVerifiedAt: item.LastVerifiedAt,
		}
		if strings.TrimSpace(item.DiscordChannelID) != "" && strings.TrimSpace(item.Status) != "missing" {
			status.Configured = true
			status.CanOpen = true
			status.Status = "ready"
			status.AudioChannel.OpenURL = discordAudioChannelURL(linkRecord.DiscordGuildID, item.DiscordChannelID)
			if status.VoiceStateTracking.Available {
				stateRows := presenceStore.Snapshot(linkRecord.DiscordGuildID, item.DiscordChannelID)
				participantIDs := make([]string, 0, len(stateRows))
				for _, state := range stateRows {
					participantIDs = append(participantIDs, state.UserID)
				}
				linkedUsers, err := loadDiscordAudioLinkedUsers(ctx, pool, participantIDs)
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "linked_user_lookup_failed"})
					return
				}
				status.Participants = buildDiscordAudioParticipants(stateRows, linkedUsers)
			}
		} else {
			status.Status = "needs_repair"
			status.Message = "Needs repair in Producer's Office."
			status.RepairHint = "Open Producer's Office and run Create / Repair Discord Channels."
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": status})
	}
}

func loadDiscordAudioVenueContext(ctx context.Context, pool *pgxpool.Pool, venueSlug string) (discordAudioVenueContext, error) {
	var venue discordAudioVenueContext
	err := pool.QueryRow(ctx, `
		SELECT
			v.id::text,
			v.slug,
			v.name,
			l.id::text,
			l.slug,
			l.name
		FROM venues v
		JOIN lots lo ON lo.id = v.lot_id
		JOIN locations l ON l.id = lo.location_id
		WHERE v.slug = $1
		LIMIT 1
	`, strings.TrimSpace(venueSlug)).Scan(&venue.VenueID, &venue.VenueSlug, &venue.VenueName, &venue.LocationID, &venue.LocationSlug, &venue.LocationName)
	if err != nil {
		return discordAudioVenueContext{}, err
	}
	return venue, nil
}

func discordAudioChannelURL(guildID, channelID string) string {
	guildID = strings.TrimSpace(guildID)
	channelID = strings.TrimSpace(channelID)
	if guildID == "" || channelID == "" {
		return ""
	}
	return "https://discord.com/channels/" + guildID + "/" + channelID
}

type discordAudioLinkedUser struct {
	UserID      string
	Handle      string
	DisplayName string
	VictoryName string
}

func loadDiscordAudioLinkedUsers(ctx context.Context, pool *pgxpool.Pool, discordUserIDs []string) (map[string]discordAudioLinkedUser, error) {
	result := map[string]discordAudioLinkedUser{}
	if len(discordUserIDs) == 0 {
		return result, nil
	}

	unique := make([]string, 0, len(discordUserIDs))
	seen := map[string]struct{}{}
	for _, id := range discordUserIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return result, nil
	}

	rows, err := pool.Query(ctx, `
		SELECT
			d.discord_user_id,
			u.id::text,
			COALESCE(NULLIF(u.handle, ''), ''),
			COALESCE(NULLIF(u.display_name, ''), '')
		FROM auth.discord_identities d
		JOIN users u ON u.id = d.user_id
		WHERE d.discord_user_id = ANY($1::text[])
	`, unique)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var discordUserID string
		var userID string
		var handle string
		var displayName string
		if err := rows.Scan(&discordUserID, &userID, &handle, &displayName); err != nil {
			return nil, err
		}
		result[strings.TrimSpace(discordUserID)] = discordAudioLinkedUser{
			UserID:      strings.TrimSpace(userID),
			Handle:      strings.TrimSpace(handle),
			DisplayName: strings.TrimSpace(displayName),
			VictoryName: firstNonEmpty(strings.TrimSpace(displayName), strings.TrimSpace(handle)),
		}
	}
	return result, rows.Err()
}

func buildDiscordAudioParticipants(states []DiscordAudioPresenceState, linkedUsers map[string]discordAudioLinkedUser) []DiscordAudioParticipant {
	participants := make([]DiscordAudioParticipant, 0, len(states))
	for _, state := range states {
		participant := DiscordAudioParticipant{
			DiscordUserID:      strings.TrimSpace(state.UserID),
			DiscordDisplayName: discordPresenceDisplayName(state),
			DisplayName:        discordPresenceDisplayName(state),
			AvatarURL:          discordAvatarURL(state.UserID, state.AvatarHash, state.Discriminator),
			Speaking:           false,
			Status:             voiceParticipantStatus(state),
		}
		if linked, ok := linkedUsers[strings.TrimSpace(state.UserID)]; ok {
			participant.LinkedUserID = linked.UserID
			participant.VictoryDisplayName = linked.VictoryName
			participant.DisplayName = firstNonEmpty(linked.VictoryName, participant.DisplayName)
			participant.SourceLabel = ""
		} else {
			participant.SourceLabel = "via Discord"
		}
		participants = append(participants, participant)
	}
	return participants
}

func discordPresenceDisplayName(state DiscordAudioPresenceState) string {
	return firstNonEmpty(strings.TrimSpace(state.Nick), strings.TrimSpace(state.GlobalName), strings.TrimSpace(state.Username), strings.TrimSpace(state.UserID))
}

func voiceParticipantStatus(state DiscordAudioPresenceState) string {
	if state.SelfMute || state.SelfDeaf || state.Mute || state.Deaf {
		return "muted"
	}
	return "listening"
}

func discordAvatarURL(userID, avatarHash, discriminator string) string {
	userID = strings.TrimSpace(userID)
	avatarHash = strings.TrimSpace(avatarHash)
	if userID == "" {
		return ""
	}
	if avatarHash != "" {
		return "https://cdn.discordapp.com/avatars/" + userID + "/" + avatarHash + ".png?size=96"
	}
	index := 0
	for _, r := range strings.TrimSpace(userID) {
		index = (index + int(r)) % 6
	}
	if strings.TrimSpace(discriminator) != "" && discriminator != "0" {
		sum := 0
		for _, r := range discriminator {
			sum += int(r)
		}
		index = (index + sum) % 6
	}
	return "https://cdn.discordapp.com/embed/avatars/" + string(rune('0'+index)) + ".png"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
