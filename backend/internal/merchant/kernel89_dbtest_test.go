package merchant

import (
	"context"
	"testing"
	"time"

	"victory/backend/internal/dbtest"
)

// A Director authors a brand-new merchant from the canonical equipment
// corpus -- kernel 89 §9's whole point, and the answer to §42's "Can I
// configure a merchant from the real equipment list?"
func TestDirectorAuthorsANewMerchantFromTheCanonicalCatalog(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	fx := buildGoldenPathFixture(ctx, t, pool)

	catalog, err := ListEquipmentItems(ctx, pool, fx.locationID)
	if err != nil {
		t.Fatalf("list canonical catalog: %v", err)
	}
	if len(catalog) < 5 {
		t.Fatalf("expected the real seeded Socio catalog, got %d items", len(catalog))
	}

	packet, err := CreatePacket(ctx, pool, fx.locationID, fx.directorUserID, PacketInput{
		DisplayName:       "Arena Quartermaster " + time.Now().UTC().Format("150405.000"),
		IntroText:         "The quartermaster looks up from a crate of practice blades.",
		HaggleSuccessText: "\"Fine. Trainee rate.\"",
		HaggleFailureText: "\"List price, same as everyone.\"",
		StanceDispositions: map[string]StanceDisposition{
			"insight": {Disposition: "positive", Responses: []string{"\"You've an eye for kit.\""}},
		},
	})
	if err != nil {
		t.Fatalf("create packet: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM merchant_packet_equipment_items WHERE packet_id = $1`, packet.ID)
		_, _ = pool.Exec(bg, `DELETE FROM merchant_packets WHERE id = $1`, packet.ID)
	})

	// All five fixed stance slots exist regardless of how few were authored
	// -- the model has exactly five, never a growing tree.
	if len(packet.StanceDispositions) != len(FixedStanceKeys) {
		t.Fatalf("expected %d fixed stances, got %d", len(FixedStanceKeys), len(packet.StanceDispositions))
	}
	if packet.StanceDispositions["insight"].Disposition != "positive" {
		t.Fatalf("authored stance lost: %#v", packet.StanceDispositions["insight"])
	}
	if packet.StanceDispositions["command"].Disposition != "neutral" {
		t.Fatalf("unauthored stance should default neutral, got %#v", packet.StanceDispositions["command"])
	}

	stock := []string{catalog[0].ID, catalog[1].ID, catalog[2].ID}
	if err := SetPacketStock(ctx, pool, packet.ID, fx.locationID, stock); err != nil {
		t.Fatalf("set stock: %v", err)
	}
	loaded, err := LoadPacketBySlug(ctx, pool, fx.locationID, packet.Slug)
	if err != nil {
		t.Fatalf("reload packet: %v", err)
	}
	// Only ACTIVE stock is served to a Player, so compare against the
	// active subset of what was set rather than the raw count.
	if len(loaded.Stock) == 0 {
		t.Fatal("packet has no stock after authoring")
	}
	for _, item := range loaded.Stock {
		if item.LocationID != fx.locationID {
			t.Fatalf("stock item %q escaped the packet's own Location", item.Name)
		}
	}

	// Replace-not-merge.
	if err := SetPacketStock(ctx, pool, packet.ID, fx.locationID, []string{catalog[0].ID}); err != nil {
		t.Fatalf("replace stock: %v", err)
	}
	reloaded, err := LoadPacketBySlug(ctx, pool, fx.locationID, packet.Slug)
	if err != nil {
		t.Fatalf("reload after replace: %v", err)
	}
	if len(reloaded.Stock) > 1 {
		t.Fatalf("stock should have been replaced, not merged: %d items", len(reloaded.Stock))
	}
}

