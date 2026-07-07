package playerprofile

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// legacyToCatalogueKey maps performer_profiles columns onto their nearest
// Kernel 61 catalogue equivalent, where one exists (Kernel 61 §9.3). A
// legacy column with no listed mapping (e.g. display_name,
// performance_age_range) is preserved verbatim under its own legacy_* key
// instead -- DeriveEffectiveFacts keeps unknown keys as private facts, and
// BuildProjectedFields only ever surfaces catalogue-known, face_eligible
// keys, so unmapped legacy data can never appear on Face by accident.
var legacyToCatalogueKey = map[string]string{
	"pronouns":               "pronouns",
	"headshot_url":           "portrait_url",
	"bio":                    "about_text",
	"credits":                "credits_text",
	"skills":                 "practical_skills",
	"availability":           "general_availability",
	"public_links":           "portfolio_links",
	"favorite_fun":           "favorite_fun",
	"most_relaxed":           "most_relaxed",
	"favorite_color":         "favorite_color",
	"favorite_artist":        "favorite_artist",
	"favorite_food":          "favorite_food",
	"favorite_song":          "favorite_song",
	"favorite_place":         "favorite_place",
	"favorite_movie_or_show": "favorite_movie_or_show",
	"hidden_talent":          "hidden_talent",
	"ideal_day":              "ideal_day",
}

// legacyFieldOrder fixes the column order used by the draft_*/published_*
// scalar SELECT lists in ImportLegacyProfiles -- it must match those lists
// exactly since scanning positionally is what lets one loop populate both
// the draft and published field maps.
var legacyFieldOrder = []string{
	"pronouns", "headshot_url", "bio", "credits", "skills",
	"availability", "public_links", "favorite_fun", "most_relaxed",
	"favorite_color", "favorite_artist", "favorite_food", "favorite_song",
	"favorite_place", "favorite_movie_or_show", "hidden_talent", "ideal_day",
}

type legacyProfileRow struct {
	userID      string
	handle      string
	isPublished bool

	draftDisplayName string
	draftStageName   string
	draftFields      map[string]string
	draftHistory     []map[string]string

	publishedStageName string
	publishedFields    map[string]string
	publishedHistory   []map[string]string
}

