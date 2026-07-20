package merchant

import (
	"context"
	"testing"
	"time"

	"victory/backend/internal/dbtest"
)

// TestNonRosteredUserCannotOpenEquipMode closes spec S12's "a non-rostered
// user cannot open Equip Mode" proof -- an authenticated user with no
// show_run_roster_members row at all (never bought a ticket, never joined)
// must be refused, distinct from the audience-role case already covered by
// TestKessaGoldenPath.
func TestNonRosteredUserCannotOpenEquipMode(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildGoldenPathFixture(ctx, t, pool)

	var strangerUserID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name) VALUES ($1, $1) RETURNING id::text
	`, "k73_stranger_"+time.Now().UTC().Format("150405.000")).Scan(&strangerUserID); err != nil {
		t.Fatalf("fixture stranger user: %v", err)
	}

	if _, err := OpenEquipMode(ctx, pool, strangerUserID, fx.interactionID); err == nil {
		t.Fatal("a user with no roster row at all should not be able to open Equip Mode")
	}
}

// TestArchivedCharacterCannotReceiveEquipment closes spec S12's "archived
// Character cannot receive equipment" proof.
func TestArchivedCharacterCannotReceiveEquipment(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildGoldenPathFixture(ctx, t, pool)

	if _, err := pool.Exec(ctx, `UPDATE character_cards SET is_deleted = TRUE WHERE id = $1`, fx.characterCardID); err != nil {
		t.Fatalf("archive fixture character: %v", err)
	}

	if _, err := OpenEquipMode(ctx, pool, fx.playerUserID, fx.interactionID); err == nil {
		t.Fatal("an archived (is_deleted) selected Character should refuse Equip Mode access")
	}
}

// TestForgedInteractionIDIsRejected closes spec S12's "a forged Show/Scene/
// interaction ID is rejected" proof for the interaction-ID axis.
func TestForgedInteractionIDIsRejected(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildGoldenPathFixture(ctx, t, pool)

	forgedID := "00000000-0000-0000-0000-000000000000"
	if _, err := OpenEquipMode(ctx, pool, fx.playerUserID, forgedID); err == nil {
		t.Fatal("a forged interaction id should be rejected")
	}
	if _, err := AttemptPurchase(ctx, pool, fx.playerUserID, forgedID, "any-item", "any-key"); err == nil {
		t.Fatal("a forged interaction id should be rejected on purchase too")
	}
}

// TestSwitchingSelectedCharacterDoesNotRewriteHistoricalInventory closes
// spec S2.10/S12's "historical acquisition metadata is not rewritten when
// the Player later switches Characters" proof: an item purchased onto
// Character A stays recorded against Character A even after the Player
// selects Character B for the Show Run and could, in principle, purchase
// again (which would land on B, not touch A's row).
func TestSwitchingSelectedCharacterDoesNotRewriteHistoricalInventory(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fx := buildGoldenPathFixture(ctx, t, pool)

	equip, err := OpenEquipMode(ctx, pool, fx.playerUserID, fx.interactionID)
	if err != nil {
		t.Fatalf("OpenEquipMode: %v", err)
	}
	item := equip.Packet.Stock[0]

	firstEntry, err := AttemptPurchase(ctx, pool, fx.playerUserID, fx.interactionID, item.ID, "switch-test-key-1")
	if err != nil {
		t.Fatalf("purchase onto original character: %v", err)
	}
	if firstEntry.CharacterCardID != fx.characterCardID {
		t.Fatalf("purchase landed on %q, want %q", firstEntry.CharacterCardID, fx.characterCardID)
	}

	// Switch the roster's selected character to the unskilled fixture
	// character (owned by the same Player, so this is the legitimate
	// self-service switch, not a security bypass).
	if _, err := pool.Exec(ctx, `
		UPDATE show_run_roster_members SET character_card_id = $2::uuid
		WHERE show_run_id = $1::uuid AND user_id = $3::uuid
	`, fx.showRunID, fx.unskilledCharacterCardID, fx.playerUserID); err != nil {
		t.Fatalf("switch selected character: %v", err)
	}

	secondEntry, err := AttemptPurchase(ctx, pool, fx.playerUserID, fx.interactionID, item.ID, "switch-test-key-2")
	if err != nil {
		t.Fatalf("purchase after switching character: %v", err)
	}
	if secondEntry.CharacterCardID != fx.unskilledCharacterCardID {
		t.Fatalf("post-switch purchase landed on %q, want the newly-selected %q", secondEntry.CharacterCardID, fx.unskilledCharacterCardID)
	}

	// The original character's inventory row must be untouched.
	originalInventory, err := ListInventoryForCharacter(ctx, pool, fx.playerUserID, fx.characterCardID)
	if err != nil {
		t.Fatalf("ListInventoryForCharacter (original): %v", err)
	}
	found := false
	for _, e := range originalInventory {
		if e.EquipmentItemID == item.ID {
			found = true
			if e.Quantity != 1 {
				t.Fatalf("original character's item quantity changed to %d, want unchanged 1", e.Quantity)
			}
		}
	}
	if !found {
		t.Fatal("original character's purchase disappeared after switching selected character")
	}
}