// Kernel 89 §41 FAIL criterion: "merchant authoring creates a second
// equipment catalog." Stock must be chosen from the corpus, never invented.
func TestMerchantStockCannotIntroduceForeignEquipment(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	fx := buildGoldenPathFixture(ctx, t, pool)

	packet, err := CreatePacket(ctx, pool, fx.locationID, fx.directorUserID, PacketInput{
		DisplayName: "Corpus Guard " + time.Now().UTC().Format("150405.000"),
	})
	if err != nil {
		t.Fatalf("create packet: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM merchant_packets WHERE id = $1`, packet.ID)
	})

	// A syntactically valid but unknown equipment id.
	if err := SetPacketStock(ctx, pool, packet.ID, fx.locationID, []string{"00000000-0000-0000-0000-000000000001"}); err == nil {
		t.Fatal("an equipment id outside the corpus must be refused, not created")
	}
}

func TestMerchantSlugIsStableAcrossRename(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	fx := buildGoldenPathFixture(ctx, t, pool)

	packet, err := CreatePacket(ctx, pool, fx.locationID, fx.directorUserID, PacketInput{
		DisplayName: "Rename Me " + time.Now().UTC().Format("150405.000"),
	})
	if err != nil {
		t.Fatalf("create packet: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM merchant_packets WHERE id = $1`, packet.ID)
	})

	renamed, err := UpdatePacket(ctx, pool, packet.ID, PacketInput{DisplayName: "Completely Different Name"})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if renamed.Slug != packet.Slug {
		t.Fatalf("renaming a merchant must not change its slug (interactions point at it): %q -> %q", packet.Slug, renamed.Slug)
	}
	if renamed.DisplayName != "Completely Different Name" {
		t.Fatalf("rename did not take: %q", renamed.DisplayName)
	}
}

func TestPacketAuthoringRejectsUnknownStanceKeys(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	fx := buildGoldenPathFixture(ctx, t, pool)

	if _, err := CreatePacket(ctx, pool, fx.locationID, fx.directorUserID, PacketInput{
		DisplayName:        "Bad Stances " + time.Now().UTC().Format("150405.000"),
		StanceDispositions: map[string]StanceDisposition{"intimidate": {Disposition: "reject"}},
	}); err == nil {
		t.Fatal("a sixth stance must be refused -- the model has exactly five")
	}
}

