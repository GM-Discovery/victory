package ewrite

// Kernel 79 Goal C: proves the four new ewrite_object_links arms (Cue,
// index card, Scene element, dialogue Topic -- spec 7, 8.1-8.3) each
// resolve through their own location-scope lookup correctly: a non-Crew+
// actor is denied, a draft target never resolves into a player-facing
// payload, and a published target resolves with the right section anchor.
// Mirrors links_dbtest_test.go's TestEquipmentRuleLinkVertical shape;
// doesn't re-prove the shared SET NULL section-degrade behavior already
// covered there, since that's a DB FK property, not new code here.

// Deliberately imports none of internal/cues, internal/shows, or internal/
// scenes: cues imports internal/actions (execute.go), shows imports
// internal/network (http.go/sessions.go), and scenes imports internal/
// shows (placements.go) -- all three eventually reach internal/characters,
// which imports internal/ewrite (Kernel 79A's RuleLinksForCharacterSkills),
// so importing any of them here would close an import cycle for this
// package's own test binary. The whole fixture chain (production, show
// run, show, scene, placement, stage element, cue) is therefore built by
// direct SQL against the same columns those packages' own Create functions
// would fill, rather than via their APIs.

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
)

// buildGoalCFixture builds a location -> production -> show run -> show ->
// scene -> placement chain by direct SQL (see import-cycle note above for
// why not via showruns/shows/scenes' own Create functions).
type goalCFixture struct {
	locationID  string
	showID      string
	sceneID     string
	placementID string
}

func buildGoalCFixture(t *testing.T, pool *pgxpool.Pool, producerUserID string) goalCFixture {
	t.Helper()
	ctx := context.Background()
	suffix := testSuffix(t) + "_" + time.Now().UTC().Format("150405.000000000")

	var f goalCFixture
	f.locationID = amurrayLocation(t, pool)
	grantRole(t, pool, f.locationID, producerUserID, "producer")

	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1, $2, $3) RETURNING id::text
	`, f.locationID, "Goal C Test Production "+suffix, "goalc-test-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID) })

	var showRunID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO show_runs (location_id, production_id, title, slug, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text
	`, f.locationID, productionID, "Goal C Test Show Run "+suffix, "goalc-test-show-run-"+suffix, producerUserID).Scan(&showRunID); err != nil {
		t.Fatalf("insert show run fixture: %v", err)
	}

	if err := pool.QueryRow(ctx, `
		INSERT INTO shows (show_run_id, slug, title, created_by_user_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text
	`, showRunID, "goalc-test-show-"+suffix, "Goal C Test Show "+suffix, producerUserID).Scan(&f.showID); err != nil {
		t.Fatalf("insert show fixture: %v", err)
	}

	if err := pool.QueryRow(ctx, `
		INSERT INTO scenes (location_id, source_production_id, slug, title, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text
	`, f.locationID, productionID, "goalc-test-scene-"+suffix, "Goal C Test Scene", producerUserID).Scan(&f.sceneID); err != nil {
		t.Fatalf("insert scene fixture: %v", err)
	}

	if err := pool.QueryRow(ctx, `
		INSERT INTO show_scene_placements (show_id, scene_id, created_by_user_id)
		VALUES ($1, $2, $3)
		RETURNING id::text
	`, f.showID, f.sceneID, producerUserID).Scan(&f.placementID); err != nil {
		t.Fatalf("insert placement fixture: %v", err)
	}

	return f
}

