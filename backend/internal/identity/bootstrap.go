package identity

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrBootstrapLocationNotFound = errors.New("bootstrap location not found")
	ErrBootstrapUserNotFound     = errors.New("bootstrap user not found")
	ErrBootstrapTargetRequired   = errors.New("bootstrap target required")
)

type BootstrapUser struct {
	ID          string
	Handle      string
	DisplayName string
	DiscordID   string
	Email       string
}

type BootstrapLocation struct {
	ID   string
	Name string
	Slug string
}

type BootstrapProducerResult struct {
	User           BootstrapUser
	Location       BootstrapLocation
	AlreadyExisted bool
}

type BootstrapProducerInput struct {
	DiscordUserID string
	UserID        string
	Handle        string
	LocationSlug  string
}

func ResolveBootstrapLocation(ctx context.Context, pool *pgxpool.Pool, slug string) (BootstrapLocation, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		slug = "amurray-family"
	}

	var location BootstrapLocation
	err := pool.QueryRow(ctx, `
		SELECT id::text, name, slug
		FROM locations
		WHERE slug = $1
		LIMIT 1
	`, slug).Scan(&location.ID, &location.Name, &location.Slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BootstrapLocation{}, ErrBootstrapLocationNotFound
		}
		return BootstrapLocation{}, err
	}

	return location, nil
}

func ResolveBootstrapUserByDiscordID(ctx context.Context, pool *pgxpool.Pool, discordUserID string) (BootstrapUser, error) {
	discordUserID = strings.TrimSpace(discordUserID)
	if discordUserID == "" {
		return BootstrapUser{}, ErrBootstrapTargetRequired
	}

	var user BootstrapUser
	err := pool.QueryRow(ctx, `
		SELECT
		  u.id::text,
		  COALESCE(NULLIF(u.handle, ''), ''),
		  COALESCE(NULLIF(u.display_name, ''), ''),
		  d.discord_user_id,
		  COALESCE(NULLIF(u.email, ''), '')
		FROM auth.discord_identities d
		JOIN users u ON u.id = d.user_id
		WHERE d.discord_user_id = $1::text
		LIMIT 1
	`, discordUserID).Scan(&user.ID, &user.Handle, &user.DisplayName, &user.DiscordID, &user.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BootstrapUser{}, ErrBootstrapUserNotFound
		}
		return BootstrapUser{}, err
	}

	return user, nil
}

func ResolveBootstrapUserByID(ctx context.Context, pool *pgxpool.Pool, userID string) (BootstrapUser, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return BootstrapUser{}, ErrBootstrapTargetRequired
	}

	var user BootstrapUser
	err := pool.QueryRow(ctx, `
		SELECT
		  id::text,
		  COALESCE(NULLIF(handle, ''), ''),
		  COALESCE(NULLIF(display_name, ''), ''),
		  COALESCE(NULLIF(email, ''), '')
		FROM users
		WHERE id = $1::uuid
		LIMIT 1
	`, userID).Scan(&user.ID, &user.Handle, &user.DisplayName, &user.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BootstrapUser{}, ErrBootstrapUserNotFound
		}
		return BootstrapUser{}, err
	}

	return user, nil
}

func ResolveBootstrapUserByHandle(ctx context.Context, pool *pgxpool.Pool, handle string) (BootstrapUser, error) {
	handle = normalizeHandle(handle)
	if handle == "" {
		return BootstrapUser{}, ErrBootstrapTargetRequired
	}

	var user BootstrapUser
	err := pool.QueryRow(ctx, `
		SELECT
		  id::text,
		  COALESCE(NULLIF(handle, ''), ''),
		  COALESCE(NULLIF(display_name, ''), ''),
		  COALESCE(NULLIF(email, ''), '')
		FROM users
		WHERE lower(handle) = $1::text
		LIMIT 1
	`, handle).Scan(&user.ID, &user.Handle, &user.DisplayName, &user.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BootstrapUser{}, ErrBootstrapUserNotFound
		}
		return BootstrapUser{}, err
	}

	return user, nil
}

func BootstrapProducer(ctx context.Context, pool *pgxpool.Pool, input BootstrapProducerInput) (BootstrapProducerResult, error) {
	location, err := ResolveBootstrapLocation(ctx, pool, input.LocationSlug)
	if err != nil {
		return BootstrapProducerResult{}, err
	}

	var user BootstrapUser
	switch {
	case strings.TrimSpace(input.DiscordUserID) != "":
		user, err = ResolveBootstrapUserByDiscordID(ctx, pool, input.DiscordUserID)
	case strings.TrimSpace(input.UserID) != "":
		user, err = ResolveBootstrapUserByID(ctx, pool, input.UserID)
	case strings.TrimSpace(input.Handle) != "":
		user, err = ResolveBootstrapUserByHandle(ctx, pool, input.Handle)
	default:
		err = ErrBootstrapTargetRequired
	}
	if err != nil {
		return BootstrapProducerResult{}, err
	}

	alreadyExisted, err := ensureLocationProducerMembership(ctx, pool, location.ID, user.ID)
	if err != nil {
		return BootstrapProducerResult{}, err
	}

	return BootstrapProducerResult{
		User:           user,
		Location:       location,
		AlreadyExisted: alreadyExisted,
	}, nil
}

func ensureLocationProducerMembership(ctx context.Context, pool *pgxpool.Pool, locationID, userID string) (bool, error) {
	var existingActive bool
	err := pool.QueryRow(ctx, `
		SELECT active
		FROM location_memberships
		WHERE location_id = $1::uuid
		  AND user_id = $2::uuid
		  AND role = 'producer'
		LIMIT 1
	`, locationID, userID).Scan(&existingActive)
	switch {
	case err == nil:
		if existingActive {
			return true, nil
		}

		_, err = pool.Exec(ctx, `
			UPDATE location_memberships
			SET active = TRUE
			WHERE location_id = $1::uuid
			  AND user_id = $2::uuid
			  AND role = 'producer'
		`, locationID, userID)
		if err != nil {
			return true, err
		}
		return true, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return false, err
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, active)
		VALUES ($1::uuid, $2::uuid, 'producer', TRUE)
	`, locationID, userID)
	if err != nil {
		return false, err
	}

	return false, nil
}
