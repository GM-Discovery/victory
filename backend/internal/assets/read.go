package assets

import (
	"context"
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

		var id, originalFilename, sourceMime, sniffedMime, ownerState, assetType, originalPath string
		var width, height, byteSize int
		var producerUserID, uploaderUserID, ownerUserID, locationID string
		var tags []string

		err = pool.QueryRow(ctx, `
			SELECT
				id,
				producer_user_id,
				uploader_user_id,
				owner_user_id,
				location_id,
				owner_state::text,
				COALESCE(NULLIF(asset_type, ''), 'generic') AS asset_type,
				COALESCE(tags, '{}'::text[]) AS tags,
				original_filename,
				source_mime,
				sniffed_mime,
				width,
				height,
				byte_size,
				COALESCE(original_path, '')
			FROM assets
			WHERE id = $1
			  AND is_deleted = FALSE
			LIMIT 1
		`, assetID).Scan(
			&id,
			&producerUserID,
			&uploaderUserID,
			&ownerUserID,
			&locationID,
			&ownerState,
			&assetType,
			&tags,
			&originalFilename,
			&sourceMime,
			&sniffedMime,
			&width,
			&height,
			&byteSize,
			&originalPath,
		)
		if err != nil {
			if err == pgx.ErrNoRows {
				writeJSON(w, http.StatusNotFound, map[string]any{
					"ok":    false,
					"error": "asset_not_found",
				})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "asset_lookup_failed",
			})
			return
		}

		if wantContent {
			if strings.EqualFold(strings.TrimSpace(assetType), "map") {
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

				contentPath := originalPath
				contentType := sourceMime
				if strings.TrimSpace(contentType) == "" {
					contentType = sniffedMime
				}
				var derivativePath, derivativeMime string
				if err := pool.QueryRow(ctx, `
					SELECT path, COALESCE(NULLIF(mime, ''), 'image/jpeg')
					FROM asset_derivatives
					WHERE asset_id = $1
					  AND variant_key = '1024'
					LIMIT 1
				`, assetID).Scan(&derivativePath, &derivativeMime); err == nil && strings.TrimSpace(derivativePath) != "" {
					contentPath = derivativePath
					contentType = derivativeMime
				}
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

		allowed, err := userCanReadAsset(ctx, pool, userID, producerUserID, uploaderUserID, ownerUserID, locationID)
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

		rows, err := pool.Query(ctx, `
			SELECT variant_key, width, height, mime
			FROM asset_derivatives
			WHERE asset_id = $1
			ORDER BY width ASC
		`, assetID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "asset_derivatives_lookup_failed",
			})
			return
		}
		defer rows.Close()

		var variants []map[string]any
		for rows.Next() {
			var variantKey, mime string
			var vWidth, vHeight int
			if err := rows.Scan(&variantKey, &vWidth, &vHeight, &mime); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"ok":    false,
					"error": "asset_derivatives_scan_failed",
				})
				return
			}
			variants = append(variants, map[string]any{
				"variant_key": variantKey,
				"width":       vWidth,
				"height":      vHeight,
				"mime":        mime,
			})
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"data": map[string]any{
				"id":                id,
				"producer_user_id":  producerUserID,
				"uploader_user_id":  uploaderUserID,
				"owner_user_id":     ownerUserID,
				"owner_state":       ownerState,
				"asset_type":        assetType,
				"tags":              tags,
				"original_filename": originalFilename,
				"source_mime":       sourceMime,
				"sniffed_mime":      sniffedMime,
				"width":             width,
				"height":            height,
				"byte_size":         byteSize,
				"content_url":       "/api/assets/" + id + "/content",
				"variants":          variants,
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
