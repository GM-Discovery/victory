// k90fixture builds a real Show with TWO Cohorts, three Players, an Audience
// watcher, a dedicated Scene carrying durable stage objects, a bound
// participant interaction and a drawing object, in the DISPOSABLE TEST
// DATABASE, and prints the ids and session cookies the Kernel 90 proofs drive.
//
// Shape and safety rails are inherited from backend/cmd/k89fixture: it refuses
// any database whose URL is not obviously a test database, mints session
// cookies through sessions.CreateSession (never hand-inserting a token hash),
// sets req.RemoteAddr so the inet column gets a valid value, and grants BOTH a
// location_memberships role and an access_grants row -- the split that makes
// every HTTP call succeed while the WebSocket upgrade 403s.
//
// What Kernel 90 needs beyond Kernel 89's fixture, and why:
//
//   - TWO Cohorts with a Player in each, because §16's whole claim is that one
//     object can be visible to Cohort A and hidden from Cohort B. A single
//     Cohort cannot distinguish "scoping works" from "hiding works".
//   - A third, Ungrouped Player, because the empty-cohort case is where a
//     naive scope comparison leaks every cohort-scoped object at once.
//   - An Audience viewer, because §18 forbids assuming the Audience sees what
//     the Cast sees, and that needs a real audience-tier session.
//   - Its OWN Scene with its OWN elements, rather than borrowing whichever
//     Scene happens to exist at the location. The proof asserts on exact
//     objects, so it must own them.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/sessions"
	"victory/backend/internal/showings"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
)

type out struct {
	LocationID  string `json:"location_id"`
	VenueSlug   string `json:"venue_slug"`
	ShowRunID   string `json:"show_run_id"`
	ShowID      string `json:"show_id"`
	SessionID   string `json:"session_id"`
	SceneID     string `json:"scene_id"`
	PlacementID string `json:"placement_id"`

	DirectorID    string `json:"director_user_id"`
	DirectorToken string `json:"director_cookie"`

	CohortAID     string `json:"cohort_a_id"`
	CohortBID     string `json:"cohort_b_id"`
	PlayerAID     string `json:"player_a_user_id"`
	PlayerAToken  string `json:"player_a_cookie"`
	PlayerACardID string `json:"player_a_character_card_id"`
	PlayerBID     string `json:"player_b_user_id"`
	PlayerBToken  string `json:"player_b_cookie"`
	PlayerBCardID string `json:"player_b_character_card_id"`
	// The Ungrouped Player: on the roster, in no Cohort.
	PlayerCID     string `json:"player_c_user_id"`
	PlayerCToken  string `json:"player_c_cookie"`
	PlayerCCardID string `json:"player_c_character_card_id"`

	AudienceID    string `json:"audience_user_id"`
	AudienceToken string `json:"audience_cookie"`

	// Durable stage objects, all on one Scene so §16/§47's "one Scene, scoped
	// projection, no forks" is what the proof actually exercises.
	TokenAID string `json:"token_a_scene_stage_element_id"`
	TokenBID string `json:"token_b_scene_stage_element_id"`
	// A map_backdrop, present ONLY so the proof can demonstrate §3's boundary
	// against a real row rather than an invented id.
	MapID string `json:"map_scene_stage_element_id"`
	// The interaction bound to TokenB, which is what carries interaction state.
	InteractionID string `json:"participant_interaction_id"`
	DrawingID     string `json:"drawing_object_id"`
}

