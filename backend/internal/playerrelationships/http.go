package playerrelationships

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

// writeError maps error codes to status. Non-owner access to a relationship
// always surfaces as *_not_found -> 404, never 403, so the existence of a
// private record is not confirmed to anyone but its observer
// (Kernel 62 §12.1).
func writeError(w http.ResponseWriter, err error) {
	code := err.Error()
	status := http.StatusBadRequest
	switch {
	case code == "not_authenticated":
		status = http.StatusUnauthorized
	case code == "forbidden":
		status = http.StatusForbidden
	case strings.HasSuffix(code, "_not_found") || code == "unknown_page":
		status = http.StatusNotFound
	}
	writeJSON(w, status, response{Ok: false, Data: map[string]any{"error": code}})
}

func notFound(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, response{Ok: false})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
}

// HandleCatalogue serves the versioned relationship-workbook catalogue plus
// the fixed qualitative vocabularies and category set that drive both
// backend validation and frontend rendering (Kernel 62 §6, §7).
func HandleCatalogue(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if _, err := requireAuthenticatedUser(ctx, pool, r); err != nil {
			writeError(w, err)
			return
		}

		cat, err := LoadCatalogue()
		if err != nil {
			writeError(w, err)
			return
		}

		categoryTerms := make([]VocabTerm, 0, len(CategoryKeys))
		for _, key := range CategoryKeys {
			categoryTerms = append(categoryTerms, VocabTerm{Key: key, Label: CategoryLabels[key]})
		}

		writeOK(w, map[string]any{
			"catalogue": cat,
			"vocabularies": map[string][]VocabTerm{
				"trust_level":              TrustVocab,
				"closeness_level":          ClosenessVocab,
				"reliability_level":        ReliabilityVocab,
				"communication_ease_level": CommunicationEaseVocab,
				"relationship_state":       RelationshipStateVocab,
			},
			"categories": categoryTerms,
		})
	}
}

type createRelationshipRequest struct {
	SubjectProfileID string `json:"subject_profile_id"`
}

// HandleCollection handles the exact path /api/player-relationships:
// GET lists the observer's own relationships (optionally looked up by
// subject profile ID for the Trailer button); POST creates one from a
// subject's opaque profile ID (Kernel 62 §9.1, §10). Any client-supplied
// observer identity is ignored -- the observer is always the session user.
func HandleCollection(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		observerUserID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}

		switch r.Method {
		case http.MethodGet:
			if subjectProfileID := strings.TrimSpace(r.URL.Query().Get("subject_profile_id")); subjectProfileID != "" {
				rel, err := GetRelationshipBySubjectProfile(ctx, pool, observerUserID, subjectProfileID)
				if err != nil {
					if err.Error() == "relationship_not_found" {
						writeOK(w, map[string]any{"relationship": nil})
						return
					}
					writeError(w, err)
					return
				}
				item, err := buildListItem(ctx, pool, observerUserID, rel.ID)
				if err != nil {
					writeError(w, err)
					return
				}
				writeOK(w, map[string]any{"relationship": item})
				return
			}

			filter := ListFilter(strings.TrimSpace(r.URL.Query().Get("state")))
			if filter != FilterArchived && filter != FilterAll {
				filter = FilterActive
			}
			items, err := ListRelationships(ctx, pool, observerUserID, filter)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"relationships": items, "state": string(filter)})

		case http.MethodPost:
			var input createRelationshipRequest
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				writeError(w, errors.New("invalid_json"))
				return
			}
			rel, created, err := EnsureRelationship(ctx, pool, observerUserID, input.SubjectProfileID)
			if err != nil {
				writeError(w, err)
				return
			}
			item, err := buildListItem(ctx, pool, observerUserID, rel.ID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"relationship": item, "created": created})

		default:
			methodNotAllowed(w)
		}
	}
}

