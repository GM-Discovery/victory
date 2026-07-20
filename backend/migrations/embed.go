// Package migrations embeds every SQL migration file into the backend
// binary. As of Kernel 72 this directory (backend/migrations/) is the single
// source of schema truth: no Go code outside backend/internal/migrate may
// execute DDL, and the runner in internal/migrate applies these files at
// startup against a schema_migrations ledger.
//
// Files must be named NNN_description.sql and sort in apply order. Never
// edit a file after it has shipped — the runner refuses to boot on a
// checksum mismatch. Add a new file instead.
package migrations

import "embed"

//go:embed *.sql
var Files embed.FS
