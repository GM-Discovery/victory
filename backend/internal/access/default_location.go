package access

import (
	"context"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultLocationSlug is the location an install treats as its own primary
// space wherever code needs one and nothing more specific applies (fallback
// Location lookups, audience auto-membership on signup, Kessa's Locked
// Courtyard opening, and similar). Kernel 100: this must come from what an
// install's own Operator named their lot at first-run (DEFAULT_LOCATION_SLUG),
// never a fixed slug -- hardcoding "amurray-family" throughout these call
// sites was harmless while this was a single-tenant app, but leaks Grant's
// own real location into every other install the moment it runs on anyone
// else's machine. Falls back to "victory-theater", the neutral location
// Kernel 42 seeded for exactly this purpose, only for deployments that
// predate this env var.
func DefaultLocationSlug() string {
	slug := strings.TrimSpace(os.Getenv("DEFAULT_LOCATION_SLUG"))
	if slug == "" {
		slug = "victory-theater"
	}
	return slug
}

// EnsureDefaultLocation creates the locations/lots rows for
// DefaultLocationSlug() if they don't already exist. Every existing
// Ensure*Surface bootstrap (venues, profiles, the Courtyard, the Socio
// manuscript) only ever SELECTs a location by slug -- none of them create
// one -- because until Kernel 100 the only slugs in play were the two
// migrations already seed unconditionally ("amurray-family",
// "victory-theater"). A fresh install naming its own lot something else
// entirely (the whole point of asking at first-run) would otherwise leave
// every one of those bootstraps silently seeding nothing, onto a location
// that never exists. Must run before all of them. Safe to call on every
// boot: ON CONFLICT DO NOTHING once the rows exist.
func EnsureDefaultLocation(ctx context.Context, pool *pgxpool.Pool) error {
	slug := DefaultLocationSlug()

	name := strings.TrimSpace(os.Getenv("DEFAULT_LOCATION_NAME"))
	if name == "" {
		name = humanizeSlug(slug)
	}

	var locationID string
	err := pool.QueryRow(ctx, `
		INSERT INTO locations (slug, name)
		VALUES ($1, $2)
		ON CONFLICT (slug) DO UPDATE SET slug = EXCLUDED.slug
		RETURNING id::text
	`, slug, name).Scan(&locationID)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO lots (location_id, name, slug)
		VALUES ($1, 'main-lot', 'main-lot')
		ON CONFLICT (location_id, slug) DO NOTHING
	`, locationID)
	return err
}

func humanizeSlug(slug string) string {
	words := strings.FieldsFunc(slug, func(r rune) bool { return r == '-' || r == '_' })
	if len(words) == 0 {
		return slug
	}
	for i, w := range words {
		if w == "" {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}
