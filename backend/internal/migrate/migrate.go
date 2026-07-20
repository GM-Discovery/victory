// Package migrate applies the embedded SQL migrations (backend/migrations)
// at startup, tracked in a schema_migrations ledger. Kernel 72 (track O3):
// before this existed, migrations were applied by hand and the Kernel 70A
// deploy discovered prod had silently drifted behind the repo.
//
// Semantics:
//   - Every embedded file gets a row in schema_migrations (filename,
//     sha256 checksum, applied_at) once applied.
//   - A recorded file whose embedded content changed, or which is no longer
//     embedded, refuses boot — history is append-only.
//   - Pending files are applied in filename order. Each file is executed as
//     one multi-statement Exec WITHOUT a wrapping transaction, because
//     several historical files carry their own BEGIN/COMMIT. Every file is
//     required to be idempotent (the repo-wide convention since 000), so a
//     partial failure is retried safely on next boot.
//   - When at least one file is pending and the database is non-empty, a
//     pg_dump custom-format backup is written to BackupDir first; a failed
//     backup aborts the migration and the boot.
//   - Options.ApplyOnBoot=false (MIGRATE_ON_BOOT=false) verifies only:
//     pending migrations name themselves in the refusal error.
package migrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/migrations"
)

type Options struct {
	// DatabaseURL is used by pg_dump for the pre-apply backup.
	DatabaseURL string
	// BackupDir receives pre-apply pg_dump files. Created if missing.
	BackupDir string
	// ApplyOnBoot false = verify only; refuse to boot while anything is pending.
	ApplyOnBoot bool
	// DangerouslySkipBackup is for tests against disposable databases only.
	// Production boots must never set this.
	DangerouslySkipBackup bool
}

type file struct {
	Name     string
	Content  string
	Checksum string
}

func embeddedFiles() ([]file, error) {
	entries, err := migrations.Files.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}
	files := make([]file, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		raw, err := migrations.Files.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read embedded migration %s: %w", entry.Name(), err)
		}
		sum := sha256.Sum256(raw)
		files = append(files, file{
			Name:     entry.Name(),
			Content:  string(raw),
			Checksum: hex.EncodeToString(sum[:]),
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	if len(files) == 0 {
		return nil, fmt.Errorf("no embedded migration files — build is broken")
	}
	return files, nil
}

// plan splits embedded files into already-applied and pending, refusing on
// any divergence between the ledger and the embedded set.
func plan(files []file, ledger map[string]string) (pending []file, err error) {
	byName := make(map[string]file, len(files))
	for _, f := range files {
		byName[f.Name] = f
	}
	for name, checksum := range ledger {
		f, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("schema_migrations records %q but it is not embedded in this binary — migration history is append-only; restore the file", name)
		}
		if f.Checksum != checksum {
			return nil, fmt.Errorf("embedded migration %q does not match the checksum recorded when it was applied — never edit an applied migration; add a new file instead", name)
		}
	}
	lastApplied := ""
	for name := range ledger {
		if name > lastApplied {
			lastApplied = name
		}
	}
	for _, f := range files {
		if _, done := ledger[f.Name]; done {
			continue
		}
		if lastApplied != "" && f.Name < lastApplied {
			log.Printf("migrate: warning: pending %q sorts before already-applied %q (out-of-order addition)", f.Name, lastApplied)
		}
		pending = append(pending, f)
	}
	return pending, nil
}

