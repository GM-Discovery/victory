package assets

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"victory/backend/internal/access"
	"victory/backend/internal/sessions"

	xdraw "golang.org/x/image/draw"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	DefaultWarehouseHardLimitBytes    = int64(8 * 1024 * 1024 * 1024)
	DefaultWarehouseMaxUploadBytes    = int64(25 * 1024 * 1024)
	DefaultTokenMasterMaxDimension    = 1024
	DefaultTokenStageMaxDimension     = 512
	DefaultTokenThumbnailMaxDimension = 128
	WarehousePhysicalReserveBytes     = int64(8 * 1024 * 1024 * 1024)
	constructionFallbackAssetPath     = "frontend/assets/construction.png"
)

func defaultWarehouseStorageSettings(locationID string) WarehouseStorageSettings {
	return WarehouseStorageSettings{
		LocationID:                 locationID,
		HardLimitBytes:             DefaultWarehouseHardLimitBytes,
		WarningThresholdPercent:    80,
		CriticalThresholdPercent:   90,
		MaxUploadBytes:             DefaultWarehouseMaxUploadBytes,
		RetainOriginalsDefault:     false,
		TokenMasterMaxDimension:    DefaultTokenMasterMaxDimension,
		TokenStageMaxDimension:     DefaultTokenStageMaxDimension,
		TokenThumbnailMaxDimension: DefaultTokenThumbnailMaxDimension,
		ImageQuality:               85,
	}
}

type WarehouseStorageSettings struct {
	LocationID                 string    `json:"location_id"`
	HardLimitBytes             int64     `json:"hard_limit_bytes"`
	WarningThresholdPercent    int       `json:"warning_threshold_percent"`
	CriticalThresholdPercent   int       `json:"critical_threshold_percent"`
	MaxUploadBytes             int64     `json:"max_upload_bytes"`
	RetainOriginalsDefault     bool      `json:"retain_originals_default"`
	TokenMasterMaxDimension    int       `json:"token_master_max_dimension"`
	TokenStageMaxDimension     int       `json:"token_stage_max_dimension"`
	TokenThumbnailMaxDimension int       `json:"token_thumbnail_max_dimension"`
	ImageQuality               int       `json:"image_quality"`
	UpdatedBy                  string    `json:"updated_by,omitempty"`
	UpdatedAt                  time.Time `json:"updated_at,omitempty"`
}

type warehouseAssetRecord struct {
	ID                string           `json:"id"`
	ProducerUserID    string           `json:"producer_user_id"`
	UploaderUserID    string           `json:"uploader_user_id"`
	OwnerUserID       string           `json:"owner_user_id"`
	LocationID        string           `json:"location_id"`
	OwnerState        string           `json:"owner_state"`
	AssetType         string           `json:"asset_type"`
	Name              string           `json:"name"`
	Shape             string           `json:"shape"`
	DefaultGridWidth  int              `json:"default_grid_width"`
	DefaultGridHeight int              `json:"default_grid_height"`
	RetainOriginal    bool             `json:"retain_original"`
	Status            string           `json:"status"`
	OriginalFilename  string           `json:"original_filename"`
	SourceMime        string           `json:"source_mime"`
	SniffedMime       string           `json:"sniffed_mime"`
	Width             int              `json:"width"`
	Height            int              `json:"height"`
	ByteSize          int64            `json:"byte_size"`
	StoredBytes       int64            `json:"stored_bytes"`
	ChecksumSHA256    []byte           `json:"-"`
	OriginalPath      string           `json:"-"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
	LastUsedAt        *time.Time       `json:"last_used_at,omitempty"`
	ReferenceCount    int              `json:"reference_count"`
	DeletedAt         *time.Time       `json:"deleted_at,omitempty"`
	MissingAsset      bool             `json:"missing_asset"`
	OriginalAssetID   string           `json:"original_asset_id,omitempty"`
	OriginalAssetName string           `json:"original_asset_name,omitempty"`
	Tags              []string         `json:"tags"`
	Variants          []map[string]any `json:"variants,omitempty"`
}

type warehouseAssetListItem struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	AssetType         string     `json:"asset_type"`
	Shape             string     `json:"shape"`
	Status            string     `json:"status"`
	DefaultGridWidth  int        `json:"default_grid_width"`
	DefaultGridHeight int        `json:"default_grid_height"`
	RetainOriginal    bool       `json:"retain_original"`
	OriginalFilename  string     `json:"original_filename"`
	SourceMime        string     `json:"source_mime"`
	SniffedMime       string     `json:"sniffed_mime"`
	Width             int        `json:"width"`
	Height            int        `json:"height"`
	ByteSize          int64      `json:"byte_size"`
	StoredBytes       int64      `json:"stored_bytes"`
	ReferenceCount    int        `json:"reference_count"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	LastUsedAt        *time.Time `json:"last_used_at,omitempty"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty"`
	ThumbnailURL      string     `json:"thumbnail_url"`
	ContentURL        string     `json:"content_url"`
}

type warehouseAssetStats struct {
	TotalStoredBytes int64  `json:"total_stored_bytes"`
	TotalLimitBytes  int64  `json:"total_limit_bytes"`
	PercentUsed      int    `json:"percent_used"`
	WarningState     string `json:"warning_state"`
	ActiveAssets     int64  `json:"active_assets"`
	TombstonedAssets int64  `json:"tombstoned_assets"`
	MapBytes         int64  `json:"map_bytes"`
	TokenBytes       int64  `json:"token_bytes"`
	OriginalsBytes   int64  `json:"originals_bytes"`
}

func defaultWarehouseAssetStats() warehouseAssetStats {
	return warehouseAssetStats{
		WarningState: "normal",
	}
}

type warehouseFilesystemStats struct {
	Path              string `json:"path"`
	TotalBytes        int64  `json:"total_bytes"`
	FreeBytes         int64  `json:"free_bytes"`
	AvailableBytes    int64  `json:"available_bytes"`
	UsedBytes         int64  `json:"used_bytes"`
	ReserveBytes      int64  `json:"reserve_bytes"`
	UsableUploadBytes int64  `json:"usable_upload_bytes"`
}