// ImportLegacyProfiles migrates every performer_profiles row into the
// Player Workbook model (Kernel 61 §9.3). It is idempotent: a workbook that
// already has a legacy import event is left untouched on subsequent runs
// (AC-28), and it never mutates `performer_profiles` itself (§11.5).
func ImportLegacyProfiles(ctx context.Context, pool *pgxpool.Pool) error {
	var tableExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'performer_profiles')
	`).Scan(&tableExists); err != nil {
		return err
	}
	if !tableExists {
		return nil
	}

	rows, err := pool.Query(ctx, `
		SELECT
			pp.user_id::text,
			COALESCE(NULLIF(u.handle, ''), LEFT(u.id::text, 8)),
			pp.is_published,
			pp.draft_display_name, pp.draft_stage_name,
			pp.draft_pronouns, pp.draft_headshot_url, pp.draft_bio, pp.draft_credits, pp.draft_skills,
			pp.draft_availability, pp.draft_public_links, pp.draft_favorite_fun, pp.draft_most_relaxed,
			pp.draft_favorite_color, pp.draft_favorite_artist, pp.draft_favorite_food, pp.draft_favorite_song,
			pp.draft_favorite_place, pp.draft_favorite_movie_or_show, pp.draft_hidden_talent, pp.draft_ideal_day,
			pp.draft_history,
			pp.published_stage_name,
			pp.published_pronouns, pp.published_headshot_url, pp.published_bio, pp.published_credits, pp.published_skills,
			pp.published_availability, pp.published_public_links, pp.published_favorite_fun, pp.published_most_relaxed,
			pp.published_favorite_color, pp.published_favorite_artist, pp.published_favorite_food, pp.published_favorite_song,
			pp.published_favorite_place, pp.published_favorite_movie_or_show, pp.published_hidden_talent, pp.published_ideal_day,
			pp.published_history
		FROM performer_profiles pp
		JOIN users u ON u.id = pp.user_id
	`)
	if err != nil {
		return err
	}

	var toImport []legacyProfileRow
	for rows.Next() {
		var (
			userID, handle      string
			isPublished         bool
			draftDisplayName    string
			draftStageName      string
			draftHistoryRaw     string
			publishedStageName  string
			publishedHistoryRaw string
		)
		draftVals := make([]string, len(legacyFieldOrder))
		publishedVals := make([]string, len(legacyFieldOrder))

		dest := []any{&userID, &handle, &isPublished, &draftDisplayName, &draftStageName}
		for i := range draftVals {
			dest = append(dest, &draftVals[i])
		}
		dest = append(dest, &draftHistoryRaw, &publishedStageName)
		for i := range publishedVals {
			dest = append(dest, &publishedVals[i])
		}
		dest = append(dest, &publishedHistoryRaw)

		if err := rows.Scan(dest...); err != nil {
			rows.Close()
			return err
		}

		r := legacyProfileRow{
			userID:             userID,
			handle:             handle,
			isPublished:        isPublished,
			draftDisplayName:   draftDisplayName,
			draftStageName:     draftStageName,
			draftFields:        map[string]string{},
			publishedStageName: publishedStageName,
			publishedFields:    map[string]string{},
		}
		for i, key := range legacyFieldOrder {
			r.draftFields[key] = draftVals[i]
			r.publishedFields[key] = publishedVals[i]
		}
		r.draftHistory = decodeLegacyHistory(draftHistoryRaw)
		r.publishedHistory = decodeLegacyHistory(publishedHistoryRaw)
		toImport = append(toImport, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	for _, r := range toImport {
		if err := importOneLegacyProfile(ctx, pool, r); err != nil {
			return fmt.Errorf("import legacy profile for user %s: %w", r.userID, err)
		}
	}

	return nil
}

func importOneLegacyProfile(ctx context.Context, pool *pgxpool.Pool, r legacyProfileRow) error {
	wb, err := EnsureWorkbook(ctx, pool, r.userID)
	if err != nil {
		return err
	}

	var alreadyImported bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM player_profile_events
			WHERE workbook_id = $1 AND event_type IN ('legacy_published_profile', 'legacy_draft_profile')
		)
	`, wb.ID).Scan(&alreadyImported); err != nil {
		return err
	}
	if alreadyImported {
		// Facts are always fully rebuilt from whatever events exist, even on
		// a run that imports nothing new -- this is what makes a partial
		// prior failure (e.g. an event that got inserted but whose fact
		// recompute then errored) self-heal on the next bootstrap instead of
		// being silently skipped forever.
		return RecomputePlayerFacts(ctx, pool, wb.ID)
	}

	if err := seedLegacyStageName(ctx, pool, r); err != nil {
		return err
	}

	publishedPayload := map[string]any{}
	if r.isPublished {
		for legacyKey, val := range r.publishedFields {
			if strings.TrimSpace(val) == "" {
				continue
			}
			key := legacyKey
			if mapped, ok := legacyToCatalogueKey[legacyKey]; ok {
				key = mapped
			} else {
				key = "legacy_" + legacyKey
			}
			publishedPayload[key] = strings.TrimSpace(val)
		}
		if len(r.publishedHistory) > 0 {
			publishedPayload["legacy_production_history"] = formatLegacyHistory(r.publishedHistory)
		}
	}

	draftPayload := map[string]any{}
	for legacyKey, val := range r.draftFields {
		trimmed := strings.TrimSpace(val)
		if trimmed == "" {
			continue
		}
		publishedVal := strings.TrimSpace(r.publishedFields[legacyKey])
		if r.isPublished && trimmed == publishedVal {
			continue
		}
		draftPayload["legacy_draft__"+legacyKey] = trimmed
	}
	if strings.TrimSpace(r.draftDisplayName) != "" {
		draftPayload["legacy_draft__display_name"] = strings.TrimSpace(r.draftDisplayName)
	}
	if len(r.draftHistory) > 0 && (!r.isPublished || fmt.Sprint(r.draftHistory) != fmt.Sprint(r.publishedHistory)) {
		draftPayload["legacy_draft__production_history"] = formatLegacyHistory(r.draftHistory)
	}

	if len(publishedPayload) > 0 {
		event, err := insertProfileEvent(ctx, pool, wb.ID, "legacy_published_profile", "legacy_import", "legacy-import", publishedPayload,
			"Imported published legacy profile data.", r.userID)
		if err != nil {
			return err
		}
		_ = event
	}

	if len(draftPayload) > 0 {
		_, err := insertProfileEvent(ctx, pool, wb.ID, "legacy_draft_profile", "legacy_import", "legacy-import", draftPayload,
			"Preserved legacy draft-only profile data (private; not shown on Face automatically).", r.userID)
		if err != nil {
			return err
		}
	}

	if len(publishedPayload) == 0 && len(draftPayload) == 0 {
		return nil
	}

	if err := RecomputePlayerFacts(ctx, pool, wb.ID); err != nil {
		return err
	}

	cat, err := LoadCatalogue()
	if err != nil {
		return err
	}
	for key := range publishedPayload {
		field, ok := cat.FieldByKey(key)
		if !ok || !field.FaceEligible {
			continue
		}
		if err := SetFaceVisibility(ctx, pool, r.userID, key, VisibilityShown); err != nil {
			return err
		}
	}

	return touchWorkbookProjection(ctx, pool, wb.ID)
}

