package merchant

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/actions"
	"victory/backend/internal/dice"
	"victory/backend/internal/ewrite"
	"victory/backend/internal/scenes"
	"victory/backend/internal/showruns"
	"victory/backend/internal/shows"
	"victory/backend/internal/stageobjects"
)

const participantInteractionColumns = `
	id::text, show_scene_placement_id::text, internal_name, stage_button_label,
	interaction_type, configuration_json, enabled, sort_order,
	COALESCE(created_by_user_id::text, ''), created_at, updated_at
`

func scanInteraction(row pgx.Row) (ParticipantInteraction, error) {
	var it ParticipantInteraction
	var configRaw []byte
	if err := row.Scan(
		&it.ID, &it.ShowScenePlacementID, &it.InternalName, &it.StageButtonLabel,
		&it.InteractionType, &configRaw, &it.Enabled, &it.SortOrder,
		&it.CreatedByUserID, &it.CreatedAt, &it.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ParticipantInteraction{}, errors.New("interaction_not_found")
		}
		return ParticipantInteraction{}, err
	}
	_ = json.Unmarshal(configRaw, &it.ConfigurationJSON)
	return it, nil
}

func LoadInteractionByID(ctx context.Context, pool *pgxpool.Pool, id string) (ParticipantInteraction, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return ParticipantInteraction{}, errors.New("interaction_not_found")
	}
	row := pool.QueryRow(ctx, `SELECT `+participantInteractionColumns+` FROM participant_interactions WHERE id = $1`, id)
	return scanInteraction(row)
}

