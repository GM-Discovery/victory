// Package firstrun runs once, synchronously, right after the very first
// account on a fresh Victory install signs up (identity's bootstrap
// window, see identity.HandleSignup). Kernel 100: a fresh install should
// land its new Operator straight on the campus map with a running show
// already mounted, not an empty stage. Swapping or unmounting that show
// afterward stays exactly where it already is -- an advanced Stage
// Management action, not something first-run needs to know about.
//
// This is a separate package, not a function inside identity itself,
// because showtime (needed to actually start the show live) already
// imports identity for DiscordServerLinkConfig -- identity importing
// showtime back would cycle. identity instead calls this package through
// a function value it never has to know the concrete type of (wired in
// cmd/victory/main.go), the same pattern as a driver registration.
package firstrun

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
	"victory/backend/internal/identity"
	"victory/backend/internal/merchant"
	"victory/backend/internal/showings"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
	"victory/backend/internal/showtime"
)

// BootstrapFirstOperator grants the new account producer authority at this
// install's own location, mounts the Locked Courtyard opening into a real
// Show, and starts it live. Every failure is logged and swallowed, never
// surfaced to the caller: the account itself already exists by the time
// this runs, and a welcome-content hiccup must never look like a failed
// signup. A Director can always finish the job by hand in Stage
// Management afterward -- this is a convenience, not a guarantee.
func BootstrapFirstOperator(ctx context.Context, pool *pgxpool.Pool, userID string) {
	locationSlug := access.DefaultLocationSlug()

	if _, err := identity.BootstrapProducer(ctx, pool, identity.BootstrapProducerInput{
		UserID:       userID,
		LocationSlug: locationSlug,
	}); err != nil {
		log.Printf("firstrun: producer grant failed: %v", err)
		return
	}

	productionID, locationID, err := ensureOpeningProduction(ctx, pool, userID, locationSlug)
	if err != nil {
		log.Printf("firstrun: production bootstrap failed: %v", err)
		return
	}

	run, err := showruns.CreateShowRun(ctx, pool, userID, productionID, showruns.CreateShowRunInput{
		Title: "Opening Night",
		Slug:  "opening-night",
	})
	if err != nil {
		log.Printf("firstrun: show run creation failed: %v", err)
		return
	}

	// Status is left at CreateShow's own "draft" default here -- corrected
	// to "live" explicitly below, after Start, once there's a real Session
	// to point at. (Kernel 101: showtime.Start only ever transitions the
	// Session/Showing to "live" -- it never touches the Show's own status
	// column. A Show still stuck at "draft" is invisible as "mounted" to a
	// genuinely fresh Audience viewer, which is exactly the onboarding
	// experience this whole function exists to deliver. Found live on
	// murray-vserver's first real fresh install; this is the code-level
	// fix for the data patch applied there.)
	show, err := shows.CreateShow(ctx, pool, userID, run.ID, shows.CreateShowInput{
		Title: "The Locked Courtyard",
		Slug:  "the-locked-courtyard",
	})
	if err != nil {
		log.Printf("firstrun: show creation failed: %v", err)
		return
	}

	if _, err := merchant.PrepareLockedCourtyardOpening(ctx, pool, userID, show.ID); err != nil {
		log.Printf("firstrun: locked courtyard prepare failed: %v", err)
		return
	}

	startResult, err := showtime.Start(ctx, pool, identity.DiscordServerLinkConfig{}, userID, "", show.ID, false, "catharsis")
	if err != nil {
		log.Printf("firstrun: show start failed: %v", err)
		return
	}

	// Kernel 101: Start only ever transitions the Session/Showing to
	// "live" -- the Show's own status column is untouched by it and stays
	// at CreateShow's "draft" default forever unless something else sets
	// it, same as a Director would from Stage Management's Show settings
	// panel. Do that explicitly here; this is exactly what a fresh
	// install's "already mounted" promise depends on.
	liveStatus := "live"
	startedAt := time.Now().UTC().Format(time.RFC3339)
	if _, err := shows.UpdateShow(ctx, pool, userID, show.ID, shows.UpdateShowPatch{
		Status:        &liveStatus,
		ActualStartAt: &startedAt,
	}); err != nil {
		log.Printf("firstrun: show status update to live failed: %v", err)
	}

	// Start alone leaves the session in "rehearsal" (Director-console
	// jargon for "exists but nobody watching yet") -- fine for a Director
	// setting up their own run, wrong for onboarding, where the whole
	// point is a new person lands on a stage that's already going. Flip
	// audience visibility on directly rather than relying on whatever
	// later toggles a Director would normally click.
	if showing, err := showings.LoadBySession(ctx, pool, startResult.SessionID); err != nil {
		log.Printf("firstrun: load showing for audience view failed: %v", err)
	} else if _, err := showings.UpdateAudienceViewByID(ctx, pool, showing.ID, true); err != nil {
		log.Printf("firstrun: enable audience view failed: %v", err)
	}

	if err := sendWelcomeMessage(ctx, pool, userID, locationID); err != nil {
		log.Printf("firstrun: welcome message failed: %v", err)
	}
}