func TestCueRuleLinkVertical(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	producer := insertTestUser(t, pool, "ew_cue_producer")
	cast := insertTestUser(t, pool, "ew_cue_cast")
	fixture := buildGoalCFixture(t, pool, producer)
	grantRole(t, pool, fixture.locationID, cast, "cast")

	var cueID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO cues (show_scene_placement_id, internal_name, trigger_scope)
		VALUES ($1, 'Test Cue', 'director_crew_only')
		RETURNING id::text
	`, fixture.placementID).Scan(&cueID); err != nil {
		t.Fatalf("insert cue: %v", err)
	}

	ruleset := mustCreateCollection(t, pool, producer, fixture.locationID, "", "ruleset", "Cue Rule Ruleset")
	pub := mustCreatePublication(t, pool, producer, ruleset.ID, "Cue Rules")
	mustSave(t, pool, producer, pub.ID, "# Cue Rules\n\n## Trigger Timing\n\ntext", "")
	sections, err := LoadSections(ctx, pool, pub.ID)
	if err != nil {
		t.Fatalf("sections: %v", err)
	}
	var section Section
	for _, s := range sections {
		if s.Title == "Trigger Timing" {
			section = s
		}
	}

	if _, err := SetCueRuleLink(ctx, pool, cast, cueID, pub.ID, section.ID); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected cast denial, got %v", err)
	}
	if _, err := SetCueRuleLink(ctx, pool, producer, cueID, pub.ID, section.ID); err != nil {
		t.Fatalf("set link: %v", err)
	}

	links, err := RuleLinksForCues(ctx, pool, []string{cueID})
	if err != nil {
		t.Fatalf("resolve (draft): %v", err)
	}
	if len(links) != 0 {
		t.Fatalf("draft publication must not resolve, got %+v", links)
	}

	if _, err := PublishPublication(ctx, pool, producer, pub.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	links, err = RuleLinksForCues(ctx, pool, []string{cueID})
	if err != nil {
		t.Fatalf("resolve (published): %v", err)
	}
	l, ok := links[cueID]
	if !ok || l.SectionAnchor != "trigger-timing" {
		t.Fatalf("expected resolved cue rule link, got %+v", links)
	}
}

func TestIndexCardRuleLinkVertical(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)
	crew := insertTestUser(t, pool, "ew_idxcard_crew")
	cast := insertTestUser(t, pool, "ew_idxcard_cast")
	grantRole(t, pool, loc, crew, "crew")
	grantRole(t, pool, loc, cast, "cast")

	var libraryID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO libraries (location_id, name) VALUES ($1, 'goal-c-test-library-'||$2) RETURNING id::text
	`, loc, testSuffix(t)).Scan(&libraryID); err != nil {
		t.Fatalf("insert library: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM libraries WHERE id = $1`, libraryID) })

	var elementID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO elements (library_id, name, slug, element_type, context_class, data)
		VALUES ($1, 'Test Index Card', $2, 'index_card', 'card', '{}'::jsonb)
		RETURNING id::text
	`, libraryID, "goal-c-index-card-"+testSuffix(t)).Scan(&elementID); err != nil {
		t.Fatalf("insert index card element: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM elements WHERE id = $1`, elementID) })

	ruleset := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Index Card Ruleset")
	pub := mustCreatePublication(t, pool, crew, ruleset.ID, "Index Card Rules")
	mustSave(t, pool, crew, pub.ID, "# Index Card Rules\n\n## Card Handling\n\ntext", "")
	sections, err := LoadSections(ctx, pool, pub.ID)
	if err != nil {
		t.Fatalf("sections: %v", err)
	}
	var section Section
	for _, s := range sections {
		if s.Title == "Card Handling" {
			section = s
		}
	}

	if _, err := SetIndexCardRuleLink(ctx, pool, cast, elementID, pub.ID, section.ID); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected cast denial, got %v", err)
	}
	if _, err := SetIndexCardRuleLink(ctx, pool, crew, elementID, pub.ID, section.ID); err != nil {
		t.Fatalf("set link: %v", err)
	}

	links, err := RuleLinksForIndexCards(ctx, pool, []string{elementID})
	if err != nil {
		t.Fatalf("resolve (draft): %v", err)
	}
	if len(links) != 0 {
		t.Fatalf("draft publication must not resolve, got %+v", links)
	}
	if _, err := PublishPublication(ctx, pool, crew, pub.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	links, err = RuleLinksForIndexCards(ctx, pool, []string{elementID})
	if err != nil {
		t.Fatalf("resolve (published): %v", err)
	}
	l, ok := links[elementID]
	if !ok || l.SectionAnchor != "card-handling" {
		t.Fatalf("expected resolved index card rule link, got %+v", links)
	}
}

