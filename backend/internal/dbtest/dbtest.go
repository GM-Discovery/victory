// Package dbtest is the single safety gate DB-touching tests (and any
// destructive reset/truncate tooling) must pass through before they are
// allowed to open a database connection. It exists so that no test can
// silently fall back to the live/shared Victory database: see Kernel 64.
package dbtest

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/db"
)

// requiredMarker must appear in the test database's name. It is what keeps a
// misconfigured TEST_DATABASE_URL from ever resolving to the live app
// database, whatever the rest of the URL looks like.
const requiredMarker = "test"

// blockedNames are rejected outright even though they don't contain
// "prod" or "production" as a substring - these are the live/shared
// Victory app database and the Postgres default admin database.
var blockedNames = map[string]bool{
	"":         true,
	"victory":  true,
	"postgres": true,
}

// ValidateTestDatabaseURL rejects any TEST_DATABASE_URL that is not clearly
// a dedicated, disposable test database. liveDatabaseURL is typically
// os.Getenv("DATABASE_URL"); pass "" if it isn't set or isn't relevant.
//
// This is the one place that decides whether a database is safe to write to
// or destroy from a test/reset context - every DB-touching test and every
// destructive schema-reset helper must call it first.
func ValidateTestDatabaseURL(testDatabaseURL, liveDatabaseURL string) error {
	trimmed := strings.TrimSpace(testDatabaseURL)
	if trimmed == "" {
		return fmt.Errorf("TEST_DATABASE_URL is required for database-touching tests (see Construction/Process/kernel-maker-field-guide.md, Backend Test Commands)")
	}

	liveTrimmed := strings.TrimSpace(liveDatabaseURL)
	if liveTrimmed != "" && trimmed == liveTrimmed {
		return fmt.Errorf("TEST_DATABASE_URL must not be the same as DATABASE_URL (the live app database)")
	}

	cfg, err := pgxpool.ParseConfig(trimmed)
	if err != nil {
		return fmt.Errorf("TEST_DATABASE_URL is not a valid postgres connection string: %w", err)
	}

	dbName := strings.ToLower(strings.TrimSpace(cfg.ConnConfig.Database))
	if blockedNames[dbName] {
		return fmt.Errorf("TEST_DATABASE_URL must not point at the live/shared database %q", dbName)
	}
	if strings.Contains(dbName, "prod") {
		return fmt.Errorf("TEST_DATABASE_URL must not point at a production-looking database %q", dbName)
	}
	if !strings.Contains(dbName, requiredMarker) {
		return fmt.Errorf("TEST_DATABASE_URL database name %q must contain %q to be recognized as a dedicated test database (e.g. victory_test)", dbName, requiredMarker)
	}

	return nil
}

// OpenTestPool is the standard entry point for DB-touching tests. It reads
// TEST_DATABASE_URL, validates it, and hard-fails the test (t.Fatalf, not
// t.Skip) if the variable is missing or unsafe, per Kernel 64's operator
// decision: a silently skipped DB test is exactly what let real regressions
// through before.
func OpenTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	// Kernel 96: access.DefaultLocationSlug() falls back to "victory-theater"
	// when this is unset -- fine for a real boot, where the launcher/compose
	// always sets it first, but a `go test` process never does. Several
	// Ensure* seed functions (e.g. ewrite.EnsureSkillDirectory) call
	// DefaultLocationSlug() directly rather than taking a location id as a
	// parameter, and a handful of tests call those functions directly
	// without going through a full server boot first. Before this pass,
	// that accidentally worked because migration 025 unconditionally
	// created a "victory-theater" location on every database; neutralizing
	// that phantom-location migration (the whole point of this pass) means
	// nothing does anymore. scripts/test/lib-migrate-and-bootstrap.sh
	// already establishes "amurray-family" as this fixture's real name;
	// this is that same convention, for tests that never went through that
	// script's own bootstrap boot.
	if os.Getenv("DEFAULT_LOCATION_SLUG") == "" {
		_ = os.Setenv("DEFAULT_LOCATION_SLUG", "amurray-family")
	}

	testDatabaseURL := os.Getenv("TEST_DATABASE_URL")
	if err := ValidateTestDatabaseURL(testDatabaseURL, os.Getenv("DATABASE_URL")); err != nil {
		t.Fatalf("dbtest: %v", err)
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, testDatabaseURL)
	if err != nil {
		t.Fatalf("dbtest: connect to TEST_DATABASE_URL: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("dbtest: ping TEST_DATABASE_URL: %v", err)
	}

	t.Cleanup(pool.Close)
	return pool
}
