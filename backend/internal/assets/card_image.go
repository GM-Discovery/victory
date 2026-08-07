package assets

// CreateReferencedImageAsset (Kernel 81) creates a single Victory asset
// from already-read image bytes for a feature that wants to attach one
// image somewhere it isn't itself Production-scoped -- Storyboards being
// the first example: boards are personal (no location/producer of their
// own, per Kernel 80), but every asset in this system is still stored and
// storage-quota-accounted against a producer + location (assetDir layout,
// checkWarehouseUploadCapacity). Callers resolve that storage scope
// themselves (storyboards uses the board owner's own producer membership,
// falling back the same way resolveProducerScope's Operator branch does)
// and must run their own feature-specific authority check before calling
// this -- there is no membership/role gate in here, unlike
// HandleWorkshopUpload's Producer-only requirement, which is exactly what
// made that handler unusable for a Crew-tier Storyboards card editor.
//
// Deliberately minimal next to HandleWorkshopUpload: one "thumbnail"
// derivative instead of three pixel-keyed ones, no crop/mask/token-shape
// pipeline. resolveAssetContentPathFromParts already falls back to the
// stored original for any variant name it doesn't recognize, so a caller
// can request ?variant=original for a full-size view with nothing extra
// needed here.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image/jpeg"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CardImageThumbnailMaxDimension bounds the one derivative this helper
// generates -- a card face thumbnail, not a token master/stage pipeline.
const CardImageThumbnailMaxDimension = 480

// CreatedImageAsset is what CreateReferencedImageAsset returns on success.
type CreatedImageAsset struct {
	AssetID    string
	ContentURL string
}

// CreateReferencedImageAsset validates, decodes, stores, and inserts one
// image asset row plus a "thumbnail" derivative. maxBytes bounds len(data)
// (callers should already have applied http.MaxBytesReader before reading
// the body; this is a second, cheap guard). assetType is stored verbatim
// (Storyboards passes "storyboard_card") purely for warehouse
// listing/accounting -- it carries no access-control meaning here.
func CreateReferencedImageAsset(
	ctx context.Context,
	pool *pgxpool.Pool,
	storageRoot string,
	uploaderUserID, producerUserID, locationID, assetType, originalFilename, contentType string,
	data []byte,
	maxBytes int64,
) (*CreatedImageAsset, error) {
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("file_too_large")
	}
	detectedMime, err := sniffAllowedMime(data)
	if err != nil {
		return nil, err
	}
	img, width, height, err := decodeImage(data, detectedMime)
	if err != nil {
		return nil, fmt.Errorf("image_decode_failed")
	}
	if width <= 0 || height <= 0 || width > MaxWidth || height > MaxHeight {
		return nil, fmt.Errorf("image_dimensions_out_of_range")
	}

	settings, err := loadWarehouseStorageSettingsByLocationID(ctx, pool, locationID)
	if err != nil {
		settings = WarehouseStorageSettings{
			LocationID:     locationID,
			HardLimitBytes: DefaultWarehouseHardLimitBytes,
			MaxUploadBytes: DefaultWarehouseMaxUploadBytes,
			ImageQuality:   85,
		}
	}
	if int64(len(data)) > settings.MaxUploadBytes {
		return nil, fmt.Errorf("file_too_large")
	}

	checksum := sha256.Sum256(data)
	sourceExt := normalizeSourceExt(originalFilename, detectedMime)

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var assetID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO assets (
			producer_user_id, location_id, uploader_user_id, owner_user_id, owner_state,
			asset_type, tags, original_filename, source_ext, source_mime, sniffed_mime,
			width, height, byte_size, checksum_sha256, storage_root, original_path
		)
		VALUES ($1, $2, $3, $3, 'uploader_owned', $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, '')
		RETURNING id
	`,
		producerUserID, locationID, uploaderUserID,
		assetType, []string{assetType}, originalFilename, sourceExt, contentType, detectedMime,
		width, height, len(data), checksum[:], storageRoot,
	).Scan(&assetID); err != nil {
		return nil, err
	}

	assetDir := filepath.Join(storageRoot, "producers", producerUserID, "assets", assetID)
	origDir := filepath.Join(assetDir, "original")
	derivedDir := filepath.Join(assetDir, "derived")
	if err := os.MkdirAll(origDir, 0o755); err != nil {
		return nil, fmt.Errorf("storage_create_failed")
	}
	if err := os.MkdirAll(derivedDir, 0o755); err != nil {
		return nil, fmt.Errorf("storage_create_failed")
	}

	originalPath := filepath.Join(origDir, "source"+sourceExt)
	if err := os.WriteFile(originalPath, data, 0o644); err != nil {
		return nil, fmt.Errorf("original_write_failed")
	}
	if _, err := tx.Exec(ctx, `UPDATE assets SET original_path = $2, updated_at = NOW() WHERE id = $1`, assetID, originalPath); err != nil {
		return nil, err
	}

	thumbImg, thumbW, thumbH := resizeToMax(img, CardImageThumbnailMaxDimension)
	var thumbBuf bytes.Buffer
	quality := settings.ImageQuality
	if quality <= 0 {
		quality = 85
	}
	if err := jpeg.Encode(&thumbBuf, thumbImg, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("derivative_encode_failed")
	}
	thumbPath := filepath.Join(derivedDir, "thumbnail.jpg")
	if err := os.WriteFile(thumbPath, thumbBuf.Bytes(), 0o644); err != nil {
		return nil, fmt.Errorf("derivative_write_failed")
	}
	thumbSum := sha256.Sum256(thumbBuf.Bytes())
	if _, err := tx.Exec(ctx, `
		INSERT INTO asset_derivatives (asset_id, variant_key, width, height, mime, byte_size, checksum_sha256, path)
		VALUES ($1, 'thumbnail', $2, $3, 'image/jpeg', $4, $5, $6)
	`, assetID, thumbW, thumbH, thumbBuf.Len(), thumbSum[:], thumbPath); err != nil {
		return nil, err
	}

	storedBytes := int64(len(data)) + int64(thumbBuf.Len())
	if errorCode, err := checkWarehouseUploadCapacity(ctx, pool, locationID, storageRoot, storedBytes, settings.HardLimitBytes); err != nil {
		_ = os.RemoveAll(assetDir)
		return nil, err
	} else if errorCode != "" {
		_ = os.RemoveAll(assetDir)
		return nil, fmt.Errorf("%s", errorCode)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE assets
		SET stored_bytes = $2, name = $3, shape = 'raw', default_grid_width = 1, default_grid_height = 1,
		    retain_original = TRUE, status = 'active', crop_x = 0.5, crop_y = 0.5, zoom = 1, updated_at = NOW()
		WHERE id = $1
	`, assetID, storedBytes, inheritedAssetName(originalFilename)); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &CreatedImageAsset{AssetID: assetID, ContentURL: "/api/assets/" + assetID + "/content"}, nil
}
