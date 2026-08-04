package ewrite

// HTTP handlers for the Writer's Room (/api/ewrite/*). Library reading
// routes live in library_http.go. Envelope, auth, and error conventions
// copied from showruns/http.go. All routes are registered method-prefixed
// in main.go (kernel 77 CSRF posture: no bare routes, no mutating GETs).

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
)

type response struct {
	Ok   bool `json:"ok"`
	Data any  `json:"data,omitempty"`
}

// maxSourceBytes caps a single publication's Markdown source. The global
// body cap is 4MiB; 2MiB of Markdown is ~6x the full Socio rulebook.
const maxSourceBytes = 2 * 1024 * 1024

func context30(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 30*time.Second)
}

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func requireAuthenticatedUser(ctx context.Context, pool *pgxpool.Pool, r *http.Request) (string, error) {
	sessionCookie := ""
	if c, err := r.Cookie("victory_session"); err == nil {
		sessionCookie = c.Value
	}
	userID, err := access.CurrentUserIDFromRequest(ctx, pool, sessionCookie)
	if err != nil || strings.TrimSpace(userID) == "" {
		return "", errors.New("not_authenticated")
	}
	return userID, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, response{Ok: true, Data: data})
}

func writeError(w http.ResponseWriter, err error) {
	code := err.Error()
	status := http.StatusBadRequest
	switch {
	case code == "not_authenticated":
		status = http.StatusUnauthorized
	case code == "not_authorized":
		status = http.StatusForbidden
	case strings.HasSuffix(code, "_not_found"):
		status = http.StatusNotFound
	case code == "revision_conflict":
		status = http.StatusConflict
	}
	writeJSON(w, status, response{Ok: false, Data: map[string]any{"error": code}})
}

// HandleTree serves GET /api/ewrite/tree -- the Writer's Room bootstrap
// and access probe.
func HandleTree(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		tree, err := LoadAuthoringTree(ctx, pool, userID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, tree)
	}
}

// HandleCreateCollection serves POST /api/ewrite/collections.
func HandleCreateCollection(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		var body struct {
			LocationID string `json:"location_id"`
			ParentID   string `json:"parent_id"`
			Kind       string `json:"kind"`
			Title      string `json:"title"`
			Summary    string `json:"summary"`
			Visibility string `json:"visibility"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		c, err := CreateCollection(ctx, pool, userID, body.LocationID, body.ParentID, body.Kind, body.Title, body.Summary, body.Visibility)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"collection": c})
	}
}

// HandleCollectionItem serves PATCH and DELETE /api/ewrite/collections/{collection_id}.
func HandleCollectionItem(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		id := strings.TrimSpace(r.PathValue("collection_id"))
		switch r.Method {
		case http.MethodPatch:
			var fields map[string]any
			if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
				writeError(w, errors.New("invalid_json"))
				return
			}
			c, err := PatchCollection(ctx, pool, userID, id, fields)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"collection": c})
		case http.MethodDelete:
			if err := DeleteCollection(ctx, pool, userID, id); err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"deleted": true})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
		}
	}
}

// HandleCreatePublication serves POST /api/ewrite/publications.
func HandleCreatePublication(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		var body struct {
			CollectionID string `json:"collection_id"`
			Title        string `json:"title"`
			Summary      string `json:"summary"`
			Visibility   string `json:"visibility"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		p, err := CreatePublication(ctx, pool, userID, body.CollectionID, body.Title, body.Summary, body.Visibility)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"publication": p})
	}
}

// HandlePublicationItem serves GET/PATCH/DELETE /api/ewrite/publications/{publication_id}.
// GET is the authoring view: full source, sections, editors, links.
func HandlePublicationItem(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		id := strings.TrimSpace(r.PathValue("publication_id"))
		switch r.Method {
		case http.MethodGet:
			p, err := LoadPublication(ctx, pool, id)
			if err != nil {
				writeError(w, err)
				return
			}
			if ok, err := CanEditPublication(ctx, pool, userID, p); err != nil {
				writeError(w, err)
				return
			} else if !ok {
				writeError(w, errors.New("not_authorized"))
				return
			}
			sections, err := LoadSections(ctx, pool, id)
			if err != nil {
				writeError(w, err)
				return
			}
			editors, err := ListEditors(ctx, pool, id)
			if err != nil {
				writeError(w, err)
				return
			}
			links, err := ListObjectLinksForPublication(ctx, pool, id)
			if err != nil {
				writeError(w, err)
				return
			}
			canPublish, err := CanPublishPublication(ctx, pool, userID, p)
			if err != nil {
				writeError(w, err)
				return
			}
			currentNumber := 0
			if p.CurrentRevisionID != "" {
				if rev, err := LoadRevision(ctx, pool, p.CurrentRevisionID); err == nil {
					currentNumber = rev.RevisionNumber
				}
			}
			writeOK(w, map[string]any{
				"publication":             p,
				"sections":                sections,
				"editors":                 editors,
				"object_links":            links,
				"can_publish":             canPublish,
				"current_revision_number": currentNumber,
			})
		case http.MethodPatch:
			var fields map[string]any
			if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
				writeError(w, errors.New("invalid_json"))
				return
			}
			p, err := PatchPublication(ctx, pool, userID, id, fields)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"publication": p})
		case http.MethodDelete:
			if err := DeletePublication(ctx, pool, userID, id); err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"deleted": true})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
		}
	}
}

