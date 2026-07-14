package network

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
)

func openNetworkTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return dbtest.OpenTestPool(t)
}

func testSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
}

func TestVenueSceneRehearsalEnabled(t *testing.T) {
	pool := openNetworkTestPool(t)

	for _, slug := range []string{"first-theater", "catharsis"} {
		enabled, err := VenueSceneRehearsalEnabled(context.Background(), pool, slug)
		if err != nil {
			t.Fatalf("check %s: %v", slug, err)
		}
		if !enabled {
			t.Fatalf("expected scene_rehearsal_enabled=true for %s", slug)
		}
	}

	enabled, err := VenueSceneRehearsalEnabled(context.Background(), pool, "middle-school-stage")
	if err != nil {
		t.Fatalf("check middle-school-stage: %v", err)
	}
	if enabled {
		t.Fatalf("expected scene_rehearsal_enabled=false (absent) for middle-school-stage")
	}
}

// TestBroadcastShowStageInvalidationOnlyReachesSessionsLinkedToShow proves
// the invalidation is scoped: a client whose session is linked to the
// target Show receives it; a client on an unrelated session (linked to a
// different Show, or unlinked) does not.
func TestBroadcastShowStageInvalidationOnlyReachesSessionsLinkedToShow(t *testing.T) {
	pool := openNetworkTestPool(t)
	ctx := context.Background()

	var venueID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM venues LIMIT 1`).Scan(&venueID); err != nil {
		t.Fatalf("load a venue: %v", err)
	}

	showID := insertMinimalShow(t, pool)
	otherShowID := insertMinimalShow(t, pool)

	linkedSession := insertMinimalSession(t, pool, venueID, showID)
	otherShowSession := insertMinimalSession(t, pool, venueID, otherShowID)
	unlinkedSession := insertMinimalSession(t, pool, venueID, "")

	hub := NewHub()
	linkedClient := &Client{Send: make(chan []byte, 4), SessionID: linkedSession}
	otherClient := &Client{Send: make(chan []byte, 4), SessionID: otherShowSession}
	unlinkedClient := &Client{Send: make(chan []byte, 4), SessionID: unlinkedSession}
	hub.Add(linkedClient)
	hub.Add(otherClient)
	hub.Add(unlinkedClient)

	BroadcastShowStageInvalidation(ctx, hub, pool, showID, "test_reason")

	select {
	case msg := <-linkedClient.Send:
		var payload map[string]any
		if err := json.Unmarshal(msg, &payload); err != nil {
			t.Fatalf("unmarshal invalidation payload: %v", err)
		}
		if payload["type"] != "show/stage_updated" || payload["show_id"] != showID {
			t.Fatalf("unexpected invalidation payload: %+v", payload)
		}
	case <-time.After(time.Second):
		t.Fatalf("expected the linked client to receive the invalidation")
	}

	select {
	case msg := <-otherClient.Send:
		t.Fatalf("expected a client linked to a different Show to receive nothing, got %s", msg)
	default:
	}
	select {
	case msg := <-unlinkedClient.Send:
		t.Fatalf("expected an unlinked client to receive nothing, got %s", msg)
	default:
	}
}

func insertMinimalShow(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	ctx := context.Background()
	suffix := testSuffix(t) + "_" + time.Now().UTC().Format("150405.000000000")

	var locationID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID); err != nil {
		t.Fatalf("load amurray-family location: %v", err)
	}
	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name) VALUES ($1, $2) RETURNING id::text
	`, "net_show_"+suffix, "Net Show Fixture").Scan(&userID); err != nil {
		t.Fatalf("insert fixture user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID) })

	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1, $2, $3) RETURNING id::text
	`, locationID, "Net Show Production "+suffix, "net-show-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID) })

	var showRunID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_runs (location_id, production_id, title, slug, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5) RETURNING id::text
	`, locationID, productionID, "Net Show Run "+suffix, "net-show-run-"+suffix, userID).Scan(&showRunID); err != nil {
		t.Fatalf("insert show run: %v", err)
	}

	var showID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO shows (show_run_id, title, slug, created_by_user_id)
		VALUES ($1, $2, $3, $4) RETURNING id::text
	`, showRunID, "Net Show "+suffix, "net-show-"+suffix, userID).Scan(&showID); err != nil {
		t.Fatalf("insert show: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM shows WHERE id = $1`, showID) })
	return showID
}

func insertMinimalSession(t *testing.T, pool *pgxpool.Pool, venueID, showID string) string {
	t.Helper()
	ctx := context.Background()
	var showIDArg *string
	if showID != "" {
		showIDArg = &showID
	}
	var sessionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO sessions (venue_id, status, show_id) VALUES ($1, 'live', $2) RETURNING id::text
	`, venueID, showIDArg).Scan(&sessionID); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM sessions WHERE id = $1`, sessionID) })
	return sessionID
}
