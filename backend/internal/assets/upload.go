package assets

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"victory/backend/internal/access"
	"victory/backend/internal/identity"
	"victory/backend/internal/sessions"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/webp"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	MaxUploadBytes = 25 * 1024 * 1024
	MaxWidth       = 4096
	MaxHeight      = 4096
)

var DerivativeSizes = []int{512, 1024, 2048}

type UploadResponse struct {
	AssetID    string   `json:"asset_id"`
	AssetType  string   `json:"asset_type,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	ContentURL string   `json:"content_url,omitempty"`
}

type workshopAssetListItem struct {
	AssetID          string    `json:"asset_id"`
	OriginalFilename string    `json:"original_filename"`
	SourceMime       string    `json:"source_mime"`
	SniffedMime      string    `json:"sniffed_mime"`
	Width            int       `json:"width"`
	Height           int       `json:"height"`
	ByteSize         int       `json:"byte_size"`
	AssetType        string    `json:"asset_type"`
	Tags             []string  `json:"tags"`
	ContentURL       string    `json:"content_url"`
	CreatedAt        time.Time `json:"created_at"`
}

func HandleWorkshopUpload(pool *pgxpool.Pool, storageRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("asset_type")), "map") {
				handleWorkshopMapAssetList(w, r, pool)
				return
			}
			handleWorkshopAssetList(w, r, pool)
			return
		}
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"ok":    false,
				"error": "method_not_allowed",
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
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

		ok, producerUserID, locationID, err := resolveProducerScope(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "producer_scope_lookup_failed",
			})
			return
		}
		if !ok {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"ok":    false,
				"error": "producer_membership_required",
			})
			return
		}

		settings, err := loadWarehouseStorageSettingsByLocationID(ctx, pool, locationID)
		if err != nil {
			settings = WarehouseStorageSettings{
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

		r.Body = http.MaxBytesReader(w, r.Body, settings.MaxUploadBytes)
		if err := r.ParseMultipartForm(settings.MaxUploadBytes); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": "invalid_or_oversize_multipart",
			})
			return
		}

		assetType := normalizeAssetType(r.FormValue("asset_type"))
		if assetType == "token" {
			handleTokenWorkshopUpload(w, r, ctx, pool, storageRoot, userID, producerUserID, locationID, settings)
			return
		}

		tags := normalizeAssetTags(r.FormValue("tags"))
		if assetType == "map" && !containsString(tags, "map") {
			tags = append(tags, "map")
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": "file_required",
			})
			return
		}
		defer file.Close()

		data, err := readBounded(file, settings.MaxUploadBytes)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": "file_read_failed",
			})
			return
		}

		detectedMime, err := sniffAllowedMime(data)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": err.Error(),
			})
			return
		}

		img, width, height, err := decodeImage(data, detectedMime)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": "image_decode_failed",
			})
			return
		}

		if width <= 0 || height <= 0 || width > MaxWidth || height > MaxHeight {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": "image_dimensions_out_of_range",
			})
			return
		}

		checksum := sha256.Sum256(data)

		tx, err := pool.Begin(ctx)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "tx_begin_failed",
			})
			return
		}
		defer tx.Rollback(ctx)

		var assetID string
		var originalPath string
		sourceExt := normalizeSourceExt(header.Filename, detectedMime)

		err = tx.QueryRow(ctx, `
			INSERT INTO assets (
				producer_user_id,
				location_id,
				uploader_user_id,
				owner_user_id,
				owner_state,
				asset_type,
				tags,
				original_filename,
				source_ext,
				source_mime,
				sniffed_mime,
				width,
				height,
				byte_size,
				checksum_sha256,
				storage_root,
				original_path
			)
			VALUES (
				$1, $2, $3, $3, 'uploader_owned',
				$4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, ''
			)
			RETURNING id
		`,
			producerUserID,
			locationID,
			userID,
			assetType,
			tags,
			header.Filename,
			sourceExt,
			header.Header.Get("Content-Type"),
			detectedMime,
			width,
			height,
			len(data),
			checksum[:],
			storageRoot,
		).Scan(&assetID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":     false,
				"error":  "asset_insert_failed",
				"detail": err.Error(),
			})
			return
		}

		assetDir := filepath.Join(storageRoot, "producers", producerUserID, "assets", assetID)
		origDir := filepath.Join(assetDir, "original")
		derivedDir := filepath.Join(assetDir, "derived")

		if err := os.MkdirAll(origDir, 0o755); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "storage_create_failed",
			})
			return
		}
		if err := os.MkdirAll(derivedDir, 0o755); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "storage_create_failed",
			})
			return
		}

		originalPath = filepath.Join(origDir, "source"+sourceExt)
		if err := os.WriteFile(originalPath, data, 0o644); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "original_write_failed",
			})
			return
		}

		_, err = tx.Exec(ctx, `
			UPDATE assets
			SET original_path = $2,
			    updated_at = NOW()
			WHERE id = $1
		`, assetID, originalPath)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "asset_update_failed",
			})
			return
		}

		for _, size := range DerivativeSizes {
			dstImg, actualW, actualH := resizeToMax(img, size)

			outPath := filepath.Join(derivedDir, fmt.Sprintf("%d.jpg", size))
			var buf bytes.Buffer
			if err := jpeg.Encode(&buf, dstImg, &jpeg.Options{Quality: 85}); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"ok":    false,
					"error": "derivative_encode_failed",
				})
				return
			}

			if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"ok":    false,
					"error": "derivative_write_failed",
				})
				return
			}

			sum := sha256.Sum256(buf.Bytes())

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
			`, assetID, fmt.Sprintf("%d", size), actualW, actualH, "image/jpeg", len(buf.Bytes()), sum[:], outPath)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"ok":    false,
					"error": "derivative_insert_failed",
				})
				return
			}
		}

		var storedBytes int64 = int64(len(data))
		for _, size := range DerivativeSizes {
			var variantBytes int64
			if err := tx.QueryRow(ctx, `
					SELECT COALESCE(byte_size, 0)
				FROM asset_derivatives
				WHERE asset_id = $1
				  AND variant_key = $2
				LIMIT 1
				`, assetID, fmt.Sprintf("%d", size)).Scan(&variantBytes); err == nil {
				storedBytes += variantBytes
			}
		}
		if errorCode, err := checkWarehouseUploadCapacity(ctx, pool, locationID, storageRoot, storedBytes, settings.HardLimitBytes); err != nil {
			_ = os.RemoveAll(assetDir)
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": errorCode,
			})
			return
		} else if errorCode != "" {
			_ = os.RemoveAll(assetDir)
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": errorCode,
			})
			return
		}
		if _, err := tx.Exec(ctx, `
				UPDATE assets
				SET stored_bytes = $2,
				    name = $3,
			    shape = 'raw',
			    default_grid_width = 1,
			    default_grid_height = 1,
			    retain_original = TRUE,
			    status = 'active',
			    crop_x = 0.5,
			    crop_y = 0.5,
			    zoom = 1,
			    updated_at = NOW()
			WHERE id = $1
		`, assetID, storedBytes, inheritedAssetName(header.Filename)); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "asset_update_failed",
			})
			return
		}

		if err := tx.Commit(ctx); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "tx_commit_failed",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"data": UploadResponse{
				AssetID:    assetID,
				AssetType:  assetType,
				Tags:       tags,
				ContentURL: "/api/assets/" + assetID + "/content",
			},
		})
	}
}

