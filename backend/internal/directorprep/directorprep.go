package directorprep

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/announcements"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
)

const preparationColumns = `
	id::text, show_id::text, kind, label, payload, sort_order,
	COALESCE(created_by_user_id::text, ''), created_at, updated_at
`

func scanPreparation(row pgx.Row) (Preparation, error) {
	var p Preparation
	var payloadRaw []byte
	if err := row.Scan(
		&p.ID, &p.ShowID, &p.Kind, &p.Label, &payloadRaw, &p.SortOrder,
		&p.CreatedByUserID, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Preparation{}, errors.New("preparation_not_found")
		}
		return Preparation{}, err
	}
	p.Payload = map[string]any{}
	_ = json.Unmarshal(payloadRaw, &p.Payload)
	return p, nil
}

// RequireDirector is the one authority gate for this package. It resolves
// the Show's Show Run location and asks showruns.CanManageShowRun -- the
// same producer/director check cohorts.requireManage and
// socio.requireShowCharacterAuthority use, rather than a fourth opinion
// about what "Director+" means.
func RequireDirector(ctx context.Context, pool *pgxpool.Pool, actorUserID, showID string) error {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return errors.New("not_authenticated")
	}
	s, err := shows.LoadShowByID(ctx, pool, showID)
	if err != nil {
		return err
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return err
	}
	ok, err := showruns.CanManageShowRun(ctx, pool, actorUserID, sr.LocationID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("not_authorized")
	}
	return nil
}

// ValidatePayload is the per-kind typed validator. It is the reason
// migration 105's JSONB column is not an executable surface: every field a
// consumer will ever read is named and bounded here, and an unrecognised
// key is dropped rather than stored, so a client cannot smuggle extra
// instructions into a preparation and hope some later reader honours them.
func ValidatePayload(kind string, in map[string]any) (map[string]any, error) {
	switch kind {
	case KindTargetComplexity:
		value, err := intField(in, "value")
		if err != nil {
			return nil, errors.New("target_complexity_value_required")
		}
		if value < MinComplexityValue || value > MaxComplexityValue {
			return nil, errors.New("target_complexity_out_of_range")
		}
		note := strings.TrimSpace(stringField(in, "note"))
		if len([]rune(note)) > MaxNoteLength {
			return nil, errors.New("note_too_long")
		}
		out := map[string]any{"value": value}
		if note != "" {
			out["note"] = note
		}
		return out, nil

	case KindAnnouncement:
		// Reuse the palette's own validator rather than re-listing style
		// keys here -- a saved announcement preset and a live one must
		// agree about what styles exist, and they do because there is only
		// one Compose.
		style, text, err := announcements.Compose(stringField(in, "style"), stringField(in, "text"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"style": style.Key, "text": text}, nil

	default:
		return nil, errors.New("unknown_preparation_kind")
	}
}

func stringField(in map[string]any, key string) string {
	if in == nil {
		return ""
	}
	if v, ok := in[key].(string); ok {
		return v
	}
	return ""
}

// intField accepts a JSON number (float64 after decoding) or a numeric
// string, since a plain <input type="number"> hands back a string when the
// caller forgets to coerce it and rejecting that would be a papercut, not
// a safety property.
func intField(in map[string]any, key string) (int, error) {
	if in == nil {
		return 0, errors.New("missing")
	}
	switch v := in[key].(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, errors.New("missing")
		}
		n := 0
		neg := false
		for i, r := range trimmed {
			if i == 0 && r == '-' {
				neg = true
				continue
			}
			if r < '0' || r > '9' {
				return 0, errors.New("not_a_number")
			}
			n = n*10 + int(r-'0')
			if n > 100000 {
				return 0, errors.New("not_a_number")
			}
		}
		if neg {
			n = -n
		}
		return n, nil
	default:
		return 0, errors.New("missing")
	}
}

