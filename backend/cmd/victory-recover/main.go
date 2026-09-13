// victory-recover is the Kernel 76 break-glass tool (Kernel 76 §2.2).
//
// It exists so that losing a Victory login is never the same thing as losing
// Victory. It runs from the trusted server shell against DATABASE_URL, needs no
// existing Victory session, and performs only the operations required to put
// the operator back in the application:
//
//	whoami    -- show how a user is identified and what authority they hold
//	bootstrap -- create the operator account on a database that has none
//	claim     -- give an account the operator handle, moving it off any account holding it
//	grant     -- restore Location memberships
//	revoke    -- revoke a user's sessions, or every session
//	recover   -- mint a one-time password-reset link the operator can redeem
//
// It deliberately cannot: create a hidden superuser (operator authority is the
// OPERATOR_HANDLE handle and nothing else, so every claim is visible in the
// users table), set a password directly (recovery goes through the ordinary
// reset-confirm endpoint), or read private content. Every subcommand is a
// no-op without an explicit target, and every mutation prints what it changed.
//
// Documented in Construction/Domains/Operations/victory-account-recovery-runbook.md.
package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
)

const usage = `victory-recover -- Victory break-glass account recovery

Usage:
  victory-recover whoami   (--handle H | --email E | --discord-id D | --user-id U)
  victory-recover bootstrap --operator-handle YOUR_HANDLE [--location SLUG] [--email E]
  victory-recover claim    (--handle H | --email E | --discord-id D | --user-id U) --operator-handle YOUR_HANDLE
  victory-recover grant    (--handle H | ...) --location SLUG --role producer
  victory-recover revoke   (--handle H | ...) [--all-sessions]
  victory-recover recover  (--handle H | ...) --base-url https://your-install.example

Reads DATABASE_URL from the environment (or --database-url).
Run from the server shell. See Construction/Domains/Operations/victory-account-recovery-runbook.md.
`

type target struct {
	handle    string
	email     string
	discordID string
	userID    string
}

func (t target) empty() bool {
	return strings.TrimSpace(t.handle) == "" &&
		strings.TrimSpace(t.email) == "" &&
		strings.TrimSpace(t.discordID) == "" &&
		strings.TrimSpace(t.userID) == ""
}