func main() {
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		log.Fatal("TEST_DATABASE_URL is required -- this program only runs against a test database")
	}
	// Same shape as internal/dbtest's guard, duplicated rather than imported
	// because dbtest's gate takes a *testing.T.
	if !strings.Contains(dsn, "victory_test") {
		log.Fatalf("refusing to run against a database that is not obviously a test database: %s", dsn)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	suffix := time.Now().UTC().Format("150405")
	req, _ := http.NewRequest("GET", "http://127.0.0.1/", nil)
	// clientIP reads RemoteAddr; an empty one produces an invalid inet.
	req.RemoteAddr = "127.0.0.1:0"
	var o out

	must := func(label string, err error) {
		if err != nil {
			log.Fatalf("%s: %v", label, err)
		}
	}

	must("location", pool.QueryRow(ctx,
		`SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&o.LocationID))

	// The real `catharsis` row, for the same reason k89fixture uses it: /ws/*
	// and /api/world/* routes exist only for the venues main.go registers by
	// name, so a throwaway slug 404s. Any leftover open session is closed
	// first, and scripts/smoke/kernel90-run.sh closes the one this opens --
	// Catharsis is a singleton-session venue and a stray open session there
	// breaks internal/shows and internal/showtime for the whole shared test
	// database. Do not run this concurrently with `go test ./internal/shows/...`.
	o.VenueSlug = "catharsis"
	var venueID string
	must("venue", pool.QueryRow(ctx,
		`SELECT id::text FROM venues WHERE slug = $1`, o.VenueSlug).Scan(&venueID))
	if _, err := pool.Exec(ctx, `
		UPDATE sessions SET status = 'closed', ended_at = NOW()
		WHERE venue_id = $1::uuid AND status IN ('rehearsal', 'live')
	`, venueID); err != nil {
		log.Fatalf("close leftover sessions: %v", err)
	}

	mkUser := func(handle string) (string, string) {
		var id string
		must("user "+handle, pool.QueryRow(ctx,
			`INSERT INTO users (handle, display_name) VALUES ($1, $2) RETURNING id::text`,
			handle+"_"+suffix, handle).Scan(&id))
		token, _, err := sessions.CreateSession(ctx, pool, id, 24*time.Hour, req)
		must("session for "+handle, err)
		return id, token
	}

	grantVenue := func(userID, role string) {
		_, err := pool.Exec(ctx, `
			INSERT INTO location_memberships (location_id, user_id, role, active)
			VALUES ($1::uuid, $2::uuid, $3::location_role, TRUE)
			ON CONFLICT (location_id, user_id, role) DO UPDATE SET active = TRUE
		`, o.LocationID, userID, role)
		must("location membership "+role, err)
		_, err = pool.Exec(ctx, `
			INSERT INTO access_grants (location_id, user_id, grant_type, venue_id, granted_by_user_id)
			VALUES ($1::uuid, $2::uuid, 'venue_access', $3::uuid, $2::uuid)
		`, o.LocationID, userID, venueID)
		must("venue access grant", err)
	}

	o.DirectorID, o.DirectorToken = mkUser("k90director")
	o.PlayerAID, o.PlayerAToken = mkUser("k90playera")
	o.PlayerBID, o.PlayerBToken = mkUser("k90playerb")
	o.PlayerCID, o.PlayerCToken = mkUser("k90playerc")
	o.AudienceID, o.AudienceToken = mkUser("k90audience")

	grantVenue(o.DirectorID, "director")
	grantVenue(o.PlayerAID, "cast")
	grantVenue(o.PlayerBID, "cast")
	grantVenue(o.PlayerCID, "cast")
	// The Audience viewer is granted a 'cast' LOCATION membership and is
	// deliberately left OFF the Show Run roster. That combination is what
	// produces a genuine audience-tier viewer, and the reason is worth
	// recording because it is counter-intuitive:
	//
	//   - access.ResolveVisibleVenues gates Catharsis on
	//     lm.role IN ('producer','director','cast','crew') AND an access_grants
	//     row. A membership role of literally 'audience' therefore cannot
	//     reach the venue AT ALL -- a pre-existing platform condition, the
	//     same shape as Kernel 89's First Theater finding, and not something
	//     Kernel 90 widens (see the reportback).
	//   - participation.ResolveParticipationContext only promotes
	//     producer/director from a location membership (step 2) and everything
	//     else from the Show Run roster (step 3). A venue-admitted user with
	//     no roster row falls through to step 4 and resolves to
	//     ViewerModeAudience.
	//
	// So this user genuinely IS the Audience: a spectator who may enter the
	// venue but is not in this Show's cast. viewerRole resolves to "audience",
	// which is what the §18 assertions depend on -- they would be worthless if
	// this user resolved to "player".
	grantVenue(o.AudienceID, "cast")

	var productionID string
	must("production", pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1::uuid, $2, $3) RETURNING id::text
	`, o.LocationID, "K90 Production "+suffix, "k90-production-"+suffix).Scan(&productionID))

	sr, err := showruns.CreateShowRun(ctx, pool, o.DirectorID, productionID, showruns.CreateShowRunInput{
		Title: "K90 Visibility Run " + suffix, Slug: "k90-run-" + suffix,
	})
	must("show run", err)
	o.ShowRunID = sr.ID

	s, err := shows.CreateShow(ctx, pool, o.DirectorID, sr.ID, shows.CreateShowInput{
		Title: "K90 Visibility Show " + suffix, Slug: "k90-show-" + suffix,
	})
	must("show", err)
	o.ShowID = s.ID

	// Roster rows inserted directly rather than through the two-punch ticket
	// flow: this fixture proves Kernel 90, and tickets.SecondPunch has its own
	// package's tests.
	addRoster := func(userID, cardName string) string {
		_, err := pool.Exec(ctx, `
			INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id)
			VALUES ($1::uuid, $2::uuid, 'player', $3::uuid)
		`, o.ShowRunID, userID, o.DirectorID)
		must("roster", err)

		var cardID string
		must("character", pool.QueryRow(ctx, `
			INSERT INTO character_cards (owner_user_id, location_id, name)
			VALUES ($1::uuid, $2::uuid, $3) RETURNING id::text
		`, userID, o.LocationID, cardName).Scan(&cardID))
		_, err = pool.Exec(ctx, `
			UPDATE show_run_roster_members SET character_card_id = $2::uuid
			WHERE show_run_id = $1::uuid AND user_id = $3::uuid
		`, o.ShowRunID, cardID, userID)
		must("select character", err)
		return cardID
	}
	o.PlayerACardID = addRoster(o.PlayerAID, "Arena Recruit A")
	o.PlayerBCardID = addRoster(o.PlayerBID, "Arena Recruit B")
	o.PlayerCCardID = addRoster(o.PlayerCID, "Arena Recruit C")

	// A dedicated Scene, so the proof owns every object it asserts on.
	must("scene", pool.QueryRow(ctx, `
		INSERT INTO scenes (location_id, source_production_id, slug, title, default_venue_id, status, created_by_user_id)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5::uuid, 'ready', $6::uuid)
		RETURNING id::text
	`, o.LocationID, productionID, "k90-arena-"+suffix, "K90 Training Arena "+suffix, venueID, o.DirectorID).Scan(&o.SceneID))

	must("placement", pool.QueryRow(ctx, `
		INSERT INTO show_scene_placements (show_id, scene_id, venue_id, status)
		VALUES ($1::uuid, $2::uuid, $3::uuid, 'ready') RETURNING id::text
	`, o.ShowID, o.SceneID, venueID).Scan(&o.PlacementID))
	_, err = pool.Exec(ctx,
		`UPDATE shows SET current_show_scene_placement_id = $2::uuid WHERE id = $1::uuid`, o.ShowID, o.PlacementID)
	must("current placement", err)

	mkElement := func(kind, label string, sortOrder int) string {
		var id string
		must("stage element "+kind, pool.QueryRow(ctx, `
			INSERT INTO scene_stage_elements (scene_id, kind, label, position, sort_order, created_by_user_id)
			VALUES ($1::uuid, $2, $3, $4::jsonb, $5, $6::uuid) RETURNING id::text
		`, o.SceneID, kind, label,
			fmt.Sprintf(`{"anchor":"stage","x":%d,"y":%d}`, 100+sortOrder*80, 120),
			sortOrder, o.DirectorID).Scan(&id))
		return id
	}
	o.TokenAID = mkElement("token", "Practice Dummy", 1)
	o.TokenBID = mkElement("token", "Weapon Cabinet", 2)
	o.MapID = mkElement("map_backdrop", "Arena Floor", 0)

	// A participant interaction bound to TokenB, so the proof has a real
	// interaction to disable and a real invoke path to be refused on.
	must("interaction", pool.QueryRow(ctx, `
		INSERT INTO participant_interactions (
			show_scene_placement_id, internal_name, stage_button_label,
			interaction_type, configuration_json, enabled, sort_order, created_by_user_id
		)
		VALUES ($1::uuid, $2, 'Open the weapon cabinet', 'open_equip_mode', '{}'::jsonb, TRUE, 0, $3::uuid)
		RETURNING id::text
	`, o.PlacementID, "k90-cabinet-"+suffix, o.DirectorID).Scan(&o.InteractionID))
	_, err = pool.Exec(ctx, `
		INSERT INTO stage_element_bindings (
			scene_stage_element_id, binding_type, participant_interaction_id, created_by_user_id
		)
		VALUES ($1::uuid, 'participant_interaction', $2::uuid, $3::uuid)
	`, o.TokenBID, o.InteractionID, o.DirectorID)
	must("binding", err)

	// A Kernel 87 drawing object, Show-scoped and un-cohorted so its ONLY
	// scoping comes from canonical state (§28).
	must("drawing", pool.QueryRow(ctx, `
		INSERT INTO drawing_objects (show_id, creator_user_id, object_type, geometry)
		VALUES ($1::uuid, $2::uuid, 'rectangle', '{"x":40,"y":40,"width":90,"height":60}'::jsonb)
		RETURNING id::text
	`, o.ShowID, o.DirectorID).Scan(&o.DrawingID))

	// A live Session linked to the Show, plus the showing and participant rows
	// CanAct requires. The Audience is a real audience-tier participant, not a
	// Player with a different label -- §18's claim depends on that.
	must("session", pool.QueryRow(ctx, `
		INSERT INTO sessions (venue_id, status, show_id)
		VALUES ($1::uuid, 'live', $2::uuid) RETURNING id::text
	`, venueID, o.ShowID).Scan(&o.SessionID))
	_, err = showings.EnsureForSession(ctx, pool, o.SessionID, o.DirectorID)
	must("showing", err)
	for _, p := range []struct{ id, role string }{
		{o.DirectorID, "director"},
		{o.PlayerAID, "cast"}, {o.PlayerBID, "cast"}, {o.PlayerCID, "cast"},
		{o.AudienceID, "audience"},
	} {
		_, err = pool.Exec(ctx, `
			INSERT INTO session_participants (session_id, user_id, role)
			VALUES ($1::uuid, $2::uuid, $3::location_role)
			ON CONFLICT DO NOTHING
		`, o.SessionID, p.id, p.role)
		must("session participant "+p.role, err)
	}

	// Two Cohorts. Player A in A, Player B in B, Player C in neither.
	mkCohort := func(serial int, slug, name string) string {
		var id string
		must("cohort "+name, pool.QueryRow(ctx, `
			INSERT INTO show_cohorts (show_id, serial_number, slug, name, created_by_user_id)
			VALUES ($1::uuid, $2, $3, $4, $5::uuid) RETURNING id::text
		`, o.ShowID, serial, slug+"-"+suffix, name, o.DirectorID).Scan(&id))
		return id
	}
	o.CohortAID = mkCohort(1, "k90-cohort-a", "Cohort A")
	o.CohortBID = mkCohort(2, "k90-cohort-b", "Cohort B")
	for _, a := range []struct{ userID, cohortID string }{
		{o.PlayerAID, o.CohortAID}, {o.PlayerBID, o.CohortBID},
	} {
		_, err = pool.Exec(ctx, `
			INSERT INTO show_cohort_assignments (show_id, cohort_id, user_id, assigned_by_user_id)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid)
		`, o.ShowID, a.cohortID, a.userID, o.DirectorID)
		must("cohort assignment", err)
	}

	encoded, _ := json.MarshalIndent(o, "", "  ")
	fmt.Println(string(encoded))
}
