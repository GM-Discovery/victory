package identity

import (
	"context"
	"database/sql"
	"net/http"
	"net/url"
	"strings"
	"time"

	"victory/backend/internal/access"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	discordChannelMappingKindCoreCategory  = "core_category"
	discordChannelMappingKindCoreChannel   = "core_channel"
	discordChannelMappingKindVenueCategory = "venue_category"
	discordChannelScopeKindLocation        = "location"
	discordChannelScopeKindCoreChannel     = "core_channel"
	discordChannelScopeKindVenue           = "venue"
	discordChannelTypeCategory             = "category"
	discordChannelTypeText                 = "text"
)

type DiscordChannelMappingItem struct {
	MappingKind            string `json:"mapping_kind"`
	VictoryScopeKind       string `json:"victory_scope_kind"`
	VictoryScopeSlug       string `json:"victory_scope_slug"`
	ExpectedName           string `json:"expected_name"`
	ActualName             string `json:"actual_name"`
	DiscordChannelID       string `json:"discord_channel_id"`
	DiscordChannelType     string `json:"discord_channel_type"`
	ParentDiscordChannelID string `json:"parent_discord_channel_id,omitempty"`
	Status                 string `json:"status"`
	LastVerifiedAt         string `json:"last_verified_at,omitempty"`
}

type DiscordChannelMappingGroup struct {
	Category DiscordChannelMappingItem   `json:"category"`
	Channels []DiscordChannelMappingItem `json:"channels"`
}

type DiscordChannelMappingStatus struct {
	Linked         bool                                `json:"linked"`
	DiscordServer  *DiscordServerLinkServer            `json:"discord_server,omitempty"`
	CanManage      bool                                `json:"can_manage"`
	SetupAvailable bool                                `json:"setup_available"`
	Core           DiscordChannelMappingGroup          `json:"core"`
	Venues         []DiscordChannelMappingItem         `json:"venues"`
	RepairSummary  *DiscordChannelMappingRepairSummary `json:"repair_summary,omitempty"`
}

type DiscordChannelMappingRepairSummary struct {
	Created       []string `json:"created"`
	Found         []string `json:"found"`
	Updated       []string `json:"updated"`
	Failed        []string `json:"failed"`
	FailedDetails []string `json:"failed_details,omitempty"`
}

type discordChannelMappingRow struct {
	LocationID             string
	DiscordServerID        string
	DiscordChannelID       string
	DiscordChannelName     string
	DiscordChannelType     string
	MappingKind            string
	VictoryScopeKind       string
	VictoryScopeID         string
	VictoryScopeSlug       string
	ExpectedName           string
	ParentDiscordChannelID string
	LastVerifiedAt         *time.Time
}

type discordChannelMappingSpec struct {
	MappingKind      string
	VictoryScopeKind string
	VictoryScopeSlug string
	ExpectedName     string
	ChannelType      string
	ParentScopeKind  string
	ParentScopeSlug  string
}

