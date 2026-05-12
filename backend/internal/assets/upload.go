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
	MaxUploadBytes = 10 * 1024 * 1024
	MaxWidth       = 4096
	MaxHeight      = 4096
)

var DerivativeSizes = []int{512, 1024, 2048}

type UploadResponse struct {
	AssetID string `json:"asset_id"`
}

func HandleWorkshopUpload(pool *pgxpool.Pool, storageRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
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

		r.Body = http.MaxBytesReader(w, r.Body, MaxUploadBytes)
		if err := r.ParseMultipartForm(MaxUploadBytes); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"ok":    false,
				"error": "invalid_or_oversize_multipart",
			})
			return
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

		data, err := readBounded(file, MaxUploadBytes)
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
				$4, $5, $6, $7,
				$8, $9, $10, $11, $12, ''
			)
			RETURNING id
		`,
			producerUserID,
			locationID,
			userID,
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
				"ok":    false,
				"error": "asset_insert_failed",
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
				AssetID: assetID,
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

func resolveProducerScope(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, string, string, error) {
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err == nil && ok {
		var locationID string
		err = pool.QueryRow(ctx, `
			SELECT id
			FROM locations
			WHERE slug = 'amurray-family'
			LIMIT 1
		`).Scan(&locationID)
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