func EnsureKernel49WarehouseStorageSurface(ctx context.Context, pool *pgxpool.Pool) error {
	// Schema DDL lives in backend/migrations/053_kernel72_warehouse_surface_ddl.sql;
	// this bootstrap only seeds/repairs warehouse settings and asset metadata.
	var locationID string
	if err := pool.QueryRow(ctx, `
		SELECT id::text
		FROM locations
		WHERE slug = 'amurray-family'
		LIMIT 1
	`).Scan(&locationID); err != nil {
		return err
	}

	_, err := pool.Exec(ctx, `
		INSERT INTO warehouse_storage_settings (
			location_id,
			hard_limit_bytes,
			warning_threshold_percent,
			critical_threshold_percent,
			max_upload_bytes,
			retain_originals_default,
			token_master_max_dimension,
			token_stage_max_dimension,
			token_thumbnail_max_dimension,
			image_quality
		)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (location_id) DO NOTHING
	`, locationID, DefaultWarehouseHardLimitBytes, 80, 90, DefaultWarehouseMaxUploadBytes, false, DefaultTokenMasterMaxDimension, DefaultTokenStageMaxDimension, DefaultTokenThumbnailMaxDimension, 85)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		UPDATE warehouse_storage_settings
		SET hard_limit_bytes = $2
		WHERE location_id = $1::uuid
		  AND hard_limit_bytes = 16106127360
	`, locationID, DefaultWarehouseHardLimitBytes)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		UPDATE assets
		SET stored_bytes = COALESCE(stored_bytes, 0),
		    name = COALESCE(NULLIF(name, ''), COALESCE(NULLIF(original_filename, ''), id::text)),
		    shape = COALESCE(NULLIF(shape, ''), 'circle')
		WHERE COALESCE(stored_bytes, 0) = 0
	`)
	return err
}

func HandleWarehouseStorage(pool *pgxpool.Pool, storageRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, ok, err := requireWarehouseAccess(ctx, pool, r)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed"})
			return
		}
		if !ok {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		locationID, err := resolveWarehouseStorageLocationID(ctx, pool)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "location_lookup_failed"})
			return
		}

		settings, err := loadWarehouseStorageSettingsByLocationID(ctx, pool, locationID)
		if err != nil {
			settings = defaultWarehouseStorageSettings(locationID)
		}

		stats, err := loadWarehouseStorageStats(ctx, pool, locationID)
		if err != nil {
			stats = defaultWarehouseAssetStats()
		}
		stats.TotalLimitBytes = settings.HardLimitBytes
		stats.PercentUsed = 0
		if stats.TotalLimitBytes > 0 {
			stats.PercentUsed = int(math.Round((float64(stats.TotalStoredBytes) / float64(stats.TotalLimitBytes)) * 100))
		}
		switch {
		case stats.PercentUsed >= 100:
			stats.WarningState = "critical"
		case stats.PercentUsed >= 90:
			stats.WarningState = "critical"
		case stats.PercentUsed >= 80:
			stats.WarningState = "warning"
		default:
			stats.WarningState = "normal"
		}

		canManageHardLimit, canManagePolicy := warehouseStoragePermission(ctx, pool, userID)
		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"data": map[string]any{
				"settings":              settings,
				"stats":                 stats,
				"can_manage_hard_limit": canManageHardLimit,
				"can_manage_policy":     canManagePolicy,
			},
		})
	}
}

func HandleWarehouseFilesystemStorage(pool *pgxpool.Pool, storageRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		_, ok, err := requireWarehouseAccess(ctx, pool, r)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed"})
			return
		}
		if !ok {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		stats, err := loadWarehouseFilesystemStats(storageRoot)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "filesystem_stats_lookup_failed"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": stats})
	}
}

func resolveWarehouseStorageLocationID(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	var locationID string
	err := pool.QueryRow(ctx, `
		SELECT id::text
		FROM locations
		WHERE slug = 'amurray-family'
		LIMIT 1
	`).Scan(&locationID)
	if err != nil {
		return "", err
	}
	return locationID, nil
}

