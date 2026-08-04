package ewrite

// Object links: existing Victory objects bound to exact rule sections
// (kernel Goal H). Typed-FK binding table (see migration 084's rationale);
// Kernel 78 ships object_type='equipment_item' and the shape for more.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RuleLink is the read-side projection attached to object payloads
// (merchant stock, character inventory): everything a client needs to
// build a Library deep link.
type RuleLink struct {
	PublicationID    string `json:"publication_id"`
	PublicationTitle string `json:"publication_title"`
	SectionAnchor    string `json:"section_anchor,omitempty"`
	SectionTitle     string `json:"section_title,omitempty"`
}

// ListObjectLinksForPublication lists a publication's outbound object
// bindings for the Writer's Room "Linked objects" panel.
func ListObjectLinksForPublication(ctx context.Context, pool queryer, publicationID string) ([]ObjectLink, error) {
	rows, err := pool.Query(ctx, `
		SELECT ol.id::text, ol.object_type, COALESCE(ol.equipment_item_id::text, ''), ol.publication_id::text,
		       COALESCE(ol.section_id::text, ''), COALESCE(s.anchor, ''), COALESCE(s.title, '')
		FROM ewrite_object_links ol
		LEFT JOIN ewrite_sections s ON s.id = ol.section_id
		WHERE ol.publication_id = $1
		ORDER BY ol.created_at
	`, publicationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ObjectLink{}
	for rows.Next() {
		var l ObjectLink
		if err := rows.Scan(&l.ID, &l.ObjectType, &l.EquipmentItemID, &l.PublicationID, &l.SectionID, &l.SectionAnchor, &l.SectionTitle); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// RuleLinksForEquipmentItems resolves rule links for a set of equipment
// items in one query. Only PUBLISHED targets resolve -- a draft rule must
// never leak its title into a player-facing payload (spec 6.3).
func RuleLinksForEquipmentItems(ctx context.Context, pool queryer, itemIDs []string) (map[string]RuleLink, error) {
	out := map[string]RuleLink{}
	if len(itemIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT ol.equipment_item_id::text, p.id::text, p.title, COALESCE(s.anchor, ''), COALESCE(s.title, '')
		FROM ewrite_object_links ol
		JOIN ewrite_publications p ON p.id = ol.publication_id
		LEFT JOIN ewrite_sections s ON s.id = ol.section_id
		WHERE ol.object_type = 'equipment_item'
		  AND ol.equipment_item_id = ANY($1)
		  AND p.status = 'published'
	`, itemIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var itemID string
		var l RuleLink
		if err := rows.Scan(&itemID, &l.PublicationID, &l.PublicationTitle, &l.SectionAnchor, &l.SectionTitle); err != nil {
			return nil, err
		}
		out[itemID] = l
	}
	return out, rows.Err()
}

// SetEquipmentItemRuleLink creates or replaces the rule link for one
// equipment item. Authority: edit authority on the publication AND Crew+
// at the equipment item's own location (both objects' scopes must
// consent -- an author must not annotate another production's items).
func SetEquipmentItemRuleLink(ctx context.Context, pool *pgxpool.Pool, userID, equipmentItemID, publicationID, sectionID string) (*ObjectLink, error) {
	p, err := LoadPublication(ctx, pool, strings.TrimSpace(publicationID))
	if err != nil {
		return nil, err
	}
	if ok, err := CanEditPublication(ctx, pool, userID, p); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("not_authorized")
	}

	var itemLocationID string
	err = pool.QueryRow(ctx, `SELECT location_id::text FROM equipment_items WHERE id = $1`, strings.TrimSpace(equipmentItemID)).Scan(&itemLocationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("equipment_item_not_found")
	}
	if err != nil {
		return nil, err
	}
	if ok, err := CanAuthorInScope(ctx, pool, userID, itemLocationID); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("not_authorized")
	}

	sectionID = strings.TrimSpace(sectionID)
	if sectionID != "" {
		var sectionPub string
		err := pool.QueryRow(ctx, `SELECT publication_id::text FROM ewrite_sections WHERE id = $1`, sectionID).Scan(&sectionPub)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("section_not_found")
		}
		if err != nil {
			return nil, err
		}
		if sectionPub != p.ID {
			return nil, errors.New("section_publication_mismatch")
		}
	}

	var l ObjectLink
	err = pool.QueryRow(ctx, `
		INSERT INTO ewrite_object_links (object_type, equipment_item_id, publication_id, section_id, created_by)
		VALUES ('equipment_item', $1, $2, NULLIF($3, '')::uuid, $4)
		ON CONFLICT (equipment_item_id, object_type) DO UPDATE
			SET publication_id = EXCLUDED.publication_id,
			    section_id = EXCLUDED.section_id,
			    updated_at = NOW()
		RETURNING id::text, COALESCE(section_id::text, '')
	`, equipmentItemID, p.ID, sectionID, userID).Scan(&l.ID, &l.SectionID)
	if err != nil {
		return nil, err
	}
	l.ObjectType = "equipment_item"
	l.EquipmentItemID = equipmentItemID
	l.PublicationID = p.ID
	return &l, nil
}

// RemoveObjectLink deletes a binding by id; edit authority on the target
// publication is the gate.
func RemoveObjectLink(ctx context.Context, pool *pgxpool.Pool, userID, linkID string) error {
	var publicationID string
	err := pool.QueryRow(ctx, `SELECT publication_id::text FROM ewrite_object_links WHERE id = $1`, linkID).Scan(&publicationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("link_not_found")
	}
	if err != nil {
		return err
	}
	p, err := LoadPublication(ctx, pool, publicationID)
	if err != nil {
		return err
	}
	if ok, err := CanEditPublication(ctx, pool, userID, p); err != nil {
		return err
	} else if !ok {
		return errors.New("not_authorized")
	}
	_, err = pool.Exec(ctx, `DELETE FROM ewrite_object_links WHERE id = $1`, linkID)
	return err
}

// HandleObjectLinks serves POST /api/ewrite/object-links and
// GET /api/ewrite/object-links?object_type=equipment_item&object_id=...
func HandleObjectLinks(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		switch r.Method {
		case http.MethodPost:
			var body struct {
				ObjectType      string `json:"object_type"`
				EquipmentItemID string `json:"equipment_item_id"`
				PublicationID   string `json:"publication_id"`
				SectionID       string `json:"section_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_json"))
				return
			}
			if body.ObjectType != "equipment_item" {
				writeError(w, errors.New("unsupported_object_type"))
				return
			}
			l, err := SetEquipmentItemRuleLink(ctx, pool, userID, body.EquipmentItemID, body.PublicationID, body.SectionID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"link": l})
		case http.MethodGet:
			objectType := r.URL.Query().Get("object_type")
			objectID := strings.TrimSpace(r.URL.Query().Get("object_id"))
			if objectType != "equipment_item" || objectID == "" {
				writeError(w, errors.New("unsupported_object_type"))
				return
			}
			links, err := RuleLinksForEquipmentItems(ctx, pool, []string{objectID})
			if err != nil {
				writeError(w, err)
				return
			}
			writeOK(w, map[string]any{"links": links})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
		}
	}
}

// HandleObjectLinkItem serves DELETE /api/ewrite/object-links/{link_id}.
func HandleObjectLinkItem(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeError(w, err)
			return
		}
		if err := RemoveObjectLink(ctx, pool, userID, strings.TrimSpace(r.PathValue("link_id"))); err != nil {
			writeError(w, err)
			return
		}
		writeOK(w, map[string]any{"deleted": true})
	}
}
