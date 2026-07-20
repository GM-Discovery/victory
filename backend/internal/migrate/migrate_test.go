package migrate

import (
	"context"
	"strings"
	"testing"
	"time"

	"victory/backend/internal/dbtest"
)

func TestEmbeddedFilesSortedAndChecksummed(t *testing.T) {
	files, err := embeddedFiles()
	if err != nil {
		t.Fatalf("embeddedFiles: %v", err)
	}
	if len(files) < 49 {
		t.Fatalf("expected at least the 49 historical migrations, got %d", len(files))
	}
	for i := 1; i < len(files); i++ {
		if files[i-1].Name >= files[i].Name {
			t.Fatalf("files not strictly sorted: %q >= %q", files[i-1].Name, files[i].Name)
		}
	}
	for _, f := range files {
		if len(f.Checksum) != 64 {
			t.Fatalf("%s: bad checksum %q", f.Name, f.Checksum)
		}
		if strings.TrimSpace(f.Content) == "" {
			t.Fatalf("%s: empty migration content", f.Name)
		}
	}
}

func TestPlanRefusesChecksumDrift(t *testing.T) {
	files := []file{{Name: "001_a.sql", Checksum: "aaa"}, {Name: "002_b.sql", Checksum: "bbb"}}
	if _, err := plan(files, map[string]string{"001_a.sql": "TAMPERED"}); err == nil {
		t.Fatal("expected checksum-drift refusal, got nil")
	} else if !strings.Contains(err.Error(), "001_a.sql") {
		t.Fatalf("drift error should name the file, got: %v", err)
	}
}

func TestPlanRefusesMissingEmbeddedFile(t *testing.T) {
	files := []file{{Name: "002_b.sql", Checksum: "bbb"}}
	if _, err := plan(files, map[string]string{"001_a.sql": "aaa"}); err == nil {
		t.Fatal("expected missing-file refusal, got nil")
	}
}

func TestPlanReturnsPendingInOrder(t *testing.T) {
	files := []file{
		{Name: "001_a.sql", Checksum: "aaa"},
		{Name: "002_b.sql", Checksum: "bbb"},
		{Name: "003_c.sql", Checksum: "ccc"},
	}
	pending, err := plan(files, map[string]string{"001_a.sql": "aaa"})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(pending) != 2 || pending[0].Name != "002_b.sql" || pending[1].Name != "003_c.sql" {
		t.Fatalf("unexpected pending set: %+v", pending)
	}
}

// TestRunAdoptsAndThenNoOps proves the full lifecycle against the disposable
// test database: first Run stamps every embedded migration (re-applying them
// idempotently — the same adoption path prod takes), the second Run applies
// nothing, and a tampered ledger row refuses the boot.
func TestRunAdoptsAndThenNoOps(t *testing.T) {
	pool := dbtest.OpenTestPool(t)
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	opts := Options{ApplyOnBoot: true, DangerouslySkipBackup: true}

	if err := Run(ctx, pool, opts); err != nil {
		t.Fatalf("first Run (adoption): %v", err)
	}

	files, err := embeddedFiles()
	if err != nil {
		t.Fatalf("embeddedFiles: %v", err)
	}
	var recorded int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&recorded); err != nil {
		t.Fatalf("count ledger: %v", err)
	}
	if recorded != len(files) {
		t.Fatalf("ledger has %d rows, want %d", recorded, len(files))
	}

	if err := Run(ctx, pool, Options{ApplyOnBoot: false}); err != nil {
		t.Fatalf("verify-only Run on current schema should pass: %v", err)
	}

	// Tamper with one recorded checksum: boot must refuse.
	if _, err := pool.Exec(ctx, `
		UPDATE schema_migrations SET checksum = 'tampered'
		WHERE filename = (SELECT filename FROM schema_migrations ORDER BY filename LIMIT 1)
	`); err != nil {
		t.Fatalf("tamper ledger: %v", err)
	}
	err = Run(ctx, pool, opts)
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("expected checksum refusal after tamper, got: %v", err)
	}
	// Repair for other suites sharing this database.
	files0 := files[0]
	if _, err := pool.Exec(ctx, `UPDATE schema_migrations SET checksum = $2 WHERE filename = $1`, files0.Name, files0.Checksum); err != nil {
		t.Fatalf("repair ledger: %v", err)
	}

	// Simulate one pending migration with MIGRATE_ON_BOOT=false: refuse, and
	// name the file.
	if _, err := pool.Exec(ctx, `DELETE FROM schema_migrations WHERE filename = $1`, files[len(files)-1].Name); err != nil {
		t.Fatalf("unstamp last migration: %v", err)
	}
	err = Run(ctx, pool, Options{ApplyOnBoot: false})
	if err == nil || !strings.Contains(err.Error(), files[len(files)-1].Name) {
		t.Fatalf("expected pending refusal naming %s, got: %v", files[len(files)-1].Name, err)
	}
	// And with ApplyOnBoot it re-applies cleanly.
	if err := Run(ctx, pool, opts); err != nil {
		t.Fatalf("re-apply after unstamp: %v", err)
	}
}

func TestRequireDisposableDatabaseGuard(t *testing.T) {
	if err := requireDisposableDatabase(""); err != nil {
		t.Fatalf("empty URL (pool-only test callers) should be allowed: %v", err)
	}
	if err := requireDisposableDatabase("postgres://u:REDACTED@localhost:5432/victory_test"); err != nil {
		t.Fatalf("test database should be allowed: %v", err)
	}
	if err := requireDisposableDatabase("postgres://u:REDACTED@localhost:5432/victory_fresh_install_1"); err != nil {
		t.Fatalf("fresh database should be allowed: %v", err)
	}
	if err := requireDisposableDatabase("postgres://u:REDACTED@localhost:5432/victory"); err == nil {
		t.Fatal("live database must refuse the skip-backup escape hatch")
	}
}