func handleWorkshopAssetList(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
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

	sessionID, err := identity.ResolveActiveCaveSessionID(ctx, pool, rec.UserID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"ok":    false,
			"error": "no_active_session",
		})
		return
	}

	var locationID string
	err = pool.QueryRow(ctx, `
		SELECT l.id::text
		FROM sessions s
		JOIN venues v ON v.id = s.venue_id
		JOIN lots lo ON lo.id = v.lot_id
		JOIN locations l ON l.id = lo.location_id
		WHERE s.id = $1
		LIMIT 1
	`, sessionID).Scan(&locationID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"ok":    false,
			"error": "session_not_found",
		})
		return
	}

	rows, err := pool.Query(ctx, `
		SELECT
			e.id::text,
			e.slug,
			e.name,
			e.element_type,
			COALESCE(NULLIF(e.context_class, ''), '') AS context_class,
			e.data
		FROM libraries l
		JOIN elements e ON e.library_id = l.id
		WHERE l.location_id = $1
		  AND COALESCE(e.state::text, '') <> 'deleted'
		  AND e.element_type IN ('index_card', 'prop', 'scenery', 'surface', 'actor', 'media')
		ORDER BY e.element_type, e.name
	`, locationID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "asset_lookup_failed",
		})
		return
	}
	defer rows.Close()

	type workshopAsset struct {
		ElementID    string         `json:"element_id"`
		Slug         string         `json:"slug"`
		Name         string         `json:"name"`
		ElementType  string         `json:"element_type"`
		ContextClass string         `json:"context_class"`
		ThumbnailURL string         `json:"thumbnail_url"`
		Data         map[string]any `json:"data"`
	}

	assets := make([]workshopAsset, 0)
	for rows.Next() {
		var item workshopAsset
		var dataRaw []byte
		if err := rows.Scan(&item.ElementID, &item.Slug, &item.Name, &item.ElementType, &item.ContextClass, &dataRaw); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "asset_scan_failed",
			})
			return
		}
		if err := json.Unmarshal(dataRaw, &item.Data); err != nil {
			item.Data = map[string]any{}
		}
		if item.ContextClass == "" {
			if item.ElementType == "index_card" {
				item.ContextClass = "card"
			} else {
				item.ContextClass = item.ElementType
			}
		}
		if strings.TrimSpace(item.ThumbnailURL) == "" {
			item.ThumbnailURL = "/api/assets/" + item.ElementID + "/content?variant=thumbnail"
		}
		assets = append(assets, item)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "asset_lookup_failed",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": assets,
	})
}