// ensureOpeningProduction creates (or reuses, on a re-run) the single
// Production every fresh install's opening show hangs off of. No package
// already exposes a Create/Ensure for Productions -- the only existing
// creation path is the Director-facing HTTP handler in
// identity/permissions.go, which identity itself can't call from a
// package it doesn't import -- so this is a direct, minimal insert
// mirroring that handler's own INSERT INTO productions statement.
func ensureOpeningProduction(ctx context.Context, pool *pgxpool.Pool, userID, locationSlug string) (productionID, locationID string, err error) {
	if err := pool.QueryRow(ctx, `
		SELECT id::text FROM locations WHERE slug = $1 LIMIT 1
	`, locationSlug).Scan(&locationID); err != nil {
		return "", "", err
	}

	err = pool.QueryRow(ctx, `
		INSERT INTO productions (location_id, name, slug, created_by_user_id)
		VALUES ($1, 'Opening Production', 'opening-production', $2)
		ON CONFLICT (location_id, slug) DO UPDATE SET slug = EXCLUDED.slug
		RETURNING id::text
	`, locationID, userID).Scan(&productionID)
	if err != nil {
		return "", "", err
	}
	return productionID, locationID, nil
}

// sendWelcomeMessage delivers the "what to do next, how to invite others"
// content as an in-app Mailbox message rather than real outbound email --
// a self-hosted consumer install has no SMTP account of its own to send
// through, and asking a home user to go get one during setup would defeat
// the entire point of Kernel 100. A direct insert, not messages.createMessage:
// the messages package already imports identity for session-identity
// types, so identity calling back into messages would cycle the same way
// showtime does.
func sendWelcomeMessage(ctx context.Context, pool *pgxpool.Pool, userID, locationID string) error {
	var locationName string
	if err := pool.QueryRow(ctx, `SELECT name FROM locations WHERE id = $1`, locationID).Scan(&locationName); err != nil {
		locationName = "your instance"
	}

	subject := "Welcome to " + locationName
	body := "Your lot is live, with an opening show already running in The Courtyard -- head to the map to see it.\n\n" +
		"To invite others: they'll need Victory accounts of their own. Once you've linked a Discord server " +
		"(Producer's Office > Discord), anyone who signs in there can join. Password signup stays closed after " +
		"this first account, on purpose -- Discord is how everyone after you gets in.\n\n" +
		"Want a different opening show, or to take this one down? That's Stage Management -- Producer's Office has the door."

	_, err := pool.Exec(ctx, `
		INSERT INTO messages (to_user_id, from_user_id, subject, body, is_read)
		VALUES ($1, NULL, $2, $3, FALSE)
	`, userID, subject, body)
	return err
}
