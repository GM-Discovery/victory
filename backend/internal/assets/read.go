package assets

import (
	"context"
	"net/http"
	"strings"
	"time"

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

		assetID := strings.TrimPrefix(r.URL.Path, "/api/assets/")
		assetID = strings.TrimSpace(assetID)
		if assetID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": "asset_id_required",
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		rawSession, err := sessions.ReadSessionCookie(r)
		if err != nil || strings.TrimSpace(rawSession) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":    false,
				"error": "not_authenticated",
			})
			return
		}

		rec, err := sessions.GetSessionByRawToken(ctx, pool, rawSession)
		if err != nil || strings.TrimSpace(rec.UserID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":    false,
				"error": "not_authenticated",
			})
			return
		}

		userID := rec.UserID

		var id, originalFilename, sourceMime, sniffedMime, ownerState string
		var width, height, byteSize int
		var producerUserID, uploaderUserID, ownerUserID, locationID string

		err = pool.QueryRow(ctx, `
			SELECT
				id,
				producer_user_id,
				uploader_user_id,
				owner_user_id,
				location_id,
				owner_state::text,
				original_filename,
				source_mime,
				sniffed_mime,
				width,
				height,
				byte_size
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
			&originalFilename,
			&sourceMime,
			&sniffedMime,
			&width,
			&height,
			&byteSize,
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

		allowed, err := userCanReadAsset(ctx, pool, userID, producerUserID, uploaderUserID, ownerUserID, locationID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "asset_access_check_failed",
			})
			return
		}
		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":    false,
				"error": "forbidden",
			})
			return
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
				"original_filename": originalFilename,
				"source_mime":       sourceMime,
				"sniffed_mime":      sniffedMime,
				"width":             width,
				"height":            height,
				"byte_size":         byteSize,
				"variants":          variants,
			},
		})
	}
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
