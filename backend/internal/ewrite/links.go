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
	// ID is the ewrite_object_links row's own id -- present so a client
	// can DELETE /api/ewrite/object-links/{id} to remove the link it's
	// displaying. Empty for RuleLinksForCharacterSkills (Kernel 79A),
	// whose links live in ewrite_directory_entries, not this table --
	// removing a Skill Directory entry's target is a Crew+ curation
	// action, not something a Character-sheet reader can trigger.
	ID               string `json:"id,omitempty"`
	PublicationID    string `json:"publication_id"`
	PublicationTitle string `json:"publication_title"`
	SectionAnchor    string `json:"section_anchor,omitempty"`
	SectionTitle     string `json:"section_title,omitempty"`
}

// ListObjectLinksForPublication lists a publication's outbound object
// bindings for the Writer's Room "Linked objects" panel.
func ListObjectLinksForPublication(ctx context.Context, pool queryer, publicationID string) ([]ObjectLink, error) {
	rows, err := pool.Query(ctx, `
		SELECT ol.id::text, ol.object_type,
		       COALESCE(ol.equipment_item_id::text, ''), COALESCE(ol.cue_id::text, ''),
		       COALESCE(ol.index_card_element_id::text, ''), COALESCE(ol.scene_element_id::text, ''),
		       COALESCE(ol.dialogue_topic_id::text, ''),
		       ol.publication_id::text,
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
		if err := rows.Scan(&l.ID, &l.ObjectType, &l.EquipmentItemID, &l.CueID, &l.IndexCardElementID, &l.SceneElementID, &l.DialogueTopicID,
			&l.PublicationID, &l.SectionID, &l.SectionAnchor, &l.SectionTitle); err != nil {
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
		SELECT ol.id::text, ol.equipment_item_id::text, p.id::text, p.title, COALESCE(s.anchor, ''), COALESCE(s.title, '')
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
		if err := rows.Scan(&l.ID, &itemID, &l.PublicationID, &l.PublicationTitle, &l.SectionAnchor, &l.SectionTitle); err != nil {
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

// SetCueRuleLink creates or replaces the rule link for one Cue (spec 8.2).
// Authority: edit authority on the publication AND Crew+ at the Cue's own
// location, resolved through show_scene_placement -> show -> show_run
// (the only path from a Cue to a location; Cues have no location_id of
// their own).
func SetCueRuleLink(ctx context.Context, pool *pgxpool.Pool, userID, cueID, publicationID, sectionID string) (*ObjectLink, error) {
	p, err := LoadPublication(ctx, pool, strings.TrimSpace(publicationID))
	if err != nil {
		return nil, err
	}
	if ok, err := CanEditPublication(ctx, pool, userID, p); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("not_authorized")
	}

	var cueLocationID string
	err = pool.QueryRow(ctx, `
		SELECT sr.location_id::text
		FROM cues c
		JOIN show_scene_placements ssp ON ssp.id = c.show_scene_placement_id
		JOIN shows sh ON sh.id = ssp.show_id
		JOIN show_runs sr ON sr.id = sh.show_run_id
		WHERE c.id = $1
	`, strings.TrimSpace(cueID)).Scan(&cueLocationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("cue_not_found")
	}
	if err != nil {
		return nil, err
	}
	if ok, err := CanAuthorInScope(ctx, pool, userID, cueLocationID); err != nil {
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
		INSERT INTO ewrite_object_links (object_type, cue_id, publication_id, section_id, created_by)
		VALUES ('cue', $1, $2, NULLIF($3, '')::uuid, $4)
		ON CONFLICT (cue_id) WHERE object_type = 'cue' DO UPDATE
			SET publication_id = EXCLUDED.publication_id,
			    section_id = EXCLUDED.section_id,
			    updated_at = NOW()
		RETURNING id::text, COALESCE(section_id::text, '')
	`, cueID, p.ID, sectionID, userID).Scan(&l.ID, &l.SectionID)
	if err != nil {
		return nil, err
	}
	l.ObjectType = "cue"
	l.CueID = cueID
	l.PublicationID = p.ID
	return &l, nil
}