func HandleDiscordChannelMappingStatus(pool *pgxpool.Pool, cfg DiscordServerLinkConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		location, err := resolveProducerOfficeLocation(ctx, pool)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "location_lookup_failed"})
			return
		}

		linkRecord, err := loadDiscordServerLinkRecord(ctx, pool, location.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "link_lookup_failed"})
			return
		}

		runtimeCfg, err := resolveDiscordServerLinkRuntimeConfig(ctx, pool, cfg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "config_lookup_failed"})
			return
		}

		canManage, err := access.IsOperatorUser(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed"})
			return
		}

		status := DiscordChannelMappingStatus{
			Linked:         linkRecord.Active && strings.TrimSpace(linkRecord.DiscordGuildID) != "",
			CanManage:      canManage,
			SetupAvailable: DiscordServerLinkConfigured(runtimeCfg),
			Core:           DiscordChannelMappingGroup{},
		}
		if status.Linked {
			status.DiscordServer = &DiscordServerLinkServer{
				ID:   linkRecord.DiscordGuildID,
				Name: fallbackString(linkRecord.DiscordGuildName, linkRecord.DiscordGuildID),
			}
		}

		if !status.Linked || !status.SetupAvailable {
			status.Core.Category = missingMappingItem(discordChannelMappingKindCoreCategory, discordChannelScopeKindLocation, location.Slug, "Victory Theater")
			status.Core.Channels = defaultCoreChannelMissingItems()
			status.Venues = defaultVenueMissingItems()
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": status})
			return
		}

		channels, err := fetchDiscordServerChannels(ctx, runtimeCfg, linkRecord.DiscordGuildID)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": "discord_channel_list_failed"})
			return
		}

		rows, err := loadDiscordChannelMappings(ctx, pool, location.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "mapping_lookup_failed"})
			return
		}

		status.Core.Category = mappingStatusForSpec(rows, channels, discordChannelMappingSpec{
			MappingKind:      discordChannelMappingKindCoreCategory,
			VictoryScopeKind: discordChannelScopeKindLocation,
			VictoryScopeSlug: location.Slug,
			ExpectedName:     "Victory Theater",
			ChannelType:      discordChannelTypeCategory,
		})
		for _, spec := range coreChannelSpecs() {
			status.Core.Channels = append(status.Core.Channels, mappingStatusForSpec(rows, channels, spec))
		}
		for _, spec := range venueCategorySpecs() {
			status.Venues = append(status.Venues, mappingStatusForSpec(rows, channels, spec))
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": status})
	}
}

func HandleDiscordChannelMappingRepair(pool *pgxpool.Pool, cfg DiscordServerLinkConfig) http.HandlerFunc {
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

		linkRecord, err := loadDiscordServerLinkRecord(ctx, pool, location.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "link_lookup_failed"})
			return
		}
		if !linkRecord.Active || strings.TrimSpace(linkRecord.DiscordGuildID) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "discord_server_not_linked"})
			return
		}

		runtimeCfg, err := resolveDiscordServerLinkRuntimeConfig(ctx, pool, cfg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "config_lookup_failed"})
			return
		}
		if !DiscordServerLinkConfigured(runtimeCfg) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "discord_server_link_unavailable"})
			return
		}

		channels, err := fetchDiscordServerChannels(ctx, runtimeCfg, linkRecord.DiscordGuildID)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": "discord_channel_list_failed"})
			return
		}

		rows, err := loadDiscordChannelMappings(ctx, pool, location.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "mapping_lookup_failed"})
			return
		}

		summary := &DiscordChannelMappingRepairSummary{}
		createdBy := userID

		coreCategory, action, err := ensureDiscordChannelMapping(ctx, pool, runtimeCfg, location.ID, linkRecord.DiscordGuildID, createdBy, channels, rows, discordChannelMappingSpec{
			MappingKind:      discordChannelMappingKindCoreCategory,
			VictoryScopeKind: discordChannelScopeKindLocation,
			VictoryScopeSlug: location.Slug,
			ExpectedName:     "Victory Theater",
			ChannelType:      discordChannelTypeCategory,
		}, "")
		if err != nil {
			summary.Failed = append(summary.Failed, "Victory Theater")
			summary.FailedDetails = append(summary.FailedDetails, err.Error())
		} else {
			summary.add(action, coreCategory.ExpectedName)
			rows = upsertRowCache(rows, coreCategory)
		}

		coreCategoryID := coreCategory.DiscordChannelID
		if strings.TrimSpace(coreCategoryID) == "" {
			for _, spec := range coreChannelSpecs() {
				summary.Failed = append(summary.Failed, spec.ExpectedName)
			}
		} else {
			for _, spec := range coreChannelSpecs() {
				item, action, err := ensureDiscordChannelMapping(ctx, pool, runtimeCfg, location.ID, linkRecord.DiscordGuildID, createdBy, channels, rows, spec, coreCategoryID)
				if err != nil {
					summary.Failed = append(summary.Failed, spec.ExpectedName)
					summary.FailedDetails = append(summary.FailedDetails, spec.ExpectedName+": "+err.Error())
					continue
				}
				summary.add(action, item.ExpectedName)
				rows = upsertRowCache(rows, item)
			}
		}

		for _, spec := range venueCategorySpecs() {
			item, action, err := ensureDiscordChannelMapping(ctx, pool, runtimeCfg, location.ID, linkRecord.DiscordGuildID, createdBy, channels, rows, spec, "")
			if err != nil {
				summary.Failed = append(summary.Failed, spec.ExpectedName)
				summary.FailedDetails = append(summary.FailedDetails, spec.ExpectedName+": "+err.Error())
				continue
			}
			summary.add(action, item.ExpectedName)
			rows = upsertRowCache(rows, item)
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"data": summary,
		})
	}
}