func HandleWarehouseStorageSettings(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, ok, err := requireWarehouseAccess(ctx, pool, r)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed"})
			return
		}
		if !ok {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		settings, err := loadWarehouseStorageSettings(ctx, pool)
		if err != nil {
			settings = defaultWarehouseStorageSettings("")
		}

		var req map[string]any
		if err := jsonDecode(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_json"})
			return
		}

		newSettings := settings
		if v, ok := req["hard_limit_bytes"]; ok {
			if !warehouseCanManageHardLimit(ctx, pool, userID) {
				writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
				return
			}
			if parsed, ok := parseInt64Value(v); ok && parsed > 0 {
				newSettings.HardLimitBytes = parsed
			}
		}
		if v, ok := req["max_upload_bytes"]; ok {
			if !warehouseCanManageHardLimit(ctx, pool, userID) {
				writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
				return
			}
			if parsed, ok := parseInt64Value(v); ok && parsed > 0 {
				newSettings.MaxUploadBytes = parsed
			}
		}
		if v, ok := req["warning_threshold_percent"]; ok {
			if parsed, ok := parseIntValue(v); ok {
				newSettings.WarningThresholdPercent = parsed
			}
		}
		if v, ok := req["critical_threshold_percent"]; ok {
			if parsed, ok := parseIntValue(v); ok {
				newSettings.CriticalThresholdPercent = parsed
			}
		}
		if v, ok := req["retain_originals_default"]; ok {
			if parsed, ok := parseBoolValue(v); ok {
				newSettings.RetainOriginalsDefault = parsed
			}
		}
		if v, ok := req["token_master_max_dimension"]; ok {
			if parsed, ok := parseIntValue(v); ok && parsed > 0 {
				newSettings.TokenMasterMaxDimension = parsed
			}
		}
		if v, ok := req["token_stage_max_dimension"]; ok {
			if parsed, ok := parseIntValue(v); ok && parsed > 0 {
				newSettings.TokenStageMaxDimension = parsed
			}
		}
		if v, ok := req["token_thumbnail_max_dimension"]; ok {
			if parsed, ok := parseIntValue(v); ok && parsed > 0 {
				newSettings.TokenThumbnailMaxDimension = parsed
			}
		}

		if newSettings.WarningThresholdPercent < 1 || newSettings.WarningThresholdPercent >= newSettings.CriticalThresholdPercent {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_warning_thresholds"})
			return
		}
		if newSettings.CriticalThresholdPercent <= newSettings.WarningThresholdPercent || newSettings.CriticalThresholdPercent > 100 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_critical_thresholds"})
			return
		}

		if newSettings.HardLimitBytes <= 0 {
			newSettings.HardLimitBytes = DefaultWarehouseHardLimitBytes
		}
		if newSettings.MaxUploadBytes <= 0 {
			newSettings.MaxUploadBytes = DefaultWarehouseMaxUploadBytes
		}
		if newSettings.TokenMasterMaxDimension <= 0 {
			newSettings.TokenMasterMaxDimension = DefaultTokenMasterMaxDimension
		}
		if newSettings.TokenStageMaxDimension <= 0 {
			newSettings.TokenStageMaxDimension = DefaultTokenStageMaxDimension
		}
		if newSettings.TokenThumbnailMaxDimension <= 0 {
			newSettings.TokenThumbnailMaxDimension = DefaultTokenThumbnailMaxDimension
		}

		if _, err := pool.Exec(ctx, `
			UPDATE warehouse_storage_settings
			SET hard_limit_bytes = $2,
			    warning_threshold_percent = $3,
			    critical_threshold_percent = $4,
			    max_upload_bytes = $5,
			    retain_originals_default = $6,
			    token_master_max_dimension = $7,
			    token_stage_max_dimension = $8,
			    token_thumbnail_max_dimension = $9,
			    updated_by = $10::uuid,
			    updated_at = NOW()
			WHERE location_id = $1::uuid
		`, newSettings.LocationID, newSettings.HardLimitBytes, newSettings.WarningThresholdPercent, newSettings.CriticalThresholdPercent, newSettings.MaxUploadBytes, newSettings.RetainOriginalsDefault, newSettings.TokenMasterMaxDimension, newSettings.TokenStageMaxDimension, newSettings.TokenThumbnailMaxDimension, userID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "storage_settings_update_failed"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": newSettings})
	}
}

func HandleWarehouseAssets(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		_, ok, err := requireWarehouseAccess(ctx, pool, r)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed"})
			return
		}
		if !ok {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		assetType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("asset_type")))
		status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
		search := strings.TrimSpace(r.URL.Query().Get("search"))

		query := `
			SELECT
				a.id::text,
				COALESCE(NULLIF(a.name, ''), COALESCE(NULLIF(a.original_filename, ''), a.id::text)) AS name,
				COALESCE(NULLIF(a.asset_type, ''), 'generic') AS asset_type,
				COALESCE(NULLIF(a.shape, ''), 'circle') AS shape,
				COALESCE(NULLIF(a.status, ''), 'active') AS status,
				COALESCE(a.default_grid_width, 1),
				COALESCE(a.default_grid_height, 1),
				COALESCE(a.retain_original, FALSE),
				COALESCE(NULLIF(a.original_filename, ''), ''),
				COALESCE(a.source_mime, ''),
				COALESCE(a.sniffed_mime, ''),
				COALESCE(a.width, 0),
				COALESCE(a.height, 0),
				COALESCE(a.byte_size, 0),
				COALESCE(a.stored_bytes, 0),
				COALESCE((SELECT count(*) FROM venue_active_maps vm WHERE vm.asset_id = a.id), 0),
				a.created_at,
				a.updated_at,
				a.last_used_at,
				a.deleted_at
			FROM assets a
			WHERE a.location_id = (SELECT id FROM locations WHERE slug = 'amurray-family' LIMIT 1)
		`

		args := make([]any, 0, 3)
		argN := 1
		if assetType != "" {
			query += fmt.Sprintf(" AND COALESCE(NULLIF(a.asset_type, ''), 'generic') = $%d", argN)
			args = append(args, assetType)
			argN++
		}
		if status != "" {
			if status == "deleted" || status == "tombstoned" {
				query += " AND (COALESCE(a.is_deleted, FALSE) = TRUE OR a.deleted_at IS NOT NULL)"
			} else if status == "active" {
				query += " AND COALESCE(a.is_deleted, FALSE) = FALSE AND a.deleted_at IS NULL"
			} else {
				query += fmt.Sprintf(" AND COALESCE(NULLIF(a.status, ''), 'active') = $%d", argN)
				args = append(args, status)
				argN++
			}
		}
		if search != "" {
			query += fmt.Sprintf(` AND (
				COALESCE(NULLIF(a.name, ''), '') ILIKE $%d OR
				COALESCE(NULLIF(a.original_filename, ''), '') ILIKE $%d OR
				COALESCE(a.tags, '{}'::text[])::text ILIKE $%d
			)`, argN, argN, argN)
			args = append(args, "%"+search+"%")
		}
		query += " ORDER BY a.created_at DESC"

		rows, err := pool.Query(ctx, query, args...)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "asset_lookup_failed"})
			return
		}
		defer rows.Close()

		items := make([]warehouseAssetListItem, 0)
		for rows.Next() {
			var item warehouseAssetListItem
			if err := rows.Scan(
				&item.ID,
				&item.Name,
				&item.AssetType,
				&item.Shape,
				&item.Status,
				&item.DefaultGridWidth,
				&item.DefaultGridHeight,
				&item.RetainOriginal,
				&item.OriginalFilename,
				&item.SourceMime,
				&item.SniffedMime,
				&item.Width,
				&item.Height,
				&item.ByteSize,
				&item.StoredBytes,
				&item.ReferenceCount,
				&item.CreatedAt,
				&item.UpdatedAt,
				&item.LastUsedAt,
				&item.DeletedAt,
			); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "asset_scan_failed"})
				return
			}
			item.ContentURL = "/api/assets/" + item.ID + "/content"
			item.ThumbnailURL = "/api/assets/" + item.ID + "/content?variant=thumbnail"
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "asset_lookup_failed"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": items})
	}
}

