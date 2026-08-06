package storyboards

import (
	"context"
	"testing"

	"victory/backend/internal/dbtest"
)

func TestRoleCapabilityMatrix(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	board := mustCreateBoard(t, pool, owner, "Matrix Board")

	tiers := []string{"audience", "cast", "crew", "director", "producer"}
	users := map[string]string{}
	for _, tier := range tiers {
		u := insertTestUser(t, pool, "sb_"+tier)
		var handle string
		pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, u).Scan(&handle)
		if _, err := AddGrant(ctx, pool, owner, board.ID, handle, tier); err != nil {
			t.Fatalf("grant %s: %v", tier, err)
		}
		users[tier] = u
	}

	// Audience/Cast: view only.
	for _, tier := range []string{"audience", "cast"} {
		u := users[tier]
		if ok, err := CanViewBoard(ctx, pool, u, board); err != nil || !ok {
			t.Fatalf("%s should view: ok=%v err=%v", tier, ok, err)
		}
		if ok, err := CanEditCards(ctx, pool, u, board); err != nil || ok {
			t.Fatalf("%s should NOT edit cards: ok=%v err=%v", tier, ok, err)
		}
		if ok, err := CanEditStructure(ctx, pool, u, board); err != nil || ok {
			t.Fatalf("%s should NOT edit structure: ok=%v err=%v", tier, ok, err)
		}
	}

	// Crew: card CRUD, not structure/sharing.
	crew := users["crew"]
	if ok, err := CanEditCards(ctx, pool, crew, board); err != nil || !ok {
		t.Fatalf("crew should edit cards: ok=%v err=%v", ok, err)
	}
	if ok, err := CanEditStructure(ctx, pool, crew, board); err != nil || ok {
		t.Fatalf("crew should NOT edit structure: ok=%v err=%v", ok, err)
	}
	if ok, err := CanManageSharing(ctx, pool, crew, board); err != nil || ok {
		t.Fatalf("crew should NOT manage sharing: ok=%v err=%v", ok, err)
	}
	if ok, err := CanExportBoard(ctx, pool, crew, board); err != nil || ok {
		t.Fatalf("crew should NOT export: ok=%v err=%v", ok, err)
	}

	// Director/Producer: structure + export, not sharing/delete.
	for _, tier := range []string{"director", "producer"} {
		u := users[tier]
		if ok, err := CanEditStructure(ctx, pool, u, board); err != nil || !ok {
			t.Fatalf("%s should edit structure: ok=%v err=%v", tier, ok, err)
		}
		if ok, err := CanExportBoard(ctx, pool, u, board); err != nil || !ok {
			t.Fatalf("%s should export: ok=%v err=%v", tier, ok, err)
		}
		if ok, err := CanManageSharing(ctx, pool, u, board); err != nil || ok {
			t.Fatalf("%s should NOT manage sharing: ok=%v err=%v", tier, ok, err)
		}
		if ok, err := CanArchiveOrDeleteBoard(ctx, pool, u, board); err != nil || ok {
			t.Fatalf("%s should NOT archive/delete: ok=%v err=%v", tier, ok, err)
		}
	}

	// Owner: everything.
	if ok, err := CanManageSharing(ctx, pool, owner, board); err != nil || !ok {
		t.Fatalf("owner should manage sharing: ok=%v err=%v", ok, err)
	}
	if ok, err := CanArchiveOrDeleteBoard(ctx, pool, owner, board); err != nil || !ok {
		t.Fatalf("owner should archive/delete: ok=%v err=%v", ok, err)
	}
}

// TestServerResolvedTierIgnoresClientClaims documents the role-forgery
// invariant at the authority-package level: resolveViewerTier never takes a
// role as input at all -- it is derived purely from server-held
// ownership/grant state, so there is no parameter a malicious client could
// even populate to forge a tier. The HTTP-layer forgery test (http_test.go,
// once http.go exists) exercises this end-to-end through a request body
// that claims an elevated role.
func TestServerResolvedTierIgnoresClientClaims(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	owner := insertTestUser(t, pool, "sb_owner")
	crewUser := insertTestUser(t, pool, "sb_crew")
	board := mustCreateBoard(t, pool, owner, "Forgery Board")
	var crewHandle string
	pool.QueryRow(ctx, `SELECT handle FROM users WHERE id = $1`, crewUser).Scan(&crewHandle)
	if _, err := AddGrant(ctx, pool, owner, board.ID, crewHandle, "crew"); err != nil {
		t.Fatalf("grant crew: %v", err)
	}

	// No matter what a caller might wish were true, the tier resolved for
	// crewUser is exactly what the DB grant says: crew, never producer.
	tier, err := ResolveViewerTier(ctx, pool, crewUser, board)
	if err != nil {
		t.Fatalf("resolve tier: %v", err)
	}
	if tier != TierCrew {
		t.Fatalf("expected crew, got %s", tier)
	}
	if ok, err := CanEditStructure(ctx, pool, crewUser, board); err != nil || ok {
		t.Fatalf("crew-tier user must not pass a structure check regardless of intent: ok=%v err=%v", ok, err)
	}
}