func fetchDiscordServerChannels(ctx context.Context, cfg DiscordServerLinkConfig, guildID string) ([]discordChannel, error) {
	var channels []discordChannel
	if err := discordServerLinkRequest(ctx, cfg, http.MethodGet, "/guilds/"+url.PathEscape(strings.TrimSpace(guildID))+"/channels", nil, &channels); err != nil {
		return nil, err
	}
	return channels, nil
}

func loadDiscordChannelMappings(ctx context.Context, pool *pgxpool.Pool, locationID string) (map[string]discordChannelMappingRow, error) {
	rows := map[string]discordChannelMappingRow{}
	rowsRaw, err := pool.Query(ctx, `
		SELECT
			location_id::text,
			discord_server_id,
			discord_channel_id,
			discord_channel_name,
			discord_channel_type,
			mapping_kind,
			victory_scope_kind,
			COALESCE(victory_scope_id::text, ''),
			COALESCE(victory_scope_slug, ''),
			expected_name,
			COALESCE(parent_discord_channel_id, ''),
			last_verified_at
		FROM auth.discord_channel_mappings
		WHERE location_id = $1::uuid
	`, locationID)
	if err != nil {
		return nil, err
	}
	defer rowsRaw.Close()
	for rowsRaw.Next() {
		var row discordChannelMappingRow
		var lastVerifiedAt sql.NullTime
		if err := rowsRaw.Scan(&row.LocationID, &row.DiscordServerID, &row.DiscordChannelID, &row.DiscordChannelName, &row.DiscordChannelType, &row.MappingKind, &row.VictoryScopeKind, &row.VictoryScopeID, &row.VictoryScopeSlug, &row.ExpectedName, &row.ParentDiscordChannelID, &lastVerifiedAt); err != nil {
			return nil, err
		}
		if lastVerifiedAt.Valid {
			ts := lastVerifiedAt.Time
			row.LastVerifiedAt = &ts
		}
		rows[mappingKey(row.MappingKind, row.VictoryScopeKind, row.VictoryScopeSlug)] = row
	}
	return rows, rowsRaw.Err()
}

func mappingKey(kind, scopeKind, scopeSlug string) string {
	return strings.Join([]string{kind, scopeKind, scopeSlug}, "|")
}

func upsertRowCache(rows map[string]discordChannelMappingRow, item DiscordChannelMappingItem) map[string]discordChannelMappingRow {
	rows[mappingKey(item.MappingKind, item.VictoryScopeKind, item.VictoryScopeSlug)] = discordChannelMappingRow{
		DiscordChannelID:       item.DiscordChannelID,
		DiscordChannelName:     item.ActualName,
		DiscordChannelType:     item.DiscordChannelType,
		MappingKind:            item.MappingKind,
		VictoryScopeKind:       item.VictoryScopeKind,
		VictoryScopeSlug:       item.VictoryScopeSlug,
		ExpectedName:           item.ExpectedName,
		ParentDiscordChannelID: item.ParentDiscordChannelID,
	}
	return rows
}