func HandleWarehouseAssetByID(pool *pgxpool.Pool, storageRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		assetID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/warehouse/assets/"))
		if assetID == "" || strings.Contains(assetID, "/") {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "asset_id_required"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		_, ok, err := requireWarehouseAccess(ctx, pool, r)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "access_check_failed"})
			return
		}
		if !ok {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "forbidden"})
			return
		}

		switch r.Method {
		case http.MethodGet:
			rec, err := loadWarehouseAssetRecord(ctx, pool, assetID, true)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "asset_not_found"})
					return
				}
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "asset_lookup_failed"})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": rec})
			return
		case http.MethodDelete:
			recovered, err := tombstoneWarehouseAsset(ctx, pool, storageRoot, assetID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "asset_not_found"})
					return
				}
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "asset_delete_failed"})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": recovered})
			return
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}
	}
}

func HandleTokenUploadAsset(pool *pgxpool.Pool, storageRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method_not_allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()

		sessionCookie, err := sessions.ReadSessionCookie(r)
		if err != nil || strings.TrimSpace(sessionCookie) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		rec, err := sessions.GetSessionByRawToken(ctx, pool, sessionCookie)
		if err != nil || strings.TrimSpace(rec.UserID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "not_authenticated"})
			return
		}

		userID := rec.UserID
		ok, producerUserID, locationID, err := resolveProducerScope(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "producer_scope_lookup_failed"})
			return
		}
		if !ok {
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "producer_membership_required"})
			return
		}

		settings, err := loadWarehouseStorageSettingsByLocationID(ctx, pool, locationID)
		if err != nil {
			settings = defaultWarehouseStorageSettings(locationID)
		}

		r.Body = http.MaxBytesReader(w, r.Body, settings.MaxUploadBytes)
		if err := r.ParseMultipartForm(settings.MaxUploadBytes); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_or_oversize_multipart"})
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "file_required"})
			return
		}
		defer file.Close()

		data, err := readBounded(file, settings.MaxUploadBytes)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "file_read_failed"})
			return
		}

		detectedMime, err := sniffAllowedMime(data)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
			return
		}

		srcImg, width, height, err := decodeImage(data, detectedMime)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "image_decode_failed"})
			return
		}
		if width <= 0 || height <= 0 || width > 8192 || height > 8192 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "image_dimensions_out_of_range"})
			return
		}

		tokenReq := parseTokenUploadRequest(r, header.Filename, settings.RetainOriginalsDefault)
		if tokenReq.Name == "" {
			tokenReq.Name = inheritedAssetName(header.Filename)
		}

		checksum := sha256.Sum256(data)
		duplicateWarning, _ := findDuplicateAssetID(ctx, pool, checksum[:], locationID)

		tempDir, err := os.MkdirTemp(storageRoot, "warehouse-token-*")
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "storage_create_failed"})
			return
		}
		defer func() {
			_ = os.RemoveAll(tempDir)
		}()

		prepared, err := prepareTokenVariantFiles(tempDir, srcImg, tokenReq, settings)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
			return
		}

		storedBytes := int64(0)
		for _, file := range prepared.files {
			storedBytes += int64(len(file.bytes))
		}
		if tokenReq.RetainOriginal {
			storedBytes += int64(len(data))
		}

		if errorCode, err := checkWarehouseUploadCapacity(ctx, pool, locationID, storageRoot, storedBytes, settings.HardLimitBytes); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": errorCode})
			return
		} else if errorCode != "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": errorCode})
			return
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "tx_begin_failed"})
			return
		}
		defer tx.Rollback(ctx)

		assetName := tokenReq.Name
		var assetID string
		err = tx.QueryRow(ctx, `
			INSERT INTO assets (
				producer_user_id,
				location_id,
				uploader_user_id,
				owner_user_id,
				owner_state,
				asset_type,
				name,
				shape,
				default_grid_width,
				default_grid_height,
				retain_original,
				status,
				crop_x,
				crop_y,
				zoom,
				original_filename,
				source_ext,
				source_mime,
				sniffed_mime,
				width,
				height,
				byte_size,
				stored_bytes,
				checksum_sha256,
				storage_root,
				original_path
			)
			VALUES (
				$1, $2, $3, $3, 'uploader_owned',
				'token', $4, $5, 1, 1,
				$6, 'active', $7, $8, $9,
				$10, $11, $12, $13, $14, $15, $16, $17, $18, $19, ''
			)
			RETURNING id
		`,
			producerUserID,
			locationID,
			userID,
			assetName,
			tokenReq.Shape,
			tokenReq.RetainOriginal,
			tokenReq.CropX,
			tokenReq.CropY,
			tokenReq.Zoom,
			header.Filename,
			normalizeSourceExt(header.Filename, detectedMime),
			header.Header.Get("Content-Type"),
			detectedMime,
			width,
			height,
			len(data),
			storedBytes,
			checksum[:],
			storageRoot,
		).Scan(&assetID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "asset_insert_failed"})
			return
		}

		assetDir := filepath.Join(storageRoot, "producers", producerUserID, "assets", assetID)
		if err := os.MkdirAll(assetDir, 0o755); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "storage_create_failed"})
			return
		}

		if tokenReq.RetainOriginal {
			originalPath := filepath.Join(assetDir, "original"+normalizeSourceExt(header.Filename, detectedMime))
			if err := os.WriteFile(originalPath, data, 0o644); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "original_write_failed"})
				return
			}
			if _, err := tx.Exec(ctx, `UPDATE assets SET original_path = $2 WHERE id = $1`, assetID, originalPath); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "asset_update_failed"})
				return
			}
		}

		for _, file := range prepared.files {
			outPath := filepath.Join(assetDir, file.filename)
			if err := os.WriteFile(outPath, file.bytes, 0o644); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "variant_write_failed"})
				return
			}
			sum := sha256.Sum256(file.bytes)
			_, err = tx.Exec(ctx, `
				INSERT INTO asset_derivatives (
					asset_id,
					variant_key,
					width,
					height,
					mime,
					byte_size,
					checksum_sha256,
					path
				)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`, assetID, file.variantKey, file.width, file.height, file.mime, len(file.bytes), sum[:], outPath)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "variant_insert_failed"})
				return
			}
		}

		if _, err := tx.Exec(ctx, `
			UPDATE assets
			SET stored_bytes = $2,
			    updated_at = NOW()
			WHERE id = $1
		`, assetID, storedBytes); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "asset_update_failed"})
			return
		}

		if err := tx.Commit(ctx); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "tx_commit_failed"})
			return
		}

		response := map[string]any{
			"asset_id":     assetID,
			"asset_type":   "token",
			"shape":        tokenReq.Shape,
			"stored_bytes": storedBytes,
			"content_url":  "/api/assets/" + assetID + "/content?variant=stage",
		}
		if duplicateWarning != "" {
			response["duplicate_warning"] = duplicateWarning
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": response})
	}
}

