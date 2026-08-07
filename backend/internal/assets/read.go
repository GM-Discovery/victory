package assets

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"victory/backend/internal/access"
	"victory/backend/internal/ewrite"
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

		if wantContent {
			log.Printf("asset content request start: asset=%s variant=%s", assetID, r.URL.Query().Get("variant"))
			rec, err := loadAssetContentRecord(ctx, pool, assetID)
			if err != nil {
				log.Printf("asset content load failed: asset=%s err=%v", assetID, err)
				if err == pgx.ErrNoRows {
					serveConstructionFallback(w, r)
					return
				}
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "asset_lookup_failed"})
				return
			}

			if rec.MissingAsset {
				log.Printf("asset content missing asset fallback: asset=%s", assetID)
				serveConstructionFallback(w, r)
				return
			}

			contentPath, contentType := resolveAssetContentPathFromParts(rec.Variants, rec.OriginalPath, rec.SourceMime, rec.SniffedMime, r.URL.Query().Get("variant"))
			log.Printf("asset content resolved: asset=%s path=%s type=%s variants=%d", assetID, contentPath, contentType, len(rec.Variants))
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

			// Kernel 79: every content request must be authorized, not just
			// map assets. Prior to this, a non-map asset (including every
			// eWrite-embedded image) served with zero access check.
			warehouseRec, whErr := loadWarehouseAssetRecord(ctx, pool, assetID, true)
			if whErr != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"ok":    false,
					"error": "asset_access_check_failed",
				})
				return
			}
			allowed, err := userCanReadAssetConsideringEwrite(ctx, pool, userID, warehouseRec)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"ok":    false,
					"error": "asset_access_check_failed",
				})
				return
			}
			if !allowed && strings.EqualFold(strings.TrimSpace(rec.AssetType), "map") {
				venueMapVisible, venueErr := assetIsActiveTheaterMap(ctx, pool, assetID)
				if venueErr != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{
						"ok":    false,
						"error": "asset_access_check_failed",
					})
					return
				}
				venueAccessible, venueErr := userCanAccessTheaterMapVenue(ctx, pool, userID)
				if venueErr != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{
						"ok":    false,
						"error": "asset_access_check_failed",
					})
					return
				}
				allowed = venueMapVisible && venueAccessible
			}
			if !allowed {
				writeJSON(w, http.StatusForbidden, map[string]any{
					"ok":    false,
					"error": "forbidden",
				})
				return
			}

			file, openErr := os.Open(filepath.Clean(contentPath))
			if openErr != nil {
				log.Printf("asset content open failed: asset=%s path=%s err=%v", assetID, contentPath, openErr)
				writeJSON(w, http.StatusNotFound, map[string]any{
					"ok":    false,
					"error": "asset_content_not_found",
				})
				return
			}
			defer file.Close()

			if stat, statErr := file.Stat(); statErr != nil || stat.IsDir() {
				writeJSON(w, http.StatusNotFound, map[string]any{
					"ok":    false,
					"error": "asset_content_not_found",
				})
				return
			}

			w.Header().Set("Content-Type", contentType)
			w.Header().Set("Cache-Control", "private, max-age=0, must-revalidate")
			w.WriteHeader(http.StatusOK)
			if _, err := io.Copy(w, file); err != nil {
				log.Printf("asset content copy failed: asset=%s path=%s err=%v", assetID, contentPath, err)
			}
			return
		}

		rec, err := loadWarehouseAssetRecord(ctx, pool, assetID, true)
		if err != nil {
			if err == pgx.ErrNoRows {
				writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "asset_not_found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "asset_lookup_failed"})
			return
		}

		allowed, err := userCanReadAssetConsideringEwrite(ctx, pool, userID, rec)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "asset_access_check_failed",
			})
			return
		}
		if !allowed {
			venueMapVisible, venueErr := assetIsActiveTheaterMap(ctx, pool, assetID)
			if venueErr != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"ok":    false,
					"error": "asset_access_check_failed",
				})
				return
			}
			venueAccessible, venueErr := userCanAccessTheaterMapVenue(ctx, pool, userID)
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

		// Kernel 79: visibility can change over time (draft->published,
		// unpublish, an eWrite reference added/removed) -- never let an
		// intermediary cache a stale access decision.
		w.Header().Set("Cache-Control", "private, max-age=0, must-revalidate")
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