func ensureDiscordChannelMapping(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg DiscordServerLinkConfig,
	locationID, guildID, userID string,
	channels []discordChannel,
	rows map[string]discordChannelMappingRow,
	spec discordChannelMappingSpec,
	parentID string,
) (DiscordChannelMappingItem, string, error) {
	key := mappingKey(spec.MappingKind, spec.VictoryScopeKind, spec.VictoryScopeSlug)
	expectedParent := strings.TrimSpace(parentID)
	liveByID := map[string]discordChannel{}
	for _, ch := range channels {
		liveByID[strings.TrimSpace(ch.ID)] = ch
	}

	if row, ok := rows[key]; ok {
		if live, ok := liveByID[strings.TrimSpace(row.DiscordChannelID)]; ok {
			action := "found"
			if !strings.EqualFold(strings.TrimSpace(live.Name), strings.TrimSpace(row.DiscordChannelName)) || channelTypeLabel(live.Type) != strings.TrimSpace(row.DiscordChannelType) || strings.TrimSpace(live.ParentID) != strings.TrimSpace(row.ParentDiscordChannelID) {
				action = "updated"
			}
			if err := saveDiscordChannelMapping(ctx, pool, locationID, guildID, userID, spec, live.ID, live.Name, channelTypeLabel(live.Type), expectedParent, nowPtr()); err != nil {
				return DiscordChannelMappingItem{}, "", err
			}
			return mappingItemFromRow(discordChannelMappingRow{
				LocationID:             locationID,
				DiscordServerID:        guildID,
				DiscordChannelID:       live.ID,
				DiscordChannelName:     live.Name,
				DiscordChannelType:     channelTypeLabel(live.Type),
				MappingKind:            spec.MappingKind,
				VictoryScopeKind:       spec.VictoryScopeKind,
				VictoryScopeSlug:       spec.VictoryScopeSlug,
				ExpectedName:           spec.ExpectedName,
				ParentDiscordChannelID: expectedParent,
			}), action, nil
		}
	}

	if found := findDiscordChannelByName(channels, spec.ExpectedName, spec.ChannelType, expectedParent); found != nil {
		if err := saveDiscordChannelMapping(ctx, pool, locationID, guildID, userID, spec, found.ID, found.Name, channelTypeLabel(found.Type), expectedParent, nowPtr()); err != nil {
			return DiscordChannelMappingItem{}, "", err
		}
		return mappingItemFromRow(discordChannelMappingRow{
			LocationID:             locationID,
			DiscordServerID:        guildID,
			DiscordChannelID:       found.ID,
			DiscordChannelName:     found.Name,
			DiscordChannelType:     channelTypeLabel(found.Type),
			MappingKind:            spec.MappingKind,
			VictoryScopeKind:       spec.VictoryScopeKind,
			VictoryScopeSlug:       spec.VictoryScopeSlug,
			ExpectedName:           spec.ExpectedName,
			ParentDiscordChannelID: expectedParent,
		}), "found", nil
	}

	created, err := createDiscordChannel(ctx, cfg, guildID, spec, expectedParent)
	if err != nil {
		return DiscordChannelMappingItem{}, "", err
	}
	if err := saveDiscordChannelMapping(ctx, pool, locationID, guildID, userID, spec, created.ID, created.Name, channelTypeLabel(created.Type), expectedParent, nowPtr()); err != nil {
		return DiscordChannelMappingItem{}, "", err
	}
	return mappingItemFromRow(discordChannelMappingRow{
		LocationID:             locationID,
		DiscordServerID:        guildID,
		DiscordChannelID:       created.ID,
		DiscordChannelName:     created.Name,
		DiscordChannelType:     channelTypeLabel(created.Type),
		MappingKind:            spec.MappingKind,
		VictoryScopeKind:       spec.VictoryScopeKind,
		VictoryScopeSlug:       spec.VictoryScopeSlug,
		ExpectedName:           spec.ExpectedName,
		ParentDiscordChannelID: expectedParent,
	}), "created", nil
}