func HandleWorkshopMapUpload(ctx context.Context, pool *pgxpool.Pool, storageRoot string, r *http.Request) (map[string]any, int, error) {
	_ = ctx
	_ = pool
	_ = storageRoot
	_ = r
	return nil, 0, nil
}

func loadWarehouseStorageSettings(ctx context.Context, pool *pgxpool.Pool) (WarehouseStorageSettings, error) {
	var settings WarehouseStorageSettings
	err := pool.QueryRow(ctx, `
		SELECT
			l.id::text,
			COALESCE(s.hard_limit_bytes, $2),
			COALESCE(s.warning_threshold_percent, 80),
			COALESCE(s.critical_threshold_percent, 90),
			COALESCE(s.max_upload_bytes, $3),
			COALESCE(s.retain_originals_default, FALSE),
			COALESCE(s.token_master_max_dimension, $4),
			COALESCE(s.token_stage_max_dimension, $5),
			COALESCE(s.token_thumbnail_max_dimension, $6),
			COALESCE(s.image_quality, 85),
			COALESCE(s.updated_by::text, ''),
			COALESCE(s.updated_at, NOW())
		FROM locations l
		LEFT JOIN warehouse_storage_settings s ON s.location_id = l.id
		WHERE l.slug = 'amurray-family'
		LIMIT 1
	`, DefaultWarehouseHardLimitBytes, DefaultWarehouseMaxUploadBytes, DefaultTokenMasterMaxDimension, DefaultTokenStageMaxDimension, DefaultTokenThumbnailMaxDimension).Scan(
		&settings.LocationID,
		&settings.HardLimitBytes,
		&settings.WarningThresholdPercent,
		&settings.CriticalThresholdPercent,
		&settings.MaxUploadBytes,
		&settings.RetainOriginalsDefault,
		&settings.TokenMasterMaxDimension,
		&settings.TokenStageMaxDimension,
		&settings.TokenThumbnailMaxDimension,
		&settings.ImageQuality,
		&settings.UpdatedBy,
		&settings.UpdatedAt,
	)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return defaultWarehouseStorageSettings(""), nil
	}
	return settings, err
}

func loadWarehouseStorageSettingsByLocationID(ctx context.Context, pool *pgxpool.Pool, locationID string) (WarehouseStorageSettings, error) {
	var settings WarehouseStorageSettings
	err := pool.QueryRow(ctx, `
		SELECT
			location_id::text,
			hard_limit_bytes,
			warning_threshold_percent,
			critical_threshold_percent,
			max_upload_bytes,
			retain_originals_default,
			token_master_max_dimension,
			token_stage_max_dimension,
			token_thumbnail_max_dimension,
			image_quality,
			COALESCE(updated_by::text, ''),
			updated_at
		FROM warehouse_storage_settings
		WHERE location_id = $1::uuid
		LIMIT 1
	`, locationID).Scan(
		&settings.LocationID,
		&settings.HardLimitBytes,
		&settings.WarningThresholdPercent,
		&settings.CriticalThresholdPercent,
		&settings.MaxUploadBytes,
		&settings.RetainOriginalsDefault,
		&settings.TokenMasterMaxDimension,
		&settings.TokenStageMaxDimension,
		&settings.TokenThumbnailMaxDimension,
		&settings.ImageQuality,
		&settings.UpdatedBy,
		&settings.UpdatedAt,
	)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return WarehouseStorageSettings{
			LocationID:                 locationID,
			HardLimitBytes:             DefaultWarehouseHardLimitBytes,
			WarningThresholdPercent:    80,
			CriticalThresholdPercent:   90,
			MaxUploadBytes:             DefaultWarehouseMaxUploadBytes,
			RetainOriginalsDefault:     false,
			TokenMasterMaxDimension:    DefaultTokenMasterMaxDimension,
			TokenStageMaxDimension:     DefaultTokenStageMaxDimension,
			TokenThumbnailMaxDimension: DefaultTokenThumbnailMaxDimension,
			ImageQuality:               85,
		}, nil
	}
	return settings, err
}

func loadWarehouseStorageStats(ctx context.Context, pool *pgxpool.Pool, locationID string) (warehouseAssetStats, error) {
	var stats warehouseAssetStats
	err := pool.QueryRow(ctx, `
		SELECT
			COALESCE(sum(CASE WHEN COALESCE(a.is_deleted, FALSE) = FALSE AND a.deleted_at IS NULL THEN a.stored_bytes ELSE 0 END), 0),
			COUNT(*) FILTER (WHERE COALESCE(a.is_deleted, FALSE) = FALSE AND a.deleted_at IS NULL),
			COUNT(*) FILTER (WHERE COALESCE(a.is_deleted, FALSE) = TRUE OR a.deleted_at IS NOT NULL),
			COALESCE(sum(CASE WHEN COALESCE(a.is_deleted, FALSE) = FALSE AND a.deleted_at IS NULL AND COALESCE(a.asset_type, 'generic') = 'map' THEN a.stored_bytes ELSE 0 END), 0),
			COALESCE(sum(CASE WHEN COALESCE(a.is_deleted, FALSE) = FALSE AND a.deleted_at IS NULL AND COALESCE(a.asset_type, 'generic') = 'token' THEN a.stored_bytes ELSE 0 END), 0),
			COALESCE(sum(CASE WHEN COALESCE(a.is_deleted, FALSE) = FALSE AND a.deleted_at IS NULL AND a.retain_original THEN a.byte_size ELSE 0 END), 0)
		FROM assets a
		WHERE a.location_id = $1::uuid
	`, locationID).Scan(&stats.TotalStoredBytes, &stats.ActiveAssets, &stats.TombstonedAssets, &stats.MapBytes, &stats.TokenBytes, &stats.OriginalsBytes)
	if err != nil {
		return warehouseAssetStats{}, err
	}
	return stats, nil
}