func assetIsActiveTheaterMap(ctx context.Context, pool *pgxpool.Pool, assetID string) (bool, error) {
	var found bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM venue_active_maps vm
			JOIN venues v ON v.id = vm.venue_id
			WHERE vm.asset_id = $1
			  AND v.slug IN ('first-theater', 'catharsis')
		)
	`, assetID).Scan(&found)
	if err != nil {
		return false, err
	}
	return found, nil
}

func userCanAccessTheaterMapVenue(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, error) {
	for _, slug := range []string{"first-theater", "catharsis"} {
		allowed, err := access.UserCanAccessVenueSlug(ctx, pool, userID, slug)
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}
	return false, nil
}

type assetContentRecord struct {
	AssetType    string
	MissingAsset bool
	OriginalPath string
	SourceMime   string
	SniffedMime  string
	Variants     []map[string]any
}

func loadAssetContentRecord(ctx context.Context, pool *pgxpool.Pool, assetID string) (assetContentRecord, error) {
	var rec assetContentRecord
	var deleted bool
	err := pool.QueryRow(ctx, `
		SELECT
			COALESCE(NULLIF(a.asset_type, ''), 'generic'),
			COALESCE(a.deleted_at IS NOT NULL, FALSE),
			COALESCE(NULLIF(a.original_path, ''), ''),
			COALESCE(a.source_mime, ''),
			COALESCE(a.sniffed_mime, '')
		FROM assets a
		WHERE a.id = $1
		LIMIT 1
	`, assetID).Scan(
		&rec.AssetType,
		&deleted,
		&rec.OriginalPath,
		&rec.SourceMime,
		&rec.SniffedMime,
	)
	if err != nil {
		return assetContentRecord{}, err
	}
	if deleted {
		rec.MissingAsset = true
	}

	rows, err := pool.Query(ctx, `
		SELECT variant_key, mime, path
		FROM asset_derivatives
		WHERE asset_id = $1
		ORDER BY width ASC
	`, assetID)
	if err != nil {
		return assetContentRecord{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var variantKey, mime, path string
		if err := rows.Scan(&variantKey, &mime, &path); err != nil {
			return assetContentRecord{}, err
		}
		rec.Variants = append(rec.Variants, map[string]any{
			"variant_key": variantKey,
			"mime":        mime,
			"path":        path,
			"url":         "/api/assets/" + assetID + "/content?variant=" + variantKey,
		})
	}
	if err := rows.Err(); err != nil {
		return assetContentRecord{}, err
	}
	return rec, nil
}

func resolveAssetContentPath(rec warehouseAssetRecord, variant string) (string, string) {
	return resolveAssetContentPathFromParts(rec.Variants, rec.OriginalPath, rec.SourceMime, rec.SniffedMime, variant)
}

func resolveAssetContentPathFromParts(variants []map[string]any, originalPath, sourceMime, sniffedMime, variant string) (string, string) {
	variant = strings.ToLower(strings.TrimSpace(variant))
	switch variant {
	case "thumbnail", "stage", "master", "original":
		for _, item := range variants {
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
			for _, item := range variants {
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

	if originalPath != "" {
		contentType := sourceMime
		if strings.TrimSpace(contentType) == "" {
			contentType = sniffedMime
		}
		return originalPath, contentType
	}

	for _, item := range variants {
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

// userCanReadAssetConsideringEwrite is the entry point every asset read
// (content and meta) must go through. Kernel 79: if an asset is referenced
// by one or more eWrite publications (ewrite_publication_assets), its
// access is governed by those publications' own visibility via
// ewrite.CanReadPublication instead of the asset's incidental location
// membership -- this closes both directions of the Kernel 78 gap: a
// public/authenticated eWriting's image is readable by any signed-in
// reader regardless of location membership, and a draft/production-scoped
// eWriting's image is denied to anyone without editor authority or
// production membership, even if they hold unrelated membership at the
// asset's own location. Ownership (producer/uploader/owner) always keeps
// access, matching userCanReadAsset's existing precedent. Assets not
// referenced by any eWrite publication are completely unaffected --
// unchanged legacy behavior.
func userCanReadAssetConsideringEwrite(ctx context.Context, pool *pgxpool.Pool, userID string, rec warehouseAssetRecord) (bool, error) {
	if userID == rec.ProducerUserID || userID == rec.UploaderUserID || userID == rec.OwnerUserID {
		return true, nil
	}

	pubIDs, err := ewritePublicationsForAsset(ctx, pool, rec.ID)
	if err != nil {
		return false, err
	}
	if len(pubIDs) > 0 {
		for _, pubID := range pubIDs {
			pub, err := ewrite.LoadPublication(ctx, pool, pubID)
			if err != nil {
				continue
			}
			if ok, err := ewrite.CanReadPublication(ctx, pool, userID, pub); err == nil && ok {
				return true, nil
			}
		}
		return false, nil
	}

	// Kernel 81: a card-image asset's real authority is the Storyboard it's
	// pinned to, not incidental location membership at whatever fallback
	// storage location the upload was scoped to (see
	// storyboards/card_image.go's resolveCardImageStorageScope) -- a
	// Storyboard grant holder with no location relationship to the board
	// owner would otherwise be 403'd loading a card image they're fully
	// authorized to see on the board itself. Found via real Playwright
	// testing with a throwaway crew account that had no location
	// memberships at all. This is a raw-SQL inline check, not an import of
	// the storyboards package, because storyboards already imports assets
	// (for the upload helper) -- assets importing storyboards back would
	// close a direct two-package cycle.
	if allowed, err := storyboardCardImageViewerAllowed(ctx, pool, userID, rec.ID); err != nil {
		return false, err
	} else if allowed {
		return true, nil
	}

	return userCanReadAsset(ctx, pool, userID, rec.ProducerUserID, rec.UploaderUserID, rec.OwnerUserID, rec.LocationID)
}

// storyboardCardImageViewerAllowed mirrors storyboards/authority.go's
// CanViewBoard ("any tier with a resolved grant or ownership") without
// importing that package. Returns false, nil (not an error) whenever
// assetID isn't a storyboard card's pinned image at all, so every other
// asset type falls through to the normal location-membership check
// unchanged.
func storyboardCardImageViewerAllowed(ctx context.Context, pool *pgxpool.Pool, userID, assetID string) (bool, error) {
	var boardID, ownerUserID string
	err := pool.QueryRow(ctx, `
		SELECT sb.id::text, sb.owner_user_id::text
		FROM storyboard_cards sc
		JOIN storyboards sb ON sb.id = sc.storyboard_id
		WHERE sc.image_asset_id = $1
		LIMIT 1
	`, assetID).Scan(&boardID, &ownerUserID)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if userID == ownerUserID {
		return true, nil
	}
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err == nil && ok {
		return true, nil
	}
	var grantedRole string
	err = pool.QueryRow(ctx, `
		SELECT granted_role FROM storyboard_grants WHERE storyboard_id = $1 AND user_id = $2
	`, boardID, userID).Scan(&grantedRole)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func ewritePublicationsForAsset(ctx context.Context, pool *pgxpool.Pool, assetID string) ([]string, error) {
	rows, err := pool.Query(ctx, `
		SELECT publication_id::text FROM ewrite_publication_assets WHERE asset_id = $1
	`, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
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