func Run(ctx context.Context, pool *pgxpool.Pool, opts Options) error {
	files, err := embeddedFiles()
	if err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
		  filename TEXT PRIMARY KEY,
		  checksum TEXT NOT NULL,
		  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`); err != nil {
		return fmt.Errorf("ensure schema_migrations ledger: %w", err)
	}

	ledger := map[string]string{}
	rows, err := pool.Query(ctx, `SELECT filename, checksum FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("read schema_migrations ledger: %w", err)
	}
	for rows.Next() {
		var name, checksum string
		if err := rows.Scan(&name, &checksum); err != nil {
			rows.Close()
			return fmt.Errorf("scan schema_migrations row: %w", err)
		}
		ledger[name] = checksum
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read schema_migrations ledger: %w", err)
	}

	pending, err := plan(files, ledger)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		log.Printf("migrate: schema current (%d migrations recorded)", len(ledger))
		return nil
	}

	names := make([]string, len(pending))
	for i, f := range pending {
		names[i] = f.Name
	}
	if !opts.ApplyOnBoot {
		return fmt.Errorf("MIGRATE_ON_BOOT=false and %d migration(s) pending: %s — refusing to boot against a stale schema", len(pending), strings.Join(names, ", "))
	}
	log.Printf("migrate: %d pending migration(s): %s", len(pending), strings.Join(names, ", "))

	empty, err := databaseEmpty(ctx, pool)
	if err != nil {
		return err
	}
	if empty {
		log.Printf("migrate: fresh database, skipping pre-apply backup")
	} else if opts.DangerouslySkipBackup {
		if err := requireDisposableDatabase(opts.DatabaseURL); err != nil {
			return err
		}
		log.Printf("migrate: DangerouslySkipBackup set — no pre-apply backup (disposable databases only)")
	} else {
		if err := backup(ctx, opts, len(pending)); err != nil {
			return fmt.Errorf("pre-apply backup failed, refusing to migrate: %w", err)
		}
	}

	for _, f := range pending {
		started := time.Now()
		if _, err := pool.Exec(ctx, f.Content); err != nil {
			return fmt.Errorf("apply %s: %w", f.Name, err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO schema_migrations (filename, checksum) VALUES ($1, $2)
			 ON CONFLICT (filename) DO NOTHING`,
			f.Name, f.Checksum); err != nil {
			return fmt.Errorf("record %s in schema_migrations: %w", f.Name, err)
		}
		log.Printf("migrate: applied %s in %s", f.Name, time.Since(started).Round(time.Millisecond))
	}
	log.Printf("migrate: %d migration(s) applied, schema current", len(pending))
	return nil
}

// databaseEmpty reports whether the database has any base table beyond the
// ledger itself — used to skip the backup on a fresh install.
func databaseEmpty(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM information_schema.tables
		  WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
		    AND table_type = 'BASE TABLE'
		    AND table_name <> 'schema_migrations'
		)
	`).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check database emptiness: %w", err)
	}
	return !exists, nil
}

func backup(ctx context.Context, opts Options, pendingCount int) error {
	if strings.TrimSpace(opts.DatabaseURL) == "" {
		return fmt.Errorf("no DatabaseURL configured for pg_dump")
	}
	dir := opts.BackupDir
	if strings.TrimSpace(dir) == "" {
		dir = "/opt/victory/backups"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create backup dir %s: %w", dir, err)
	}
	out := filepath.Join(dir, fmt.Sprintf("victory_pre_migrate_%s_%dpending.dump",
		time.Now().Format("20060102_150405"), pendingCount))
	cmd := exec.CommandContext(ctx, "pg_dump", "--format=custom", "--file", out, "--dbname", opts.DatabaseURL)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pg_dump: %w: %s", err, strings.TrimSpace(string(output)))
	}
	info, err := os.Stat(out)
	if err != nil || info.Size() == 0 {
		return fmt.Errorf("pg_dump produced no usable file at %s", out)
	}
	log.Printf("migrate: pre-apply backup written to %s (%d bytes)", out, info.Size())
	return nil
}

// requireDisposableDatabase is the guard on the skip-backup escape hatch: a
// non-empty database may only skip the pre-apply backup when its name marks
// it clearly disposable ("test" or "fresh"), mirroring internal/dbtest's
// safety-gate philosophy. An empty DatabaseURL (pool-only test callers) is
// allowed — those run under dbtest's own TEST_DATABASE_URL validation.
func requireDisposableDatabase(databaseURL string) error {
	trimmed := strings.TrimSpace(databaseURL)
	if trimmed == "" {
		return nil
	}
	cfg, err := pgxpool.ParseConfig(trimmed)
	if err != nil {
		return fmt.Errorf("skip-backup guard: cannot parse database URL: %w", err)
	}
	name := strings.ToLower(strings.TrimSpace(cfg.ConnConfig.Database))
	if strings.Contains(name, "test") || strings.Contains(name, "fresh") {
		return nil
	}
	return fmt.Errorf("refusing to skip the pre-apply backup against non-disposable database %q — unset MIGRATE_DANGEROUSLY_SKIP_BACKUP", name)
}

// OptionsFromEnv builds production Options from the environment.
func OptionsFromEnv(databaseURL string) Options {
	applyOnBoot := true
	if raw, ok := os.LookupEnv("MIGRATE_ON_BOOT"); ok {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "0", "false", "f", "no", "n", "off":
			applyOnBoot = false
		}
	}
	backupDir := strings.TrimSpace(os.Getenv("BACKUP_DIR"))
	if backupDir == "" {
		backupDir = "/opt/victory/backups"
	}
	skipBackup := false
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MIGRATE_DANGEROUSLY_SKIP_BACKUP"))) {
	case "1", "true", "yes", "on":
		skipBackup = true // guarded: Run refuses this against non-disposable DB names
	}
	return Options{
		DatabaseURL:           databaseURL,
		BackupDir:             backupDir,
		ApplyOnBoot:           applyOnBoot,
		DangerouslySkipBackup: skipBackup,
	}
}