func loadWarehouseFilesystemStats(storageRoot string) (warehouseFilesystemStats, error) {
	root := strings.TrimSpace(storageRoot)
	if root == "" {
		root = "/opt/victory/storage"
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(root, &stat); err != nil {
		return warehouseFilesystemStats{}, err
	}

	blockSize := int64(stat.Bsize)
	totalBytes := int64(stat.Blocks) * blockSize
	freeBytes := int64(stat.Bfree) * blockSize
	availableBytes := int64(stat.Bavail) * blockSize
	usedBytes := totalBytes - freeBytes
	usableUploadBytes := availableBytes - WarehousePhysicalReserveBytes
	if usableUploadBytes < 0 {
		usableUploadBytes = 0
	}

	return warehouseFilesystemStats{
		Path:              root,
		TotalBytes:        totalBytes,
		FreeBytes:         freeBytes,
		AvailableBytes:    availableBytes,
		UsedBytes:         usedBytes,
		ReserveBytes:      WarehousePhysicalReserveBytes,
		UsableUploadBytes: usableUploadBytes,
	}, nil
}

func checkWarehouseUploadCapacity(ctx context.Context, pool *pgxpool.Pool, locationID, storageRoot string, additionalBytes, hardLimitBytes int64) (string, error) {
	usage, err := loadWarehouseStoredBytes(ctx, pool, locationID)
	if err != nil {
		return "storage_usage_lookup_failed", err
	}
	if usage+additionalBytes > hardLimitBytes {
		return "warehouse_capacity_exceeded", nil
	}

	filesystemStats, err := loadWarehouseFilesystemStats(storageRoot)
	if err != nil {
		return "filesystem_stats_lookup_failed", err
	}
	if filesystemStats.AvailableBytes-additionalBytes < WarehousePhysicalReserveBytes {
		return "warehouse_physical_reserve_exceeded", nil
	}

	return "", nil
}

func loadWarehouseStoredBytes(ctx context.Context, pool *pgxpool.Pool, locationID string) (int64, error) {
	var storedBytes int64
	err := pool.QueryRow(ctx, `
		SELECT COALESCE(sum(COALESCE(stored_bytes, 0)), 0)
		FROM assets
		WHERE location_id = $1::uuid
		  AND COALESCE(is_deleted, FALSE) = FALSE
		  AND deleted_at IS NULL
	`, locationID).Scan(&storedBytes)
	return storedBytes, err
}

func requireWarehouseAccess(ctx context.Context, pool *pgxpool.Pool, r *http.Request) (string, bool, error) {
	rawSession, err := sessions.ReadSessionCookie(r)
	if err != nil || strings.TrimSpace(rawSession) == "" {
		return "", false, nil
	}

	rec, err := sessions.GetSessionByRawToken(ctx, pool, rawSession)
	if err != nil || strings.TrimSpace(rec.UserID) == "" {
		return "", false, nil
	}

	userID := rec.UserID
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err == nil && ok {
		return userID, true, nil
	} else if err != nil {
		return "", false, err
	}

	role, err := access.CurrentLocationRole(ctx, pool, userID)
	if err != nil {
		return "", false, err
	}
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "producer", "director":
		return userID, true, nil
	default:
		return userID, false, nil
	}
}

func warehouseStoragePermission(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, bool) {
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err == nil && ok {
		return true, true
	}
	if role, err := access.CurrentLocationRole(ctx, pool, userID); err == nil {
		switch strings.ToLower(strings.TrimSpace(role)) {
		case "producer":
			return false, true
		case "director":
			return false, true
		}
	}
	return false, false
}

func warehouseCanManageHardLimit(ctx context.Context, pool *pgxpool.Pool, userID string) bool {
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err == nil && ok {
		return true
	}
	return false
}

func inheritedAssetName(filename string) string {
	base := strings.TrimSpace(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
	if base == "" {
		base = strings.TrimSpace(filepath.Base(filename))
	}
	if base == "" {
		return "Untitled Token"
	}
	return base
}

type tokenUploadRequest struct {
	Name           string
	Shape          string
	CropX          float64
	CropY          float64
	Zoom           float64
	RetainOriginal bool
}

func parseTokenUploadRequest(r *http.Request, fallbackName string, defaultRetain bool) tokenUploadRequest {
	req := tokenUploadRequest{
		Name:           strings.TrimSpace(r.FormValue("name")),
		Shape:          normalizeTokenShape(r.FormValue("shape")),
		CropX:          parseFloatFormValue(r.FormValue("crop_x"), 0.5),
		CropY:          parseFloatFormValue(r.FormValue("crop_y"), 0.5),
		Zoom:           parseFloatFormValue(r.FormValue("zoom"), 1),
		RetainOriginal: defaultRetain,
	}
	if req.Name == "" {
		req.Name = inheritedAssetName(fallbackName)
	}
	if raw := strings.TrimSpace(r.FormValue("retain_original")); raw != "" {
		if parsed, ok := parseBoolString(raw); ok {
			req.RetainOriginal = parsed
		}
	}
	req.CropX = clampFloat(req.CropX, 0, 1, 0.5)
	req.CropY = clampFloat(req.CropY, 0, 1, 0.5)
	req.Zoom = clampFloat(req.Zoom, 0.25, 4, 1)
	return req
}

func normalizeTokenShape(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "square", "hex", "raw":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "circle"
	}
}

type preparedTokenVariant struct {
	variantKey string
	filename   string
	width      int
	height     int
	mime       string
	bytes      []byte
}

type preparedTokenBundle struct {
	files []preparedTokenVariant
}