func handleTokenWorkshopUpload(w http.ResponseWriter, r *http.Request, ctx context.Context, pool *pgxpool.Pool, storageRoot string, userID string, producerUserID string, locationID string, settings WarehouseStorageSettings) {
	if r.MultipartForm == nil {
		r.Body = http.MaxBytesReader(w, r.Body, settings.MaxUploadBytes)
		if err := r.ParseMultipartForm(settings.MaxUploadBytes); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": "invalid_or_oversize_multipart",
			})
			return
		}
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": "file_required",
		})
		return
	}
	defer file.Close()

	data, err := readBounded(file, settings.MaxUploadBytes)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": "file_read_failed",
		})
		return
	}

	detectedMime, err := sniffAllowedMime(data)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	srcImg, width, height, err := decodeImage(data, detectedMime)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": "image_decode_failed",
		})
		return
	}
	if width <= 0 || height <= 0 || width > 8192 || height > 8192 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": "image_dimensions_out_of_range",
		})
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

	usage, err := loadWarehouseStoredBytes(ctx, pool, locationID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "storage_usage_lookup_failed"})
		return
	}
	if usage+storedBytes > settings.HardLimitBytes {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "warehouse_capacity_exceeded"})
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
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":     false,
			"error":  "asset_insert_failed",
			"detail": err.Error(),
		})
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

func handleWorkshopMapAssetList(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
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

	ok, _, locationID, err := resolveProducerScope(ctx, pool, rec.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "producer_scope_lookup_failed",
		})
		return
	}
	if !ok {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"ok":    false,
			"error": "producer_membership_required",
		})
		return
	}

	rows, err := pool.Query(ctx, `
		SELECT
			id::text,
			COALESCE(NULLIF(original_filename, ''), id::text),
			COALESCE(source_mime, ''),
			COALESCE(sniffed_mime, ''),
			COALESCE(width, 0),
			COALESCE(height, 0),
			COALESCE(byte_size, 0),
			COALESCE(NULLIF(asset_type, ''), 'generic'),
			COALESCE(tags, '{}'::text[]),
			created_at
		FROM assets
		WHERE location_id = $1
		  AND COALESCE(asset_type, 'generic') = 'map'
		  AND is_deleted = FALSE
		ORDER BY created_at DESC
	`, locationID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "asset_lookup_failed",
		})
		return
	}
	defer rows.Close()

	items := make([]workshopAssetListItem, 0)
	for rows.Next() {
		var item workshopAssetListItem
		if err := rows.Scan(
			&item.AssetID,
			&item.OriginalFilename,
			&item.SourceMime,
			&item.SniffedMime,
			&item.Width,
			&item.Height,
			&item.ByteSize,
			&item.AssetType,
			&item.Tags,
			&item.CreatedAt,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "asset_scan_failed",
			})
			return
		}
		item.ContentURL = "/api/assets/" + item.AssetID + "/content"
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "asset_lookup_failed",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"data": items,
	})
}

