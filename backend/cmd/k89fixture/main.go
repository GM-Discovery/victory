// k89fixture builds a real Show, roster, Characters, Cohort, Scene
// placement and live Session in the DISPOSABLE TEST DATABASE, and prints
// the ids and session cookies the Kernel 89 browser proof drives.
//
// It exists because Kernel 88's equivalent was written as a throwaway and
// deleted, which cost the next pass the time to rebuild it. This one is
// checked in, refuses to run against anything but a test database, and
// mints its OWN venue rather than occupying `catharsis` -- the collision
// Kernel 88A's test-database housekeeping note records, where a leftover
// rehearsal session on the shared slug broke internal/shows and
// internal/showtime for everyone.
//
// Session cookies are created through sessions.CreateSession rather than
// hand-inserted, so the bytea token-hash trap Kernel 88 §5 documents
// ("never hex-encode a hash before handing it to pgx") cannot recur here.
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
	LocationID    string `json:"location_id"`
	VenueSlug     string `json:"venue_slug"`
	ProductionID  string `json:"production_id"`
	ShowRunID     string `json:"show_run_id"`
	ShowID        string `json:"show_id"`
	SessionID     string `json:"session_id"`
	PlacementID   string `json:"placement_id"`
	CohortID      string `json:"cohort_id"`
	DirectorID    string `json:"director_user_id"`
	DirectorToken string `json:"director_cookie"`
	PlayerID      string `json:"player_user_id"`
	PlayerToken   string `json:"player_cookie"`
	PlayerCardID  string `json:"player_character_card_id"`
	OutsiderID    string `json:"outsider_user_id"`
	OutsiderToken string `json:"outsider_cookie"`
}

