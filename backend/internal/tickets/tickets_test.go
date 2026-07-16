package tickets_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/dbtest"
	"victory/backend/internal/showruns"
	"victory/backend/internal/tickets"
)

func ticketTestSuffix(t *testing.T) string {
	t.Helper()
	return strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000000")
}

func ticketTestUser(t *testing.T, pool *pgxpool.Pool, handlePrefix string) string {
	t.Helper()
	ctx := context.Background()
	handle := handlePrefix + "_" + ticketTestSuffix(t)

	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		RETURNING id::text
	`, handle, handlePrefix).Scan(&userID); err != nil {
		t.Fatalf("insert user %q: %v", handle, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_run_tickets WHERE user_id = $1 OR player_punched_by_user_id = $1 OR director_punched_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_run_roster_members WHERE user_id = $1 OR added_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM access_grants WHERE user_id = $1 OR granted_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM show_runs WHERE created_by_user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM player_profile_workbooks WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID
}

func ticketTestProfileID(t *testing.T, pool *pgxpool.Pool, userID string) string {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO player_profile_workbooks (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING
	`, userID); err != nil {
		t.Fatalf("ensure profile for %s: %v", userID, err)
	}
	var profileID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM player_profile_workbooks WHERE user_id = $1`, userID).Scan(&profileID); err != nil {
		t.Fatalf("load profile id for %s: %v", userID, err)
	}
	return profileID
}

type ticketFixture struct {
	locationID string
	showRunID  string
}

func buildTicketFixture(t *testing.T, pool *pgxpool.Pool, producerUserID string) ticketFixture {
	t.Helper()
	ctx := context.Background()
	suffix := ticketTestSuffix(t)

	var f ticketFixture
	if err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&f.locationID); err != nil {
		t.Fatalf("load amurray-family location: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1, $2, 'producer', TRUE)
		ON CONFLICT (location_id, user_id, role) DO UPDATE SET active = TRUE
	`, f.locationID, producerUserID); err != nil {
		t.Fatalf("grant producer role: %v", err)
	}

	var productionID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug) VALUES ($1, $2, $3) RETURNING id::text
	`, f.locationID, "Ticket Test Production "+suffix, "ticket-test-production-"+suffix).Scan(&productionID); err != nil {
		t.Fatalf("insert production: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM productions WHERE id = $1`, productionID) })

	sr, err := showruns.CreateShowRun(ctx, pool, producerUserID, productionID, showruns.CreateShowRunInput{
		Title: "Ticket Test Show Run " + suffix, Slug: "ticket-test-show-run-" + suffix,
	})
	if err != nil {
		t.Fatalf("create show run fixture: %v", err)
	}
	f.showRunID = sr.ID
	return f
}

func rosterRole(t *testing.T, pool *pgxpool.Pool, showRunID, userID string) string {
	t.Helper()
	var role string
	err := pool.QueryRow(context.Background(), `
		SELECT role FROM show_run_roster_members
		WHERE show_run_id = $1 AND user_id = $2 AND removed_at IS NULL
	`, showRunID, userID).Scan(&role)
	if err != nil {
		return ""
	}
	return role
}

// TestPlayerRequestThenDirectorApprovalValidatesAtomically covers the
// Player-initiated direction end to end: first punch creates a
// pending_director ticket granting nothing, the second punch (Director
// approval) atomically marks it valid and creates exactly one Player
// roster row.
func TestPlayerRequestThenDirectorApprovalValidatesAtomically(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	producer := ticketTestUser(t, pool, "ticket_producer")
	player := ticketTestUser(t, pool, "ticket_player")
	f := buildTicketFixture(t, pool, producer)

	req, err := tickets.RequestFromPlayer(context.Background(), pool, player, f.showRunID, "let me in")
	if err != nil {
		t.Fatalf("RequestFromPlayer: %v", err)
	}
	if req.Status != tickets.StatusPendingDirector {
		t.Fatalf("expected status pending_director after first punch, got %q", req.Status)
	}
	if rosterRole(t, pool, f.showRunID, player) != "" {
		t.Fatalf("expected no roster row after one punch, found role %q", rosterRole(t, pool, f.showRunID, player))
	}

	valid, err := tickets.SecondPunch(context.Background(), pool, producer, req.ID)
	if err != nil {
		t.Fatalf("SecondPunch (director approval): %v", err)
	}
	if valid.Status != tickets.StatusValid {
		t.Fatalf("expected status valid, got %q", valid.Status)
	}
	if valid.RosterMembershipID == "" {
		t.Fatalf("expected roster_membership_id to be set on the valid ticket")
	}
	if role := rosterRole(t, pool, f.showRunID, player); role != "player" {
		t.Fatalf("expected active player roster row, got role %q", role)
	}
}