// RuleLinksForCues resolves rule links for a set of Cues in one query.
// Same shallow published-only filter as RuleLinksForEquipmentItems --
// deep-link metadata on an already-authority-gated payload.
func RuleLinksForCues(ctx context.Context, pool queryer, cueIDs []string) (map[string]RuleLink, error) {
	out := map[string]RuleLink{}
	if len(cueIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT ol.id::text, ol.cue_id::text, p.id::text, p.title, COALESCE(s.anchor, ''), COALESCE(s.title, '')
		FROM ewrite_object_links ol
		JOIN ewrite_publications p ON p.id = ol.publication_id
		LEFT JOIN ewrite_sections s ON s.id = ol.section_id
		WHERE ol.object_type = 'cue' AND ol.cue_id = ANY($1) AND p.status = 'published'
	`, cueIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var cueID string
		var l RuleLink
		if err := rows.Scan(&l.ID, &cueID, &l.PublicationID, &l.PublicationTitle, &l.SectionAnchor, &l.SectionTitle); err != nil {
			return nil, err
		}
		out[cueID] = l
	}
	return out, rows.Err()
}

// SetIndexCardRuleLink creates or replaces the rule link for one index
// card (spec 8.1). Authority: edit authority on the publication AND Crew+
// at the card's owning library's location (elements.library_id ->
// libraries.location_id -- the same scope resolveIndexCardLibrary uses to
// create the card in the first place).
func SetIndexCardRuleLink(ctx context.Context, pool *pgxpool.Pool, userID, elementID, publicationID, sectionID string) (*ObjectLink, error) {
	p, err := LoadPublication(ctx, pool, strings.TrimSpace(publicationID))
	if err != nil {
		return nil, err
	}
	if ok, err := CanEditPublication(ctx, pool, userID, p); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("not_authorized")
	}

	var cardLocationID string
	err = pool.QueryRow(ctx, `
		SELECT lib.location_id::text
		FROM elements e
		JOIN libraries lib ON lib.id = e.library_id
		WHERE e.id = $1 AND e.element_type = 'index_card'
	`, strings.TrimSpace(elementID)).Scan(&cardLocationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("index_card_not_found")
	}
	if err != nil {
		return nil, err
	}
	if ok, err := CanAuthorInScope(ctx, pool, userID, cardLocationID); err != nil {
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
		INSERT INTO ewrite_object_links (object_type, index_card_element_id, publication_id, section_id, created_by)
		VALUES ('index_card', $1, $2, NULLIF($3, '')::uuid, $4)
		ON CONFLICT (index_card_element_id) WHERE object_type = 'index_card' DO UPDATE
			SET publication_id = EXCLUDED.publication_id,
			    section_id = EXCLUDED.section_id,
			    updated_at = NOW()
		RETURNING id::text, COALESCE(section_id::text, '')
	`, elementID, p.ID, sectionID, userID).Scan(&l.ID, &l.SectionID)
	if err != nil {
		return nil, err
	}
	l.ObjectType = "index_card"
	l.IndexCardElementID = elementID
	l.PublicationID = p.ID
	return &l, nil
}

