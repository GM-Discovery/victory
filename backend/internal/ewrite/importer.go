package ewrite

// Markdown import (kernel spec 7): pasted source or an uploaded .md file,
// through the same SavePublicationSource path as every other save -- import
// is not a second write path, it is a save with a report. UTF-8 is
// enforced, CRLF is normalized, and the original (normalized) source is
// what gets stored: export returns exactly what import accepted (spec 7.3).

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
)

// maxImportBytes caps one imported manuscript. Inside the global 4MiB body
// cap; ~6x the full Socio v1.1 rulebook.
const maxImportBytes = 2 * 1024 * 1024

func normalizeSource(raw []byte) (string, error) {
	if !utf8.Valid(raw) {
		return "", errors.New("source_not_utf8")
	}
	s := strings.ReplaceAll(string(raw), "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	// Strip a UTF-8 BOM if present -- Google Docs exports sometimes carry
	// one, and it would otherwise hide the first heading marker.
	s = strings.TrimPrefix(s, "\ufeff")
	return s, nil
}

// HandleImport serves POST /api/ewrite/publications/{publication_id}/import.
// multipart/form-data: fields "file" (.md upload) or "source" (paste), plus
// "base_revision_id". Also accepts a JSON body {source_markdown,
// base_revision_id} for paste-only clients.
func HandleImport(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context30(r)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		publicationID := strings.TrimSpace(r.PathValue("publication_id"))
		r.Body = http.MaxBytesReader(w, r.Body, maxImportBytes+64*1024)

		var raw []byte
		var filename, baseRevisionID string

		contentType := r.Header.Get("Content-Type")
		if strings.HasPrefix(contentType, "multipart/form-data") {
			if err := r.ParseMultipartForm(maxImportBytes); err != nil {
				writeError(w, errors.New("import_too_large"))
				return
			}
			baseRevisionID = r.FormValue("base_revision_id")
			if file, header, err := r.FormFile("file"); err == nil {
				defer file.Close()
				if !strings.HasSuffix(strings.ToLower(header.Filename), ".md") &&
					!strings.HasSuffix(strings.ToLower(header.Filename), ".markdown") &&
					!strings.HasSuffix(strings.ToLower(header.Filename), ".txt") {
					writeError(w, errors.New("unsupported_file_type"))
					return
				}
				filename = header.Filename
				raw, err = io.ReadAll(io.LimitReader(file, maxImportBytes+1))
				if err != nil {
					writeError(w, errors.New("import_read_failed"))
					return
				}
			} else {
				raw = []byte(r.FormValue("source"))
			}
		} else {
			var body struct {
				SourceMarkdown string `json:"source_markdown"`
				BaseRevisionID string `json:"base_revision_id"`
			}
			if err := decodeJSON(r, &body); err != nil {
				writeError(w, errors.New("invalid_json"))
				return
			}
			raw = []byte(body.SourceMarkdown)
			baseRevisionID = body.BaseRevisionID
		}

		if len(raw) > maxImportBytes {
			writeError(w, errors.New("import_too_large"))
			return
		}
		if len(raw) == 0 {
			writeError(w, errors.New("source_required"))
			return
		}

		source, err := normalizeSource(raw)
		if err != nil {
			writeError(w, err)
			return
		}

		start := time.Now()
		result, conflict, err := SavePublicationSource(ctx, pool, userID, publicationID, source, baseRevisionID)
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
		result.Report.SourceFilename = filename
		writeOK(w, map[string]any{
			"publication":     result.Publication,
			"revision_id":     result.RevisionID,
			"revision_number": result.RevisionNumber,
			"report":          result.Report,
			"import_ms":       time.Since(start).Milliseconds(),
		})
	}
}
