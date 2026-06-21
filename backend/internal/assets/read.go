package assets

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"victory/backend/internal/access"
	"victory/backend/internal/sessions"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func HandleGetAssetMeta(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"ok":    false,
				"error": "method_not_allowed",
			})
			return
		}

		assetID, wantContent := parseAssetRequestPath(r.URL.Path)
		if assetID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": "asset_id_required",
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID := ""
		rawSession, err := sessions.ReadSessionCookie(r)
		if err == nil && strings.TrimSpace(rawSession) != "" {
			if rec, sessionErr := sessions.GetSessionByRawToken(ctx, pool, rawSession); sessionErr == nil && strings.TrimSpace(rec.UserID) != "" {
				userID = rec.UserID
			}
		}

		rec, err := loadWarehouseAssetRecord(ctx, pool, assetID, true)
		if err != nil {
			if err == pgx.ErrNoRows {
				if wantContent {
					serveConstructionFallback(w, r)
					return
				}
				writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "asset_not_found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "asset_lookup_failed"})
			return
		}

		if wantContent {
			if rec.MissingAsset {
				serveConstructionFallback(w, r)
				return
			}

			if strings.EqualFold(strings.TrimSpace(rec.AssetType), "map") {
				venueMapVisible, venueErr := assetIsActiveFirstTheaterMap(ctx, pool, assetID)
				if venueErr != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{
						"ok":    false,
						"error": "asset_access_check_failed",
					})
					return
				}
				venueAccessible, venueErr := access.UserCanAccessVenueSlug(ctx, pool, userID, "first-theater")
				if venueErr != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{
						"ok":    false,
						"error": "asset_access_check_failed",
					})
					return
				}
				if !(venueMapVisible && venueAccessible) {
					writeJSON(w, http.StatusForbidden, map[string]any{
						"ok":    false,
						"error": "forbidden",
					})
					return
				}

				contentPath, contentType := resolveAssetContentPath(rec, r.URL.Query().Get("variant"))
				if strings.TrimSpace(contentPath) == "" {
					writeJSON(w, http.StatusNotFound, map[string]any{
						"ok":    false,
						"error": "asset_content_not_found",
					})
					return
				}
				if stat, statErr := os.Stat(contentPath); statErr != nil || stat.IsDir() {
					writeJSON(w, http.StatusNotFound, map[string]any{
						"ok":    false,
						"error": "asset_content_not_found",
					})
					return
				}
				if strings.TrimSpace(contentType) == "" {
					contentType = "application/octet-stream"
				}
				w.Header().Set("Content-Type", contentType)
				w.Header().Set("Cache-Control", "private, max-age=0, must-revalidate")
				http.ServeFile(w, r, filepath.Clean(contentPath))
				return
			}
		}

		allowed, err := userCanReadAsset(ctx, pool, userID, rec.ProducerUserID, rec.UploaderUserID, rec.OwnerUserID, rec.LocationID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "asset_access_check_failed",
			})
			return
		}
		if !allowed {
			venueMapVisible, venueErr := assetIsActiveFirstTheaterMap(ctx, pool, assetID)
			if venueErr != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"ok":    false,
					"error": "asset_access_check_failed",
				})
				return
			}
			venueAccessible, venueErr := access.UserCanAccessVenueSlug(ctx, pool, userID, "first-theater")
			if venueErr != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"ok":    false,
					"error": "asset_access_check_failed",
				})
				return
			}
			if !(venueMapVisible && venueAccessible) {
				writeJSON(w, http.StatusForbidden, map[string]any{
					"ok":    false,
					"error": "forbidden",
				})
				return
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"data": map[string]any{
				"id":                  rec.ID,
				"producer_user_id":    rec.ProducerUserID,
				"uploader_user_id":    rec.UploaderUserID,
				"owner_user_id":       rec.OwnerUserID,
				"owner_state":         rec.OwnerState,
				"asset_type":          rec.AssetType,
				"name":                rec.Name,
				"shape":               rec.Shape,
				"default_grid_width":  rec.DefaultGridWidth,
				"default_grid_height": rec.DefaultGridHeight,
				"retain_original":     rec.RetainOriginal,
				"status":              rec.Status,
				"tags":                rec.Tags,
				"original_filename":   rec.OriginalFilename,
				"source_mime":         rec.SourceMime,
				"sniffed_mime":        rec.SniffedMime,
				"width":               rec.Width,
				"height":              rec.Height,
				"byte_size":           rec.ByteSize,
				"stored_bytes":        rec.StoredBytes,
				"content_url":         "/api/assets/" + rec.ID + "/content",
				"variants":            rec.Variants,
				"missing_asset":       rec.MissingAsset,
				"original_asset_id":   rec.OriginalAssetID,
				"original_asset_name": rec.OriginalAssetName,
				"last_used_at":        rec.LastUsedAt,
				"deleted_at":          rec.DeletedAt,
			},
		})
	}
}