// HandleSaveSource serves PUT /api/ewrite/publications/{publication_id}/source.
func HandleSaveSource(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxSourceBytes)
		var body struct {
			SourceMarkdown string `json:"source_markdown"`
			BaseRevisionID string `json:"base_revision_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		id := strings.TrimSpace(r.PathValue("publication_id"))
		result, conflict, err := SavePublicationSource(ctx, pool, userID, id, body.SourceMarkdown, body.BaseRevisionID)
		if errors.Is(err, ErrRevisionConflict) {
			writeJSON(w, http.StatusConflict, response{Ok: false, Data: map[string]any{
				"error":    "revision_conflict",
				"conflict": conflict,
			}})
			return
		}
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, result)
	}
}

// HandlePublish serves POST /api/ewrite/publications/{publication_id}/publish.
func HandlePublish(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		p, err := PublishPublication(ctx, pool, userID, strings.TrimSpace(r.PathValue("publication_id")))
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"publication": p})
	}
}

// HandleUnpublish serves POST /api/ewrite/publications/{publication_id}/unpublish.
func HandleUnpublish(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		p, err := UnpublishPublication(ctx, pool, userID, strings.TrimSpace(r.PathValue("publication_id")))
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"publication": p})
	}
}

// HandleRevisionList serves GET /api/ewrite/publications/{publication_id}/revisions.
func HandleRevisionList(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		id := strings.TrimSpace(r.PathValue("publication_id"))
		p, err := LoadPublication(ctx, pool, id)
		if err != nil {
			writeError(w, err)
			return
		}
		if ok, err := CanEditPublication(ctx, pool, userID, p); err != nil {
			writeError(w, err)
			return
		} else if !ok {
			writeError(w, errors.New("not_authorized"))
			return
		}
		revs, err := ListRevisions(ctx, pool, id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"revisions": revs})
	}
}

// HandleRevisionItem serves GET /api/ewrite/revisions/{revision_id}.
func HandleRevisionItem(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		rev, err := LoadRevision(ctx, pool, strings.TrimSpace(r.PathValue("revision_id")))
		if err != nil {
			writeError(w, err)
			return
		}
		p, err := LoadPublication(ctx, pool, rev.PublicationID)
		if err != nil {
			writeError(w, err)
			return
		}
		if ok, err := CanEditPublication(ctx, pool, userID, p); err != nil {
			writeError(w, err)
			return
		} else if !ok {
			writeError(w, errors.New("not_authorized"))
			return
		}
		writeOK(w, map[string]any{"revision": rev})
	}
}

// HandlePreview serves POST /api/ewrite/preview: unsaved source in,
// sanitized HTML + outline + report out. Requires an authoring role
// somewhere -- previews are not an anonymous rendering service.
func HandlePreview(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		ids, _, err := authoringLocationIDs(ctx, pool, userID)
		if err != nil {
			writeError(w, err)
			return
		}
		if len(ids) == 0 {
			writeError(w, errors.New("not_authorized"))
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxSourceBytes)
		var body struct {
			SourceMarkdown string `json:"source_markdown"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		res, err := Render(body.SourceMarkdown)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{
			"html":    res.HTML,
			"report":  buildImportReport(body.SourceMarkdown, res),
			"outline": res.Outline,
		})
	}
}

// HandleEditors serves GET and POST /api/ewrite/publications/{publication_id}/editors.
func HandleEditors(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		id := strings.TrimSpace(r.PathValue("publication_id"))
		switch r.Method {
		case http.MethodGet:
			p, err := LoadPublication(ctx, pool, id)
			if err != nil {
				writeError(w, err)
				return
			}
			if ok, err := CanEditPublication(ctx, pool, userID, p); err != nil {
				writeError(w, err)
				return
			} else if !ok {
				writeError(w, errors.New("not_authorized"))
				return
			}
			editors, err := ListEditors(ctx, pool, id)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"editors": editors})
		case http.MethodPost:
			var body struct {
				UserHandle string `json:"user_handle"`
				GrantKind  string `json:"grant_kind"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_json"))
				return
			}
			g, err := AddEditor(ctx, pool, userID, id, body.UserHandle, body.GrantKind)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"editor": g})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
		}
	}
}

// HandleEditorItem serves DELETE /api/ewrite/editors/{grant_id}.
func HandleEditorItem(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		if err := RemoveEditor(ctx, pool, userID, strings.TrimSpace(r.PathValue("grant_id"))); err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"deleted": true})
	}
}
