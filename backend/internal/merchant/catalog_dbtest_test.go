package merchant

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
)

// TestSocioEquipmentCatalogSeed proves migration 063's catalog import and
// Kessa curation, not just that SOME stock exists (TestKessaGoldenPath
// already covers the purchase flow itself).
func TestSocioEquipmentCatalogSeed(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var locationID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family'`).Scan(&locationID); err != nil {
		t.Fatalf("location lookup: %v", err)
	}

	// The 5 Kernel 73 placeholders are deactivated, not deleted, and no
	// longer in Kessa's stock.
	for _, slug := range []string{"travelers-cloak", "coil-of-rope", "traveling-rations", "sturdy-boots", "simple-dagger"} {
		item, err := loadEquipmentItemBySlug(ctx, pool, locationID, slug)
		if err != nil {
			t.Fatalf("placeholder %s should still exist (deactivated, not deleted): %v", slug, err)
		}
		if item.Active {
			t.Fatalf("placeholder %s should be deactivated by migration 063", slug)
		}
	}

	// A representative weapon carries its reference-only catalog fields.
	longsword, err := loadEquipmentItemBySlug(ctx, pool, locationID, "longsword")
	if err != nil {
		t.Fatalf("longsword lookup: %v", err)
	}
	if longsword.Category != "weapon_blade" {
		t.Fatalf("longsword category = %q, want weapon_blade", longsword.Category)
	}
	if longsword.CostCredits == nil || *longsword.CostCredits != 80 {
		t.Fatalf("longsword cost_credits = %v, want 80", longsword.CostCredits)
	}
	if longsword.StatsJSON["damage"] != "d8" {
		t.Fatalf("longsword stats.damage = %v, want d8", longsword.StatsJSON["damage"])
	}

	// A fractional-cost consumable round-trips correctly.
	bread, err := loadEquipmentItemBySlug(ctx, pool, locationID, "fresh-bread")
	if err != nil {
		t.Fatalf("fresh-bread lookup: %v", err)
	}
	if bread.CostCredits == nil || *bread.CostCredits != 0.5 {
		t.Fatalf("fresh-bread cost_credits = %v, want 0.5", bread.CostCredits)
	}

	// The catalog itself is large (full reference data), but Kessa's own
	// shop is a small curated subset -- not everything in the catalog is
	// purchasable from her.
	var packetID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM merchant_packets WHERE location_id = $1::uuid AND slug = 'kessa'`, locationID).Scan(&packetID); err != nil {
		t.Fatalf("kessa packet lookup: %v", err)
	}
	var kessaStockCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM merchant_packet_equipment_items WHERE packet_id = $1::uuid`, packetID).Scan(&kessaStockCount); err != nil {
		t.Fatalf("kessa stock count: %v", err)
	}
	if kessaStockCount != 19 {
		t.Fatalf("kessa stock count = %d, want 19 (the curated starter subset)", kessaStockCount)
	}

	var catalogCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM equipment_items WHERE location_id = $1::uuid AND active = TRUE`, locationID).Scan(&catalogCount); err != nil {
		t.Fatalf("catalog count: %v", err)
	}
	if catalogCount < 145 {
		t.Fatalf("active catalog count = %d, want at least 145 (the full Socio catalog)", catalogCount)
	}
	if kessaStockCount >= catalogCount {
		t.Fatal("Kessa's curated stock should be a small subset of the full catalog, not everything in it")
	}

	// An item deliberately NOT in Kessa's stock (Enchanted Plate -- endgame
	// gear, narratively wrong for a fresh Courtyard character) exists in
	// the catalog but is not purchasable from her.
	plate, err := loadEquipmentItemBySlug(ctx, pool, locationID, "enchanted-plate")
	if err != nil {
		t.Fatalf("enchanted-plate should exist in the general catalog: %v", err)
	}
	var inKessaStock bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM merchant_packet_equipment_items WHERE packet_id = $1::uuid AND equipment_item_id = $2::uuid)
	`, packetID, plate.ID).Scan(&inKessaStock); err != nil {
		t.Fatalf("kessa stock membership check: %v", err)
	}
	if inKessaStock {
		t.Fatal("Enchanted Plate should not be in Kessa's curated starter stock")
	}
}

func loadEquipmentItemBySlug(ctx context.Context, pool *pgxpool.Pool, locationID, slug string) (EquipmentItem, error) {
	row := pool.QueryRow(ctx, `SELECT `+equipmentItemColumns+` FROM equipment_items WHERE location_id = $1 AND slug = $2`, locationID, slug)
	return scanEquipmentItem(row)
}