func ListInteractionsForPlacement(ctx context.Context, pool *pgxpool.Pool, placementID string) ([]ParticipantInteraction, error) {
	rows, err := pool.Query(ctx, `
		SELECT `+participantInteractionColumns+`
		FROM participant_interactions
		WHERE show_scene_placement_id = $1
		ORDER BY sort_order ASC, created_at ASC
	`, placementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ParticipantInteraction
	for rows.Next() {
		it, err := scanInteraction(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// placementShowShowRunLocation mirrors cues.placementShowShowRunLocation
// (unexported to that package) -- resolving a placement up through Show and
// Show Run to a locationID is common to every placement-scoped authoring
// surface.
func placementShowShowRunLocation(ctx context.Context, pool *pgxpool.Pool, placementID string) (showID, showRunID, locationID string, err error) {
	p, err := scenes.LoadPlacementByID(ctx, pool, placementID)
	if err != nil {
		return "", "", "", err
	}
	s, err := shows.LoadShowByID(ctx, pool, p.ShowID)
	if err != nil {
		return "", "", "", err
	}
	sr, err := showruns.LoadShowRunByID(ctx, pool, s.ShowRunID)
	if err != nil {
		return "", "", "", err
	}
	return p.ShowID, sr.ID, sr.LocationID, nil
}

// resolvePlacementVenueSlug resolves the venue a placement actually renders
// in -- the placement's own venue override if set, otherwise the Scene's
// default venue -- so ResolveEligibleContext can check the
// participant_interactions_enabled capability flag against the right venue.
func resolvePlacementVenueSlug(ctx context.Context, pool *pgxpool.Pool, placementID string) (string, error) {
	var venueSlug string
	err := pool.QueryRow(ctx, `
		SELECT v.slug
		FROM show_scene_placements p
		JOIN scenes s ON s.id = p.scene_id
		JOIN venues v ON v.id = COALESCE(p.venue_id, s.default_venue_id)
		WHERE p.id = $1
	`, placementID).Scan(&venueSlug)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("unknown_target")
	}
	if err != nil {
		return "", err
	}
	return venueSlug, nil
}

type CreateInteractionInput struct {
	InternalName     string
	StageButtonLabel string
	InteractionType  string
	Configuration    map[string]any
	SortOrder        int
	Enabled          *bool
}

// CreateInteraction is the Director authoring surface's create path (spec
// S11). Authority mirrors cues.CreateCue exactly: CanCrewPerformNonDestructiveEdit
// at the placement's Show Run location -- no new authority concept.
func CreateInteraction(ctx context.Context, pool *pgxpool.Pool, actorUserID, placementID string, in CreateInteractionInput) (ParticipantInteraction, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return ParticipantInteraction{}, errors.New("not_authenticated")
	}
	if strings.TrimSpace(in.InternalName) == "" {
		return ParticipantInteraction{}, errors.New("internal_name_required")
	}
	if strings.TrimSpace(in.StageButtonLabel) == "" {
		return ParticipantInteraction{}, errors.New("stage_button_label_required")
	}
	interactionType := strings.TrimSpace(in.InteractionType)
	if !validInteractionTypes[interactionType] {
		return ParticipantInteraction{}, errors.New("invalid_interaction_type")
	}

	_, _, locationID, err := placementShowShowRunLocation(ctx, pool, placementID)
	if err != nil {
		return ParticipantInteraction{}, err
	}
	allowed, err := showruns.CanCrewPerformNonDestructiveEdit(ctx, pool, actorUserID, locationID)
	if err != nil {
		return ParticipantInteraction{}, err
	}
	if !allowed {
		return ParticipantInteraction{}, errors.New("not_authorized")
	}

	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	config := in.Configuration
	if config == nil {
		config = map[string]any{}
	}
	configJSON, _ := json.Marshal(config)

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO participant_interactions (
			show_scene_placement_id, internal_name, stage_button_label,
			interaction_type, configuration_json, enabled, sort_order, created_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8::uuid)
		RETURNING id::text
	`, placementID, strings.TrimSpace(in.InternalName), strings.TrimSpace(in.StageButtonLabel),
		interactionType, configJSON, enabled, in.SortOrder, actorUserID).Scan(&id); err != nil {
		return ParticipantInteraction{}, err
	}
	return LoadInteractionByID(ctx, pool, id)
}

type UpdateInteractionPatch struct {
	StageButtonLabel *string
	Configuration    *map[string]any
	Enabled          *bool
	SortOrder        *int
}

func UpdateInteraction(ctx context.Context, pool *pgxpool.Pool, actorUserID, interactionID string, patch UpdateInteractionPatch) (ParticipantInteraction, error) {
	existing, err := LoadInteractionByID(ctx, pool, interactionID)
	if err != nil {
		return ParticipantInteraction{}, err
	}
	_, _, locationID, err := placementShowShowRunLocation(ctx, pool, existing.ShowScenePlacementID)
	if err != nil {
		return ParticipantInteraction{}, err
	}
	allowed, err := showruns.CanCrewPerformNonDestructiveEdit(ctx, pool, actorUserID, locationID)
	if err != nil {
		return ParticipantInteraction{}, err
	}
	if !allowed {
		return ParticipantInteraction{}, errors.New("not_authorized")
	}

	label := existing.StageButtonLabel
	if patch.StageButtonLabel != nil {
		label = strings.TrimSpace(*patch.StageButtonLabel)
	}
	enabled := existing.Enabled
	if patch.Enabled != nil {
		enabled = *patch.Enabled
	}
	sortOrder := existing.SortOrder
	if patch.SortOrder != nil {
		sortOrder = *patch.SortOrder
	}
	configJSON, _ := json.Marshal(existing.ConfigurationJSON)
	if patch.Configuration != nil {
		configJSON, _ = json.Marshal(*patch.Configuration)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE participant_interactions
		SET stage_button_label = $2, configuration_json = $3::jsonb, enabled = $4,
		    sort_order = $5, updated_at = NOW()
		WHERE id = $1
	`, interactionID, label, configJSON, enabled, sortOrder); err != nil {
		return ParticipantInteraction{}, err
	}
	return LoadInteractionByID(ctx, pool, interactionID)
}

// --- Player-side eligibility and Program context -----------------------

// EligibleContext is the fully-resolved, server-derived context every
// mutating Equip Mode action (stance attempt, Haggle attempt, purchase)
// requires (spec S7.2, S10 -- never trust user/Character/Show/Scene
// identity from client payloads). Resolving this once per request is what
// makes every downstream action's identity server-authoritative.
type EligibleContext struct {
	Interaction ParticipantInteraction
	// ActorUserID is the authenticated caller this context was resolved
	// for. Carried explicitly (Kernel 74) so downstream helpers building a
	// tutorial.Participation cannot accidentally pair one caller's resolved
	// Character/Show with a different caller's user ID.
	ActorUserID     string
	PlacementID     string
	ShowID          string
	ShowRunID       string
	LocationID      string
	SessionID       string
	CharacterCardID string
}

// ResolveEligibleContext enforces every criterion in spec S7.2: authenticated;
// active Player roster member; Show's current Scene Placement matches the
// interaction's placement; selected Character active and owned; interaction
// enabled. Audience and non-roster users are refused by construction (no
// active roster row => "not_a_roster_member"; role != player =>
// "insufficient_role"), matching cues.CanTriggerCue's Audience hard floor.
func ResolveEligibleContext(ctx context.Context, pool *pgxpool.Pool, actorUserID, interactionID string) (EligibleContext, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return EligibleContext{}, errors.New("not_authenticated")
	}

	interaction, err := LoadInteractionByID(ctx, pool, interactionID)
	if err != nil {
		return EligibleContext{}, err
	}
	if !interaction.Enabled {
		return EligibleContext{}, errors.New("interaction_disabled")
	}

	showID, showRunID, locationID, err := placementShowShowRunLocation(ctx, pool, interaction.ShowScenePlacementID)
	if err != nil {
		return EligibleContext{}, err
	}

	// Kernel 90 §35/§53: the Director may have disabled this interaction for
	// the rest of this Show. Enforced HERE, at the same single Player-
	// eligibility gate Kernel 89 §9.3's cohort targeting chose, so
	// exposure/open/stance/haggle/purchase all inherit it without a second
	// check to keep in sync.
	//
	// This is the difference between an interaction that is not offered and
	// one that is disabled: world/snapshot.go stops advertising it to the
	// client, and this refuses the Player who calls anyway. Distinct from
	// interaction.Enabled checked above -- that is the global authoring
	// kill-switch, this is in-play Show state (see the bridge documented on
	// migration 106's interaction_enabled column).
	invocable, err := stageobjects.InteractionInvocable(ctx, pool, showID, interaction.ID)
	if err != nil {
		return EligibleContext{}, err
	}
	if !invocable {
		return EligibleContext{}, errors.New("interaction_disabled")
	}

	venueSlug, err := resolvePlacementVenueSlug(ctx, pool, interaction.ShowScenePlacementID)
	if err != nil {
		return EligibleContext{}, err
	}
	enabled, err := VenueParticipantInteractionsEnabled(ctx, pool, venueSlug)
	if err != nil {
		return EligibleContext{}, err
	}
	if !enabled {
		return EligibleContext{}, errors.New("unknown_target")
	}

	show, err := shows.LoadShowByID(ctx, pool, showID)
	if err != nil {
		return EligibleContext{}, err
	}
	currentPlacementID := ""
	if show.CurrentShowScenePlacementID != nil {
		currentPlacementID = strings.TrimSpace(*show.CurrentShowScenePlacementID)
	}
	if currentPlacementID != strings.TrimSpace(interaction.ShowScenePlacementID) {
		return EligibleContext{}, errors.New("scene_not_current")
	}

	// Kernel 75 extracted the roster-and-Character half of this gate into
	// ResolveShowParticipation, so Aftercare (which is Show-keyed and may be
	// written when no Session is live) can reuse it rather than growing a
	// second copy. The checks are unchanged; only their location moved.
	participation, err := ResolveShowParticipation(ctx, pool, actorUserID, showID)
	if err != nil {
		return EligibleContext{}, err
	}

	// Kernel 89 §9.3: an interaction may be aimed at ONE Cohort rather than
	// at every eligible Player on the placement. Enforced here, at the one
	// Player-eligibility gate every participant action already funnels
	// through, so exposure/open/stance/haggle/purchase all inherit it
	// without a second check to keep in sync -- and so a Player who guesses
	// the interaction id still cannot open a merchant aimed at another
	// Cohort. There is no whole-Show targeting keyword and no user-list
	// targeting: §9.3 explicitly keeps that out of this kernel.
	if targetCohortID := targetCohortIDFromConfig(interaction.ConfigurationJSON); targetCohortID != "" {
		var inCohort bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM show_cohort_assignments a
				JOIN show_cohorts c ON c.id = a.cohort_id
				WHERE a.show_id = $1 AND a.user_id = $2
				  AND a.cohort_id = $3::uuid AND c.archived_at IS NULL
			)
		`, showID, actorUserID, targetCohortID).Scan(&inCohort); err != nil {
			return EligibleContext{}, err
		}
		if !inCohort {
			return EligibleContext{}, errors.New("not_targeted")
		}
	}

	var sessionID string
	err = pool.QueryRow(ctx, `
		SELECT id::text FROM sessions
		WHERE show_id = $1 AND status IN ('rehearsal', 'live')
		ORDER BY started_at DESC
		LIMIT 1
	`, showID).Scan(&sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return EligibleContext{}, errors.New("no_active_session")
	}
	if err != nil {
		return EligibleContext{}, err
	}

	return EligibleContext{
		Interaction:     interaction,
		ActorUserID:     actorUserID,
		PlacementID:     interaction.ShowScenePlacementID,
		ShowID:          showID,
		ShowRunID:       showRunID,
		LocationID:      locationID,
		SessionID:       sessionID,
		CharacterCardID: participation.CharacterCardID,
	}, nil
}

// ListTriggerableInteractionsForViewer is the curated player-facing stage-
// button listing (mirrors cues.ListTriggerableCuesForViewer's shape), but
// unlike Cues -- which Director/Producer/Operator may always trigger
// regardless of trigger_scope -- a participant interaction is Player-only by
// construction; there is no backstage-override branch here.
func ListTriggerableInteractionsForViewer(ctx context.Context, pool *pgxpool.Pool, viewerUserID, placementID string) ([]PlayerVisibleInteraction, error) {
	all, err := ListInteractionsForPlacement(ctx, pool, placementID)
	if err != nil {
		return nil, err
	}

	// Kernel 73A bound a token to an interaction so it's reachable by clicking
	// the token on stage. This floating listing predates that (Kernel 73) and
	// would otherwise show the exact same interaction a second time as a
	// disembodied button -- exclude anything already bound to a token so the
	// token stays the one true entry point, leaving this listing as the
	// fallback for interactions that aren't placed on stage yet.
	boundIDs := map[string]bool{}
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT b.participant_interaction_id::text
		FROM stage_element_bindings b
		JOIN participant_interactions pi ON pi.id = b.participant_interaction_id
		WHERE pi.show_scene_placement_id = $1
	`, placementID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		boundIDs[id] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]PlayerVisibleInteraction, 0, len(all))
	for _, it := range all {
		if !it.Enabled {
			continue
		}
		if boundIDs[it.ID] {
			continue
		}
		if _, err := ResolveEligibleContext(ctx, pool, viewerUserID, it.ID); err != nil {
			continue
		}
		out = append(out, PlayerVisibleInteraction{ID: it.ID, Label: it.StageButtonLabel, InteractionType: it.InteractionType})
	}
	return out, nil
}

