package shows

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/showruns"
)

// ListInternalRosterForShow and ListAudienceProgramForShow are thin
// pass-throughs to the parent Show Run's roster -- a Show has no roster
// table of its own this kernel; it inherits its Show Run's roster and
// authority wholesale (Kernel 67 §1.7).
func ListInternalRosterForShow(ctx context.Context, pool *pgxpool.Pool, show Show) ([]showruns.RosterMember, error) {
	return showruns.ListInternalRoster(ctx, pool, show.ShowRunID)
}

func ListAudienceProgramForShow(ctx context.Context, pool *pgxpool.Pool, show Show) ([]showruns.RosterMember, error) {
	return showruns.ListAudienceProgramMembers(ctx, pool, show.ShowRunID)
}
