package ewrite

// Kernel 78 Goal H tests: an equipment item bound to an exact rule
// section, resolved only while the target is published, degrading to the
// publication top when the heading disappears (section_id SET NULL).

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
)

func insertTestEquipmentItem(t *testing.T, pool *pgxpool.Pool, locationID, creatorID, name string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO equipment_items (location_id, name, slug, short_description, created_by_user_id)
		VALUES ($1, $2, $3, 'test item', $4)
		RETURNING id::text
	`, locationID, name, "ew-test-"+testSuffix(t), creatorID).Scan(&id); err != nil {
		t.Fatalf("insert equipment item: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM equipment_items WHERE id = $1`, id)
	})
	return id
}

func TestEquipmentRuleLinkVertical(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)
	crew := insertTestUser(t, pool, "ew_link_crew")
	cast := insertTestUser(t, pool, "ew_link_cast")
	grantRole(t, pool, loc, crew, "crew")
	grantRole(t, pool, loc, cast, "cast")

	item := insertTestEquipmentItem(t, pool, loc, crew, "Test Dagger of Linking")

	ruleset := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Link Ruleset")
	pub := mustCreatePublication(t, pool, crew, ruleset.ID, "Weapon Rules")
	r1 := mustSave(t, pool, crew, pub.ID, "# Weapon Rules\n\n## Daggers\n\nstabby rules", "")

	sections, err := LoadSections(ctx, pool, pub.ID)
	if err != nil {
		t.Fatalf("sections: %v", err)
	}
	var daggers Section
	for _, s := range sections {
		if s.Title == "Daggers" {
			daggers = s
		}
	}
	if daggers.ID == "" {
		t.Fatal("daggers section missing")
	}

	// Cast cannot create links (client manipulation is not authority).
	if _, err := SetEquipmentItemRuleLink(ctx, pool, cast, item, pub.ID, daggers.ID); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected cast link denial, got %v", err)
	}

	if _, err := SetEquipmentItemRuleLink(ctx, pool, crew, item, pub.ID, daggers.ID); err != nil {
		t.Fatalf("set link: %v", err)
	}

	// Draft target: link exists but resolves to nothing player-facing.
	links, err := RuleLinksForEquipmentItems(ctx, pool, []string{item})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(links) != 0 {
		t.Fatalf("draft publication must not resolve into player payloads: %+v", links)
	}

	if _, err := PublishPublication(ctx, pool, crew, pub.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	links, err = RuleLinksForEquipmentItems(ctx, pool, []string{item})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	l, ok := links[item]
	if !ok || l.SectionAnchor != "daggers" || l.PublicationID != pub.ID {
		t.Fatalf("expected published rule link to daggers, got %+v", links)
	}

	// Remove the heading: section row deleted, link degrades to
	// publication top (SET NULL), never dangles or vanishes.
	if _, _, err := SavePublicationSource(ctx, pool, crew, pub.ID, "# Weapon Rules\n\nno more daggers section", r1.RevisionID); err != nil {
		t.Fatalf("resave: %v", err)
	}
	links, err = RuleLinksForEquipmentItems(ctx, pool, []string{item})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	l, ok = links[item]
	if !ok || l.SectionAnchor != "" || l.PublicationID != pub.ID {
		t.Fatalf("expected degraded link to publication top, got %+v", links)
	}
}