// RuleLinksForIndexCards resolves rule links for a set of index card
// element IDs in one query.
func RuleLinksForIndexCards(ctx context.Context, pool queryer, elementIDs []string) (map[string]RuleLink, error) {
	out := map[string]RuleLink{}
	if len(elementIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT ol.id::text, ol.index_card_element_id::text, p.id::text, p.title, COALESCE(s.anchor, ''), COALESCE(s.title, '')
		FROM ewrite_object_links ol
		JOIN ewrite_publications p ON p.id = ol.publication_id
		LEFT JOIN ewrite_sections s ON s.id = ol.section_id
		WHERE ol.object_type = 'index_card' AND ol.index_card_element_id = ANY($1) AND p.status = 'published'
	`, elementIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var elementID string
		var l RuleLink
		if err := rows.Scan(&l.ID, &elementID, &l.PublicationID, &l.PublicationTitle, &l.SectionAnchor, &l.SectionTitle); err != nil {
			return nil, err
		}
		out[elementID] = l
	}
	return out, rows.Err()
}

// SetSceneElementRuleLink creates or replaces the rule link for one Scene
// stage element (spec 8.3). Authority: edit authority on the publication
// AND Crew+ at the element's own Scene's location (scene_stage_elements.
// scene_id -> scenes.location_id, Kernel 70's ownership-scope column).
func SetSceneElementRuleLink(ctx context.Context, pool *pgxpool.Pool, userID, sceneElementID, publicationID, sectionID string) (*ObjectLink, error) {
	p, err := LoadPublication(ctx, pool, strings.TrimSpace(publicationID))
	if err != nil {
		return nil, err
	}
	if ok, err := CanEditPublication(ctx, pool, userID, p); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("not_authorized")
	}

	var elementLocationID string
	err = pool.QueryRow(ctx, `
		SELECT sc.location_id::text
		FROM scene_stage_elements sse
		JOIN scenes sc ON sc.id = sse.scene_id
		WHERE sse.id = $1
	`, strings.TrimSpace(sceneElementID)).Scan(&elementLocationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("scene_element_not_found")
	}
	if err != nil {
		return nil, err
	}
	if ok, err := CanAuthorInScope(ctx, pool, userID, elementLocationID); err != nil {
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
		INSERT INTO ewrite_object_links (object_type, scene_element_id, publication_id, section_id, created_by)
		VALUES ('scene_element', $1, $2, NULLIF($3, '')::uuid, $4)
		ON CONFLICT (scene_element_id) WHERE object_type = 'scene_element' DO UPDATE
			SET publication_id = EXCLUDED.publication_id,
			    section_id = EXCLUDED.section_id,
			    updated_at = NOW()
		RETURNING id::text, COALESCE(section_id::text, '')
	`, sceneElementID, p.ID, sectionID, userID).Scan(&l.ID, &l.SectionID)
	if err != nil {
		return nil, err
	}
	l.ObjectType = "scene_element"
	l.SceneElementID = sceneElementID
	l.PublicationID = p.ID
	return &l, nil
}

// RuleLinksForSceneElements resolves rule links for a set of Scene stage
// element IDs in one query.
func RuleLinksForSceneElements(ctx context.Context, pool queryer, sceneElementIDs []string) (map[string]RuleLink, error) {
	out := map[string]RuleLink{}
	if len(sceneElementIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT ol.id::text, ol.scene_element_id::text, p.id::text, p.title, COALESCE(s.anchor, ''), COALESCE(s.title, '')
		FROM ewrite_object_links ol
		JOIN ewrite_publications p ON p.id = ol.publication_id
		LEFT JOIN ewrite_sections s ON s.id = ol.section_id
		WHERE ol.object_type = 'scene_element' AND ol.scene_element_id = ANY($1) AND p.status = 'published'
	`, sceneElementIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var elementID string
		var l RuleLink
		if err := rows.Scan(&l.ID, &elementID, &l.PublicationID, &l.PublicationTitle, &l.SectionAnchor, &l.SectionTitle); err != nil {
			return nil, err
		}
		out[elementID] = l
	}
	return out, rows.Err()
}

// SetDialogueTopicRuleLink creates or replaces the rule link for one
// guided-dialogue Topic (spec 7 -- the one concrete, stable "tutorial
// choice" object that exists today). Authority: edit authority on the
// publication AND Crew+ at the Topic's own Packet's location
// (dialogue_topics.packet_id -> dialogue_packets.location_id).
func SetDialogueTopicRuleLink(ctx context.Context, pool *pgxpool.Pool, userID, topicID, publicationID, sectionID string) (*ObjectLink, error) {
	p, err := LoadPublication(ctx, pool, strings.TrimSpace(publicationID))
	if err != nil {
		return nil, err
	}
	if ok, err := CanEditPublication(ctx, pool, userID, p); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("not_authorized")
	}

	var topicLocationID string
	err = pool.QueryRow(ctx, `
		SELECT dp.location_id::text
		FROM dialogue_topics dt
		JOIN dialogue_packets dp ON dp.id = dt.packet_id
		WHERE dt.id = $1
	`, strings.TrimSpace(topicID)).Scan(&topicLocationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("dialogue_topic_not_found")
	}
	if err != nil {
		return nil, err
	}
	if ok, err := CanAuthorInScope(ctx, pool, userID, topicLocationID); err != nil {
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
		INSERT INTO ewrite_object_links (object_type, dialogue_topic_id, publication_id, section_id, created_by)
		VALUES ('dialogue_topic', $1, $2, NULLIF($3, '')::uuid, $4)
		ON CONFLICT (dialogue_topic_id) WHERE object_type = 'dialogue_topic' DO UPDATE
			SET publication_id = EXCLUDED.publication_id,
			    section_id = EXCLUDED.section_id,
			    updated_at = NOW()
		RETURNING id::text, COALESCE(section_id::text, '')
	`, topicID, p.ID, sectionID, userID).Scan(&l.ID, &l.SectionID)
	if err != nil {
		return nil, err
	}
	l.ObjectType = "dialogue_topic"
	l.DialogueTopicID = topicID
	l.PublicationID = p.ID
	return &l, nil
}