type account struct {
	userID      string
	handle      string
	email       string
	displayName string
	discordID   string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}

	command := os.Args[1]
	fs := flag.NewFlagSet(command, flag.ExitOnError)

	var (
		t              target
		databaseURL    = fs.String("database-url", os.Getenv("DATABASE_URL"), "PostgreSQL connection string")
		operatorHandle = fs.String("operator-handle", os.Getenv("OPERATOR_HANDLE"), "handle that carries operator authority")
		location       = fs.String("location", "", "Location slug (grant)")
		role           = fs.String("role", "", "Location role: producer|director|cast|crew|audience (grant)")
		allSessions    = fs.Bool("all-sessions", false, "revoke every session for every user (revoke)")
		baseURL        = fs.String("base-url", os.Getenv("VICTORY_BASE_URL"), "public base URL used to build the recovery link")
	)
	fs.StringVar(&t.handle, "handle", "", "target by Victory handle")
	fs.StringVar(&t.email, "email", "", "target by email address")
	fs.StringVar(&t.discordID, "discord-id", "", "target by stable Discord user ID")
	fs.StringVar(&t.userID, "user-id", "", "target by internal user ID")

	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}

	if strings.TrimSpace(*databaseURL) == "" {
		fail(errors.New("DATABASE_URL is not set (pass --database-url)"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, *databaseURL)
	if err != nil {
		fail(fmt.Errorf("database connect failed: %w", err))
	}
	defer pool.Close()

	switch command {
	case "whoami":
		err = runWhoami(ctx, pool, t, *operatorHandle)
	case "bootstrap":
		err = runBootstrap(ctx, pool, *operatorHandle, *location, t.email, *baseURL)
	case "claim":
		err = runClaim(ctx, pool, t, *operatorHandle)
	case "grant":
		err = runGrant(ctx, pool, t, *location, *role)
	case "revoke":
		err = runRevoke(ctx, pool, t, *allSessions)
	case "recover":
		err = runRecover(ctx, pool, t, *baseURL)
	case "help", "-h", "--help":
		fmt.Print(usage)
		return
	default:
		fmt.Print(usage)
		os.Exit(2)
	}

	if err != nil {
		fail(err)
	}
}

// resolveAccount finds exactly one account. It refuses to guess: an ambiguous
// or missing target is an error, never a silently chosen row.
func resolveAccount(ctx context.Context, pool *pgxpool.Pool, t target) (account, error) {
	if t.empty() {
		return account{}, errors.New("no target given: pass --handle, --email, --discord-id, or --user-id")
	}

	const query = `
		SELECT u.id::text,
		       COALESCE(u.handle, ''),
		       COALESCE(u.email, ''),
		       COALESCE(u.display_name, ''),
		       COALESCE(di.discord_user_id, '')
		FROM users u
		LEFT JOIN auth.discord_identities di ON di.user_id = u.id
		WHERE ($1 <> '' AND lower(u.handle) = lower($1))
		   OR ($2 <> '' AND lower(u.email) = lower($2))
		   OR ($3 <> '' AND di.discord_user_id = $3)
		   OR ($4 <> '' AND u.id::text = $4)
	`

	rows, err := pool.Query(ctx, query,
		strings.TrimSpace(t.handle),
		strings.TrimSpace(t.email),
		strings.TrimSpace(t.discordID),
		strings.TrimSpace(t.userID),
	)
	if err != nil {
		return account{}, err
	}
	defer rows.Close()

	var found []account
	for rows.Next() {
		var a account
		if err := rows.Scan(&a.userID, &a.handle, &a.email, &a.displayName, &a.discordID); err != nil {
			return account{}, err
		}
		found = append(found, a)
	}
	if err := rows.Err(); err != nil {
		return account{}, err
	}

	switch len(found) {
	case 0:
		return account{}, errors.New("no account matched that target")
	case 1:
		return found[0], nil
	default:
		var ids []string
		for _, a := range found {
			ids = append(ids, fmt.Sprintf("%s (%s)", a.handle, a.userID))
		}
		return account{}, fmt.Errorf("target matched %d accounts, refusing to guess: %s",
			len(found), strings.Join(ids, ", "))
	}
}

func runWhoami(ctx context.Context, pool *pgxpool.Pool, t target, operatorHandle string) error {
	a, err := resolveAccount(ctx, pool, t)
	if err != nil {
		return err
	}

	fmt.Printf("user_id:       %s\n", a.userID)
	fmt.Printf("handle:        %s\n", orNone(a.handle))
	fmt.Printf("display_name:  %s\n", orNone(a.displayName))
	fmt.Printf("email:         %s\n", orNone(a.email))
	fmt.Printf("discord_id:    %s\n", orNone(a.discordID))
	fmt.Printf("is_operator:   %t (operator handle is %q)\n",
		strings.EqualFold(a.handle, operatorHandle), operatorHandle)

	var hasPassword bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM auth.password_credentials WHERE user_id = $1)
	`, a.userID).Scan(&hasPassword); err != nil {
		return err
	}
	fmt.Printf("password_set:  %t\n", hasPassword)

	var activeSessions int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM auth.sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`, a.userID).Scan(&activeSessions); err != nil {
		return err
	}
	fmt.Printf("live_sessions: %d\n", activeSessions)

	rows, err := pool.Query(ctx, `
		SELECT l.slug, lm.role::text, lm.active
		FROM location_memberships lm
		JOIN locations l ON l.id = lm.location_id
		WHERE lm.user_id = $1
		ORDER BY l.slug, lm.role
	`, a.userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Println("memberships:")
	any := false
	for rows.Next() {
		var slug, membershipRole string
		var active bool
		if err := rows.Scan(&slug, &membershipRole, &active); err != nil {
			return err
		}
		any = true
		state := "active"
		if !active {
			state = "INACTIVE"
		}
		fmt.Printf("  - %s: %s (%s)\n", slug, membershipRole, state)
	}
	if !any {
		fmt.Println("  (none)")
	}
	return rows.Err()
}