// TestDirectorInviteThenPlayerAcceptanceValidatesAtomically covers the
// Director-initiated direction.
func TestDirectorInviteThenPlayerAcceptanceValidatesAtomically(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	producer := ticketTestUser(t, pool, "ticket_producer")
	player := ticketTestUser(t, pool, "ticket_player")
	f := buildTicketFixture(t, pool, producer)
	profileID := ticketTestProfileID(t, pool, player)

	invite, err := tickets.InviteFromDirector(context.Background(), pool, producer, f.showRunID, profileID, "join us")
	if err != nil {
		t.Fatalf("InviteFromDirector: %v", err)
	}
	if invite.Status != tickets.StatusPendingPlayer {
		t.Fatalf("expected status pending_player after director's first punch, got %q", invite.Status)
	}
	if rosterRole(t, pool, f.showRunID, player) != "" {
		t.Fatalf("expected no roster row after one punch")
	}

	valid, err := tickets.SecondPunch(context.Background(), pool, player, invite.ID)
	if err != nil {
		t.Fatalf("SecondPunch (player acceptance): %v", err)
	}
	if valid.Status != tickets.StatusValid {
		t.Fatalf("expected status valid, got %q", valid.Status)
	}
	if role := rosterRole(t, pool, f.showRunID, player); role != "player" {
		t.Fatalf("expected active player roster row, got role %q", role)
	}
}

