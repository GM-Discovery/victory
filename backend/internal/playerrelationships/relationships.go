package playerrelationships

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// resolveSubjectUserID maps an opaque Player Workbook / profile ID to the
// subject's stable account UUID (Kernel 61 contract reuse -- Kernel 62 §3,
// §9.1). Ordinary URLs and API payloads never carry the raw account UUID.
func resolveSubjectUserID(ctx context.Context, pool *pgxpool.Pool, subjectProfileID string) (string, error) {
	subjectProfileID = strings.TrimSpace(subjectProfileID)
	if subjectProfileID == "" {
		return "", errors.New("subject_profile_id_required")
	}
	var userID string
	if err := pool.QueryRow(ctx, `
		SELECT user_id::text FROM player_profile_workbooks WHERE id = $1
	`, subjectProfileID).Scan(&userID); err != nil {
		return "", errors.New("profile_not_found")
	}
	return userID, nil
}

// EnsureRelationship creates (or returns) the observer's private directional
// record about the subject (Kernel 62 §9.1). The observer always comes from
// the session; a self-relationship is rejected before touching the table.
func EnsureRelationship(ctx context.Context, pool *pgxpool.Pool, observerUserID, subjectProfileID string) (Relationship, bool, error) {
	observerUserID = strings.TrimSpace(observerUserID)
	if observerUserID == "" {
		return Relationship{}, false, errors.New("not_authenticated")
	}

	subjectUserID, err := resolveSubjectUserID(ctx, pool, subjectProfileID)
	if err != nil {
		return Relationship{}, false, err
	}
	if subjectUserID == observerUserID {
		return Relationship{}, false, errors.New("cannot_relate_to_self")
	}

	tag, err := pool.Exec(ctx, `
		INSERT INTO player_relationships (observer_user_id, subject_user_id)
		VALUES ($1, $2)
		ON CONFLICT (observer_user_id, subject_user_id) DO NOTHING
	`, observerUserID, subjectUserID)
	if err != nil {
		return Relationship{}, false, err
	}
	created := tag.RowsAffected() > 0

	rel, err := loadRelationshipBySubject(ctx, pool, observerUserID, subjectUserID)
	return rel, created, err
}

const relationshipColumns = `
	id::text, observer_user_id::text, subject_user_id::text,
	private_nickname, relationship_state,
	trust_level, closeness_level, reliability_level, communication_ease_level,
	archived_at, created_at, updated_at
`

func scanRelationship(row pgx.Row) (Relationship, error) {
	var r Relationship
	if err := row.Scan(
		&r.ID, &r.ObserverUserID, &r.SubjectUserID,
		&r.PrivateNickname, &r.RelationshipState,
		&r.TrustLevel, &r.ClosenessLevel, &r.ReliabilityLevel, &r.CommunicationEaseLevel,
		&r.ArchivedAt, &r.CreatedAt, &r.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Relationship{}, errors.New("relationship_not_found")
		}
		return Relationship{}, err
	}
	return r, nil
}

// loadRelationshipOwned is the single ownership gate every relationship
// operation flows through: no row for (id, observer) means 404-shaped
// "relationship_not_found" -- never 403, so existence is not confirmed to
// non-owners (Kernel 62 §12.1).
func loadRelationshipOwned(ctx context.Context, pool *pgxpool.Pool, observerUserID, relationshipID string) (Relationship, error) {
	observerUserID = strings.TrimSpace(observerUserID)
	relationshipID = strings.TrimSpace(relationshipID)
	if observerUserID == "" {
		return Relationship{}, errors.New("not_authenticated")
	}
	if relationshipID == "" {
		return Relationship{}, errors.New("relationship_not_found")
	}
	row := pool.QueryRow(ctx, `
		SELECT `+relationshipColumns+`
		FROM player_relationships
		WHERE id = $1 AND observer_user_id = $2
	`, relationshipID, observerUserID)
	return scanRelationship(row)
}