func normalizeLabel(label string) (string, error) {
	label = strings.Join(strings.Fields(label), " ")
	if label == "" {
		return "", errors.New("label_required")
	}
	if len([]rune(label)) > MaxLabelLength {
		return "", errors.New("label_too_long")
	}
	return label, nil
}

// CreateInput is a new preparation. Kind and payload are validated before
// any write; the actor is authorized by the caller through RequireDirector.
type CreateInput struct {
	ShowID    string
	Kind      string
	Label     string
	Payload   map[string]any
	SortOrder int
	ActorID   string
}

func Create(ctx context.Context, pool *pgxpool.Pool, in CreateInput) (Preparation, error) {
	label, err := normalizeLabel(in.Label)
	if err != nil {
		return Preparation{}, err
	}
	payload, err := ValidatePayload(in.Kind, in.Payload)
	if err != nil {
		return Preparation{}, err
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return Preparation{}, err
	}
	row := pool.QueryRow(ctx, `
		INSERT INTO director_preparations (show_id, kind, label, payload, sort_order, created_by_user_id)
		VALUES ($1, $2, $3, $4::jsonb, $5, NULLIF($6, '')::uuid)
		RETURNING `+preparationColumns,
		in.ShowID, in.Kind, label, encoded, in.SortOrder, strings.TrimSpace(in.ActorID))
	return scanPreparation(row)
}

// UpdateInput patches a preparation. Nil fields are left alone, so a
// Director changing a target complexity live (kernel 89 §7's "Director can
// change the value live") sends only the number.
type UpdateInput struct {
	Label     *string
	Payload   *map[string]any
	SortOrder *int
}

func Update(ctx context.Context, pool *pgxpool.Pool, preparationID string, in UpdateInput) (Preparation, error) {
	existing, err := LoadByID(ctx, pool, preparationID)
	if err != nil {
		return Preparation{}, err
	}

	label := existing.Label
	if in.Label != nil {
		label, err = normalizeLabel(*in.Label)
		if err != nil {
			return Preparation{}, err
		}
	}

	payload := existing.Payload
	if in.Payload != nil {
		payload, err = ValidatePayload(existing.Kind, *in.Payload)
		if err != nil {
			return Preparation{}, err
		}
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return Preparation{}, err
	}

	sortOrder := existing.SortOrder
	if in.SortOrder != nil {
		sortOrder = *in.SortOrder
	}

	row := pool.QueryRow(ctx, `
		UPDATE director_preparations
		SET label = $2, payload = $3::jsonb, sort_order = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING `+preparationColumns,
		preparationID, label, encoded, sortOrder)
	return scanPreparation(row)
}

func Delete(ctx context.Context, pool *pgxpool.Pool, preparationID string) error {
	tag, err := pool.Exec(ctx, `DELETE FROM director_preparations WHERE id = $1`, preparationID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("preparation_not_found")
	}
	return nil
}

func LoadByID(ctx context.Context, pool *pgxpool.Pool, preparationID string) (Preparation, error) {
	preparationID = strings.TrimSpace(preparationID)
	if preparationID == "" {
		return Preparation{}, errors.New("preparation_not_found")
	}
	row := pool.QueryRow(ctx, `SELECT `+preparationColumns+` FROM director_preparations WHERE id = $1`, preparationID)
	return scanPreparation(row)
}

// List returns a Show's preparations, optionally narrowed to one kind.
// Ordered so the Director's own arrangement (sort_order) wins and creation
// order breaks ties -- a recall list that reshuffles itself between opens
// would be useless mid-scene.
func List(ctx context.Context, pool *pgxpool.Pool, showID, kind string) ([]Preparation, error) {
	kind = strings.TrimSpace(kind)
	rows, err := pool.Query(ctx, `
		SELECT `+preparationColumns+`
		FROM director_preparations
		WHERE show_id = $1 AND ($2 = '' OR kind = $2)
		ORDER BY kind, sort_order, created_at
	`, showID, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Preparation{}
	for rows.Next() {
		p, err := scanPreparation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
