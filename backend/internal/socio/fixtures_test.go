package socio

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/playerprofile"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
	"victory/backend/internal/tickets"
)

func openTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return dbtest.OpenTestPool(t)
}

func testSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000000")
}

func insertTestUser(t *testing.T, pool *pgxpool.Pool, handlePrefix string) string {
	t.Helper()
	ctx := context.Background()
	handle := handlePrefix + "_" + testSuffix(t)

	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name) VALUES ($1, $2) RETURNING id::text
	`, handle, handlePrefix).Scan(&userID); err != nil {
		t.Fatalf("insert user %q: %v", handle, err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM character_socio_status_effects WHERE applied_by_user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM character_socio_state WHERE character_card_id IN (SELECT id FROM character_cards WHERE owner_user_id = $1)`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM show_run_roster_members WHERE user_id = $1 OR added_by_user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM character_cards WHERE owner_user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM shows WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM show_runs WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM player_profile_workbooks WHERE user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(bg, `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID
}

func grantLocationRole(t *testing.T, pool *pgxpool.Pool, locationID, userID, role string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, $3::location_role, TRUE)
		ON CONFLICT (location_id, user_id, role) DO UPDATE SET active = TRUE
	`, locationID, userID, role); err != nil {
		t.Fatalf("grant location role %q: %v", role, err)
	}
}

func showFixture(t *testing.T, pool *pgxpool.Pool, directorUserID string) (showRunID, showID, locationID string) {
	t.Helper()
	ctx := context.Background()
	suffix := testSuffix(t)

	if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID); err != nil {
		t.Fatalf("load amurray-family location: %v", err)
	}
	grantLocationRole(t, pool, locationID, directorUserID, "director")

	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1, $2, $3) RETURNING id::text
	`, locationID, "Test Production "+suffix, "test-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID) })

	sr, err := showruns.CreateShowRun(ctx, pool, directorUserID, productionID, showruns.CreateShowRunInput{
		Title: "Test Show Run " + suffix, Slug: "test-show-run-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show run fixture: %v", err)
	}
	s, err := shows.CreateShow(ctx, pool, directorUserID, sr.ID, shows.CreateShowInput{
		Title: "Test Show " + suffix, Slug: "test-show-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show fixture: %v", err)
	}
	return sr.ID, s.ID, locationID
}

func playerFixture(t *testing.T, pool *pgxpool.Pool, directorUserID, showRunID, locationID, handlePrefix string) (userID, characterCardID string) {
	t.Helper()
	ctx := context.Background()
	userID = insertTestUser(t, pool, handlePrefix)

	wb, err := playerprofile.EnsureWorkbook(ctx, pool, userID)
	if err != nil {
		t.Fatalf("ensure workbook for %s: %v", handlePrefix, err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO character_cards (owner_user_id, location_id, name) VALUES ($1, $2, $3) RETURNING id::text
	`, userID, locationID, handlePrefix+" Character").Scan(&characterCardID); err != nil {
		t.Fatalf("insert character card: %v", err)
	}
	ticket, err := tickets.InviteFromDirector(ctx, pool, directorUserID, showRunID, wb.ID, "")
	if err != nil {
		t.Fatalf("invite %s to show run: %v", handlePrefix, err)
	}
	if _, err := tickets.SecondPunch(ctx, pool, userID, ticket.ID); err != nil {
		t.Fatalf("accept invite for %s: %v", handlePrefix, err)
	}
	if _, err := showruns.SelectCharacter(ctx, pool, userID, showRunID, characterCardID); err != nil {
		t.Fatalf("select character: %v", err)
	}
	return userID, characterCardID
}