// seedLegacyStageName resolves and records the initial stage name for a
// migrated account using the fallback order in Kernel 61 §9.3: non-empty
// draft, then published, then handle, then a generated support code. This
// bypasses ChangeStageName's actor==target check since the system -- not
// the account itself -- performs the one-time import.
func seedLegacyStageName(ctx context.Context, pool *pgxpool.Pool, r legacyProfileRow) error {
	var hasCurrent bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM player_stage_name_history WHERE user_id = $1 AND ended_at IS NULL)
	`, r.userID).Scan(&hasCurrent); err != nil {
		return err
	}
	if hasCurrent {
		return nil
	}

	candidate := strings.TrimSpace(r.draftStageName)
	if candidate == "" {
		candidate = strings.TrimSpace(r.publishedStageName)
	}
	if candidate == "" {
		candidate = strings.TrimSpace(r.handle)
	}
	if candidate == "" {
		candidate = "Player-" + r.userID[:8]
	}

	trimmed, err := ValidateStageNameCandidate(candidate)
	if err != nil {
		trimmed = "Player-" + r.userID[:8]
	}
	normalized := NormalizeStageName(trimmed)

	_, err = pool.Exec(ctx, `
		INSERT INTO player_stage_name_history (user_id, stage_name, normalized_stage_name, changed_by_user_id, source)
		VALUES ($1, $2, $3, $1, 'legacy_import')
	`, r.userID, trimmed, normalized)
	return err
}

func decodeLegacyHistory(raw string) []map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" || raw == "[]" {
		return nil
	}
	// performer_profiles stores history as a JSON string of
	// [{production_name, role, run_period}, ...]; parse defensively so a
	// malformed legacy row can't fail the whole import.
	var entries []struct {
		ProductionName string `json:"production_name"`
		Role           string `json:"role"`
		RunPeriod      string `json:"run_period"`
	}
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return nil
	}
	out := make([]map[string]string, 0, len(entries))
	for _, e := range entries {
		if strings.TrimSpace(e.ProductionName) == "" && strings.TrimSpace(e.Role) == "" && strings.TrimSpace(e.RunPeriod) == "" {
			continue
		}
		out = append(out, map[string]string{"production_name": e.ProductionName, "role": e.Role, "run_period": e.RunPeriod})
	}
	return out
}

func formatLegacyHistory(entries []map[string]string) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, strings.TrimSpace(fmt.Sprintf("%s — %s (%s)", e["production_name"], e["role"], e["run_period"])))
	}
	return out
}