func resolveProducerScope(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, string, string, error) {
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err == nil && ok {
		var locationID string
		err = pool.QueryRow(ctx, `
			SELECT id
			FROM locations
			WHERE slug = $1
			LIMIT 1
		`, access.DefaultLocationSlug()).Scan(&locationID)
		if err != nil {
			return false, "", "", err
		}

		return true, userID, locationID, nil
	}

	var producerUserID string
	var locationID string

	err := pool.QueryRow(ctx, `
		SELECT lm.user_id, lm.location_id
		FROM location_memberships lm
		WHERE lm.user_id = $1
		  AND lm.role = 'producer'
		  AND lm.active = TRUE
		ORDER BY lm.created_at ASC
		LIMIT 1
	`, userID).Scan(&producerUserID, &locationID)
	if err != nil {
		return false, "", "", nil
	}

	return true, producerUserID, locationID, nil
}

func normalizeAssetType(value string) string {
	assetType := strings.ToLower(strings.TrimSpace(value))
	switch assetType {
	case "map":
		return "map"
	case "token":
		return "token"
	case "generic", "":
		return "generic"
	default:
		return "generic"
	}
}

func normalizeAssetTags(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\t'
	})
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		tag := strings.ToLower(strings.TrimSpace(part))
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	return out
}

func containsString(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == want {
			return true
		}
	}
	return false
}

func readBounded(file multipart.File, maxBytes int64) ([]byte, error) {
	var buf bytes.Buffer
	n, err := io.CopyN(&buf, file, maxBytes+1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if n > maxBytes {
		return nil, errors.New("file_too_large")
	}
	return buf.Bytes(), nil
}

func sniffAllowedMime(data []byte) (string, error) {
	mime := http.DetectContentType(data)

	switch mime {
	case "image/png", "image/jpeg", "image/webp":
		return mime, nil
	default:
		return "", errors.New("unsupported_file_type")
	}
}

func decodeImage(data []byte, detectedMime string) (image.Image, int, int, error) {
	switch detectedMime {
	case "image/png":
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return nil, 0, 0, err
		}
		img, err := png.Decode(bytes.NewReader(data))
		return img, cfg.Width, cfg.Height, err

	case "image/jpeg":
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return nil, 0, 0, err
		}
		img, err := jpeg.Decode(bytes.NewReader(data))
		return img, cfg.Width, cfg.Height, err

	case "image/webp":
		cfg, err := webp.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return nil, 0, 0, err
		}
		img, err := webp.Decode(bytes.NewReader(data))
		return img, cfg.Width, cfg.Height, err

	default:
		return nil, 0, 0, errors.New("unsupported_file_type")
	}
}

func normalizeSourceExt(filename, detectedMime string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp":
		return ext
	}

	switch detectedMime {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}

func resizeToMax(src image.Image, maxDim int) (image.Image, int, int) {
	b := src.Bounds()
	sw := b.Dx()
	sh := b.Dy()

	if sw <= maxDim && sh <= maxDim {
		dst := image.NewRGBA(image.Rect(0, 0, sw, sh))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
		return dst, sw, sh
	}

	var dw, dh int
	if sw >= sh {
		dw = maxDim
		dh = int(float64(sh) * (float64(maxDim) / float64(sw)))
	} else {
		dh = maxDim
		dw = int(float64(sw) * (float64(maxDim) / float64(sh)))
	}

	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	return dst, dw, dh
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