// runBootstrap creates the operator account on a database that has none. A
// database rebuilt from migrations seeds Locations and Venues but no human
// accounts, so without this the first move after a reset -- signing in --
// would have nothing to sign in to, and the operator would be locked out of a
// server they own (Kernel 76 §2).
//
// The account is created with no password and no session. It becomes usable
// only when someone redeems the recovery link this prints, so a bootstrapped
// account that is never claimed is not a standing back door.
func runBootstrap(ctx context.Context, pool *pgxpool.Pool, operatorHandle, locationSlug, email, baseURL string) error {
	operatorHandle = strings.ToLower(strings.TrimSpace(operatorHandle))
	if operatorHandle == "" {
		return errors.New("--operator-handle must not be empty")
	}
	if strings.TrimSpace(locationSlug) == "" {
		locationSlug = access.DefaultLocationSlug()
	}

	var existingID string
	err := pool.QueryRow(ctx, `
		SELECT id::text FROM users WHERE lower(handle) = $1 LIMIT 1
	`, operatorHandle).Scan(&existingID)
	switch {
	case err == nil:
		fmt.Printf("operator account %q already exists (%s); nothing created\n", operatorHandle, existingID)
		fmt.Println("use `victory-recover recover` to mint a sign-in link for it")
		return nil
	case errors.Is(err, pgx.ErrNoRows):
	default:
		return err
	}

	var locationID string
	if err := pool.QueryRow(ctx, `
		SELECT id::text FROM locations WHERE slug = $1 LIMIT 1
	`, locationSlug).Scan(&locationID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("no Location with slug %q -- boot the backend once so migrations and seeds run, then retry", locationSlug)
		}
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var userID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (handle, display_name, email)
		VALUES ($1, $2, NULLIF($3, ''))
		RETURNING id::text
	`, operatorHandle, operatorHandle, strings.ToLower(strings.TrimSpace(email))).Scan(&userID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, granted_by_user_id, active)
		VALUES ($1, $2, 'producer', $2, TRUE)
		ON CONFLICT (location_id, user_id, role) DO UPDATE SET active = TRUE
	`, locationID, userID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	fmt.Printf("created operator account %q (%s) with producer at %s\n", operatorHandle, userID, locationSlug)
	fmt.Println("it has no password and no session yet; the link below is how it is claimed:")
	fmt.Println()
	return runRecover(ctx, pool, target{userID: userID}, baseURL)
}