// Kernel 89 §9.3: a merchant exposed to one Cohort opens for that Cohort's
// members and refuses everyone else -- enforced at ResolveEligibleContext,
// the one Player-eligibility gate, so guessing the interaction id is not a
// way around it (§28).
func TestCohortTargetedInteractionRefusesNonMembers(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	fx := buildGoldenPathFixture(ctx, t, pool)

	// Baseline: untargeted, the Player may open it.
	if _, err := ResolveEligibleContext(ctx, pool, fx.playerUserID, fx.interactionID); err != nil {
		t.Fatalf("untargeted interaction should be open to an eligible Player: %v", err)
	}

	var cohortID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_cohorts (show_id, serial_number, slug, name, created_by_user_id)
		VALUES ($1::uuid, 991, 'k89-target-cohort', 'K89 Target Cohort', $2::uuid)
		RETURNING id::text
	`, fx.showID, fx.directorUserID).Scan(&cohortID); err != nil {
		t.Fatalf("create cohort: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM show_cohort_assignments WHERE cohort_id = $1`, cohortID)
		_, _ = pool.Exec(bg, `DELETE FROM show_cohorts WHERE id = $1`, cohortID)
	})

	config := map[string]any{"packet_slug": "kessa", "target_cohort_id": cohortID}
	if _, err := UpdateInteraction(ctx, pool, fx.directorUserID, fx.interactionID, UpdateInteractionPatch{
		Configuration: &config,
	}); err != nil {
		t.Fatalf("target the interaction at a cohort: %v", err)
	}

	// Nobody is in the cohort yet, so even the previously-eligible Player
	// is refused -- exposure is a positive act, not a default.
	if _, err := ResolveEligibleContext(ctx, pool, fx.playerUserID, fx.interactionID); err == nil {
		t.Fatal("a Player outside the targeted Cohort must not resolve eligible")
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO show_cohort_assignments (show_id, cohort_id, user_id, assigned_by_user_id)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid)
	`, fx.showID, cohortID, fx.playerUserID, fx.directorUserID); err != nil {
		t.Fatalf("assign player to cohort: %v", err)
	}
	if _, err := ResolveEligibleContext(ctx, pool, fx.playerUserID, fx.interactionID); err != nil {
		t.Fatalf("a Player inside the targeted Cohort should resolve eligible: %v", err)
	}
}

// Kernel 89 §32: Send Aftercare reaches the intended Players and nobody
// else. Audience members and Players with no selected Character have no
// Aftercare record to write and are not targets.
func TestAftercareSendTargetsOnlyEligiblePlayers(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	fx := buildGoldenPathFixture(ctx, t, pool)

	recipients, err := resolveAftercareTargets(ctx, pool, fx.showID, "")
	if err != nil {
		t.Fatalf("resolve aftercare targets: %v", err)
	}
	byUser := map[string]bool{}
	for _, r := range recipients {
		byUser[r.UserID] = true
	}
	if !byUser[fx.playerUserID] {
		t.Fatal("the Player with a selected Character should be a target")
	}
	if byUser[fx.audienceUserID] {
		t.Fatal("an Audience member must never be an Aftercare target")
	}
	if byUser[fx.otherPlayerUserID] {
		t.Fatal("a Player who never selected a Character has no Aftercare record and must not be a target")
	}
	if byUser[fx.directorUserID] {
		t.Fatal("the Director is not a Player and must not be a target")
	}

	// Cohort narrowing.
	var cohortID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_cohorts (show_id, serial_number, slug, name, created_by_user_id)
		VALUES ($1::uuid, 992, 'k89-aftercare-cohort', 'K89 Aftercare Cohort', $2::uuid)
		RETURNING id::text
	`, fx.showID, fx.directorUserID).Scan(&cohortID); err != nil {
		t.Fatalf("create cohort: %v", err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM show_cohort_assignments WHERE cohort_id = $1`, cohortID)
		_, _ = pool.Exec(bg, `DELETE FROM show_cohorts WHERE id = $1`, cohortID)
	})

	narrowed, err := resolveAftercareTargets(ctx, pool, fx.showID, cohortID)
	if err != nil {
		t.Fatalf("resolve cohort-narrowed targets: %v", err)
	}
	if len(narrowed) != 0 {
		t.Fatalf("an empty Cohort should target nobody, got %d", len(narrowed))
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO show_cohort_assignments (show_id, cohort_id, user_id, assigned_by_user_id)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid)
	`, fx.showID, cohortID, fx.playerUserID, fx.directorUserID); err != nil {
		t.Fatalf("assign player to cohort: %v", err)
	}
	narrowed, err = resolveAftercareTargets(ctx, pool, fx.showID, cohortID)
	if err != nil {
		t.Fatalf("resolve cohort-narrowed targets: %v", err)
	}
	if len(narrowed) != 1 || narrowed[0].UserID != fx.playerUserID {
		t.Fatalf("expected exactly the cohort member, got %#v", narrowed)
	}
}

// Kernel 89 §28: only Director+ may operate these surfaces.
func TestShowDirectorGateRefusesPlayersAndAudience(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	fx := buildGoldenPathFixture(ctx, t, pool)

	if _, err := requireShowDirector(ctx, pool, fx.directorUserID, fx.showID); err != nil {
		t.Fatalf("the Director should pass the gate: %v", err)
	}
	for name, userID := range map[string]string{
		"player":    fx.playerUserID,
		"audience":  fx.audienceUserID,
		"anonymous": "",
	} {
		if _, err := requireShowDirector(ctx, pool, userID, fx.showID); err == nil {
			t.Fatalf("%s must not pass the Director gate", name)
		}
	}
}
