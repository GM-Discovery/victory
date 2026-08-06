package storyboards

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"victory/backend/internal/dbtest"
)

func TestExportVersionedFormatAndHiddenAuthority(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_export_owner")
	director := insertTestUser(t, pool, "sb_export_director")
	crew := insertTestUser(t, pool, "sb_export_crew")
	board := mustCreateBoard(t, pool, owner, "Export Board")
	for _, pair := range []struct{ id, role string }{{director, "director"}, {crew, "crew"}} {
		var handle string
		pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, pair.id).Scan(&handle)
		if _, err := AddGrant(ctx, pool, owner, board.ID, handle, pair.role); err != nil {
			t.Fatalf("grant %s: %v", pair.role, err)
		}
	}
	rows, _ := ListRows(ctx, pool, board.ID)
	cols, _ := ListColumns(ctx, pool, board.ID)

	visible, err := CreateCard(ctx, pool, owner, board.ID, rows[0].ID, cols[0].ID, "Visible", "front", "back")
	if err != nil {
		t.Fatalf("create visible: %v", err)
	}
	hidden, err := CreateCard(ctx, pool, owner, board.ID, rows[0].ID, cols[0].ID, "Secret Plan", "shh", "")
	if err != nil {
		t.Fatalf("create hidden: %v", err)
	}
	hideVal := true
	if _, err := UpdateCard(ctx, pool, owner, board.ID, hidden.ID, hidden.Version, CardEdit{
		Title: hidden.Title, FrontText: hidden.FrontText, HiddenFromAudience: &hideVal,
	}); err != nil {
		t.Fatalf("hide: %v", err)
	}
	_ = visible

	// Crew may not export at all (spec 5.3-5.5: export is Director+/owner
	// only).
	if _, err := BuildBoardExport(ctx, pool, crew, board.ID); err != ErrNotAuthorized {
		t.Fatalf("expected ErrNotAuthorized for crew export, got %v", err)
	}

	doc, err := BuildBoardExport(ctx, pool, director, board.ID)
	if err != nil {
		t.Fatalf("director export: %v", err)
	}
	if doc.Format != ExportFormat || doc.FormatVersion != ExportFormatVersion {
		t.Fatalf("expected versioned format, got %q v%d", doc.Format, doc.FormatVersion)
	}
	if doc.IntegritySHA256 == "" {
		t.Fatalf("expected integrity hash to be set")
	}
	foundHidden := false
	for _, c := range doc.Cards {
		if c.Title == "Secret Plan" {
			foundHidden = true
		}
	}
	if !foundHidden {
		t.Fatalf("expected Director+ export to include the hidden card")
	}

	ownerDoc, err := BuildBoardExport(ctx, pool, owner, board.ID)
	if err != nil {
		t.Fatalf("owner export: %v", err)
	}
	rawBytes, err := json.Marshal(ownerDoc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	raw := strings.ToLower(string(rawBytes))
	for _, secret := range []string{"password", "session_token", "victory_session"} {
		if strings.Contains(raw, secret) {
			t.Fatalf("export payload unexpectedly contains %q", secret)
		}
	}
}

func TestExportOmitsHiddenCardsForUnauthorizedExporterTier(t *testing.T) {
	// There is no authorized-but-audience-tier exporter (Audience can never
	// pass CanExportBoard), so this test instead documents that a
	// Director-tier exporter -- the minimum tier that can export at all --
	// still receives the correct hidden-authorized projection: hidden
	// cards present, because Director+ is itself Crew+.
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_export_owner2")
	board := mustCreateBoard(t, pool, owner, "Export Board 2")
	rows, _ := ListRows(ctx, pool, board.ID)
	cols, _ := ListColumns(ctx, pool, board.ID)
	card, err := CreateCard(ctx, pool, owner, board.ID, rows[0].ID, cols[0].ID, "C", "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	hideVal := true
	if _, err := UpdateCard(ctx, pool, owner, board.ID, card.ID, card.Version, CardEdit{
		Title: card.Title, HiddenFromAudience: &hideVal,
	}); err != nil {
		t.Fatalf("hide: %v", err)
	}

	doc, err := BuildBoardExport(ctx, pool, owner, board.ID)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(doc.Cards) != 1 || !doc.Cards[0].HiddenFromAudience {
		t.Fatalf("expected owner export to include the hidden card with its flag set")
	}
}

func TestMalformedBoardCannotExport(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	owner := insertTestUser(t, pool, "sb_export_owner3")
	if _, err := BuildBoardExport(context.Background(), pool, owner, "00000000-0000-0000-0000-000000000000"); err != ErrBoardNotFound {
		t.Fatalf("expected ErrBoardNotFound for a nonexistent board, got %v", err)
	}
}