func prepareTokenVariantFiles(tempDir string, src image.Image, req tokenUploadRequest, settings WarehouseStorageSettings) (preparedTokenBundle, error) {
	masterSize := maxInt(settings.TokenMasterMaxDimension, 1)
	stageSize := maxInt(settings.TokenStageMaxDimension, 1)
	thumbSize := maxInt(settings.TokenThumbnailMaxDimension, 1)

	specs := []struct {
		key  string
		size int
	}{
		{key: "master", size: masterSize},
		{key: "stage", size: stageSize},
		{key: "thumbnail", size: thumbSize},
	}

	bundle := preparedTokenBundle{files: make([]preparedTokenVariant, 0, len(specs))}
	for _, spec := range specs {
		img, width, height, err := renderTokenVariant(src, req.Shape, spec.size, req.Zoom, req.CropX, req.CropY)
		if err != nil {
			return preparedTokenBundle{}, err
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return preparedTokenBundle{}, errors.New("variant_encode_failed")
		}
		bundle.files = append(bundle.files, preparedTokenVariant{
			variantKey: spec.key,
			filename:   spec.key + ".png",
			width:      width,
			height:     height,
			mime:       "image/png",
			bytes:      buf.Bytes(),
		})
	}
	_ = tempDir
	return bundle, nil
}

func renderTokenVariant(src image.Image, shape string, maxDim int, zoom, cropX, cropY float64) (image.Image, int, int, error) {
	if maxDim <= 0 {
		maxDim = 1
	}
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW <= 0 || srcH <= 0 {
		return nil, 0, 0, errors.New("invalid_source_image")
	}

	scale := zoom * float64(maxDim) / float64(maxInt(srcW, srcH))
	if scale <= 0 {
		scale = float64(maxDim) / float64(maxInt(srcW, srcH))
	}
	scaledW := maxInt(1, int(math.Round(float64(srcW)*scale)))
	scaledH := maxInt(1, int(math.Round(float64(srcH)*scale)))
	scaled := image.NewNRGBA(image.Rect(0, 0, scaledW, scaledH))
	xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), src, bounds, xdraw.Over, nil)

	if strings.EqualFold(shape, "raw") {
		canvas := image.NewNRGBA(image.Rect(0, 0, scaledW, scaledH))
		draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.NRGBA{0, 0, 0, 0}), image.Point{}, draw.Src)
		blitTokenImage(canvas, scaled, cropX, cropY)
		return canvas, scaledW, scaledH, nil
	}

	canvas := image.NewNRGBA(image.Rect(0, 0, maxDim, maxDim))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.NRGBA{0, 0, 0, 0}), image.Point{}, draw.Src)
	blitTokenImage(canvas, scaled, cropX, cropY)
	_ = applyTokenMask(canvas, normalizeTokenShape(shape))
	return canvas, maxDim, maxDim, nil
}

func blitTokenImage(dst *image.NRGBA, src *image.NRGBA, cropX, cropY float64) {
	bounds := dst.Bounds()
	sw := src.Bounds().Dx()
	sh := src.Bounds().Dy()
	dw := bounds.Dx()
	dh := bounds.Dy()
	if sw <= 0 || sh <= 0 || dw <= 0 || dh <= 0 {
		return
	}

	offsetX := int(math.Round((cropX - 0.5) * float64(dw)))
	offsetY := int(math.Round((cropY - 0.5) * float64(dh)))
	destRect := image.Rect((dw-sw)/2+offsetX, (dh-sh)/2+offsetY, (dw-sw)/2+offsetX+sw, (dh-sh)/2+offsetY+sh)
	draw.Draw(dst, destRect, src, image.Point{}, draw.Over)
}

func applyTokenMask(dst *image.NRGBA, shape string) error {
	bounds := dst.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= 0 || h <= 0 {
		return nil
	}

	switch shape {
	case "circle":
		cx := float64(w) / 2
		cy := float64(h) / 2
		r := math.Min(cx, cy)
		r2 := r * r
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dx := float64(x) + 0.5 - cx
				dy := float64(y) + 0.5 - cy
				if dx*dx+dy*dy > r2 {
					i := dst.PixOffset(x, y)
					dst.Pix[i+3] = 0
				}
			}
		}
	case "hex":
		pts := hexMaskPoints(w, h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if !pointInPolygon(float64(x)+0.5, float64(y)+0.5, pts) {
					i := dst.PixOffset(x, y)
					dst.Pix[i+3] = 0
				}
			}
		}
	}
	return nil
}

func hexMaskPoints(w, h int) []image.Point {
	if w <= 0 || h <= 0 {
		return nil
	}
	return []image.Point{
		{X: int(math.Round(float64(w) * 0.25)), Y: 0},
		{X: int(math.Round(float64(w) * 0.75)), Y: 0},
		{X: w - 1, Y: h / 2},
		{X: int(math.Round(float64(w) * 0.75)), Y: h - 1},
		{X: int(math.Round(float64(w) * 0.25)), Y: h - 1},
		{X: 0, Y: h / 2},
	}
}

func pointInPolygon(x, y float64, points []image.Point) bool {
	if len(points) < 3 {
		return false
	}
	inside := false
	j := len(points) - 1
	for i := 0; i < len(points); i++ {
		xi := float64(points[i].X)
		yi := float64(points[i].Y)
		xj := float64(points[j].X)
		yj := float64(points[j].Y)
		intersects := ((yi > y) != (yj > y)) && (x < (xj-xi)*(y-yi)/(yj-yi)+xi)
		if intersects {
			inside = !inside
		}
		j = i
	}
	return inside
}

