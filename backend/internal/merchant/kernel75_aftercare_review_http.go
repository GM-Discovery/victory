package merchant

// Directors+ review surface for Aftercare (kernel-75 S9).
//
// READ-ONLY. There is no POST, PATCH, or DELETE here, and that absence is
// the feature: S1.15 and S9.3 forbid replies, annotations, scores, comment
// threads, and Director edits to Player responses. A Director who wants to
// respond to a Player's reflection does it as a person, not through this
// table.

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/aftercare"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
	"victory/backend/internal/tutorial"
)

// resolveShowAuthorityContext loads the Show, its Run, and the Location the
// authority helpers need. Mirrors HandleShowTutorialProgress's three lines
// exactly rather than inventing a second lookup.
func resolveShowAuthorityContext(ctx context.Context, pool *pgxpool.Pool, showID string) (shows.Show, showruns.ShowRun, error) {
	show, err := shows.LoadShowByID(ctx, pool, showID)
	if err != nil {
		return shows.Show{}, showruns.ShowRun{}, err
	}
	run, err := showruns.LoadShowRunByID(ctx, pool, show.ShowRunID)
	if err != nil {
		return shows.Show{}, showruns.ShowRun{}, err
	}
	return show, run, nil
}

// attachTutorialStatus fills in each row's human tutorial phrase from the
// existing Kernel 74 progress list, so the Director's Chair reports one
// consistent vocabulary rather than a second one invented here.
func attachTutorialStatus(ctx context.Context, pool *pgxpool.Pool, showID string, rows []aftercare.ReviewRow) error {
	progress, err := tutorial.ListShowProgress(ctx, pool, showID)
	if err != nil {
		return err
	}
	byCharacter := map[string]string{}
	for _, p := range progress {
		byCharacter[p.CharacterCardID] = p.Status
	}
	for i := range rows {
		rows[i].TutorialStatus = byCharacter[rows[i].CharacterCardID]
	}
	return nil
}

// HandleShowAftercareReview handles GET
// /api/shows/{show_id}/aftercare-review.
//
// Gate: showruns.CanViewBackstage -- Director, Producer, Operator, or active
// Crew at that Location. Crew may see who finished and who reflected,
// because that is backstage awareness.
func HandleShowAftercareReview(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showID := strings.TrimSpace(r.PathValue("show_id"))
		show, run, err := resolveShowAuthorityContext(ctx, pool, showID)
		if err != nil {
			writeError(w, err)
			return
		}
		allowed, err := showruns.CanViewBackstage(ctx, pool, userID, run.LocationID)
		if err != nil {
			writeError(w, err)
			return
		}
		if !allowed {
			writeError(w, errors.New("not_authorized"))
			return
		}

		rows, err := aftercare.ListForShow(ctx, pool, showID)
		if err != nil {
			writeError(w, err)
			return
		}
		if err := attachTutorialStatus(ctx, pool, showID, rows); err != nil {
			writeError(w, err)
			return
		}

		// can_export is computed server-side so the UI can hide a control
		// the caller cannot use. It is a courtesy, NOT the gate -- the
		// export endpoint runs its own stricter check.
		canExport, err := showruns.CanManageShowRun(ctx, pool, userID, run.LocationID)
		if err != nil {
			writeError(w, err)
			return
		}

		writeOK(w, map[string]any{
			"rows":       rows,
			"prompts":    aftercare.PromptSetV1,
			"can_export": canExport,
			"show": map[string]any{
				"id": show.ID, "title": show.Title, "short_code": show.ShortCode,
			},
			"show_run": map[string]any{"id": run.ID, "title": run.Title},
		})
	}
}

// HandleShowAftercareReviewCSV handles GET
// /api/shows/{show_id}/aftercare-review.csv.
//
// Gate: showruns.CanManageShowRun -- STRICTER than the read above, and
// deliberately so. Reading the table is backstage visibility; exporting
// takes Player-written reflection off the platform as a file that can be
// mailed, shared, or lost, and that is a Producer/Director act rather than
// a Crew one. showruns/authority.go's own comment warns against collapsing
// CanViewBackstage and CanManageShowRun; this is a case where the
// distinction earns its keep.
//
// RESPONSE-SHAPE NOTE: this is the repo's only non-JSON endpoint. Authority
// is resolved and every row buffered before a single byte is written, so any
// failure still falls back to the ordinary {ok:false} JSON envelope.
func HandleShowAftercareReviewCSV(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		showID := strings.TrimSpace(r.PathValue("show_id"))
		show, run, err := resolveShowAuthorityContext(ctx, pool, showID)
		if err != nil {
			writeError(w, err)
			return
		}
		allowed, err := showruns.CanManageShowRun(ctx, pool, userID, run.LocationID)
		if err != nil {
			writeError(w, err)
			return
		}
		if !allowed {
			writeError(w, errors.New("not_authorized"))
			return
		}

		rows, err := aftercare.ListForShow(ctx, pool, showID)
		if err != nil {
			writeError(w, err)
			return
		}
		if err := attachTutorialStatus(ctx, pool, showID, rows); err != nil {
			writeError(w, err)
			return
		}

		// Render into memory first. Everything that could fail has now
		// failed or succeeded, so the headers below are safe to commit to.
		var buf strings.Builder
		if err := aftercare.WriteCSV(&buf, run.Title, show.Title, rows); err != nil {
			writeError(w, err)
			return
		}

		filename := aftercare.ExportFilename(show.ShortCode, time.Now())
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(buf.String()))
	}
}