// TestSecondPunchIsIdempotentUnderConcurrency is the required §11/§4.3
// proof: two callers racing to second-punch the same ticket must produce
// exactly one valid result and exactly one active roster row, never a
// duplicate or an error from the loser.
func TestSecondPunchIsIdempotentUnderConcurrency(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	producer := ticketTestUser(t, pool, "ticket_producer")
	player := ticketTestUser(t, pool, "ticket_player")
	f := buildTicketFixture(t, pool, producer)

	req, err := tickets.RequestFromPlayer(context.Background(), pool, player, f.showRunID, "")
	if err != nil {
		t.Fatalf("RequestFromPlayer: %v", err)
	}

	const attempts = 8
	var wg sync.WaitGroup
	errs := make([]error, attempts)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := tickets.SecondPunch(context.Background(), pool, producer, req.ID)
			errs[i] = err
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("concurrent SecondPunch attempt %d failed: %v", i, err)
		}
	}

	var rosterRowCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM show_run_roster_members
		WHERE show_run_id = $1 AND user_id = $2 AND removed_at IS NULL
	`, f.showRunID, player).Scan(&rosterRowCount); err != nil {
		t.Fatalf("count roster rows: %v", err)
	}
	if rosterRowCount != 1 {
		t.Fatalf("expected exactly one active roster row after concurrent second punches, got %d", rosterRowCount)
	}
}

// TestDuplicateActiveTicketRejected covers the unique-active-ticket
// constraint (spec §4.1).
func TestDuplicateActiveTicketRejected(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	producer := ticketTestUser(t, pool, "ticket_producer")
	player := ticketTestUser(t, pool, "ticket_player")
	f := buildTicketFixture(t, pool, producer)

	if _, err := tickets.RequestFromPlayer(context.Background(), pool, player, f.showRunID, ""); err != nil {
		t.Fatalf("first RequestFromPlayer: %v", err)
	}
	_, err := tickets.RequestFromPlayer(context.Background(), pool, player, f.showRunID, "")
	if err == nil || err.Error() != "ticket_already_pending" {
		t.Fatalf("expected ticket_already_pending for a second pending request, got %v", err)
	}
}

// TestDeclineAndWithdrawGrantNothing covers spec §4.4: neither exit grants
// permissions or a roster row, and both are auditable (ticket row remains,
// status updated).
func TestDeclineAndWithdrawGrantNothing(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	producer := ticketTestUser(t, pool, "ticket_producer")
	player := ticketTestUser(t, pool, "ticket_player")
	f := buildTicketFixture(t, pool, producer)

	req, err := tickets.RequestFromPlayer(context.Background(), pool, player, f.showRunID, "")
	if err != nil {
		t.Fatalf("RequestFromPlayer: %v", err)
	}
	declined, err := tickets.Decline(context.Background(), pool, producer, req.ID)
	if err != nil {
		t.Fatalf("Decline: %v", err)
	}
	if declined.Status != tickets.StatusDeclined {
		t.Fatalf("expected status declined, got %q", declined.Status)
	}
	if rosterRole(t, pool, f.showRunID, player) != "" {
		t.Fatalf("expected no roster row after decline")
	}

	req2, err := tickets.RequestFromPlayer(context.Background(), pool, player, f.showRunID, "")
	if err != nil {
		t.Fatalf("second RequestFromPlayer after decline: %v", err)
	}
	withdrawn, err := tickets.Withdraw(context.Background(), pool, player, req2.ID)
	if err != nil {
		t.Fatalf("Withdraw: %v", err)
	}
	if withdrawn.Status != tickets.StatusWithdrawn {
		t.Fatalf("expected status withdrawn, got %q", withdrawn.Status)
	}
	if rosterRole(t, pool, f.showRunID, player) != "" {
		t.Fatalf("expected no roster row after withdraw")
	}
}

// TestPlayerCannotPunchForAnotherPlayer and
// TestDirectorCannotApproveOutsideTheirAuthority are the required §11
// security proofs.
func TestPlayerCannotPunchForAnotherPlayer(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	producer := ticketTestUser(t, pool, "ticket_producer")
	player := ticketTestUser(t, pool, "ticket_player")
	otherPlayer := ticketTestUser(t, pool, "ticket_other_player")
	f := buildTicketFixture(t, pool, producer)
	profileID := ticketTestProfileID(t, pool, player)

	invite, err := tickets.InviteFromDirector(context.Background(), pool, producer, f.showRunID, profileID, "")
	if err != nil {
		t.Fatalf("InviteFromDirector: %v", err)
	}

	if _, err := tickets.SecondPunch(context.Background(), pool, otherPlayer, invite.ID); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized when a different player tries to accept, got %v", err)
	}
	if rosterRole(t, pool, f.showRunID, player) != "" {
		t.Fatalf("expected no roster row after a rejected impersonation attempt")
	}
}

func TestDirectorCannotApproveOutsideTheirAuthority(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	producer := ticketTestUser(t, pool, "ticket_producer")
	player := ticketTestUser(t, pool, "ticket_player")
	stranger := ticketTestUser(t, pool, "ticket_stranger")
	f := buildTicketFixture(t, pool, producer)

	req, err := tickets.RequestFromPlayer(context.Background(), pool, player, f.showRunID, "")
	if err != nil {
		t.Fatalf("RequestFromPlayer: %v", err)
	}

	if _, err := tickets.SecondPunch(context.Background(), pool, stranger, req.ID); err == nil || err.Error() != "not_authorized" {
		t.Fatalf("expected not_authorized for a non-manager approving, got %v", err)
	}
	if rosterRole(t, pool, f.showRunID, player) != "" {
		t.Fatalf("expected no roster row after a rejected unauthorized approval")
	}
}
