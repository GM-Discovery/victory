package playerrelationships

import (
	"context"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// membershipRow is one active production membership for either side of the
// relationship, used by the pure fold below.
type membershipRow struct {
	ProductionID   string
	ProductionName string
	UserID         string
	Role           string
	CreatedAt      time.Time
}

// ProjectSharedContext derives the read-only "Victory can currently verify"
// panel from reliable server-known data only: active production memberships
// both users hold (Kernel 62 §5.7, §13.1). It never infers trust, closeness,
// attendance, or meaning, and is safe (empty) when nothing is shared
// (Kernel 62 §9.5, §13.2).
func ProjectSharedContext(ctx context.Context, pool *pgxpool.Pool, observerUserID, subjectUserID string) (SharedContext, error) {
	rows, err := pool.Query(ctx, `
		SELECT m.production_id::text, p.name, m.user_id::text, m.role::text, m.created_at
		FROM memberships m
		JOIN productions p ON p.id = m.production_id
		WHERE m.active AND m.production_id IS NOT NULL AND m.user_id IN ($1, $2)
	`, observerUserID, subjectUserID)
	if err != nil {
		return SharedContext{}, err
	}
	defer rows.Close()

	var memberships []membershipRow
	for rows.Next() {
		var m membershipRow
		if err := rows.Scan(&m.ProductionID, &m.ProductionName, &m.UserID, &m.Role, &m.CreatedAt); err != nil {
			return SharedContext{}, err
		}
		memberships = append(memberships, m)
	}
	if err := rows.Err(); err != nil {
		return SharedContext{}, err
	}

	return SharedContext{
		Productions: FoldSharedProductions(memberships, observerUserID, subjectUserID),
	}, nil
}

// FoldSharedProductions keeps only productions where BOTH users hold active
// memberships, collecting each side's roles and the moment the overlap began
// (the later of the two sides' earliest memberships). Pure so it can be
// unit-tested without a database.
func FoldSharedProductions(memberships []membershipRow, observerUserID, subjectUserID string) []SharedProduction {
	type side struct {
		roles    map[string]bool
		earliest time.Time
		present  bool
	}
	type entry struct {
		name     string
		observer side
		subject  side
	}

	byProduction := map[string]*entry{}
	for _, m := range memberships {
		e, ok := byProduction[m.ProductionID]
		if !ok {
			e = &entry{
				name:     m.ProductionName,
				observer: side{roles: map[string]bool{}},
				subject:  side{roles: map[string]bool{}},
			}
			byProduction[m.ProductionID] = e
		}

		var s *side
		switch m.UserID {
		case observerUserID:
			s = &e.observer
		case subjectUserID:
			s = &e.subject
		default:
			continue
		}
		s.roles[m.Role] = true
		if !s.present || m.CreatedAt.Before(s.earliest) {
			s.earliest = m.CreatedAt
		}
		s.present = true
	}

	out := []SharedProduction{}
	for _, e := range byProduction {
		if !e.observer.present || !e.subject.present {
			continue
		}
		since := e.observer.earliest
		if e.subject.earliest.After(since) {
			since = e.subject.earliest
		}
		out = append(out, SharedProduction{
			ProductionName: e.name,
			ObserverRoles:  sortedKeys(e.observer.roles),
			SubjectRoles:   sortedKeys(e.subject.roles),
			Since:          since,
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ProductionName < out[j].ProductionName })
	return out
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