// targetCohortIDFromConfig reads Kernel 89's optional Cohort target.
// Absent/blank means "every eligible Player on this placement", which is
// the Kernel 73 behaviour every existing interaction keeps unchanged.
func targetCohortIDFromConfig(config map[string]any) string {
	if v, ok := config["target_cohort_id"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func packetSlugFromConfig(config map[string]any) string {
	if v, ok := config["packet_slug"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

// EquipModeContext is everything the frontend needs to render Equip Mode on
// open (spec S8.2): the packet (with stock), current inventory for the
// Character, and the resolved identity to display. RuleLinks (Kernel 78)
// maps equipment_item_id -> published eWrite rule section so the shop can
// offer "View rule" without a second round trip; only published targets
// resolve, so no draft title can leak into a Player payload.
type EquipModeContext struct {
	Interaction     ParticipantInteraction     `json:"interaction"`
	Packet          MerchantPacket             `json:"packet"`
	CharacterCardID string                     `json:"character_card_id"`
	Inventory       []InventoryEntry           `json:"inventory"`
	RuleLinks       map[string]ewrite.RuleLink `json:"rule_links,omitempty"`
}

// OpenEquipMode resolves eligibility, then loads everything the Program
// Panel needs. Opening the shop never mutates shows.current_show_scene_placement_id
// (spec S2.1) -- this function only reads.
func OpenEquipMode(ctx context.Context, pool *pgxpool.Pool, actorUserID, interactionID string) (EquipModeContext, error) {
	eligible, err := ResolveEligibleContext(ctx, pool, actorUserID, interactionID)
	if err != nil {
		return EquipModeContext{}, err
	}
	packetSlug := packetSlugFromConfig(eligible.Interaction.ConfigurationJSON)
	if packetSlug == "" {
		return EquipModeContext{}, errors.New("interaction_missing_packet")
	}
	packet, err := LoadPacketBySlug(ctx, pool, eligible.LocationID, packetSlug)
	if err != nil {
		return EquipModeContext{}, err
	}
	inventory, err := ListInventoryForCharacter(ctx, pool, actorUserID, eligible.CharacterCardID)
	if err != nil {
		return EquipModeContext{}, err
	}
	stockIDs := make([]string, 0, len(packet.Stock))
	for _, item := range packet.Stock {
		stockIDs = append(stockIDs, item.ID)
	}
	ruleLinks, err := ewrite.RuleLinksForEquipmentItems(ctx, pool, stockIDs)
	if err != nil {
		return EquipModeContext{}, err
	}
	return EquipModeContext{
		Interaction:     eligible.Interaction,
		Packet:          packet,
		CharacterCardID: eligible.CharacterCardID,
		Inventory:       inventory,
		RuleLinks:       ruleLinks,
	}, nil
}

// --- Stance attempts -----------------------------------------------------

// StanceAttemptResult is what the Player sees after picking one of the five
// fixed stances. Disposition is always the packet's authored value for that
// stance (spec S2.6) -- the roll only selects which response-text variant
// is shown.
type StanceAttemptResult struct {
	Stance      string `json:"stance"`
	Disposition string `json:"disposition"`
	Response    string `json:"response"`
	Die         string `json:"die"`
	Total       int    `json:"total"`
}

func isFixedStanceKey(key string) bool {
	for _, k := range FixedStanceKeys {
		if k == key {
			return true
		}
	}
	return false
}

// AttemptStance rolls a flavor die purely to pick which authored response
// variant is shown -- the audit found no existing "stance skill" mapping in
// the codebase (kernel-73 spec's own required audit deliverable), and the
// spec never names a skill for the five stances (only Haggle names one, with
// an explicit Target Value). Deliberately NOT skill-gated: recording "which
// die, which result" (spec S9.2) doesn't require inventing a skill
// association the spec doesn't specify. A d20 is rolled purely to bucket
// into a low/mid/high response tier; the disposition itself never changes.
func AttemptStance(ctx context.Context, pool *pgxpool.Pool, actorUserID, interactionID, stanceKey string) (StanceAttemptResult, error) {
	stanceKey = strings.ToLower(strings.TrimSpace(stanceKey))
	if !isFixedStanceKey(stanceKey) {
		return StanceAttemptResult{}, errors.New("invalid_stance")
	}

	eligible, err := ResolveEligibleContext(ctx, pool, actorUserID, interactionID)
	if err != nil {
		return StanceAttemptResult{}, err
	}
	packetSlug := packetSlugFromConfig(eligible.Interaction.ConfigurationJSON)
	packet, err := LoadPacketBySlug(ctx, pool, eligible.LocationID, packetSlug)
	if err != nil {
		return StanceAttemptResult{}, err
	}
	stance, ok := packet.StanceDispositions[stanceKey]
	if !ok || len(stance.Responses) == 0 {
		return StanceAttemptResult{}, errors.New("stance_not_configured")
	}

	result, err := dice.RollExpression(ctx, "d20", dice.CryptoSource{})
	if err != nil {
		return StanceAttemptResult{}, err
	}
	tier := 0
	switch {
	case result.Total >= 15:
		tier = 2
	case result.Total >= 8:
		tier = 1
	}
	if tier >= len(stance.Responses) {
		tier = len(stance.Responses) - 1
	}
	response := stance.Responses[tier]

	// Kernel 75 S5.1: the durable, Character-keyed record, written BEFORE the
	// ephemeral actions row so a failure here can never leave an actions row
	// claiming an attempt that Character history cannot corroborate.
	if err := recordInteractionAttempt(ctx, pool, eligible, attemptRecord{
		AttemptKind:  "stance",
		PacketSlug:   packet.Slug,
		StanceKey:    stanceKey,
		Disposition:  stance.Disposition,
		ResponseTier: intPtr(tier),
		Die:          "d20",
		Total:        intPtr(result.Total),
	}); err != nil {
		return StanceAttemptResult{}, err
	}

	// StoreGameEventTrusted, not StoreGameEvent: StoreGameEvent's own CanAct
	// gate for "game/event" only allows director/producer session
	// participants (despite its doc comment's "any session participant"
	// claim) -- it would incorrectly deny the Player this whole function
	// exists for. ResolveEligibleContext above has already fully authorized
	// this specific actor for this specific interaction; re-running CanAct
	// here would add no security value, exactly like cues.ExecuteCue's own
	// use of StoreGameEventTrusted for the same reason.
	if _, err := actions.StoreGameEventTrusted(ctx, pool, actions.GameEventRequest{
		SessionID:       eligible.SessionID,
		ActorID:         actorUserID,
		EventKind:       "interaction/stance_attempted",
		CharacterCardID: eligible.CharacterCardID,
		Detail: map[string]any{
			"interaction_id": interactionID,
			"packet_slug":    packet.Slug,
			"stance":         stanceKey,
			"disposition":    stance.Disposition,
			"response_key":   tier,
			"die":            "d20",
			"total":          result.Total,
		},
	}); err != nil {
		return StanceAttemptResult{}, err
	}

	return StanceAttemptResult{
		Stance:      stanceKey,
		Disposition: stance.Disposition,
		Response:    response,
		Die:         "d20",
		Total:       result.Total,
	}, nil
}

// --- Haggle ---------------------------------------------------------------

// HaggleAttemptPreview is shown BEFORE rolling (spec S2.7/S9.3): the
// confirmation panel's die/Target-Value display. ImpossibleToReach is true
// exactly when the unskilled die's maximum face is below the target value
// (d4 max 4 < TV 5) -- the frontend uses this to require an explicit
// "Attempt Anyway" rather than rolling automatically.
type HaggleAttemptPreview struct {
	HasSkill          bool   `json:"has_skill"`
	Die               string `json:"die"`
	TargetValue       int    `json:"target_value"`
	ImpossibleToReach bool   `json:"impossible_to_reach"`
}

func dieMaxFace(expr string) int {
	spec, _, err := dice.ParseExpression(expr)
	if err != nil || len(spec.Groups) == 0 {
		return 0
	}
	max := 0
	for _, g := range spec.Groups {
		max += g.Count * g.Sides
	}
	return max + spec.Modifier
}

func PreviewHaggle(ctx context.Context, pool *pgxpool.Pool, actorUserID, interactionID string) (HaggleAttemptPreview, error) {
	eligible, err := ResolveEligibleContext(ctx, pool, actorUserID, interactionID)
	if err != nil {
		return HaggleAttemptPreview{}, err
	}
	packet, err := LoadPacketBySlug(ctx, pool, eligible.LocationID, packetSlugFromConfig(eligible.Interaction.ConfigurationJSON))
	if err != nil {
		return HaggleAttemptPreview{}, err
	}

	hasSkill, err := hasCharacterSkill(ctx, pool, eligible.CharacterCardID, packet.HaggleSkillKey)
	if err != nil {
		return HaggleAttemptPreview{}, err
	}
	die := packet.HaggleUnskilledDie
	if hasSkill {
		die = packet.HaggleSkilledDie
	}

	return HaggleAttemptPreview{
		HasSkill:          hasSkill,
		Die:               die,
		TargetValue:       packet.HaggleTargetValue,
		ImpossibleToReach: dieMaxFace(die) < packet.HaggleTargetValue,
	}, nil
}

// HaggleAttemptResult is the server-calculated outcome (spec S9.3: "Success
// is server-calculated using the audited canonical comparison" -- never
// faked in frontend JavaScript).
type HaggleAttemptResult struct {
	Die         string `json:"die"`
	Total       int    `json:"total"`
	TargetValue int    `json:"target_value"`
	Success     bool   `json:"success"`
	Text        string `json:"text"`
}

// AttemptHaggle performs the actual roll -- only reachable after a Preview
// (skilled Characters may go straight here; unskilled Characters must have
// already seen ImpossibleToReach and explicitly chosen to proceed, enforced
// by the frontend's Attempt-Anyway/Return gate, not re-checked here since
// there's no unsafe action being gated -- an unskilled attempt simply fails
// deterministically against a TV the die cannot reach).
func AttemptHaggle(ctx context.Context, pool *pgxpool.Pool, actorUserID, interactionID string) (HaggleAttemptResult, error) {
	eligible, err := ResolveEligibleContext(ctx, pool, actorUserID, interactionID)
	if err != nil {
		return HaggleAttemptResult{}, err
	}
	packet, err := LoadPacketBySlug(ctx, pool, eligible.LocationID, packetSlugFromConfig(eligible.Interaction.ConfigurationJSON))
	if err != nil {
		return HaggleAttemptResult{}, err
	}

	roll, err := RollSkillGatedDie(ctx, pool, actorUserID, eligible.CharacterCardID, packet.HaggleSkillKey, packet.HaggleSkilledDie, packet.HaggleUnskilledDie)
	if err != nil {
		return HaggleAttemptResult{}, err
	}

	success := roll.Result.Total >= packet.HaggleTargetValue
	text := packet.HaggleFailureText
	if success {
		text = packet.HaggleSuccessText
	}

	// Kernel 75 S5.1: durable record first, ephemeral actions row second --
	// see recordInteractionAttempt for why the ordering is load-bearing.
	if err := recordInteractionAttempt(ctx, pool, eligible, attemptRecord{
		AttemptKind: "haggle",
		PacketSlug:  packet.Slug,
		SkillKey:    packet.HaggleSkillKey,
		HasSkill:    boolPtr(roll.HasSkill),
		Die:         roll.Die,
		Total:       intPtr(roll.Result.Total),
		TargetValue: intPtr(packet.HaggleTargetValue),
		Success:     boolPtr(success),
	}); err != nil {
		return HaggleAttemptResult{}, err
	}

	if _, err := actions.StoreGameEventTrusted(ctx, pool, actions.GameEventRequest{
		SessionID:       eligible.SessionID,
		ActorID:         actorUserID,
		EventKind:       "interaction/haggle_attempted",
		CharacterCardID: eligible.CharacterCardID,
		Detail: map[string]any{
			"interaction_id": interactionID,
			"packet_slug":    packet.Slug,
			"has_skill":      roll.HasSkill,
			"die":            roll.Die,
			"total":          roll.Result.Total,
			"target_value":   packet.HaggleTargetValue,
			"success":        success,
		},
	}); err != nil {
		return HaggleAttemptResult{}, err
	}

	return HaggleAttemptResult{
		Die:         roll.Die,
		Total:       roll.Result.Total,
		TargetValue: packet.HaggleTargetValue,
		Success:     success,
		Text:        text,
	}, nil
}

// --- Purchase --------------------------------------------------------------

// AttemptPurchase resolves eligibility fresh (so the Character purchased
// onto is whichever is currently selected AT THE MOMENT OF PURCHASE, spec
// S2.10 -- never the Character that was selected when Equip Mode opened),
// then delegates to PurchaseEquipment and records the participation event.
func AttemptPurchase(ctx context.Context, pool *pgxpool.Pool, actorUserID, interactionID, equipmentItemID, idempotencyKey string) (InventoryEntry, error) {
	eligible, err := ResolveEligibleContext(ctx, pool, actorUserID, interactionID)
	if err != nil {
		return InventoryEntry{}, err
	}

	entry, err := PurchaseEquipment(ctx, pool, PurchaseInput{
		ActorUserID:            actorUserID,
		CharacterCardID:        eligible.CharacterCardID,
		EquipmentItemID:        equipmentItemID,
		IdempotencyKey:         idempotencyKey,
		SourceShowRunID:        eligible.ShowRunID,
		SourceShowID:           eligible.ShowID,
		SourceSessionID:        eligible.SessionID,
		SourceScenePlacementID: eligible.PlacementID,
		SourceInteractionKey:   interactionID,
	})
	if err != nil {
		return InventoryEntry{}, err
	}

	itemName := ""
	if entry.Item != nil {
		itemName = entry.Item.Name
	}
	if _, err := actions.StoreGameEventTrusted(ctx, pool, actions.GameEventRequest{
		SessionID:       eligible.SessionID,
		ActorID:         actorUserID,
		EventKind:       "interaction/equipment_acquired",
		CharacterCardID: eligible.CharacterCardID,
		Detail: map[string]any{
			"interaction_id":    interactionID,
			"equipment_item_id": equipmentItemID,
			"item_name":         itemName,
			"quantity":          entry.Quantity,
		},
	}); err != nil {
		return InventoryEntry{}, err
	}

	return entry, nil
}

// attemptPurchaseWithSession is AttemptPurchase plus the resolved
// SessionID, used only by the HTTP layer to target the post-purchase
// invalidation push at the correct session+user pair.
func attemptPurchaseWithSession(ctx context.Context, pool *pgxpool.Pool, actorUserID, interactionID, equipmentItemID, idempotencyKey string) (string, InventoryEntry, error) {
	eligible, err := ResolveEligibleContext(ctx, pool, actorUserID, interactionID)
	if err != nil {
		return "", InventoryEntry{}, err
	}
	entry, err := AttemptPurchase(ctx, pool, actorUserID, interactionID, equipmentItemID, idempotencyKey)
	if err != nil {
		return "", InventoryEntry{}, err
	}
	return eligible.SessionID, entry, nil
}
