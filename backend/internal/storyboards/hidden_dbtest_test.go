package storyboards

import (
	"context"
	"testing"

	"victory/backend/internal/dbtest"
)

func TestHiddenCardFilteredForAudienceAndCastVisibleForCrewPlus(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	audience := insertTestUser(t, pool, "sb_audience")
	crew := insertTestUser(t, pool, "sb_crew")
	board := mustCreateBoard(t, pool, owner, "Hidden Board")
	for _, pair := range []struct{ id, role string }{{audience, "audience"}, {crew, "crew"}} {
		var handle string
		pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, pair.id).Scan(&handle)
		if _, err := AddGrant(ctx, pool, owner, board.ID, handle, pair.role); err != nil {
			t.Fatalf("grant %s: %v", pair.role, err)
		}
	}
	rows, _ := ListRows(ctx, pool, board.ID)
	cols, _ := ListColumns(ctx, pool, board.ID)

	visibleCard, err := CreateCard(ctx, pool, owner, board.ID, rows[0].ID, cols[0].ID, "Visible", "", "")
	if err != nil {
		t.Fatalf("create visible card: %v", err)
	}
	hiddenCard, err := CreateCard(ctx, pool, owner, board.ID, rows[0].ID, cols[0].ID, "Secret", "front secret", "back secret")
	if err != nil {
		t.Fatalf("create hidden card: %v", err)
	}
	hideVal := true
	if _, err := UpdateCard(ctx, pool, owner, board.ID, hiddenCard.ID, hiddenCard.Version, CardEdit{
		Title: hiddenCard.Title, FrontText: hiddenCard.FrontText, BackText: hiddenCard.BackText,
		HiddenFromAudience: &hideVal,
	}); err != nil {
		t.Fatalf("hide card: %v", err)
	}

	audienceSnap, err := ProjectBoardSnapshot(ctx, pool, audience, board)
	if err != nil {
		t.Fatalf("audience snapshot: %v", err)
	}
	if len(audienceSnap.Cards) != 1 {
		t.Fatalf("expected exactly 1 visible card for audience, got %d", len(audienceSnap.Cards))
	}
	if audienceSnap.Cards[0].ID != visibleCard.ID {
		t.Fatalf("expected the visible card, got %s", audienceSnap.Cards[0].ID)
	}
	for _, c := range audienceSnap.Cards {
		if c.ID == hiddenCard.ID {
			t.Fatalf("hidden card content leaked to audience snapshot")
		}
	}

	crewSnap, err := ProjectBoardSnapshot(ctx, pool, crew, board)
	if err != nil {
		t.Fatalf("crew snapshot: %v", err)
	}
	if len(crewSnap.Cards) != 2 {
		t.Fatalf("expected crew to see both cards, got %d", len(crewSnap.Cards))
	}
	foundHidden := false
	for _, c := range crewSnap.Cards {
		if c.ID == hiddenCard.ID {
			foundHidden = true
			if !c.HiddenFromAudience {
				t.Fatalf("expected hidden flag surfaced to crew")
			}
		}
	}
	if !foundHidden {
		t.Fatalf("expected crew to see the hidden card")
	}
}

func TestHiddenCardMoveDeleteNotVisibleToAudienceTier(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	audience := insertTestUser(t, pool, "sb_audience")
	board := mustCreateBoard(t, pool, owner, "Hidden Move Board")
	var audienceHandle string
	pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, audience).Scan(&audienceHandle)
	if _, err := AddGrant(ctx, pool, owner, board.ID, audienceHandle, "audience"); err != nil {
		t.Fatalf("grant audience: %v", err)
	}
	rows, _ := ListRows(ctx, pool, board.ID)
	cols, _ := ListColumns(ctx, pool, board.ID)
	col2, err := AddColumn(ctx, pool, owner, board.ID, "Col 2")
	if err != nil {
		t.Fatalf("add col2: %v", err)
	}

	card, err := CreateCard(ctx, pool, owner, board.ID, rows[0].ID, cols[0].ID, "Secret", "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	hideVal := true
	updated, err := UpdateCard(ctx, pool, owner, board.ID, card.ID, card.Version, CardEdit{
		Title: card.Title, HiddenFromAudience: &hideVal,
	})
	if err != nil {
		t.Fatalf("hide: %v", err)
	}

	moved, err := MoveCard(ctx, pool, owner, board.ID, card.ID, updated.Version, rows[0].ID, col2.ID)
	if err != nil {
		t.Fatalf("move: %v", err)
	}

	snap, err := ProjectBoardSnapshot(ctx, pool, audience, board)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	for _, c := range snap.Cards {
		if c.ID == card.ID {
			t.Fatalf("moved hidden card should not appear in audience-tier snapshot at all")
		}
	}

	if err := DeleteCard(ctx, pool, owner, board.ID, card.ID, moved.Version); err != nil {
		t.Fatalf("delete: %v", err)
	}
	snap, _ = ProjectBoardSnapshot(ctx, pool, audience, board)
	for _, c := range snap.Cards {
		if c.ID == card.ID {
			t.Fatalf("deleted hidden card should not appear post-delete either")
		}
	}
}

func TestCrewCannotSetHiddenFlagOnlyDirectorPlus(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	crew := insertTestUser(t, pool, "sb_crew")
	board := mustCreateBoard(t, pool, owner, "Crew Hidden Board")
	var crewHandle string
	pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, crew).Scan(&crewHandle)
	if _, err := AddGrant(ctx, pool, owner, board.ID, crewHandle, "crew"); err != nil {
		t.Fatalf("grant crew: %v", err)
	}
	rows, _ := ListRows(ctx, pool, board.ID)
	cols, _ := ListColumns(ctx, pool, board.ID)

	card, err := CreateCard(ctx, pool, crew, board.ID, rows[0].ID, cols[0].ID, "Card", "", "")
	if err != nil {
		t.Fatalf("crew create: %v", err)
	}
	hideVal := true
	if _, err := UpdateCard(ctx, pool, crew, board.ID, card.ID, card.Version, CardEdit{
		Title: card.Title, HiddenFromAudience: &hideVal,
	}); err != ErrNotAuthorized {
		t.Fatalf("expected ErrNotAuthorized for crew setting hidden flag, got %v", err)
	}
}