// RuleLinksForDialogueTopics resolves rule links for a set of dialogue
// Topic IDs in one query.
func RuleLinksForDialogueTopics(ctx context.Context, pool queryer, topicIDs []string) (map[string]RuleLink, error) {
	out := map[string]RuleLink{}
	if len(topicIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT ol.id::text, ol.dialogue_topic_id::text, p.id::text, p.title, COALESCE(s.anchor, ''), COALESCE(s.title, '')
		FROM ewrite_object_links ol
		JOIN ewrite_publications p ON p.id = ol.publication_id
		LEFT JOIN ewrite_sections s ON s.id = ol.section_id
		WHERE ol.object_type = 'dialogue_topic' AND ol.dialogue_topic_id = ANY($1) AND p.status = 'published'
	`, topicIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var topicID string
		var l RuleLink
		if err := rows.Scan(&l.ID, &topicID, &l.PublicationID, &l.PublicationTitle, &l.SectionAnchor, &l.SectionTitle); err != nil {
			return nil, err
		}
		out[topicID] = l
	}
	return out, rows.Err()
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
// GET /api/ewrite/object-links?object_type=...&object_id=... across every
// linkable object type (spec 8.4 -- one reusable link shape).
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
				ObjectType         string `json:"object_type"`
				EquipmentItemID    string `json:"equipment_item_id"`
				CueID              string `json:"cue_id"`
				IndexCardElementID string `json:"index_card_element_id"`
				SceneElementID     string `json:"scene_element_id"`
				DialogueTopicID    string `json:"dialogue_topic_id"`
				PublicationID      string `json:"publication_id"`
				SectionID          string `json:"section_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, errors.New("invalid_json"))
				return
			}
			var l *ObjectLink
			var setErr error
			switch body.ObjectType {
			case "equipment_item":
				l, setErr = SetEquipmentItemRuleLink(ctx, pool, userID, body.EquipmentItemID, body.PublicationID, body.SectionID)
			case "cue":
				l, setErr = SetCueRuleLink(ctx, pool, userID, body.CueID, body.PublicationID, body.SectionID)
			case "index_card":
				l, setErr = SetIndexCardRuleLink(ctx, pool, userID, body.IndexCardElementID, body.PublicationID, body.SectionID)
			case "scene_element":
				l, setErr = SetSceneElementRuleLink(ctx, pool, userID, body.SceneElementID, body.PublicationID, body.SectionID)
			case "dialogue_topic":
				l, setErr = SetDialogueTopicRuleLink(ctx, pool, userID, body.DialogueTopicID, body.PublicationID, body.SectionID)
			default:
				writeError(w, errors.New("unsupported_object_type"))
				return
			}
			if setErr != nil {
				writeError(w, setErr)
				return
			}
			writeOK(w, map[string]any{"link": l})
		case http.MethodGet:
			objectType := r.URL.Query().Get("object_type")
			objectID := strings.TrimSpace(r.URL.Query().Get("object_id"))
			if objectID == "" {
				writeError(w, errors.New("unsupported_object_type"))
				return
			}
			var links map[string]RuleLink
			var linkErr error
			switch objectType {
			case "equipment_item":
				links, linkErr = RuleLinksForEquipmentItems(ctx, pool, []string{objectID})
			case "cue":
				links, linkErr = RuleLinksForCues(ctx, pool, []string{objectID})
			case "index_card":
				links, linkErr = RuleLinksForIndexCards(ctx, pool, []string{objectID})
			case "scene_element":
				links, linkErr = RuleLinksForSceneElements(ctx, pool, []string{objectID})
			case "dialogue_topic":
				links, linkErr = RuleLinksForDialogueTopics(ctx, pool, []string{objectID})
			default:
				writeError(w, errors.New("unsupported_object_type"))
				return
			}
			if linkErr != nil {
				writeError(w, linkErr)
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