func loadRelationshipBySubject(ctx context.Context, pool *pgxpool.Pool, observerUserID, subjectUserID string) (Relationship, error) {
	row := pool.QueryRow(ctx, `
		SELECT `+relationshipColumns+`
		FROM player_relationships
		WHERE observer_user_id = $1 AND subject_user_id = $2
	`, observerUserID, subjectUserID)
	return scanRelationship(row)
}

// GetRelationshipBySubjectProfile returns the observer's relationship about
// the subject identified by opaque profile ID, if one exists.
func GetRelationshipBySubjectProfile(ctx context.Context, pool *pgxpool.Pool, observerUserID, subjectProfileID string) (Relationship, error) {
	subjectUserID, err := resolveSubjectUserID(ctx, pool, subjectProfileID)
	if err != nil {
		return Relationship{}, err
	}
	return loadRelationshipBySubject(ctx, pool, observerUserID, subjectUserID)
}

// touchRelationship bumps updated_at so "sort by last updated" reflects any
// mutation.
func touchRelationship(ctx context.Context, pool *pgxpool.Pool, relationshipID string) error {
	_, err := pool.Exec(ctx, `
		UPDATE player_relationships SET updated_at = NOW() WHERE id = $1
	`, relationshipID)
	return err
}

// ArchiveRelationship hides the record from the default list without
// deleting anything and without any effect on the subject (Kernel 62 §14.1).
func ArchiveRelationship(ctx context.Context, pool *pgxpool.Pool, observerUserID, relationshipID string) error {
	if _, err := loadRelationshipOwned(ctx, pool, observerUserID, relationshipID); err != nil {
		return err
	}
	_, err := pool.Exec(ctx, `
		UPDATE player_relationships
		SET archived_at = COALESCE(archived_at, NOW()),
		    relationship_state = 'archived',
		    updated_at = NOW()
		WHERE id = $1 AND observer_user_id = $2
	`, relationshipID, observerUserID)
	return err
}

// UnarchiveRelationship restores the record to the default list
// (Kernel 62 §14.1).
func UnarchiveRelationship(ctx context.Context, pool *pgxpool.Pool, observerUserID, relationshipID string) error {
	if _, err := loadRelationshipOwned(ctx, pool, observerUserID, relationshipID); err != nil {
		return err
	}
	_, err := pool.Exec(ctx, `
		UPDATE player_relationships
		SET archived_at = NULL,
		    relationship_state = CASE WHEN relationship_state = 'archived' THEN 'active' ELSE relationship_state END,
		    updated_at = NOW()
		WHERE id = $1 AND observer_user_id = $2
	`, relationshipID, observerUserID)
	return err
}

// UpdateInput carries the observer's structured relationship edits. Nil
// pointers mean "leave unchanged".
type UpdateInput struct {
	PrivateNickname        *string     `json:"private_nickname"`
	RelationshipState      *string     `json:"relationship_state"`
	TrustLevel             *string     `json:"trust_level"`
	ClosenessLevel         *string     `json:"closeness_level"`
	ReliabilityLevel       *string     `json:"reliability_level"`
	CommunicationEaseLevel *string     `json:"communication_ease_level"`
	Categories             *[]Category `json:"categories"`
}