func findDuplicateAssetID(ctx context.Context, pool *pgxpool.Pool, checksum []byte, locationID string) (string, error) {
	var assetID string
	err := pool.QueryRow(ctx, `
		SELECT id::text
		FROM assets
		WHERE checksum_sha256 = $1
		  AND location_id = $2::uuid
		  AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
	`, checksum, locationID).Scan(&assetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return assetID, nil
}

func loadWarehouseAssetRecord(ctx context.Context, pool *pgxpool.Pool, assetID string, includeDeleted bool) (warehouseAssetRecord, error) {
	var rec warehouseAssetRecord
	query := `
		SELECT
			a.id::text,
			a.producer_user_id::text,
			a.uploader_user_id::text,
			a.owner_user_id::text,
			a.location_id::text,
			a.owner_state::text,
			COALESCE(NULLIF(a.asset_type, ''), 'generic'),
			COALESCE(NULLIF(a.name, ''), COALESCE(NULLIF(a.original_filename, ''), a.id::text)),
			COALESCE(NULLIF(a.shape, ''), 'circle'),
			COALESCE(a.default_grid_width, 1),
			COALESCE(a.default_grid_height, 1),
			COALESCE(a.retain_original, FALSE),
			COALESCE(NULLIF(a.status, ''), 'active'),
			COALESCE(NULLIF(a.original_filename, ''), ''),
			COALESCE(a.source_mime, ''),
			COALESCE(a.sniffed_mime, ''),
			COALESCE(a.width, 0),
			COALESCE(a.height, 0),
			COALESCE(a.byte_size, 0),
			COALESCE(a.stored_bytes, 0),
			COALESCE(a.checksum_sha256, '\x'::bytea),
			COALESCE(a.original_path, ''),
			a.created_at,
			a.updated_at,
			a.last_used_at,
			a.deleted_at,
			COALESCE(a.tags, '{}'::text[])
		FROM assets a
		WHERE a.id = $1
	`
	if !includeDeleted {
		query += " AND COALESCE(a.is_deleted, FALSE) = FALSE AND a.deleted_at IS NULL"
	}
	query += " LIMIT 1"

	err := pool.QueryRow(ctx, query, assetID).Scan(
		&rec.ID,
		&rec.ProducerUserID,
		&rec.UploaderUserID,
		&rec.OwnerUserID,
		&rec.LocationID,
		&rec.OwnerState,
		&rec.AssetType,
		&rec.Name,
		&rec.Shape,
		&rec.DefaultGridWidth,
		&rec.DefaultGridHeight,
		&rec.RetainOriginal,
		&rec.Status,
		&rec.OriginalFilename,
		&rec.SourceMime,
		&rec.SniffedMime,
		&rec.Width,
		&rec.Height,
		&rec.ByteSize,
		&rec.StoredBytes,
		&rec.ChecksumSHA256,
		&rec.OriginalPath,
		&rec.CreatedAt,
		&rec.UpdatedAt,
		&rec.LastUsedAt,
		&rec.DeletedAt,
		&rec.Tags,
	)
	if err != nil {
		return warehouseAssetRecord{}, err
	}

	if rec.DeletedAt != nil || strings.EqualFold(strings.TrimSpace(rec.Status), "deleted") {
		rec.MissingAsset = true
		rec.OriginalAssetID = rec.ID
		rec.OriginalAssetName = rec.Name
	}

	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(count(*), 0)
		FROM venue_active_maps
		WHERE asset_id = $1
	`, assetID).Scan(&rec.ReferenceCount); err != nil {
		rec.ReferenceCount = 0
	}

	rows, err := pool.Query(ctx, `
		SELECT variant_key, width, height, mime, path
		FROM asset_derivatives
		WHERE asset_id = $1
		ORDER BY width ASC
	`, assetID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var variantKey, mime, path string
			var width, height int
			if scanErr := rows.Scan(&variantKey, &width, &height, &mime, &path); scanErr != nil {
				return warehouseAssetRecord{}, scanErr
			}
			rec.Variants = append(rec.Variants, map[string]any{
				"variant_key": variantKey,
				"width":       width,
				"height":      height,
				"mime":        mime,
				"path":        path,
				"url":         "/api/assets/" + assetID + "/content?variant=" + variantKey,
			})
		}
	}

	return rec, nil
}

func tombstoneWarehouseAsset(ctx context.Context, pool *pgxpool.Pool, storageRoot, assetID string) (map[string]any, error) {
	rec, err := loadWarehouseAssetRecord(ctx, pool, assetID, true)
	if err != nil {
		return nil, err
	}

	recoveredBytes := rec.StoredBytes
	if recoveredBytes <= 0 {
		recoveredBytes = rec.ByteSize
	}

	if rec.OriginalPath != "" {
		_ = os.Remove(rec.OriginalPath)
	}
	rows, err := pool.Query(ctx, `SELECT path FROM asset_derivatives WHERE asset_id = $1`, assetID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var path string
			if scanErr := rows.Scan(&path); scanErr == nil && strings.TrimSpace(path) != "" {
				_ = os.Remove(path)
			}
		}
	}

	if _, err := pool.Exec(ctx, `
		UPDATE assets
		SET deleted_at = NOW(),
		    is_deleted = TRUE,
		    status = 'deleted',
		    stored_bytes = 0,
		    original_path = '',
		    updated_at = NOW()
		WHERE id = $1
	`, assetID); err != nil {
		return nil, err
	}
	if _, err := pool.Exec(ctx, `DELETE FROM asset_derivatives WHERE asset_id = $1`, assetID); err != nil {
		return nil, err
	}
	_ = storageRoot
	return map[string]any{
		"asset_id":            assetID,
		"asset_name":          rec.Name,
		"asset_type":          rec.AssetType,
		"recovered_bytes":     recoveredBytes,
		"reference_count":     rec.ReferenceCount,
		"missing_asset":       true,
		"original_asset_id":   rec.ID,
		"original_asset_name": rec.Name,
	}, nil
}

func parseFloatFormValue(value string, fallback float64) float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseInt64Value(value any) (int64, bool) {
	switch v := value.(type) {
	case float64:
		return int64(v), true
	case float32:
		return int64(v), true
	case int:
		return int64(v), true
	case int64:
		return v, true
	case json.Number:
		parsed, err := v.Int64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func parseIntValue(value any) (int, bool) {
	v, ok := parseInt64Value(value)
	return int(v), ok
}

func parseBoolValue(value any) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		return parseBoolString(v)
	default:
		return false, false
	}
}

func parseBoolString(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "on":
		return true, true
	case "false", "0", "no", "off":
		return false, true
	default:
		return false, false
	}
}

func clampFloat(value, minValue, maxValue, fallback float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fallback
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func jsonDecode(r *http.Request, target any) error {
	return json.NewDecoder(r.Body).Decode(target)
}
