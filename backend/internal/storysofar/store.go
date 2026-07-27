package storysofar

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Ref is the participation a generated story belongs to. Every field except
// CharacterCardID/OwnerUserID may be empty; the optional ones are stored as
// NULL context rather than being required.
type Ref struct {
	CharacterCardID string
	OwnerUserID     string
	ShowRunID       string
	ShowID          string
	SessionID       string
	PlacementID     string
}

func (r Ref) valid() error {
	if strings.TrimSpace(r.CharacterCardID) == "" {
		return errors.New("no_character_selected")
	}
	if strings.TrimSpace(r.OwnerUserID) == "" {
		return errors.New("not_authenticated")
	}
	return nil
}

func nullableID(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}

// Persist writes drafts idempotently.
//
// ON CONFLICT DO NOTHING against uq_character_story_events_dedupe is the
// whole retry story: a second Continue re-derives the same tuples (Generate
// is pure) and writes nothing. There is deliberately no read-then-write and
// no transaction wrapping the batch -- a partial write self-heals on the
// next press, and a transaction would buy nothing because each row is
// independently idempotent.
func Persist(ctx context.Context, pool *pgxpool.Pool, ref Ref, drafts []DraftEvent) error {
	if err := ref.valid(); err != nil {
		return err
	}
	for _, d := range drafts {
		occurred := d.OccurredAt
		if occurred.IsZero() {
			occurred = time.Now().UTC()
		}
		_, err := pool.Exec(ctx, `
			INSERT INTO character_story_events (
				character_card_id, owner_user_id, event_type, title, summary,
				show_run_id, show_id, session_id, show_scene_placement_id,
				source_kind, source_ref, visibility_state, generator_version, occurred_at
			)
			VALUES (
				$1, $2, $3, $4, $5,
				$6::uuid, $7::uuid, $8::uuid, $9::uuid,
				$10, $11, $12, $13, $14
			)
			ON CONFLICT (character_card_id, event_type, source_kind, source_ref) DO NOTHING
		`,
			ref.CharacterCardID, ref.OwnerUserID, d.EventType, d.Title, d.Summary,
			nullableID(ref.ShowRunID), nullableID(ref.ShowID),
			nullableID(ref.SessionID), nullableID(ref.PlacementID),
			d.SourceKind, d.SourceRef, VisibilityPrivate, GeneratorVersion, occurred,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// RecordAuthored writes a single entry attributed to a person -- today only
// a Director-authored moment arriving through /journal (S6.4). The author's
// identity is preserved rather than erased, and the entry is private by
// default like every other.
func RecordAuthored(ctx context.Context, pool *pgxpool.Pool, ref Ref, authorUserID string, d DraftEvent) error {
	if err := ref.valid(); err != nil {
		return err
	}
	if strings.TrimSpace(authorUserID) == "" {
		return errors.New("not_authenticated")
	}
	occurred := d.OccurredAt
	if occurred.IsZero() {
		occurred = time.Now().UTC()
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO character_story_events (
			character_card_id, owner_user_id, event_type, title, summary,
			show_run_id, show_id, session_id, show_scene_placement_id,
			source_kind, source_ref, visibility_state, authored_by_user_id,
			generator_version, occurred_at
		)
		VALUES (
			$1, $2, $3, $4, $5,
			$6::uuid, $7::uuid, $8::uuid, $9::uuid,
			$10, $11, $12, $13, $14, $15
		)
		ON CONFLICT (character_card_id, event_type, source_kind, source_ref) DO NOTHING
	`,
		ref.CharacterCardID, ref.OwnerUserID, d.EventType, d.Title, d.Summary,
		nullableID(ref.ShowRunID), nullableID(ref.ShowID),
		nullableID(ref.SessionID), nullableID(ref.PlacementID),
		d.SourceKind, d.SourceRef, VisibilityPrivate, authorUserID,
		GeneratorVersion, occurred,
	)
	return err
}

const storyEventColumns = `
	e.id::text, e.character_card_id::text, e.owner_user_id::text,
	e.event_type, e.title, e.summary,
	COALESCE(e.show_run_id::text, ''), COALESCE(e.show_id::text, ''),
	COALESCE(e.session_id::text, ''), COALESCE(e.show_scene_placement_id::text, ''),
	e.source_kind, e.source_ref, e.visibility_state,
	COALESCE(e.authored_by_user_id::text, ''),
	COALESCE(NULLIF(u.display_name, ''), ''),
	e.generator_version, e.occurred_at, e.created_at
`

func scanStoryEvents(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close()
}) ([]StoryEvent, error) {
	defer rows.Close()
	out := []StoryEvent{}
	for rows.Next() {
		var e StoryEvent
		if err := rows.Scan(
			&e.ID, &e.CharacterCardID, &e.OwnerUserID,
			&e.EventType, &e.Title, &e.Summary,
			&e.ShowRunID, &e.ShowID, &e.SessionID, &e.PlacementID,
			&e.SourceKind, &e.SourceRef, &e.VisibilityState,
			&e.AuthoredByUserID, &e.AuthoredByName,
			&e.GeneratorVersion, &e.OccurredAt, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ListForCharacter returns one Character's story in chronological order.
//
// ownerView is the privacy gate and it is NOT optional. S12 requires that a
// Player read only their own Character history, and that private entries do
// not reach anyone else -- including a Director with location authority,
// who passes characters.CanEditCard and would otherwise see everything.
// Callers must pass ownerView=false for any viewer who is not the Character's
// owner; this function then returns only entries the Player has explicitly
// shared with the table.
//
// The filter lives in SQL rather than in the caller on purpose: an
// omit-from-payload gate applied by each caller is the pattern that produced
// Kernel 74's authority hole.
func ListForCharacter(ctx context.Context, pool *pgxpool.Pool, characterCardID string, ownerView bool) ([]StoryEvent, error) {
	characterCardID = strings.TrimSpace(characterCardID)
	if characterCardID == "" {
		return nil, errors.New("no_character_selected")
	}
	rows, err := pool.Query(ctx, `
		SELECT `+storyEventColumns+`
		FROM character_story_events e
		LEFT JOIN users u ON u.id = e.authored_by_user_id
		WHERE e.character_card_id = $1
		  AND ($2::boolean OR e.visibility_state = 'table')
		ORDER BY e.occurred_at ASC, e.created_at ASC, e.id ASC
	`, characterCardID, ownerView)
	if err != nil {
		return nil, err
	}
	return scanStoryEvents(rows)
}

// ListForShow returns every story event recorded during one Show, for the
// Directors+ Aftercare review. Show-scoped rather than Character-scoped, and
// gated at the HTTP layer by showruns.CanViewBackstage.
func ListForShow(ctx context.Context, pool *pgxpool.Pool, showID string) ([]StoryEvent, error) {
	rows, err := pool.Query(ctx, `
		SELECT `+storyEventColumns+`
		FROM character_story_events e
		LEFT JOIN users u ON u.id = e.authored_by_user_id
		WHERE e.show_id = $1
		ORDER BY e.character_card_id, e.occurred_at ASC, e.created_at ASC, e.id ASC
	`, showID)
	if err != nil {
		return nil, err
	}
	return scanStoryEvents(rows)
}

// SetVisibility is the Player's reveal control (S1.8, S5.5).
//
// Scoped to owner_user_id in the WHERE clause, not merely checked before it:
// this is intentionally STRICTER than characters.CanEditCard, which grants
// location authority-holders write access to a Character. A Director must
// not be able to publish a Player's private reflection, so ownership is the
// only key that opens this door.
//
// Note what this does NOT allow: the Player may change an entry's visibility
// and nothing else. Generated text is not editable through any route
// (S5.5) -- there is no UPDATE ... SET summary anywhere in this package.
func SetVisibility(ctx context.Context, pool *pgxpool.Pool, ownerUserID, eventID, visibilityState string) (StoryEvent, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return StoryEvent{}, errors.New("not_authenticated")
	}
	visibilityState = strings.TrimSpace(visibilityState)
	if visibilityState != VisibilityPrivate && visibilityState != VisibilityTable {
		return StoryEvent{}, errors.New("invalid_visibility_state")
	}
	tag, err := pool.Exec(ctx, `
		UPDATE character_story_events
		SET visibility_state = $3
		WHERE id = $1 AND owner_user_id = $2
	`, eventID, ownerUserID, visibilityState)
	if err != nil {
		return StoryEvent{}, err
	}
	if tag.RowsAffected() == 0 {
		return StoryEvent{}, errors.New("not_found")
	}
	rows, err := pool.Query(ctx, `
		SELECT `+storyEventColumns+`
		FROM character_story_events e
		LEFT JOIN users u ON u.id = e.authored_by_user_id
		WHERE e.id = $1
	`, eventID)
	if err != nil {
		return StoryEvent{}, err
	}
	out, err := scanStoryEvents(rows)
	if err != nil {
		return StoryEvent{}, err
	}
	if len(out) == 0 {
		return StoryEvent{}, errors.New("not_found")
	}
	return out[0], nil
}