func findDiscordChannelByName(channels []discordChannel, expectedName, expectedType, parentID string) *discordChannel {
	expectedName = strings.ToLower(strings.TrimSpace(expectedName))
	expectedType = strings.ToLower(strings.TrimSpace(expectedType))
	parentID = strings.TrimSpace(parentID)
	for i := range channels {
		ch := channels[i]
		if strings.ToLower(strings.TrimSpace(ch.Name)) != expectedName {
			continue
		}
		if expectedType != "" && channelTypeLabel(ch.Type) != expectedType {
			continue
		}
		if parentID != "" && strings.TrimSpace(ch.ParentID) != parentID {
			continue
		}
		return &ch
	}
	return nil
}

func saveDiscordChannelMapping(ctx context.Context, pool *pgxpool.Pool, locationID, guildID, userID string, spec discordChannelMappingSpec, channelID, channelName, channelType, parentID string, verifiedAt *time.Time) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO auth.discord_channel_mappings (
			location_id,
			discord_server_id,
			discord_channel_id,
			discord_channel_name,
			discord_channel_type,
			mapping_kind,
			victory_scope_kind,
			victory_scope_slug,
			expected_name,
			parent_discord_channel_id,
			created_by_user_id,
			last_verified_at,
			updated_at
		)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, NULLIF($10, ''), $11::uuid, $12, NOW())
		ON CONFLICT (location_id, mapping_kind, victory_scope_kind, victory_scope_slug) DO UPDATE
		SET discord_server_id = EXCLUDED.discord_server_id,
			discord_channel_id = EXCLUDED.discord_channel_id,
			discord_channel_name = EXCLUDED.discord_channel_name,
			discord_channel_type = EXCLUDED.discord_channel_type,
			expected_name = EXCLUDED.expected_name,
			parent_discord_channel_id = EXCLUDED.parent_discord_channel_id,
			created_by_user_id = EXCLUDED.created_by_user_id,
			last_verified_at = EXCLUDED.last_verified_at,
			updated_at = NOW()
	`, locationID, guildID, channelID, channelName, channelType, spec.MappingKind, spec.VictoryScopeKind, spec.VictoryScopeSlug, spec.ExpectedName, parentID, userID, verifiedAt)
	return err
}

func createDiscordChannel(ctx context.Context, cfg DiscordServerLinkConfig, guildID string, spec discordChannelMappingSpec, parentID string) (discordChannel, error) {
	payload := map[string]any{
		"name": spec.ExpectedName,
		"type": discordChannelTypeID(spec.ChannelType),
	}
	if strings.TrimSpace(parentID) != "" {
		payload["parent_id"] = strings.TrimSpace(parentID)
	}
	var created discordChannel
	if err := discordServerLinkRequest(ctx, cfg, http.MethodPost, "/guilds/"+url.PathEscape(strings.TrimSpace(guildID))+"/channels", payload, &created); err != nil {
		return discordChannel{}, err
	}
	return created, nil
}

func channelTypeLabel(channelType int) string {
	switch channelType {
	case 4:
		return discordChannelTypeCategory
	case 0:
		return discordChannelTypeText
	default:
		return "unknown"
	}
}

func discordChannelTypeID(label string) int {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case discordChannelTypeCategory:
		return 4
	case discordChannelTypeText:
		return 0
	default:
		return 0
	}
}

func mappingItemFromRow(row discordChannelMappingRow) DiscordChannelMappingItem {
	status := "missing"
	if strings.TrimSpace(row.DiscordChannelID) != "" {
		status = "found"
	}
	return DiscordChannelMappingItem{
		MappingKind:            row.MappingKind,
		VictoryScopeKind:       row.VictoryScopeKind,
		VictoryScopeSlug:       row.VictoryScopeSlug,
		ExpectedName:           row.ExpectedName,
		ActualName:             row.DiscordChannelName,
		DiscordChannelID:       row.DiscordChannelID,
		DiscordChannelType:     row.DiscordChannelType,
		ParentDiscordChannelID: row.ParentDiscordChannelID,
		Status:                 status,
		LastVerifiedAt:         formatTimePtr(row.LastVerifiedAt),
	}
}

func missingMappingItem(mappingKind, scopeKind, scopeSlug, expected string) DiscordChannelMappingItem {
	return DiscordChannelMappingItem{
		MappingKind:      mappingKind,
		VictoryScopeKind: scopeKind,
		VictoryScopeSlug: scopeSlug,
		ExpectedName:     expected,
		ActualName:       "",
		Status:           "missing",
	}
}

func defaultCoreChannelMissingItems() []DiscordChannelMappingItem {
	items := make([]DiscordChannelMappingItem, 0, len(coreChannelSpecs()))
	for _, spec := range coreChannelSpecs() {
		items = append(items, missingMappingItem(spec.MappingKind, spec.VictoryScopeKind, spec.VictoryScopeSlug, spec.ExpectedName))
	}
	return items
}

func defaultVenueMissingItems() []DiscordChannelMappingItem {
	items := make([]DiscordChannelMappingItem, 0, len(venueCategorySpecs()))
	for _, spec := range venueCategorySpecs() {
		items = append(items, missingMappingItem(spec.MappingKind, spec.VictoryScopeKind, spec.VictoryScopeSlug, spec.ExpectedName))
	}
	return items
}

func mappingStatusForSpec(rows map[string]discordChannelMappingRow, channels []discordChannel, spec discordChannelMappingSpec) DiscordChannelMappingItem {
	row, ok := rows[mappingKey(spec.MappingKind, spec.VictoryScopeKind, spec.VictoryScopeSlug)]
	if !ok {
		return missingMappingItem(spec.MappingKind, spec.VictoryScopeKind, spec.VictoryScopeSlug, spec.ExpectedName)
	}
	if live := findDiscordChannelByID(channels, row.DiscordChannelID); live != nil {
		return DiscordChannelMappingItem{
			MappingKind:            spec.MappingKind,
			VictoryScopeKind:       spec.VictoryScopeKind,
			VictoryScopeSlug:       spec.VictoryScopeSlug,
			ExpectedName:           spec.ExpectedName,
			ActualName:             live.Name,
			DiscordChannelID:       live.ID,
			DiscordChannelType:     channelTypeLabel(live.Type),
			ParentDiscordChannelID: strings.TrimSpace(live.ParentID),
			Status:                 statusForLiveChannel(spec, live, row),
			LastVerifiedAt:         formatTimePtr(row.LastVerifiedAt),
		}
	}
	return DiscordChannelMappingItem{
		MappingKind:            spec.MappingKind,
		VictoryScopeKind:       spec.VictoryScopeKind,
		VictoryScopeSlug:       spec.VictoryScopeSlug,
		ExpectedName:           spec.ExpectedName,
		ActualName:             row.DiscordChannelName,
		DiscordChannelID:       row.DiscordChannelID,
		DiscordChannelType:     row.DiscordChannelType,
		ParentDiscordChannelID: row.ParentDiscordChannelID,
		Status:                 "missing",
		LastVerifiedAt:         formatTimePtr(row.LastVerifiedAt),
	}
}

func statusForLiveChannel(spec discordChannelMappingSpec, live *discordChannel, row discordChannelMappingRow) string {
	if live == nil {
		return "missing"
	}
	if strings.TrimSpace(live.Name) != strings.TrimSpace(row.DiscordChannelName) || channelTypeLabel(live.Type) != strings.TrimSpace(row.DiscordChannelType) {
		return "updated"
	}
	return "found"
}

func findDiscordChannelByID(channels []discordChannel, id string) *discordChannel {
	id = strings.TrimSpace(id)
	for i := range channels {
		if strings.TrimSpace(channels[i].ID) == id {
			return &channels[i]
		}
	}
	return nil
}

func nowPtr() *time.Time {
	now := time.Now().UTC()
	return &now
}

func formatTimePtr(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func coreChannelSpecs() []discordChannelMappingSpec {
	return []discordChannelMappingSpec{
		{MappingKind: discordChannelMappingKindCoreChannel, VictoryScopeKind: discordChannelScopeKindCoreChannel, VictoryScopeSlug: "victory-system", ExpectedName: "victory-system", ChannelType: discordChannelTypeText},
		{MappingKind: discordChannelMappingKindCoreChannel, VictoryScopeKind: discordChannelScopeKindCoreChannel, VictoryScopeSlug: "victory-announcements", ExpectedName: "victory-announcements", ChannelType: discordChannelTypeText},
		{MappingKind: discordChannelMappingKindCoreChannel, VictoryScopeKind: discordChannelScopeKindCoreChannel, VictoryScopeSlug: "victory-lobby", ExpectedName: "victory-lobby", ChannelType: discordChannelTypeText},
		{MappingKind: discordChannelMappingKindCoreChannel, VictoryScopeKind: discordChannelScopeKindCoreChannel, VictoryScopeSlug: "victory-support", ExpectedName: "victory-support", ChannelType: discordChannelTypeText},
	}
}

func venueCategorySpecs() []discordChannelMappingSpec {
	return []discordChannelMappingSpec{
		{MappingKind: discordChannelMappingKindVenueCategory, VictoryScopeKind: discordChannelScopeKindVenue, VictoryScopeSlug: "the-cave", ExpectedName: "The Cave", ChannelType: discordChannelTypeCategory},
		{MappingKind: discordChannelMappingKindVenueCategory, VictoryScopeKind: discordChannelScopeKindVenue, VictoryScopeSlug: "first-theater", ExpectedName: "First Theater", ChannelType: discordChannelTypeCategory},
		{MappingKind: discordChannelMappingKindVenueCategory, VictoryScopeKind: discordChannelScopeKindVenue, VictoryScopeSlug: "middle-school-stage", ExpectedName: "Middle School Stage", ChannelType: discordChannelTypeCategory},
		{MappingKind: discordChannelMappingKindVenueCategory, VictoryScopeKind: discordChannelScopeKindVenue, VictoryScopeSlug: "producers-office", ExpectedName: "Producer's Office", ChannelType: discordChannelTypeCategory},
		{MappingKind: discordChannelMappingKindVenueCategory, VictoryScopeKind: discordChannelScopeKindVenue, VictoryScopeSlug: "directors-chair", ExpectedName: "The Director's Chair", ChannelType: discordChannelTypeCategory},
		{MappingKind: discordChannelMappingKindVenueCategory, VictoryScopeKind: discordChannelScopeKindVenue, VictoryScopeSlug: "audition-hall", ExpectedName: "Audition Hall", ChannelType: discordChannelTypeCategory},
		{MappingKind: discordChannelMappingKindVenueCategory, VictoryScopeKind: discordChannelScopeKindVenue, VictoryScopeSlug: "greenroom", ExpectedName: "The Greenroom", ChannelType: discordChannelTypeCategory},
		{MappingKind: discordChannelMappingKindVenueCategory, VictoryScopeKind: discordChannelScopeKindVenue, VictoryScopeSlug: "trailers", ExpectedName: "Trailers", ChannelType: discordChannelTypeCategory},
		{MappingKind: discordChannelMappingKindVenueCategory, VictoryScopeKind: discordChannelScopeKindVenue, VictoryScopeSlug: "workshop", ExpectedName: "Workshop", ChannelType: discordChannelTypeCategory},
	}
}

func (s *DiscordChannelMappingRepairSummary) add(action, name string) {
	switch action {
	case "created":
		s.Created = append(s.Created, name)
	case "updated":
		s.Updated = append(s.Updated, name)
	default:
		s.Found = append(s.Found, name)
	}
}
