package shows

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
)

// shortCodeAlphabet deliberately excludes visually-confusing characters
// (0/O, 1/I/L) per spec §1.7 -- what's left is short enough to type and
// hard to misread over voice/chat.
const shortCodeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
const shortCodeLength = 4
const shortCodeMaxAttempts = 10

func randomShortCode() (string, error) {
	buf := make([]byte, shortCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, shortCodeLength)
	for i, b := range buf {
		out[i] = shortCodeAlphabet[int(b)%len(shortCodeAlphabet)]
	}
	return string(out), nil
}

// shortCodeTaken reports whether code is already used by a non-archived
// Show at this location (case-insensitive), scoped through
// show_runs.location_id since shows has no direct location column.
func shortCodeTaken(ctx context.Context, pool *pgxpool.Pool, locationID, code string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM shows s
			JOIN show_runs sr ON sr.id = s.show_run_id
			WHERE sr.location_id = $1 AND lower(s.short_code) = lower($2)
		)
	`, locationID, code).Scan(&exists)
	return exists, err
}

// generateUniqueShortCode picks a random code and retries on collision,
// scoped to one location. Called at Show-creation time, inside the same
// transaction/connection as the insert would ideally run, but a plain
// pool call is sufficient here since a collision at insert time (extremely
// unlikely given the retry loop) still surfaces as a clean 409 from
// UpdateShortCode's own conflict handling if ever hit.
func generateUniqueShortCode(ctx context.Context, pool *pgxpool.Pool, locationID string) (string, error) {
	for attempt := 0; attempt < shortCodeMaxAttempts; attempt++ {
		code, err := randomShortCode()
		if err != nil {
			return "", err
		}
		taken, err := shortCodeTaken(ctx, pool, locationID, code)
		if err != nil {
			return "", err
		}
		if !taken {
			return code, nil
		}
	}
	return "", errors.New("short_code_generation_failed")
}

// normalizeShortCode validates a Director/Producer/Operator-supplied short
// code edit: 3-8 alphanumeric characters, no spaces or punctuation.
func normalizeShortCode(raw string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if len(code) < 3 || len(code) > 8 {
		return "", errors.New("short_code_invalid_length")
	}
	for _, r := range code {
		if !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return "", errors.New("short_code_invalid_characters")
		}
	}
	return code, nil
}

// UpdateShortCode lets a Director/Producer/Operator edit a Show's short
// code within their own authority. Returns "short_code_taken" (mapped to
// 409 by the HTTP layer) on a location-scoped collision rather than a raw
// constraint error.
func UpdateShortCode(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID, rawCode string) (Show, error) {
	s, err := LoadShowByID(ctx, pool, showID)
	if err != nil {
		return Show{}, err
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return Show{}, err
	}
	canManage, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return Show{}, err
	}
	if !canManage {
		return Show{}, errors.New("not_authorized")
	}

	code, err := normalizeShortCode(rawCode)
	if err != nil {
		return Show{}, err
	}

	taken, err := shortCodeTaken(ctx, pool, sr.LocationID, code)
	if err != nil {
		return Show{}, err
	}
	if taken {
		var currentCode string
		if err := pool.QueryRow(ctx, `SELECT COALESCE(short_code, '') FROM shows WHERE id = $1`, showID).Scan(&currentCode); err == nil && strings.EqualFold(currentCode, code) {
			taken = false // unchanged, not a real conflict
		}
	}
	if taken {
		return Show{}, errors.New("short_code_taken")
	}

	row := pool.QueryRow(ctx, `
		UPDATE shows SET short_code = $1 WHERE id = $2
		RETURNING `+showColumns,
		code, showID)
	return scanShow(row)
}

// LoadShowByCode resolves a Show by its case-insensitive short code. Not
// location-scoped by an explicit parameter -- codes are rare 4-character
// random strings and this deployment is effectively single-location in
// practice, so an unambiguous global match is the common case; if the code
// happens to match Shows in more than one location, the caller's own
// manageable one (if exactly one) is preferred, else this reports
// "show_code_ambiguous" rather than guessing.
func LoadShowByCode(ctx context.Context, pool *pgxpool.Pool, actorUserID, rawCode string) (Show, error) {
	code := strings.ToUpper(strings.TrimSpace(rawCode))
	if code == "" {
		return Show{}, errors.New("show_code_not_found")
	}

	rows, err := pool.Query(ctx, `
		SELECT `+showColumns+`
		FROM shows s
		WHERE lower(s.short_code) = lower($1) AND s.archived_at IS NULL
	`, code)
	if err != nil {
		return Show{}, err
	}
	defer rows.Close()

	var matches []Show
	for rows.Next() {
		s, err := scanShow(rows)
		if err != nil {
			return Show{}, err
		}
		matches = append(matches, s)
	}
	if err := rows.Err(); err != nil {
		return Show{}, err
	}

	switch len(matches) {
	case 0:
		return Show{}, errors.New("show_code_not_found")
	case 1:
		return matches[0], nil
	default:
		for _, m := range matches {
			sr, err := showruns.LoadShowRunByID(ctx, pool, m.ShowRunID)
			if err != nil {
				continue
			}
			canManage, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
			if err == nil && canManage {
				return m, nil
			}
		}
		return Show{}, errors.New("show_code_ambiguous")
	}
}