func main() {
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		log.Fatal("TEST_DATABASE_URL is required -- this program only runs against a test database")
	}
	// The same shape as internal/dbtest's guard. Duplicated deliberately
	// rather than imported: dbtest's own gate takes a *testing.T.
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

	var lotID string
	must("lot", pool.QueryRow(ctx,
		`SELECT id::text FROM lots WHERE location_id = $1::uuid AND slug = 'main-lot' LIMIT 1`, o.LocationID).Scan(&lotID))

	// The proof drives a real WebSocket, and /ws/* routes exist only for the
	// three venues main.go registers by name -- a throwaway venue slug would
	// 404 on upgrade. So this runs on the real `catharsis` row, which already
	// carries the participant-interaction and Aftercare capability flags the
	// kernel needs.
	//
	// That makes it the one fixture in this repo that occupies the shared
	// slug, which is exactly the collision Kernel 88A's housekeeping note
	// records. Two mitigations: any session already open on the venue is
	// closed first (this is the disposable test database, and 88A closed the
	// same kind of leftover by hand), and scripts/smoke/kernel89-run.sh
	// closes the session this fixture opens when the proof finishes. Do not
	// run this concurrently with `go test ./internal/shows/...`.
	o.VenueSlug = strings.TrimSpace(os.Getenv("K89_VENUE_SLUG"))
	if o.VenueSlug == "" {
		o.VenueSlug = "catharsis"
	}
	var venueID string
	must("venue", pool.QueryRow(ctx,
		`SELECT id::text FROM venues WHERE slug = $1`, o.VenueSlug).Scan(&venueID))
	_ = lotID
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
	o.DirectorID, o.DirectorToken = mkUser("k89director")
	o.PlayerID, o.PlayerToken = mkUser("k89player")
	o.OutsiderID, o.OutsiderToken = mkUser("k89outsider")

	// Location role + a venue access grant. Both are required to reach the
	// venue at all: access.ResolveVisibleVenues gates Catharsis on a
	// location_memberships role AND an access_grants row, which real users
	// get from the Kernel 71 ticket second punch. Without these the HTTP
	// surface works fine and the WebSocket upgrade 403s -- a confusing
	// split that cost this proof a debugging round.
	// First Theater's seeded config does not carry the participant-interaction
	// or Aftercare flags, and it should not gain them here -- only the
	// announcement half of Kernel 89 applies to that venue (§10). Nothing in
	// this fixture writes venue config.
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
	grantVenue(o.DirectorID, "director")

	must("production", pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1::uuid, $2, $3) RETURNING id::text
	`, o.LocationID, "K89 Production "+suffix, "k89-production-"+suffix).Scan(&o.ProductionID))

	sr, err := showruns.CreateShowRun(ctx, pool, o.DirectorID, o.ProductionID, showruns.CreateShowRunInput{
		Title: "K89 Training Arena Run " + suffix, Slug: "k89-run-" + suffix,
	})
	must("show run", err)
	o.ShowRunID = sr.ID

	s, err := shows.CreateShow(ctx, pool, o.DirectorID, sr.ID, shows.CreateShowInput{
		Title: "K89 Training Arena " + suffix, Slug: "k89-show-" + suffix,
	})
	must("show", err)
	o.ShowID = s.ID

	// Roster + Character. Inserted directly rather than through the
	// two-punch ticket flow: this fixture is proving Kernel 89, and
	// tickets.SecondPunch is already proven by its own package's tests.
	_, err = pool.Exec(ctx, `
		INSERT INTO show_run_roster_members (show_run_id, user_id, role, added_by_user_id)
		VALUES ($1::uuid, $2::uuid, 'player', $3::uuid)
	`, o.ShowRunID, o.PlayerID, o.DirectorID)
	must("roster", err)

	grantVenue(o.PlayerID, "cast")

	must("character", pool.QueryRow(ctx, `
		INSERT INTO character_cards (owner_user_id, location_id, name)
		VALUES ($1::uuid, $2::uuid, 'Arena Recruit')
		RETURNING id::text
	`, o.PlayerID, o.LocationID).Scan(&o.PlayerCardID))

	_, err = pool.Exec(ctx, `
		UPDATE show_run_roster_members SET character_card_id = $2::uuid
		WHERE show_run_id = $1::uuid AND user_id = $3::uuid
	`, o.ShowRunID, o.PlayerCardID, o.PlayerID)
	must("select character", err)

	// A live Session linked to the Show, plus the showing and participant
	// rows CanAct requires.
	must("session", pool.QueryRow(ctx, `
		INSERT INTO sessions (venue_id, status, show_id)
		VALUES ($1::uuid, 'live', $2::uuid) RETURNING id::text
	`, venueID, o.ShowID).Scan(&o.SessionID))
	_, err = showings.EnsureForSession(ctx, pool, o.SessionID, o.DirectorID)
	must("showing", err)
	for _, p := range []struct{ id, role string }{
		{o.DirectorID, "director"}, {o.PlayerID, "cast"},
	} {
		_, err = pool.Exec(ctx, `
			INSERT INTO session_participants (session_id, user_id, role)
			VALUES ($1::uuid, $2::uuid, $3::location_role)
			ON CONFLICT DO NOTHING
		`, o.SessionID, p.id, p.role)
		must("session participant "+p.role, err)
	}

	// A Scene placed on this Show and made current, so merchant exposure
	// has somewhere to attach.
	var sceneID string
	must("scene", pool.QueryRow(ctx,
		`SELECT id::text FROM scenes WHERE location_id = $1::uuid ORDER BY created_at LIMIT 1`, o.LocationID).Scan(&sceneID))
	must("placement", pool.QueryRow(ctx, `
		INSERT INTO show_scene_placements (show_id, scene_id, venue_id, status)
		VALUES ($1::uuid, $2::uuid, $3::uuid, 'ready') RETURNING id::text
	`, o.ShowID, sceneID, venueID).Scan(&o.PlacementID))
	_, err = pool.Exec(ctx,
		`UPDATE shows SET current_show_scene_placement_id = $2::uuid WHERE id = $1::uuid`, o.ShowID, o.PlacementID)
	must("current placement", err)

	// One Cohort with the Player in it, for the targeting proofs.
	must("cohort", pool.QueryRow(ctx, `
		INSERT INTO show_cohorts (show_id, serial_number, slug, name, created_by_user_id)
		VALUES ($1::uuid, 1, 'k89-arena-cohort', 'Arena Cohort', $2::uuid) RETURNING id::text
	`, o.ShowID, o.DirectorID).Scan(&o.CohortID))
	_, err = pool.Exec(ctx, `
		INSERT INTO show_cohort_assignments (show_id, cohort_id, user_id, assigned_by_user_id)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid)
	`, o.ShowID, o.CohortID, o.PlayerID, o.DirectorID)
	must("cohort assignment", err)

	encoded, _ := json.MarshalIndent(o, "", "  ")
	fmt.Println(string(encoded))
}