// HandleByID handles every /api/player-relationships/{id}... route
// (Kernel 62 §10). Ownership is proven inside each operation via
// loadRelationshipOwned; a non-owner receives the same 404 as a
// nonexistent ID.
func HandleByID(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		observerUserID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}

		rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/player-relationships/"), "/")
		segments := []string{}
		if rest != "" {
			segments = strings.Split(rest, "/")
		}
		if len(segments) == 0 {
			notFound(w)
			return
		}
		relationshipID := segments[0]
		tail := segments[1:]

		switch {
		case len(tail) == 0:
			handleRelationshipRoot(ctx, w, r, pool, observerUserID, relationshipID)
		case len(tail) == 1 && tail[0] == "archive" && r.Method == http.MethodPost:
			respondAfter(w, ArchiveRelationship(ctx, pool, observerUserID, relationshipID), map[string]any{"archived": true})
		case len(tail) == 1 && tail[0] == "unarchive" && r.Method == http.MethodPost:
			respondAfter(w, UnarchiveRelationship(ctx, pool, observerUserID, relationshipID), map[string]any{"archived": false})
		case len(tail) == 2 && tail[0] == "pages" && r.Method == http.MethodPost:
			handlePageCommit(ctx, w, r, pool, observerUserID, relationshipID, tail[1])
		case len(tail) == 2 && tail[0] == "events" && r.Method == http.MethodDelete:
			respondAfter(w, DeleteRelationshipEvent(ctx, pool, observerUserID, relationshipID, tail[1]), map[string]any{"deleted": true})
		case len(tail) >= 1 && tail[0] == "journal":
			handleJournal(ctx, w, r, pool, observerUserID, relationshipID, tail[1:])
		case len(tail) >= 1 && tail[0] == "followups":
			handleFollowUps(ctx, w, r, pool, observerUserID, relationshipID, tail[1:])
		default:
			notFound(w)
		}
	}
}

func respondAfter(w http.ResponseWriter, err error, payload any) {
	if err != nil {
		writeError(w, err)
		return
	}
	writeOK(w, payload)
}

func handleRelationshipRoot(ctx context.Context, w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool, observerUserID, relationshipID string) {
	switch r.Method {
	case http.MethodGet:
		detail, err := ProjectRelationshipDetail(ctx, pool, observerUserID, relationshipID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, detail)

	case http.MethodPatch:
		var input UpdateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		if _, err := UpdateRelationship(ctx, pool, observerUserID, relationshipID, input); err != nil {
			writeError(w, err)
			return
		}
		item, err := buildListItem(ctx, pool, observerUserID, relationshipID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"relationship": item})

	default:
		methodNotAllowed(w)
	}
}

type pageCommitRequest struct {
	Answers map[string]any `json:"answers"`
}

func handlePageCommit(ctx context.Context, w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool, observerUserID, relationshipID, pageKey string) {
	var input pageCommitRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, errors.New("invalid_json"))
		return
	}
	event, changed, err := CommitRelationshipPage(ctx, pool, observerUserID, relationshipID, pageKey, input.Answers)
	if err != nil {
		writeError(w, err)
		return
	}
	writeOK(w, map[string]any{"changed": changed, "event": event})
}

func handleJournal(ctx context.Context, w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool, observerUserID, relationshipID string, tail []string) {
	switch {
	case len(tail) == 0 && r.Method == http.MethodGet:
		entries, err := ListJournalEntries(ctx, pool, observerUserID, relationshipID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"entries": entries})

	case len(tail) == 0 && r.Method == http.MethodPost:
		var input JournalInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		entry, err := CreateJournalEntry(ctx, pool, observerUserID, relationshipID, input)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"entry": entry})

	case len(tail) == 1 && r.Method == http.MethodPatch:
		var input JournalInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		entry, err := UpdateJournalEntry(ctx, pool, observerUserID, relationshipID, tail[0], input)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"entry": entry})

	case len(tail) == 1 && r.Method == http.MethodDelete:
		respondAfter(w, DeleteJournalEntry(ctx, pool, observerUserID, relationshipID, tail[0]), map[string]any{"deleted": true})

	default:
		notFound(w)
	}
}

func handleFollowUps(ctx context.Context, w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool, observerUserID, relationshipID string, tail []string) {
	switch {
	case len(tail) == 0 && r.Method == http.MethodGet:
		items, err := ListFollowUps(ctx, pool, observerUserID, relationshipID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"followups": items})

	case len(tail) == 0 && r.Method == http.MethodPost:
		var input FollowUpInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		item, err := CreateFollowUp(ctx, pool, observerUserID, relationshipID, input)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"followup": item})

	case len(tail) == 1 && r.Method == http.MethodPatch:
		var input FollowUpInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, errors.New("invalid_json"))
			return
		}
		item, err := UpdateFollowUp(ctx, pool, observerUserID, relationshipID, tail[0], input)
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"followup": item})

	case len(tail) == 2 && r.Method == http.MethodPost:
		var item FollowUp
		var err error
		switch tail[1] {
		case "complete":
			item, err = CompleteFollowUp(ctx, pool, observerUserID, relationshipID, tail[0])
		case "dismiss":
			item, err = DismissFollowUp(ctx, pool, observerUserID, relationshipID, tail[0])
		case "reopen":
			item, err = ReopenFollowUp(ctx, pool, observerUserID, relationshipID, tail[0])
		default:
			notFound(w)
			return
		}
		if err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"followup": item})

	default:
		notFound(w)
	}
}
