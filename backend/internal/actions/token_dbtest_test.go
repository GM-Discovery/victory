package actions

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
)

// Kernel 72A regression: the full create-token chain on the catharsis stage.
// Token creation crosses THREE venue gates (CanAct's stage_elements_enabled,
// the role policy, and resolvePlacementVenue) plus the showing/asset/library
// resolutions — and until Kernel 72A no test drove the whole chain, which is
// how catharsis shipped a stage where every token create was denied
// (unknown_target from the old hardcoded allowlist, then policy_denied from
// resolvePlacementVenue's borrowed index_cards_enabled check).
func TestStoreCreateTokenOnCatharsisStage(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fx := createTokenFixture(ctx, t, pool)

	stored, err := StoreCreateToken(ctx, pool, TokenPlacementRequest{
		SessionID: fx.sessionID,
		ActorID:   fx.userID,
		AssetID:   fx.assetID,
		VenueSlug: "catharsis",
		X:         120,
		Y:         80,
	})
	if err != nil {
		t.Fatalf("StoreCreateToken on catharsis: %v", err)
	}
	if stored == nil || strings.TrimSpace(stored.ID) == "" {
		t.Fatalf("expected a stored action, got %+v", stored)
	}

	// The placement row must exist on the catharsis stage surface —
	// resolvePlacementVenue regression guard.
	var placed bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM venue_layout_elements vle
			JOIN venues v ON v.id = vle.venue_id
			JOIN elements e ON e.id = vle.element_id
			WHERE v.slug = 'catharsis'
			  AND vle.surface = 'stage'
			  AND e.element_type = 'token'
			  AND e.data ->> 'created_by' = $1
		)
	`, fx.userID).Scan(&placed); err != nil {
		t.Fatalf("check placement row: %v", err)
	}
	if !placed {
		t.Fatal("token element was created but has no catharsis stage placement row")
	}

	// A non-producer participant is still refused by role policy.
	deniedReq := TokenPlacementRequest{
		SessionID: fx.sessionID,
		ActorID:   fx.audienceUserID,
		AssetID:   fx.assetID,
		VenueSlug: "catharsis",
	}
	if _, err := StoreCreateToken(ctx, pool, deniedReq); err == nil {
		t.Fatal("audience participant should not be able to create tokens")
	}
}

type tokenFixture struct {
	userID         string
	audienceUserID string
	sessionID      string
	assetID        string
}

func createTokenFixture(ctx context.Context, t *testing.T, pool *pgxpool.Pool) tokenFixture {
	t.Helper()
	var fx tokenFixture
	suffix := time.Now().UTC().Format("150405") + "_" + strings.ToLower(t.Name()[len(t.Name())-4:])

	var locationID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID); err != nil {
		t.Fatalf("fixture location: %v", err)
	}
	// EnsureForSession requires a production for the location (same idempotent
	// insert the Kernel 64 fixtures use).
	if _, err := pool.Exec(ctx, `
		INSERT INTO productions (location_id, name, slug)
		SELECT $1::uuid, 'Fixture Production', 'fixture-production'
		WHERE NOT EXISTS (SELECT 1 FROM productions WHERE location_id = $1::uuid)
	`, locationID); err != nil {
		t.Fatalf("fixture production: %v", err)
	}

	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name) VALUES ('tok_fx_'||$1, 'Token Fixture') RETURNING id::text
	`, suffix).Scan(&fx.userID); err != nil {
		t.Fatalf("fixture user: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name) VALUES ('tok_aud_'||$1, 'Token Audience') RETURNING id::text
	`, suffix).Scan(&fx.audienceUserID); err != nil {
		t.Fatalf("fixture audience user: %v", err)
	}

	if err := pool.QueryRow(ctx, `
		INSERT INTO sessions (venue_id, status)
		SELECT v.id, 'rehearsal' FROM venues v WHERE v.slug = 'catharsis'
		RETURNING id::text
	`).Scan(&fx.sessionID); err != nil {
		t.Fatalf("fixture session: %v", err)
	}
	t.Cleanup(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer ccancel()
		_, _ = pool.Exec(cctx, `UPDATE sessions SET status = 'closed' WHERE id = $1::uuid`, fx.sessionID)
	})

	// CanAct refuses outright without a showings row for the session.
	if _, err := pool.Exec(ctx, `
		INSERT INTO showings (session_id, production_id, venue_id, status, created_by)
		SELECT s.id, p.id, s.venue_id, 'rehearsal', $2::uuid
		FROM sessions s, productions p
		WHERE s.id = $1::uuid
		  AND p.location_id = $3::uuid
		ORDER BY p.created_at ASC
		LIMIT 1
		ON CONFLICT (session_id) DO NOTHING
	`, fx.sessionID, fx.userID, locationID); err != nil {
		t.Fatalf("fixture showing: %v", err)
	}

	for user, role := range map[string]string{fx.userID: "producer", fx.audienceUserID: "audience"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO session_participants (session_id, user_id, role)
			VALUES ($1::uuid, $2::uuid, $3::location_role)
			ON CONFLICT (session_id, user_id) DO UPDATE SET role = EXCLUDED.role
		`, fx.sessionID, user, role); err != nil {
			t.Fatalf("fixture participant %s: %v", role, err)
		}
	}

	if err := pool.QueryRow(ctx, `
		INSERT INTO assets (
			producer_user_id, location_id, uploader_user_id, owner_user_id,
			owner_state, asset_type, status, name, original_filename,
			source_mime, sniffed_mime, width, height, byte_size, stored_bytes,
			checksum_sha256, storage_root, original_path
		)
		VALUES (
			$1::uuid, $2::uuid, $1::uuid, $1::uuid,
			'uploader_owned', 'token', 'active', 'Fixture Goat', 'goat.png',
			'image/png', 'image/png', 64, 64, 1024, 1024,
			decode('00', 'hex'), '/tmp/fixture', '/tmp/fixture/goat.png'
		)
		RETURNING id::text
	`, fx.userID, locationID).Scan(&fx.assetID); err != nil {
		t.Fatalf("fixture asset: %v", err)
	}

	return fx
}