// runClaim moves the operator handle onto the target account. Operator
// authority in Victory is "your handle equals OPERATOR_HANDLE" (see
// access.IsOperatorUser), so after a database reset the Discord-created
// account -- which lands as discord_<id> -- needs this to become Grant again.
func runClaim(ctx context.Context, pool *pgxpool.Pool, t target, operatorHandle string) error {
	operatorHandle = strings.ToLower(strings.TrimSpace(operatorHandle))
	if operatorHandle == "" {
		return errors.New("--operator-handle must not be empty")
	}

	a, err := resolveAccount(ctx, pool, t)
	if err != nil {
		return err
	}

	if strings.EqualFold(a.handle, operatorHandle) {
		fmt.Printf("no change: %s already holds the operator handle %q\n", a.userID, operatorHandle)
		return nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// The handle is unique, so any prior holder must be parked first. It is
	// renamed rather than deleted: it may still own Characters and journals.
	var priorID, priorNewHandle string
	err = tx.QueryRow(ctx, `
		SELECT id::text FROM users WHERE lower(handle) = $1 LIMIT 1
	`, operatorHandle).Scan(&priorID)
	switch {
	case err == nil:
		priorNewHandle = fmt.Sprintf("%s_released_%d", operatorHandle, time.Now().UTC().Unix())
		if _, err := tx.Exec(ctx, `
			UPDATE users SET handle = $2, updated_at = NOW() WHERE id = $1
		`, priorID, priorNewHandle); err != nil {
			return fmt.Errorf("could not release the operator handle from %s: %w", priorID, err)
		}
	case errors.Is(err, pgx.ErrNoRows):
	default:
		return err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE users SET handle = $2, updated_at = NOW() WHERE id = $1
	`, a.userID, operatorHandle); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	if priorID != "" {
		fmt.Printf("released operator handle from %s (renamed to %s)\n", priorID, priorNewHandle)
	}
	fmt.Printf("granted operator handle %q to %s (was %q)\n", operatorHandle, a.userID, orNone(a.handle))
	fmt.Println("next: victory-recover grant --handle " + operatorHandle + " --location <slug> --role producer")
	return nil
}

func runGrant(ctx context.Context, pool *pgxpool.Pool, t target, locationSlug, role string) error {
	locationSlug = strings.TrimSpace(locationSlug)
	role = strings.ToLower(strings.TrimSpace(role))
	if locationSlug == "" || role == "" {
		return errors.New("grant requires --location SLUG and --role ROLE")
	}
	switch role {
	case "producer", "director", "cast", "crew", "audience":
	default:
		return fmt.Errorf("invalid role %q: use producer, director, cast, crew, or audience", role)
	}

	a, err := resolveAccount(ctx, pool, t)
	if err != nil {
		return err
	}

	var locationID string
	if err := pool.QueryRow(ctx, `
		SELECT id::text FROM locations WHERE slug = $1 LIMIT 1
	`, locationSlug).Scan(&locationID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("no Location with slug %q", locationSlug)
		}
		return err
	}

	// Idempotent: re-running reactivates an existing row rather than failing on
	// the (location, user, role) unique constraint.
	tag, err := pool.Exec(ctx, `
		INSERT INTO location_memberships (location_id, user_id, role, granted_by_user_id, active)
		VALUES ($1, $2, $3::location_role, $2, TRUE)
		ON CONFLICT (location_id, user_id, role)
		DO UPDATE SET active = TRUE
	`, locationID, a.userID, role)
	if err != nil {
		return err
	}

	fmt.Printf("granted %s at %s to %s (%s); rows affected: %d\n",
		role, locationSlug, orNone(a.handle), a.userID, tag.RowsAffected())
	return nil
}

func runRevoke(ctx context.Context, pool *pgxpool.Pool, t target, allSessions bool) error {
	if allSessions {
		if !t.empty() {
			return errors.New("--all-sessions revokes every session; do not also pass a target")
		}
		tag, err := pool.Exec(ctx, `
			UPDATE auth.sessions SET revoked_at = NOW() WHERE revoked_at IS NULL
		`)
		if err != nil {
			return err
		}
		fmt.Printf("revoked every live session; rows affected: %d\n", tag.RowsAffected())
		fmt.Println("every user must sign in again, including the operator")
		return nil
	}

	a, err := resolveAccount(ctx, pool, t)
	if err != nil {
		return err
	}

	tag, err := pool.Exec(ctx, `
		UPDATE auth.sessions SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`, a.userID)
	if err != nil {
		return err
	}

	fmt.Printf("revoked sessions for %s (%s); rows affected: %d\n",
		orNone(a.handle), a.userID, tag.RowsAffected())
	return nil
}

// runRecover mints a one-time password-reset token and prints the link that
// redeems it. This is the path that keeps account ownership from depending on
// Discord staying available (Kernel 76 §5.4). Only the SHA-256 hash is stored,
// matching what /api/auth/password-reset/confirm verifies; the raw value is
// printed once, to the operator's terminal, and never written to a log.
func runRecover(ctx context.Context, pool *pgxpool.Pool, t target, baseURL string) error {
	if strings.TrimSpace(baseURL) == "" {
		return errors.New("--base-url or VICTORY_BASE_URL must be set to this installation's own public address")
	}

	a, err := resolveAccount(ctx, pool, t)
	if err != nil {
		return err
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(token))
	expiresAt := time.Now().UTC().Add(1 * time.Hour)

	if _, err := pool.Exec(ctx, `
		INSERT INTO auth.password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, a.userID, sum[:], expiresAt); err != nil {
		return err
	}

	fmt.Printf("recovery token issued for %s (%s)\n", orNone(a.handle), a.userID)
	fmt.Printf("expires: %s (1 hour)\n\n", expiresAt.Format(time.RFC3339))
	// /auth/* is reverse-proxied to the backend by Caddy, so the redemption
	// page lives under the statically served /login/ path.
	fmt.Printf("  %s/login/reset.html?token=%s\n\n", strings.TrimRight(strings.TrimSpace(baseURL), "/"), token)
	fmt.Println("Single use. Do not paste this into a log, ticket, or chat.")
	fmt.Println("If the account has no password yet, redeeming this sets one.")
	return nil
}

func orNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(none)"
	}
	return s
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "victory-recover: %v\n", err)
	os.Exit(1)
}