func TestSceneElementRuleLinkVertical(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	producer := insertTestUser(t, pool, "ew_sceneel_producer")
	cast := insertTestUser(t, pool, "ew_sceneel_cast")
	fixture := buildGoalCFixture(t, pool, producer)
	grantRole(t, pool, fixture.locationID, cast, "cast")

	var elementID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO scene_stage_elements (scene_id, kind, label, created_by_user_id)
		VALUES ($1, 'token', 'Test Token', $2)
		RETURNING id::text
	`, fixture.sceneID, producer).Scan(&elementID); err != nil {
		t.Fatalf("insert scene stage element: %v", err)
	}

	ruleset := mustCreateCollection(t, pool, producer, fixture.locationID, "", "ruleset", "Scene Element Ruleset")
	pub := mustCreatePublication(t, pool, producer, ruleset.ID, "Scene Element Rules")
	mustSave(t, pool, producer, pub.ID, "# Scene Element Rules\n\n## Token Placement\n\ntext", "")
	sections, err := LoadSections(ctx, pool, pub.ID)
	if err != nil {
		t.Fatalf("sections: %v", err)
	}
	var section Section
	for _, s := range sections {
		if s.Title == "Token Placement" {
			section = s
		}
	}

	if _, err := SetSceneElementRuleLink(ctx, pool, cast, elementID, pub.ID, section.ID); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected cast denial, got %v", err)
	}
	if _, err := SetSceneElementRuleLink(ctx, pool, producer, elementID, pub.ID, section.ID); err != nil {
		t.Fatalf("set link: %v", err)
	}

	links, err := RuleLinksForSceneElements(ctx, pool, []string{elementID})
	if err != nil {
		t.Fatalf("resolve (draft): %v", err)
	}
	if len(links) != 0 {
		t.Fatalf("draft publication must not resolve, got %+v", links)
	}
	if _, err := PublishPublication(ctx, pool, producer, pub.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	links, err = RuleLinksForSceneElements(ctx, pool, []string{elementID})
	if err != nil {
		t.Fatalf("resolve (published): %v", err)
	}
	l, ok := links[elementID]
	if !ok || l.SectionAnchor != "token-placement" {
		t.Fatalf("expected resolved scene element rule link, got %+v", links)
	}
}

func TestDialogueTopicRuleLinkVertical(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	ctx := context.Background()
	loc := amurrayLocation(t, pool)
	crew := insertTestUser(t, pool, "ew_topic_crew")
	cast := insertTestUser(t, pool, "ew_topic_cast")
	grantRole(t, pool, loc, crew, "crew")
	grantRole(t, pool, loc, cast, "cast")

	var packetID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO dialogue_packets (location_id, slug, npc_name)
		VALUES ($1, $2, 'Test NPC')
		RETURNING id::text
	`, loc, "goal-c-test-packet-"+testSuffix(t)).Scan(&packetID); err != nil {
		t.Fatalf("insert dialogue packet: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM dialogue_packets WHERE id = $1`, packetID) })

	var topicID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO dialogue_topics (packet_id, topic_key, label, response_text)
		VALUES ($1, 'test_topic', 'Test Topic', 'A response')
		RETURNING id::text
	`, packetID).Scan(&topicID); err != nil {
		t.Fatalf("insert dialogue topic: %v", err)
	}

	ruleset := mustCreateCollection(t, pool, crew, loc, "", "ruleset", "Dialogue Topic Ruleset")
	pub := mustCreatePublication(t, pool, crew, ruleset.ID, "Dialogue Topic Rules")
	mustSave(t, pool, crew, pub.ID, "# Dialogue Topic Rules\n\n## Social Stance\n\ntext", "")
	sections, err := LoadSections(ctx, pool, pub.ID)
	if err != nil {
		t.Fatalf("sections: %v", err)
	}
	var section Section
	for _, s := range sections {
		if s.Title == "Social Stance" {
			section = s
		}
	}

	if _, err := SetDialogueTopicRuleLink(ctx, pool, cast, topicID, pub.ID, section.ID); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected cast denial, got %v", err)
	}
	if _, err := SetDialogueTopicRuleLink(ctx, pool, crew, topicID, pub.ID, section.ID); err != nil {
		t.Fatalf("set link: %v", err)
	}

	links, err := RuleLinksForDialogueTopics(ctx, pool, []string{topicID})
	if err != nil {
		t.Fatalf("resolve (draft): %v", err)
	}
	if len(links) != 0 {
		t.Fatalf("draft publication must not resolve, got %+v", links)
	}
	if _, err := PublishPublication(ctx, pool, crew, pub.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	links, err = RuleLinksForDialogueTopics(ctx, pool, []string{topicID})
	if err != nil {
		t.Fatalf("resolve (published): %v", err)
	}
	l, ok := links[topicID]
	if !ok || l.SectionAnchor != "social-stance" {
		t.Fatalf("expected resolved dialogue topic rule link, got %+v", links)
	}
}