// UpdateRelationship applies nickname, qualitative dropdown, state, and
// category edits (Kernel 62 §4.3, §4.4, §4.7). Invalid vocabulary keys are
// rejected; the archived state cannot be set here (Kernel 62 §14).
func UpdateRelationship(ctx context.Context, pool *pgxpool.Pool, observerUserID, relationshipID string, input UpdateInput) (Relationship, error) {
	rel, err := loadRelationshipOwned(ctx, pool, observerUserID, relationshipID)
	if err != nil {
		return Relationship{}, err
	}

	if input.PrivateNickname != nil {
		nickname := strings.TrimSpace(*input.PrivateNickname)
		if len([]rune(nickname)) > 120 {
			return Relationship{}, errors.New("private_nickname_too_long")
		}
		rel.PrivateNickname = nickname
	}
	if input.RelationshipState != nil {
		state := strings.TrimSpace(*input.RelationshipState)
		if !ValidateSettableStateKey(state) {
			return Relationship{}, errors.New("invalid_relationship_state")
		}
		rel.RelationshipState = state
	}
	if input.TrustLevel != nil {
		if !ValidateTrustKey(*input.TrustLevel) {
			return Relationship{}, errors.New("invalid_trust_level")
		}
		rel.TrustLevel = *input.TrustLevel
	}
	if input.ClosenessLevel != nil {
		if !ValidateClosenessKey(*input.ClosenessLevel) {
			return Relationship{}, errors.New("invalid_closeness_level")
		}
		rel.ClosenessLevel = *input.ClosenessLevel
	}
	if input.ReliabilityLevel != nil {
		if !ValidateReliabilityKey(*input.ReliabilityLevel) {
			return Relationship{}, errors.New("invalid_reliability_level")
		}
		rel.ReliabilityLevel = *input.ReliabilityLevel
	}
	if input.CommunicationEaseLevel != nil {
		if !ValidateCommunicationEaseKey(*input.CommunicationEaseLevel) {
			return Relationship{}, errors.New("invalid_communication_ease_level")
		}
		rel.CommunicationEaseLevel = *input.CommunicationEaseLevel
	}

	var categories []Category
	if input.Categories != nil {
		categories, err = ValidateCategories(*input.Categories)
		if err != nil {
			return Relationship{}, err
		}
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return Relationship{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE player_relationships
		SET private_nickname = $3,
		    relationship_state = $4,
		    trust_level = $5,
		    closeness_level = $6,
		    reliability_level = $7,
		    communication_ease_level = $8,
		    updated_at = NOW()
		WHERE id = $1 AND observer_user_id = $2
	`, relationshipID, observerUserID,
		rel.PrivateNickname, rel.RelationshipState,
		rel.TrustLevel, rel.ClosenessLevel, rel.ReliabilityLevel, rel.CommunicationEaseLevel,
	); err != nil {
		return Relationship{}, err
	}

	if input.Categories != nil {
		if _, err := tx.Exec(ctx, `
			DELETE FROM player_relationship_categories WHERE relationship_id = $1
		`, relationshipID); err != nil {
			return Relationship{}, err
		}
		for _, c := range categories {
			if _, err := tx.Exec(ctx, `
				INSERT INTO player_relationship_categories (relationship_id, category_key, custom_label)
				VALUES ($1, $2, $3)
				ON CONFLICT DO NOTHING
			`, relationshipID, c.CategoryKey, c.CustomLabel); err != nil {
				return Relationship{}, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Relationship{}, err
	}

	return loadRelationshipOwned(ctx, pool, observerUserID, relationshipID)
}

func loadCategories(ctx context.Context, pool *pgxpool.Pool, relationshipID string) ([]Category, error) {
	rows, err := pool.Query(ctx, `
		SELECT category_key, custom_label
		FROM player_relationship_categories
		WHERE relationship_id = $1
		ORDER BY category_key, custom_label
	`, relationshipID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.CategoryKey, &c.CustomLabel); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListFilter selects which relationship rows the My People list shows
// (Kernel 62 §8.2, §14.2).
type ListFilter string

const (
	FilterActive   ListFilter = "active"
	FilterArchived ListFilter = "archived"
	FilterAll      ListFilter = "all"
)

// ListRelationships returns the observer's My People rows with subject
// header, categories, and list metadata. Archived rows are excluded unless
// asked for (Kernel 62 §9.1).
func ListRelationships(ctx context.Context, pool *pgxpool.Pool, observerUserID string, filter ListFilter) ([]ListItem, error) {
	observerUserID = strings.TrimSpace(observerUserID)
	if observerUserID == "" {
		return nil, errors.New("not_authenticated")
	}

	where := "AND r.archived_at IS NULL"
	switch filter {
	case FilterArchived:
		where = "AND r.archived_at IS NOT NULL"
	case FilterAll:
		where = ""
	}

	rows, err := pool.Query(ctx, `
		SELECT r.id::text, r.private_nickname, r.relationship_state,
		       r.trust_level, r.closeness_level, r.reliability_level, r.communication_ease_level,
		       r.archived_at, r.created_at, r.updated_at,
		       w.id::text AS subject_profile_id,
		       COALESCE(s.stage_name, '') AS subject_stage_name,
		       COALESCE(pf.display_value, '') AS subject_portrait_url,
		       (SELECT MAX(j.created_at) FROM player_relationship_journal_entries j
		         WHERE j.relationship_id = r.id AND j.deleted_at IS NULL) AS last_journal_at,
		       (SELECT COUNT(*) FROM player_relationship_followups f
		         WHERE f.relationship_id = r.id AND f.status = 'open') AS open_followups,
		       (SELECT COUNT(DISTINCT m1.production_id) FROM memberships m1
		          JOIN memberships m2 ON m2.production_id = m1.production_id
		           AND m2.user_id = r.subject_user_id AND m2.active
		         WHERE m1.user_id = r.observer_user_id AND m1.active
		           AND m1.production_id IS NOT NULL) AS shared_productions
		FROM player_relationships r
		JOIN player_profile_workbooks w ON w.user_id = r.subject_user_id
		LEFT JOIN player_stage_name_history s ON s.user_id = r.subject_user_id AND s.ended_at IS NULL
		LEFT JOIN player_profile_facts pf ON pf.workbook_id = w.id AND pf.field_key = 'portrait_url'
		WHERE r.observer_user_id = $1 `+where+`
		ORDER BY r.updated_at DESC
	`, observerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []ListItem{}
	for rows.Next() {
		var item ListItem
		if err := rows.Scan(
			&item.ID, &item.PrivateNickname, &item.RelationshipState,
			&item.TrustLevel, &item.ClosenessLevel, &item.ReliabilityLevel, &item.CommunicationEaseLevel,
			&item.ArchivedAt, &item.CreatedAt, &item.UpdatedAt,
			&item.Subject.SubjectProfileID, &item.Subject.StageName, &item.Subject.PortraitURL,
			&item.LastJournalAt, &item.OpenFollowupCount, &item.SharedProductionCount,
		); err != nil {
			return nil, err
		}
		item.Archived = item.ArchivedAt != nil
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range items {
		cats, err := loadCategories(ctx, pool, items[i].ID)
		if err != nil {
			return nil, err
		}
		items[i].Categories = cats
	}

	return items, nil
}

// buildListItem loads the My People row shape for a single owned
// relationship, reusing ListRelationships so list and detail never drift.
func buildListItem(ctx context.Context, pool *pgxpool.Pool, observerUserID, relationshipID string) (ListItem, error) {
	items, err := ListRelationships(ctx, pool, observerUserID, FilterAll)
	if err != nil {
		return ListItem{}, err
	}
	for _, item := range items {
		if item.ID == relationshipID {
			return item, nil
		}
	}
	return ListItem{}, errors.New("relationship_not_found")
}

// ProjectRelationshipDetail assembles the full owner-only detail view
// (Kernel 62 §8.3): list row + facts + event history + verified shared
// context. Nothing here is ever served to anyone but the observer.
func ProjectRelationshipDetail(ctx context.Context, pool *pgxpool.Pool, observerUserID, relationshipID string) (Detail, error) {
	rel, err := loadRelationshipOwned(ctx, pool, observerUserID, relationshipID)
	if err != nil {
		return Detail{}, err
	}

	item, err := buildListItem(ctx, pool, observerUserID, relationshipID)
	if err != nil {
		return Detail{}, err
	}

	facts, err := loadCurrentFacts(ctx, pool, relationshipID)
	if err != nil {
		return Detail{}, err
	}

	events, err := loadRelationshipEvents(ctx, pool, relationshipID)
	if err != nil {
		return Detail{}, err
	}
	eventsDesc := append([]RelationshipEvent(nil), events...)
	for i, j := 0, len(eventsDesc)-1; i < j; i, j = i+1, j-1 {
		eventsDesc[i], eventsDesc[j] = eventsDesc[j], eventsDesc[i]
	}

	shared, err := ProjectSharedContext(ctx, pool, rel.ObserverUserID, rel.SubjectUserID)
	if err != nil {
		return Detail{}, err
	}

	return Detail{
		ListItem:      item,
		Facts:         facts,
		Events:        eventsDesc,
		SharedContext: shared,
	}, nil
}
