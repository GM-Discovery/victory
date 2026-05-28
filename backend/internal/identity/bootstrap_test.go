package identity

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestBootstrapProducerByDiscordUserIDIsIdempotent(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()

	ensureBootstrapSchema(t, pool)

	handle := "bootstrap_" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
	displayName := "Bootstrap Producer"
	discordID := "998877665544332211"

	userID := insertBootstrapUser(t, pool, handle, displayName)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.discord_identities WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO auth.discord_identities (
			user_id,
			discord_user_id,
			username,
			global_name,
			email,
			email_verified
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, userID, discordID, "booty", "Booty Producer", "booty@example.com", true); err != nil {
		t.Fatalf("insert discord identity: %v", err)
	}

	first, err := BootstrapProducer(ctx, pool, BootstrapProducerInput{
		DiscordUserID: discordID,
		LocationSlug:  "amurray-family",
	})
	if err != nil {
		t.Fatalf("first bootstrap: %v", err)
	}
	if first.AlreadyExisted {
		t.Fatalf("expected first grant to be new")
	}
	if first.Location.Slug != "amurray-family" {
		t.Fatalf("unexpected location slug %q", first.Location.Slug)
	}
	if first.User.ID != userID {
		t.Fatalf("unexpected user id %q", first.User.ID)
	}

	assertProducerMembershipCount(t, pool, userID, 1)

	second, err := BootstrapProducer(ctx, pool, BootstrapProducerInput{
		DiscordUserID: discordID,
		LocationSlug:  "amurray-family",
	})
	if err != nil {
		t.Fatalf("second bootstrap: %v", err)
	}
	if !second.AlreadyExisted {
		t.Fatalf("expected second grant to report already existed")
	}

	assertProducerMembershipCount(t, pool, userID, 1)
}

func TestBootstrapProducerUnknownDiscordUserFails(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()

	ensureBootstrapSchema(t, pool)

	_, err := BootstrapProducer(ctx, pool, BootstrapProducerInput{
		DiscordUserID: "definitely-not-real",
		LocationSlug:  "amurray-family",
	})
	if !errors.Is(err, ErrBootstrapUserNotFound) {
		t.Fatalf("expected ErrBootstrapUserNotFound, got %v", err)
	}
}

func TestBootstrapProducerByHandleAndUserID(t *testing.T) {
	pool := openDiscordTestPool(t)
	ctx := context.Background()

	ensureBootstrapSchema(t, pool)

	handle := "bootstrap_handle_" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405.000000")
	displayName := "Handle Producer"
	userID := insertBootstrapUser(t, pool, handle, displayName)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM location_memberships WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	byHandle, err := BootstrapProducer(ctx, pool, BootstrapProducerInput{
		Handle:       handle,
		LocationSlug: "amurray-family",
	})
	if err != nil {
		t.Fatalf("bootstrap by handle: %v", err)
	}
	if byHandle.User.ID != userID {
		t.Fatalf("unexpected user from handle path %q", byHandle.User.ID)
	}
	if byHandle.AlreadyExisted {
		t.Fatalf("expected handle-based grant to be new")
	}

	assertProducerMembershipCount(t, pool, userID, 1)

	byID, err := BootstrapProducer(ctx, pool, BootstrapProducerInput{
		UserID:       userID,
		LocationSlug: "amurray-family",
	})
	if err != nil {
		t.Fatalf("bootstrap by user id: %v", err)
	}
	if byID.User.ID != userID {
		t.Fatalf("unexpected user from id path %q", byID.User.ID)
	}
	if !byID.AlreadyExisted {
		t.Fatalf("expected id-based grant to report already existed")
	}
}

func ensureBootstrapSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()
	_, _ = pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS auth`)
	_, _ = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS auth.discord_identities (
			id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			discord_user_id text NOT NULL UNIQUE,
			username text,
			global_name text,
			discriminator text,
			avatar text,
			email text,
			email_verified boolean,
			locale text,
			last_login_at timestamptz,
			created_at timestamptz NOT NULL DEFAULT now(),
			updated_at timestamptz NOT NULL DEFAULT now()
		)
	`)
}

func insertBootstrapUser(t *testing.T, pool *pgxpool.Pool, handle, displayName string) string {
	t.Helper()

	var userID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO users (handle, display_name)
		VALUES ($1, $2)
		RETURNING id::text
	`, handle, displayName).Scan(&userID); err != nil {
		t.Fatalf("insert bootstrap user: %v", err)
	}
	return userID
}

func assertProducerMembershipCount(t *testing.T, pool *pgxpool.Pool, userID string, want int) {
	t.Helper()

	var got int
	if err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM location_memberships
		WHERE user_id = $1
		  AND role = 'producer'
		  AND active = TRUE
	`, userID).Scan(&got); err != nil {
		t.Fatalf("count producer memberships: %v", err)
	}
	if got != want {
		t.Fatalf("producer membership count = %d, want %d", got, want)
	}
}