func parseAssetRequestPath(path string) (assetID string, wantContent bool) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(path, "/api/assets/"))
	if trimmed == "" {
		return "", false
	}
	if strings.HasSuffix(trimmed, "/content") {
		return strings.TrimSuffix(trimmed, "/content"), true
	}
	return trimmed, false
}

func assetIsActiveFirstTheaterMap(ctx context.Context, pool *pgxpool.Pool, assetID string) (bool, error) {
	var found bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM venue_active_maps vm
			JOIN venues v ON v.id = vm.venue_id
			WHERE vm.asset_id = $1
			  AND v.slug = 'first-theater'
		)
	`, assetID).Scan(&found)
	if err != nil {
		return false, err
	}
	return found, nil
}

func resolveAssetContentPath(rec warehouseAssetRecord, variant string) (string, string) {
	variant = strings.ToLower(strings.TrimSpace(variant))
	switch variant {
	case "thumbnail", "stage", "master", "original":
		for _, item := range rec.Variants {
			if strings.EqualFold(strings.TrimSpace(fmt.Sprint(item["variant_key"])), variant) {
				path := strings.TrimSpace(fmt.Sprint(item["path"]))
				mime := strings.TrimSpace(fmt.Sprint(item["mime"]))
				if path != "" {
					return path, mime
				}
			}
		}
	}

	if variant == "" {
		for _, preferred := range []string{"stage", "master", "thumbnail", "original"} {
			for _, item := range rec.Variants {
				if strings.EqualFold(strings.TrimSpace(fmt.Sprint(item["variant_key"])), preferred) {
					path := strings.TrimSpace(fmt.Sprint(item["path"]))
					mime := strings.TrimSpace(fmt.Sprint(item["mime"]))
					if path != "" {
						return path, mime
					}
				}
			}
		}
	}

	if rec.OriginalPath != "" {
		contentType := rec.SourceMime
		if strings.TrimSpace(contentType) == "" {
			contentType = rec.SniffedMime
		}
		return rec.OriginalPath, contentType
	}

	for _, item := range rec.Variants {
		path := strings.TrimSpace(fmt.Sprint(item["path"]))
		mime := strings.TrimSpace(fmt.Sprint(item["mime"]))
		if path != "" {
			return path, mime
		}
	}

	return "", ""
}

func serveConstructionFallback(w http.ResponseWriter, r *http.Request) {
	if _, err := os.Stat(constructionFallbackAssetPath); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "asset_content_not_found"})
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=60")
	http.ServeFile(w, r, filepath.Clean(constructionFallbackAssetPath))
}

func userCanReadAsset(ctx context.Context, pool *pgxpool.Pool, userID, producerUserID, uploaderUserID, ownerUserID, locationID string) (bool, error) {
	if userID == producerUserID || userID == uploaderUserID || userID == ownerUserID {
		return true, nil
	}

	var found int

	// legacy/bootstrap location membership path
	err := pool.QueryRow(ctx, `
		SELECT 1
		FROM location_memberships
		WHERE user_id = $1
		  AND location_id = $2
		  AND active = TRUE
		  AND role IN ('producer', 'director', 'cast', 'crew')
		LIMIT 1
	`, userID, locationID).Scan(&found)
	if err == nil && found == 1 {
		return true, nil
	}
	if err != nil && err != pgx.ErrNoRows {
		return false, err
	}

	// kernel2 scoped membership path
	err = pool.QueryRow(ctx, `
		SELECT 1
		FROM memberships
		WHERE user_id = $1
		  AND location_id = $2
		  AND active = TRUE
		  AND role IN ('producer', 'director', 'cast', 'crew')
		LIMIT 1
	`, userID, locationID).Scan(&found)
	if err == nil && found == 1 {
		return true, nil
	}
	if err != nil && err != pgx.ErrNoRows {
		return false, err
	}

	return false, nil
}
