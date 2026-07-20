package characters

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"victory/backend/internal/access"
	"victory/backend/internal/sessions"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const DraftCapability = "character_card:draft"

const maxSheetLinks = 12

var maxCharacterCardsPerAccount = 50

func SetMaxCharacterCardsPerAccount(limit int) {
	if limit < 1 {
		return
	}
	maxCharacterCardsPerAccount = limit
}

type SheetLink struct {
	ID        string `json:"id"`
	Ruleset   string `json:"ruleset"`
	SheetType string `json:"sheet_type"`
	Label     string `json:"label"`
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"`
}

type CharacterCard struct {
	ID                string         `json:"id"`
	OwnerUserID       string         `json:"owner_user_id"`
	LocationID        string         `json:"location_id"`
	ProductionID      string         `json:"production_id,omitempty"`
	WorkbookStatus    string         `json:"workbook_status,omitempty"`
	WorkbookContext   map[string]any `json:"workbook_context,omitempty"`
	Name              string         `json:"name"`
	Pronouns          string         `json:"pronouns"`
	PortraitURL       string         `json:"portrait_url"`
	TokenAura         string         `json:"token_aura,omitempty"`
	Color             string         `json:"color,omitempty"`
	Tagline           string         `json:"tagline"`
	PublicDescription string         `json:"public_description"`
	PrivateNotes      string         `json:"private_notes,omitempty"`
	SheetLinks        []SheetLink    `json:"sheet_links"`
	CreatedAt         string         `json:"created_at"`
	UpdatedAt         string         `json:"updated_at"`
}

type CharacterCardInput struct {
	Name              string         `json:"name"`
	Pronouns          string         `json:"pronouns"`
	PortraitURL       string         `json:"portrait_url"`
	TokenAura         string         `json:"token_aura,omitempty"`
	Aura              string         `json:"aura,omitempty"`
	Color             string         `json:"color,omitempty"`
	Tagline           string         `json:"tagline"`
	PublicDescription string         `json:"public_description"`
	PrivateNotes      string         `json:"private_notes"`
	SheetLinks        *[]SheetLink   `json:"sheet_links,omitempty"`
	WorkbookStatus    string         `json:"workbook_status,omitempty"`
	WorkbookContext   map[string]any `json:"workbook_context,omitempty"`
}

type PermissionInput struct {
	TargetUserID string `json:"target_user_id"`
	ProductionID string `json:"production_id"`
}

type response struct {
	Ok   bool `json:"ok"`
	Data any  `json:"data,omitempty"`
}

type characterQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func HandleMyCharacterCards(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false, Data: map[string]any{"error": "method_not_allowed"}})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := requireUser(ctx, pool, r)
		if err != nil {
			writeAuthError(w, err)
			return
		}

		canDraft, err := CanDraftCharacter(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, response{Ok: false, Data: map[string]any{"error": "permission_lookup_failed"}})
			return
		}

		cards, err := ListOwnedCards(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, response{Ok: false, Data: map[string]any{"error": "character_cards_lookup_failed"}})
			return
		}

		persona, _ := ActivePersonaForLatestCaveSession(ctx, pool, userID)
		activeCharacter, _ := ActiveCharacterForUser(ctx, pool, userID)

		writeJSON(w, http.StatusOK, response{Ok: true, Data: map[string]any{
			"can_draft":        canDraft,
			"cards":            cards,
			"active_persona":   persona,
			"active_character": activeCharacter,
		}})
	}
}

func HandleParentageChart() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false, Data: map[string]any{"error": "method_not_allowed"}})
			return
		}

		writeJSON(w, http.StatusOK, response{Ok: true, Data: map[string]any{
			"version": ParentageChartVersionV11,
			"entries": ParentageChartV11,
		}})
	}
}

func HandleChapter2Rules() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false, Data: map[string]any{"error": "method_not_allowed"}})
			return
		}
		writeJSON(w, http.StatusOK, response{Ok: true, Data: map[string]any{
			"version":       Chapter2Version,
			"starting_fp":   Chapter2StartingFP,
			"attribute_cap": Chapter2AttributeCap,
			"attributes":    AllChapter2Attributes,
			"ratings":       AttributeScaleRatings,
			"stages":        Chapter2Rules,
		}})
	}
}

func HandleChapter2Roll(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false, Data: map[string]any{"error": "method_not_allowed"}})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireUser(ctx, pool, r)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		var body struct {
			CharacterCardID string `json:"character_card_id"`
			StageNumber     int    `json:"stage_number"`
			DieType         string `json:"die_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": "invalid_json"}})
			return
		}
		allowed, err := CanEditCard(ctx, pool, userID, strings.TrimSpace(body.CharacterCardID))
		if err != nil {
			writeCharacterError(w, err)
			return
		}
		if !allowed {
			writeJSON(w, http.StatusForbidden, response{Ok: false, Data: map[string]any{"error": "forbidden"}})
			return
		}
		value, err := RequestChapter2Roll(ctx, pool, userID, strings.TrimSpace(body.CharacterCardID), body.StageNumber, strings.TrimSpace(body.DieType))
		if err != nil {
			writeCharacterError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, response{Ok: true, Data: map[string]any{
			"stage_number": body.StageNumber,
			"die_type":     body.DieType,
			"value":        value,
		}})
	}
}

func HandleChapter2RollStatus(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false, Data: map[string]any{"error": "method_not_allowed"}})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireUser(ctx, pool, r)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		cardID := strings.TrimSpace(r.URL.Query().Get("character_card_id"))
		stageNumber, _ := strconv.Atoi(r.URL.Query().Get("stage_number"))
		allowed, err := CanEditCard(ctx, pool, userID, cardID)
		if err != nil {
			writeCharacterError(w, err)
			return
		}
		if !allowed {
			writeJSON(w, http.StatusForbidden, response{Ok: false, Data: map[string]any{"error": "forbidden"}})
			return
		}
		rolls, err := LoadChapter2StageRolls(ctx, pool, userID, cardID, stageNumber)
		if err != nil {
			writeCharacterError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, response{Ok: true, Data: map[string]any{"rolls": rolls}})
	}
}

func HandleChapter2Stage(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false, Data: map[string]any{"error": "method_not_allowed"}})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireUser(ctx, pool, r)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		var input Chapter2StageInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": "invalid_json"}})
			return
		}
		result, newContext, err := CommitChapter2Stage(ctx, pool, userID, input)
		if err != nil {
			writeCharacterError(w, err)
			return
		}
		// Persist the updated workbook context.
		wbContextJSON, merr := json.Marshal(newContext)
		if merr == nil {
			_, _ = pool.Exec(ctx, `
				UPDATE character_cards SET workbook_context = $2::jsonb, updated_at = NOW()
				WHERE id = $1 AND is_deleted = FALSE
			`, strings.TrimSpace(input.CharacterCardID), string(wbContextJSON))
		}
		// Update the module stage tracking.
		currentEvent := fmt.Sprintf("chapter2_stage_%d", result.StageNumber+1)
		if result.NextStage == 0 {
			currentEvent = "chapter2_complete"
		}
		_, _ = pool.Exec(ctx, `
			UPDATE character_workbook_modules
			SET current_stage = $2, current_event = $3, module_context = $4::jsonb, updated_at = NOW()
			WHERE character_card_id = $1 AND ruleset_key = 'socio'
		`, strings.TrimSpace(input.CharacterCardID), 2, currentEvent, string(wbContextJSON))

		writeJSON(w, http.StatusOK, response{Ok: true, Data: result})
	}
}

func HandleChapter3Archetypes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false, Data: map[string]any{"error": "method_not_allowed"}})
			return
		}
		writeJSON(w, http.StatusOK, response{Ok: true, Data: map[string]any{
			"dataset_version": Chapter3ArchetypeDatasetVersion,
			"ruleset_version": Chapter3RulesetVersion,
			"archetypes":      Chapter3Archetypes,
		}})
	}
}

func HandleChapter3Confirm(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false, Data: map[string]any{"error": "method_not_allowed"}})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireUser(ctx, pool, r)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		var body struct {
			CharacterCardID string `json:"character_card_id"`
			ArchetypeKey    string `json:"archetype_key"`
			Override        bool   `json:"override"`
			CustomArchetype *struct {
				Name               string `json:"name"`
				Summary            string `json:"summary"`
				PrimaryAttribute   string `json:"primary_attribute"`
				SecondaryAttribute string `json:"secondary_attribute"`
				MechanicalEffect   string `json:"mechanical_effect"`
			} `json:"custom_archetype"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": "invalid_json"}})
			return
		}
		var custom *Chapter3CustomArchetypeInput
		if body.CustomArchetype != nil {
			custom = &Chapter3CustomArchetypeInput{
				Name:               body.CustomArchetype.Name,
				Summary:            body.CustomArchetype.Summary,
				PrimaryAttribute:   body.CustomArchetype.PrimaryAttribute,
				SecondaryAttribute: body.CustomArchetype.SecondaryAttribute,
				MechanicalEffect:   body.CustomArchetype.MechanicalEffect,
			}
		}
		result, _, err := CommitChapter3Archetype(ctx, pool, userID, body.CharacterCardID, body.ArchetypeKey, custom, body.Override)
		if err != nil {
			writeCharacterError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, response{Ok: true, Data: result})
	}
}

func HandleChapter4Group(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false, Data: map[string]any{"error": "method_not_allowed"}})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireUser(ctx, pool, r)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		cardID := strings.TrimSpace(r.URL.Query().Get("character_card_id"))
		result, err := ResolveChapter4Group(ctx, pool, userID, cardID)
		if err != nil {
			writeCharacterError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, response{Ok: true, Data: result})
	}
}

func HandleChapter4Confirm(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false, Data: map[string]any{"error": "method_not_allowed"}})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireUser(ctx, pool, r)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		var body struct {
			CharacterCardID string `json:"character_card_id"`
			SkillID         string `json:"skill_id"`
			Override        bool   `json:"override"`
			CustomSkill     *struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"custom_skill"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": "invalid_json"}})
			return
		}
		var custom *Chapter4CustomSkillInput
		if body.CustomSkill != nil {
			custom = &Chapter4CustomSkillInput{
				Name:        body.CustomSkill.Name,
				Description: body.CustomSkill.Description,
			}
		}
		result, _, err := CommitChapter4FirstSkill(ctx, pool, userID, body.CharacterCardID, body.SkillID, custom, body.Override)
		if err != nil {
			writeCharacterError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, response{Ok: true, Data: result})
	}
}

func HandleChapter4Courtyard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false, Data: map[string]any{"error": "method_not_allowed"}})
			return
		}
		writeJSON(w, http.StatusOK, response{Ok: true, Data: LoadChapter4CourtyardScene()})
	}
}

func HandleCatharsisCoinFlip(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false, Data: map[string]any{"error": "method_not_allowed"}})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		userID, err := requireUser(ctx, pool, r)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		var body struct {
			CharacterCardID string `json:"character_card_id"`
			ParentIndex     int    `json:"parent_index"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": "invalid_json"}})
			return
		}
		allowed, err := CanEditCard(ctx, pool, userID, strings.TrimSpace(body.CharacterCardID))
		if err != nil {
			writeCharacterError(w, err)
			return
		}
		if !allowed {
			writeJSON(w, http.StatusForbidden, response{Ok: false, Data: map[string]any{"error": "forbidden"}})
			return
		}
		result, err := RequestCatharsisCoinFlip(ctx, pool, userID, strings.TrimSpace(body.CharacterCardID), body.ParentIndex)
		if err != nil {
			writeCharacterError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, response{Ok: true, Data: result})
	}
}

func HandleRequestParentageRoll(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false, Data: map[string]any{"error": "method_not_allowed"}})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireUser(ctx, pool, r)
		if err != nil {
			writeAuthError(w, err)
			return
		}

		var body struct {
			DraftToken string `json:"draft_token"`
			EventKey   string `json:"event_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": "invalid_json"}})
			return
		}

		canDraft, err := CanDraftCharacter(ctx, pool, userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, response{Ok: false, Data: map[string]any{"error": "permission_lookup_failed"}})
			return
		}
		if !canDraft {
			writeJSON(w, http.StatusForbidden, response{Ok: false, Data: map[string]any{"error": "character_draft_permission_required"}})
			return
		}

		result, err := RequestCatharsisParentageRoll(ctx, pool, userID, body.DraftToken, body.EventKey)
		if err != nil {
			writeCharacterError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, response{Ok: true, Data: result})
	}
}

func HandleCreateCharacterCard(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false, Data: map[string]any{"error": "method_not_allowed"}})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireUser(ctx, pool, r)
		if err != nil {
			writeAuthError(w, err)
			return
		}

		var input CharacterCardInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": "invalid_json"}})
			return
		}

		card, err := CreateCard(ctx, pool, userID, input)
		if err != nil {
			writeCharacterError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, response{Ok: true, Data: card})
	}
}

func HandleCharacterCardByID(pool *pgxpool.Pool, notifiers ...ProjectionChangeNotifier) http.HandlerFunc {
	var notify ProjectionChangeNotifier
	if len(notifiers) > 0 {
		notify = notifiers[0]
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch && r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false, Data: map[string]any{"error": "method_not_allowed"}})
			return
		}

		path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/character-cards/"))
		if strings.HasSuffix(path, "/activate") {
			if r.Method != http.MethodPost {
				writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false, Data: map[string]any{"error": "method_not_allowed"}})
				return
			}
			cardID := strings.TrimSpace(strings.TrimSuffix(path, "/activate"))
			if cardID == "" {
				writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": "character_card_id_required"}})
				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()

			userID, err := requireUser(ctx, pool, r)
			if err != nil {
				writeAuthError(w, err)
				return
			}

			card, err := ActivateCharacterCard(ctx, pool, userID, cardID)
			if err != nil {
				writeCharacterError(w, err)
				return
			}
			if notify != nil {
				notify(ctx, cardID, []string{"active_character"}, "")
			}

			writeJSON(w, http.StatusOK, response{Ok: true, Data: card})
			return
		}

		cardID := path
		if cardID == "" || cardID == "me" {
			writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": "character_card_id_required"}})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireUser(ctx, pool, r)
		if err != nil {
			writeAuthError(w, err)
			return
		}

		var input CharacterCardInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": "invalid_json"}})
			return
		}

		card, err := UpdateCard(ctx, pool, userID, cardID, input)
		if err != nil {
			writeCharacterError(w, err)
			return
		}
		if notify != nil {
			notify(ctx, cardID, []string{"face", "identity"}, "")
		}

		writeJSON(w, http.StatusOK, response{Ok: true, Data: card})
	}
}

func ActivateCharacterCard(ctx context.Context, pool *pgxpool.Pool, userID, cardID string) (map[string]any, error) {
	userID = strings.TrimSpace(userID)
	cardID = strings.TrimSpace(cardID)
	if userID == "" {
		return nil, errors.New("not_authenticated")
	}
	if cardID == "" {
		return nil, errors.New("character_card_id_required")
	}

	var ownerID, workbookStatus string
	if err := pool.QueryRow(ctx, `
		SELECT owner_user_id::text, workbook_status
		FROM character_cards
		WHERE id = $1
		  AND is_deleted = FALSE
		LIMIT 1
	`, cardID).Scan(&ownerID, &workbookStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("character_card_not_found")
		}
		return nil, err
	}
	if ownerID != userID {
		return nil, errors.New("forbidden")
	}
	if strings.ToLower(strings.TrimSpace(workbookStatus)) != "complete" {
		return nil, errors.New("completed_character_required")
	}

	if err := setActiveCharacter(ctx, pool, userID, cardID); err != nil {
		return nil, err
	}

	activeCharacter, err := ActiveCharacterForUser(ctx, pool, userID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"active_character": activeCharacter}, nil
}

func HandleGrantCharacterPermission(pool *pgxpool.Pool) http.HandlerFunc {
	return permissionHandler(pool, false)
}

func HandleRevokeCharacterPermission(pool *pgxpool.Pool) http.HandlerFunc {
	return permissionHandler(pool, true)
}

func permissionHandler(pool *pgxpool.Pool, revoke bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false, Data: map[string]any{"error": "method_not_allowed"}})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireUser(ctx, pool, r)
		if err != nil {
			writeAuthError(w, err)
			return
		}

		var input PermissionInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": "invalid_json"}})
			return
		}

		input.TargetUserID = strings.TrimSpace(input.TargetUserID)
		input.ProductionID = strings.TrimSpace(input.ProductionID)
		if input.TargetUserID == "" {
			writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": "target_user_id_required"}})
			return
		}

		var out map[string]any
		if revoke {
			out, err = RevokeDraftPermission(ctx, pool, userID, input.TargetUserID, input.ProductionID)
		} else {
			out, err = GrantDraftPermission(ctx, pool, userID, input.TargetUserID, input.ProductionID)
		}
		if err != nil {
			writeCharacterError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, response{Ok: true, Data: out})
	}
}

func CanDraftCharacter(ctx context.Context, q characterQuerier, userID string) (bool, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false, nil
	}

	if ok, err := hasImplicitDraftRole(ctx, q, userID); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	return false, nil
}

func CreateCard(ctx context.Context, pool *pgxpool.Pool, ownerUserID string, input CharacterCardInput) (CharacterCard, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return CharacterCard{}, errors.New("not_authenticated")
	}

	canDraft, err := CanDraftCharacter(ctx, pool, ownerUserID)
	if err != nil {
		return CharacterCard{}, err
	}
	if !canDraft {
		return CharacterCard{}, errors.New("character_draft_permission_required")
	}

	input = sanitizeInput(input)
	if input.TokenAura == "__invalid__" {
		return CharacterCard{}, errors.New("invalid_token_aura")
	}
	if strings.ToLower(strings.TrimSpace(stringValue(input.WorkbookContext["source"]))) == "catharsis" {
		resolved, err := attachCanonicalCatharsisParentageRows(ctx, pool, ownerUserID, input)
		if err != nil {
			return CharacterCard{}, err
		}
		input = resolved
	}
	input = seedCatharsisStarterDraft(input)

	locationID, productionID, err := resolveCharacterScope(ctx, pool, ownerUserID)
	if err != nil {
		return CharacterCard{}, err
	}

	workbookStatus := strings.TrimSpace(input.WorkbookStatus)
	if workbookStatus == "" {
		workbookStatus = "draft"
	}
	workbookContextJSON, err := json.Marshal(normalizeWorkbookContext(input.WorkbookContext))
	if err != nil {
		return CharacterCard{}, err
	}
	sheetLinksJSON, err := json.Marshal(sheetLinksFromInput(input))
	if err != nil {
		return CharacterCard{}, err
	}

	if existing, ok, err := findDraftCharacterForContext(ctx, pool, ownerUserID, workbookContextJSON); err != nil {
		return CharacterCard{}, err
	} else if ok {
		if err := ensureWorkbookModule(ctx, pool, existing.ID, existing.WorkbookContext, existing.WorkbookStatus); err != nil {
			return CharacterCard{}, err
		}
		return existing, nil
	}

	if ok, err := canCreateCharacterCard(ctx, pool, ownerUserID); err != nil {
		return CharacterCard{}, err
	} else if !ok {
		return CharacterCard{}, errors.New("character_card_limit_reached")
	}

	if strings.TrimSpace(input.Name) == "" {
		defaultName, err := nextUnusedRomanNumeralName(ctx, pool, ownerUserID)
		if err != nil {
			return CharacterCard{}, err
		}
		input.Name = defaultName
	}

	var card CharacterCard
	var createdAt, updatedAt time.Time
	var sheetLinksRaw []byte
	var workbookContextRaw []byte
	err = pool.QueryRow(ctx, `
		INSERT INTO character_cards (
			owner_user_id,
			location_id,
			production_id,
			workbook_status,
			workbook_context,
			name,
			pronouns,
			portrait_url,
			token_aura,
			color,
			tagline,
			public_description,
			private_notes,
			sheet_links
		)
		VALUES ($1, $2, NULLIF($3, '')::uuid, $4, $5::jsonb, $6, $7, $8, NULLIF($9, ''), $10, $11, $12, $13, $14::jsonb)
		RETURNING id::text, owner_user_id::text, location_id::text, COALESCE(production_id::text, ''), workbook_status, workbook_context, name, pronouns, portrait_url, COALESCE(token_aura, ''), color, tagline, public_description, private_notes, sheet_links, created_at, updated_at
	`, ownerUserID, locationID, productionID, workbookStatus, string(workbookContextJSON), input.Name, input.Pronouns, input.PortraitURL, input.TokenAura, input.Color, input.Tagline, input.PublicDescription, input.PrivateNotes, string(sheetLinksJSON)).
		Scan(&card.ID, &card.OwnerUserID, &card.LocationID, &card.ProductionID, &card.WorkbookStatus, &workbookContextRaw, &card.Name, &card.Pronouns, &card.PortraitURL, &card.TokenAura, &card.Color, &card.Tagline, &card.PublicDescription, &card.PrivateNotes, &sheetLinksRaw, &createdAt, &updatedAt)
	if err != nil {
		return CharacterCard{}, err
	}

	card.WorkbookContext = decodeJSONMap(workbookContextRaw)
	if err := finishCharacterCard(&card, sheetLinksRaw, createdAt, updatedAt); err != nil {
		return CharacterCard{}, err
	}
	if err := ensureWorkbookModule(ctx, pool, card.ID, card.WorkbookContext, workbookStatus); err != nil {
		return CharacterCard{}, err
	}
	return card, nil
}

func UpdateCard(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID string, input CharacterCardInput) (CharacterCard, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	cardID = strings.TrimSpace(cardID)
	if actorUserID == "" {
		return CharacterCard{}, errors.New("not_authenticated")
	}
	if cardID == "" {
		return CharacterCard{}, errors.New("character_card_id_required")
	}

	input = sanitizeInput(input)
	if input.TokenAura == "__invalid__" {
		return CharacterCard{}, errors.New("invalid_token_aura")
	}
	if input.Name == "" {
		return CharacterCard{}, errors.New("character_name_required")
	}

	allowed, err := CanEditCard(ctx, pool, actorUserID, cardID)
	if err != nil {
		return CharacterCard{}, err
	}
	if !allowed {
		return CharacterCard{}, errors.New("forbidden")
	}

	if err := rejectLockedFaceValueChanges(ctx, pool, cardID, input); err != nil {
		return CharacterCard{}, err
	}

	var sheetLinksParam any
	if input.SheetLinks != nil {
		sheetLinksJSON, err := json.Marshal(*input.SheetLinks)
		if err != nil {
			return CharacterCard{}, err
		}
		sheetLinksParam = string(sheetLinksJSON)
	}

	var workbookStatusParam any
	if trimmed := strings.TrimSpace(input.WorkbookStatus); trimmed != "" {
		workbookStatusParam = trimmed
	}

	var workbookContextParam any
	if input.WorkbookContext != nil {
		workbookContextJSON, err := json.Marshal(normalizeWorkbookContext(input.WorkbookContext))
		if err != nil {
			return CharacterCard{}, err
		}
		workbookContextParam = string(workbookContextJSON)
	}

	var card CharacterCard
	var createdAt, updatedAt time.Time
	var sheetLinksRaw []byte
	var workbookContextRaw []byte
	err = pool.QueryRow(ctx, `
		UPDATE character_cards
		SET name = $2,
		    pronouns = $3,
		    portrait_url = $4,
		    token_aura = NULLIF($5, ''),
		    color = $6,
		    tagline = $7,
		    public_description = $8,
		    private_notes = $9,
		    sheet_links = COALESCE($10::jsonb, sheet_links),
		    workbook_status = COALESCE($11, workbook_status),
		    workbook_context = COALESCE($12::jsonb, workbook_context),
		    updated_at = NOW()
		WHERE id = $1
		  AND is_deleted = FALSE
		RETURNING id::text, owner_user_id::text, location_id::text, COALESCE(production_id::text, ''), workbook_status, workbook_context, name, pronouns, portrait_url, COALESCE(token_aura, ''), color, tagline, public_description, private_notes, sheet_links, created_at, updated_at
	`, cardID, input.Name, input.Pronouns, input.PortraitURL, input.TokenAura, input.Color, input.Tagline, input.PublicDescription, input.PrivateNotes, sheetLinksParam, workbookStatusParam, workbookContextParam).
		Scan(&card.ID, &card.OwnerUserID, &card.LocationID, &card.ProductionID, &card.WorkbookStatus, &workbookContextRaw, &card.Name, &card.Pronouns, &card.PortraitURL, &card.TokenAura, &card.Color, &card.Tagline, &card.PublicDescription, &card.PrivateNotes, &sheetLinksRaw, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CharacterCard{}, errors.New("character_card_not_found")
		}
		return CharacterCard{}, err
	}

	card.WorkbookContext = decodeJSONMap(workbookContextRaw)
	if err := finishCharacterCard(&card, sheetLinksRaw, createdAt, updatedAt); err != nil {
		return CharacterCard{}, err
	}
	return card, nil
}

func rejectLockedFaceValueChanges(ctx context.Context, pool *pgxpool.Pool, cardID string, input CharacterCardInput) error {
	overrides, err := loadFaceOverrides(ctx, pool, cardID)
	if err != nil {
		return err
	}
	hasLocked := false
	for _, o := range overrides {
		if o.ValueLocked {
			hasLocked = true
			break
		}
	}
	if !hasLocked {
		return nil
	}

	var current struct {
		Name, Pronouns, PortraitURL, TokenAura, Color, Tagline, PublicDescription string
	}
	if err := pool.QueryRow(ctx, `
		SELECT name, pronouns, portrait_url, COALESCE(token_aura, ''), color, tagline, public_description
		FROM character_cards
		WHERE id = $1 AND is_deleted = FALSE
	`, cardID).Scan(&current.Name, &current.Pronouns, &current.PortraitURL, &current.TokenAura, &current.Color, &current.Tagline, &current.PublicDescription); err != nil {
		return err
	}

	check := func(key, before, after string) error {
		if o, ok := overrides[key]; ok && o.ValueLocked && strings.TrimSpace(before) != strings.TrimSpace(after) {
			return errors.New("face_value_locked")
		}
		return nil
	}
	if err := check("name", current.Name, input.Name); err != nil {
		return err
	}
	if err := check("pronouns", current.Pronouns, input.Pronouns); err != nil {
		return err
	}
	if err := check("portrait_url", current.PortraitURL, input.PortraitURL); err != nil {
		return err
	}
	if err := check("token_aura", current.TokenAura, input.TokenAura); err != nil {
		return err
	}
	if err := check("tagline", current.Tagline, input.Tagline); err != nil {
		return err
	}
	if err := check("public_description", current.PublicDescription, input.PublicDescription); err != nil {
		return err
	}
	return nil
}

// LoadCardForActor loads a character card for actorUserID, enforcing the same
// edit authority as UpdateCard. UpdateCard has no partial-patch path (it always
// overwrites every column), so callers that want to change a single field must
// load the current card first, patch just that field, and pass every other
// field back through unchanged.
func LoadCardForActor(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID string) (CharacterCard, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	cardID = strings.TrimSpace(cardID)
	if actorUserID == "" {
		return CharacterCard{}, errors.New("not_authenticated")
	}
	if cardID == "" {
		return CharacterCard{}, errors.New("character_card_id_required")
	}

	allowed, err := CanEditCard(ctx, pool, actorUserID, cardID)
	if err != nil {
		return CharacterCard{}, err
	}
	if !allowed {
		return CharacterCard{}, errors.New("forbidden")
	}

	var card CharacterCard
	var createdAt, updatedAt time.Time
	var workbookContextRaw, sheetLinksRaw []byte
	err = pool.QueryRow(ctx, `
		SELECT id::text, owner_user_id::text, location_id::text, COALESCE(production_id::text, ''), workbook_status, workbook_context, name, pronouns, portrait_url, COALESCE(token_aura, ''), color, tagline, public_description, private_notes, sheet_links, created_at, updated_at
		FROM character_cards
		WHERE id = $1
		  AND is_deleted = FALSE
		LIMIT 1
	`, cardID).Scan(&card.ID, &card.OwnerUserID, &card.LocationID, &card.ProductionID, &card.WorkbookStatus, &workbookContextRaw, &card.Name, &card.Pronouns, &card.PortraitURL, &card.TokenAura, &card.Color, &card.Tagline, &card.PublicDescription, &card.PrivateNotes, &sheetLinksRaw, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CharacterCard{}, errors.New("character_card_not_found")
		}
		return CharacterCard{}, err
	}
	card.WorkbookContext = decodeJSONMap(workbookContextRaw)
	if err := finishCharacterCard(&card, sheetLinksRaw, createdAt, updatedAt); err != nil {
		return CharacterCard{}, err
	}
	return card, nil
}

func CanEditCard(ctx context.Context, q characterQuerier, actorUserID, cardID string) (bool, error) {
	var ownerUserID, locationID string
	err := q.QueryRow(ctx, `
		SELECT owner_user_id::text, location_id::text
		FROM character_cards
		WHERE id = $1
		  AND is_deleted = FALSE
		LIMIT 1
	`, cardID).Scan(&ownerUserID, &locationID)
	if err != nil {
		return false, err
	}

	if actorUserID == ownerUserID {
		return CanDraftCharacter(ctx, q, actorUserID)
	}

	return hasAuthorityInLocation(ctx, q, actorUserID, locationID)
}

func ListOwnedCards(ctx context.Context, pool *pgxpool.Pool, userID string) ([]CharacterCard, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, owner_user_id::text, location_id::text, COALESCE(production_id::text, ''), workbook_status, workbook_context, name, pronouns, portrait_url, COALESCE(token_aura, ''), color, tagline, public_description, private_notes, sheet_links, created_at, updated_at
		FROM character_cards
		WHERE owner_user_id = $1
		  AND is_deleted = FALSE
		ORDER BY updated_at DESC, created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []CharacterCard{}
	for rows.Next() {
		var card CharacterCard
		var createdAt, updatedAt time.Time
		var sheetLinksRaw []byte
		var workbookContextRaw []byte
		if err := rows.Scan(&card.ID, &card.OwnerUserID, &card.LocationID, &card.ProductionID, &card.WorkbookStatus, &workbookContextRaw, &card.Name, &card.Pronouns, &card.PortraitURL, &card.TokenAura, &card.Color, &card.Tagline, &card.PublicDescription, &card.PrivateNotes, &sheetLinksRaw, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		card.WorkbookContext = decodeJSONMap(workbookContextRaw)
		if err := finishCharacterCard(&card, sheetLinksRaw, createdAt, updatedAt); err != nil {
			return nil, err
		}
		out = append(out, card)
	}
	return out, rows.Err()
}

func countOwnedCharacterCards(ctx context.Context, q characterQuerier, userID string) (int, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return 0, errors.New("not_authenticated")
	}

	var count int
	if err := q.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM character_cards
		WHERE owner_user_id = $1
		  AND is_deleted = FALSE
	`, userID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func canCreateCharacterCard(ctx context.Context, q characterQuerier, userID string) (bool, error) {
	if limit := maxCharacterCardsPerAccount; limit > 0 {
		count, err := countOwnedCharacterCards(ctx, q, userID)
		if err != nil {
			return false, err
		}
		if count >= limit {
			return false, nil
		}
	}
	return true, nil
}

func PersonaForCard(ctx context.Context, q characterQuerier, userID, cardID string) (map[string]any, error) {
	var ownerUserID, id, name, pronouns, portraitURL, tokenAura, color, tagline string
	err := q.QueryRow(ctx, `
		SELECT owner_user_id::text, id::text, name, pronouns, portrait_url, COALESCE(token_aura, ''), color, tagline
		FROM character_cards
		WHERE id = $1
		  AND is_deleted = FALSE
		LIMIT 1
	`, cardID).Scan(&ownerUserID, &id, &name, &pronouns, &portraitURL, &tokenAura, &color, &tagline)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(userID) != "" && ownerUserID != strings.TrimSpace(userID) {
		return nil, errors.New("character_card_not_owned")
	}

	return map[string]any{
		"character_card_id": id,
		"name":              name,
		"display_name":      name,
		"pronouns":          pronouns,
		"portrait_url":      portraitURL,
		"token_aura":        tokenAura,
		"color":             color,
		"tagline":           tagline,
	}, nil
}

func ActivePersonaForLatestCaveSession(ctx context.Context, q characterQuerier, userID string) (map[string]any, error) {
	var sessionID string
	err := q.QueryRow(ctx, `
		SELECT s.id::text
		FROM sessions s
		JOIN venues v ON v.id = s.venue_id
		JOIN session_participants sp ON sp.session_id = s.id
		WHERE v.slug = 'the-cave'
		  AND s.status IN ('rehearsal', 'live')
		  AND sp.user_id = $1
		ORDER BY s.started_at DESC
		LIMIT 1
	`, userID).Scan(&sessionID)
	if err != nil {
		return nil, err
	}
	return ActivePersonaForSession(ctx, q, sessionID, userID)
}

func ActivePersonaForSession(ctx context.Context, q characterQuerier, sessionID, userID string) (map[string]any, error) {
	var id, name, pronouns, portraitURL, tokenAura, color, tagline string
	err := q.QueryRow(ctx, `
		SELECT cc.id::text, cc.name, cc.pronouns, cc.portrait_url, COALESCE(cc.token_aura, ''), cc.color, cc.tagline
		FROM current_session_personas csp
		JOIN character_cards cc ON cc.id = csp.character_card_id
		WHERE csp.session_id = $1
		  AND csp.user_id = $2
		  AND cc.is_deleted = FALSE
		LIMIT 1
	`, sessionID, userID).Scan(&id, &name, &pronouns, &portraitURL, &tokenAura, &color, &tagline)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"character_card_id": id,
		"name":              name,
		"display_name":      name,
		"pronouns":          pronouns,
		"portrait_url":      portraitURL,
		"token_aura":        tokenAura,
		"color":             color,
		"tagline":           tagline,
	}, nil
}

func GrantDraftPermission(ctx context.Context, pool *pgxpool.Pool, granterUserID, targetUserID, productionID string) (map[string]any, error) {
	locationID, err := authorityLocation(ctx, pool, granterUserID)
	if err != nil {
		return nil, err
	}

	allowed, err := hasAuthorityInLocation(ctx, pool, granterUserID, locationID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errors.New("forbidden")
	}

	if ok, err := targetCanReceiveDraftGrant(ctx, pool, targetUserID, locationID); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("target_must_be_cast_or_crew")
	}

	if productionID == "" {
		productionID, _ = firstProductionForLocation(ctx, pool, locationID)
	}

	var grantID string
	err = pool.QueryRow(ctx, `
		INSERT INTO permission_grants (location_id, grantee_user_id, capability, production_id, granted_by_user_id)
		VALUES ($1, $2, $3, NULLIF($4, '')::uuid, $5)
		ON CONFLICT (location_id, grantee_user_id, capability, COALESCE(production_id, '00000000-0000-0000-0000-000000000000'::uuid), COALESCE(venue_id, '00000000-0000-0000-0000-000000000000'::uuid))
		WHERE revoked_at IS NULL
		DO UPDATE SET revoked_at = NULL
		RETURNING id::text
	`, locationID, targetUserID, DraftCapability, productionID, granterUserID).Scan(&grantID)
	if err != nil {
		return nil, err
	}

	return map[string]any{"grant_id": grantID, "target_user_id": targetUserID, "capability": DraftCapability}, nil
}

func RevokeDraftPermission(ctx context.Context, pool *pgxpool.Pool, revokerUserID, targetUserID, productionID string) (map[string]any, error) {
	locationID, err := authorityLocation(ctx, pool, revokerUserID)
	if err != nil {
		return nil, err
	}

	allowed, err := hasAuthorityInLocation(ctx, pool, revokerUserID, locationID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errors.New("forbidden")
	}

	tag, err := pool.Exec(ctx, `
		UPDATE permission_grants
		SET revoked_at = NOW()
		WHERE location_id = $1
		  AND grantee_user_id = $2
		  AND capability = $3
		  AND revoked_at IS NULL
		  AND ($4 = '' OR production_id = $4::uuid)
	`, locationID, targetUserID, DraftCapability, strings.TrimSpace(productionID))
	if err != nil {
		return nil, err
	}

	return map[string]any{"target_user_id": targetUserID, "capability": DraftCapability, "revoked": tag.RowsAffected()}, nil
}

func sanitizeInput(input CharacterCardInput) CharacterCardInput {
	input.Name = truncate(strings.TrimSpace(input.Name), 80)
	input.Pronouns = truncate(strings.TrimSpace(input.Pronouns), 80)
	input.PortraitURL = truncate(strings.TrimSpace(input.PortraitURL), 500)
	input.TokenAura = strings.TrimSpace(input.TokenAura)
	input.Aura = strings.TrimSpace(input.Aura)
	input.Color = strings.TrimSpace(input.Color)
	switch {
	case input.TokenAura != "":
		aura, err := normalizeTokenAura(input.TokenAura)
		if err != nil {
			input.TokenAura = "__invalid__"
			return input
		}
		input.TokenAura = aura
	case input.Aura != "":
		aura, err := normalizeTokenAura(input.Aura)
		if err != nil {
			input.TokenAura = "__invalid__"
			return input
		}
		input.TokenAura = aura
	case input.Color != "":
		aura, err := normalizeTokenAura(input.Color)
		if err != nil {
			input.TokenAura = "__invalid__"
			return input
		}
		if aura != "" && aura != "#d9c7a6" {
			input.TokenAura = aura
		}
	}
	if input.TokenAura == "" && input.Color == "" {
		input.Color = "#d9c7a6"
	}
	if input.Color == "" {
		input.Color = input.TokenAura
	}
	input.Color = normalizeColor(input.Color)
	input.Tagline = truncate(strings.TrimSpace(input.Tagline), 160)
	input.PublicDescription = truncate(strings.TrimSpace(input.PublicDescription), 1000)
	input.PrivateNotes = truncate(strings.TrimSpace(input.PrivateNotes), 2000)
	if input.SheetLinks != nil {
		links := sanitizeSheetLinks(*input.SheetLinks)
		input.SheetLinks = &links
	}
	return input
}

func sheetLinksFromInput(input CharacterCardInput) []SheetLink {
	if input.SheetLinks == nil {
		return []SheetLink{}
	}
	return *input.SheetLinks
}

func finishCharacterCard(card *CharacterCard, sheetLinksRaw []byte, createdAt, updatedAt time.Time) error {
	card.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	card.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	card.SheetLinks = []SheetLink{}
	if len(sheetLinksRaw) == 0 {
		return nil
	}
	if err := json.Unmarshal(sheetLinksRaw, &card.SheetLinks); err != nil {
		return err
	}
	if card.SheetLinks == nil {
		card.SheetLinks = []SheetLink{}
	}
	return nil
}

func sanitizeSheetLinks(links []SheetLink) []SheetLink {
	out := make([]SheetLink, 0, len(links))
	seen := map[string]bool{}
	now := time.Now().UTC().Format(time.RFC3339)

	for _, link := range links {
		if len(out) >= maxSheetLinks {
			break
		}

		clean := SheetLink{
			ID:        truncate(strings.TrimSpace(link.ID), 80),
			Ruleset:   truncate(strings.TrimSpace(link.Ruleset), 80),
			SheetType: truncate(strings.TrimSpace(link.SheetType), 80),
			Label:     truncate(strings.TrimSpace(link.Label), 160),
			URL:       truncate(strings.TrimSpace(link.URL), 500),
			CreatedAt: truncate(strings.TrimSpace(link.CreatedAt), 40),
		}

		if clean.Ruleset == "" && clean.SheetType == "" && clean.Label == "" && clean.URL == "" {
			continue
		}
		if clean.ID == "" || seen[clean.ID] {
			clean.ID = newSheetLinkID()
		}
		for attempt := 0; seen[clean.ID]; attempt++ {
			clean.ID = newSheetLinkID() + "_" + strconv.Itoa(attempt)
		}
		seen[clean.ID] = true
		if clean.CreatedAt == "" {
			clean.CreatedAt = now
		} else if _, err := time.Parse(time.RFC3339, clean.CreatedAt); err != nil {
			clean.CreatedAt = now
		}
		if clean.Label == "" {
			clean.Label = sheetLinkLabel(clean.Ruleset, clean.SheetType)
		}
		out = append(out, clean)
	}
	return out
}

func sheetLinkLabel(ruleset, sheetType string) string {
	parts := []string{}
	if ruleset != "" {
		parts = append(parts, ruleset)
	}
	if sheetType != "" {
		parts = append(parts, sheetType)
	}
	if len(parts) == 0 {
		return "Character Sheet"
	}
	return strings.Join(parts, " ") + " Sheet"
}

func newSheetLinkID() string {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return "sheet_" + hex.EncodeToString(bytes[:])
	}
	return "sheet_" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func normalizeColor(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) == 4 || len(value) == 7 {
		if strings.HasPrefix(value, "#") {
			return value
		}
	}
	return "#d9c7a6"
}

func normalizeTokenAura(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", nil
	}
	if len(value) != 7 || !strings.HasPrefix(value, "#") {
		return "", errors.New("invalid_token_aura")
	}
	for _, r := range value[1:] {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return "", errors.New("invalid_token_aura")
		}
	}
	return value, nil
}

func truncate(value string, max int) string {
	if utf8.RuneCountInString(value) <= max {
		return value
	}
	runes := []rune(value)
	return string(runes[:max])
}

func hasImplicitDraftRole(ctx context.Context, q characterQuerier, userID string) (bool, error) {
	var found int
	err := q.QueryRow(ctx, `
		SELECT 1
		FROM (
			SELECT user_id, role::text AS role
			FROM location_memberships
			WHERE user_id = $1 AND active = TRUE AND role IN ('producer', 'director', 'cast', 'crew')
			UNION ALL
			SELECT user_id, role::text AS role
			FROM memberships
			WHERE user_id = $1 AND active = TRUE AND role IN ('producer', 'director', 'cast', 'crew')
		) roles
		LIMIT 1
	`, userID).Scan(&found)
	if err == nil && found == 1 {
		return true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return false, err
}

func hasAuthorityInLocation(ctx context.Context, q characterQuerier, userID, locationID string) (bool, error) {
	var found int
	err := q.QueryRow(ctx, `
		SELECT 1
		FROM (
			SELECT user_id
			FROM location_memberships
			WHERE user_id = $1 AND location_id = $2 AND active = TRUE AND role IN ('producer', 'director')
			UNION ALL
			SELECT user_id
			FROM memberships
			WHERE user_id = $1 AND location_id = $2 AND active = TRUE AND role IN ('producer', 'director')
		) roles
		LIMIT 1
	`, userID, locationID).Scan(&found)
	if err == nil && found == 1 {
		return true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return false, err
}

func targetCanReceiveDraftGrant(ctx context.Context, q characterQuerier, targetUserID, locationID string) (bool, error) {
	var found int
	err := q.QueryRow(ctx, `
		SELECT 1
		FROM (
			SELECT user_id
			FROM location_memberships
			WHERE user_id = $1 AND location_id = $2 AND active = TRUE AND role IN ('cast', 'crew')
			UNION ALL
			SELECT user_id
			FROM memberships
			WHERE user_id = $1 AND location_id = $2 AND active = TRUE AND role IN ('cast', 'crew')
		) roles
		LIMIT 1
	`, targetUserID, locationID).Scan(&found)
	if err == nil && found == 1 {
		return true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return false, err
}

func resolveCharacterScope(ctx context.Context, q characterQuerier, userID string) (locationID, productionID string, err error) {
	err = q.QueryRow(ctx, `
		SELECT location_id::text, COALESCE(production_id::text, '')
		FROM memberships
		WHERE user_id = $1
		  AND active = TRUE
		  AND role IN ('director', 'cast', 'crew')
		  AND production_id IS NOT NULL
		ORDER BY created_at ASC
		LIMIT 1
	`, userID).Scan(&locationID, &productionID)
	if err == nil {
		return locationID, productionID, nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", "", err
	}

	err = q.QueryRow(ctx, `
		SELECT location_id::text
		FROM location_memberships
		WHERE user_id = $1
		  AND active = TRUE
		ORDER BY created_at ASC
		LIMIT 1
	`, userID).Scan(&locationID)
	if err == nil {
		productionID, _ = firstProductionForLocation(ctx, q, locationID)
		return locationID, productionID, nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", "", err
	}

	err = q.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID)
	if err != nil {
		return "", "", err
	}
	productionID, _ = firstProductionForLocation(ctx, q, locationID)
	return locationID, productionID, nil
}

func authorityLocation(ctx context.Context, pool *pgxpool.Pool, userID string) (string, error) {
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err == nil && ok {
		var locationID string
		err := pool.QueryRow(ctx, `SELECT id::text FROM locations WHERE slug = 'amurray-family' LIMIT 1`).Scan(&locationID)
		return locationID, err
	}

	var locationID string
	err := pool.QueryRow(ctx, `
		SELECT location_id::text
		FROM (
			SELECT location_id, created_at
			FROM location_memberships
			WHERE user_id = $1 AND active = TRUE AND role IN ('producer', 'director')
			UNION ALL
			SELECT location_id, created_at
			FROM memberships
			WHERE user_id = $1 AND active = TRUE AND role IN ('producer', 'director')
		) roles
		ORDER BY created_at ASC
		LIMIT 1
	`, userID).Scan(&locationID)
	return locationID, err
}

func firstProductionForLocation(ctx context.Context, q characterQuerier, locationID string) (string, error) {
	var productionID string
	err := q.QueryRow(ctx, `
		SELECT id::text
		FROM productions
		WHERE location_id = $1
		ORDER BY created_at ASC
		LIMIT 1
	`, locationID).Scan(&productionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return productionID, err
}

func requireUser(ctx context.Context, pool *pgxpool.Pool, r *http.Request) (string, error) {
	raw, err := sessions.ReadSessionCookie(r)
	if err != nil || strings.TrimSpace(raw) == "" {
		return "", errors.New("not_authenticated")
	}
	rec, err := sessions.GetSessionByRawToken(ctx, pool, raw)
	if err != nil || strings.TrimSpace(rec.UserID) == "" {
		return "", errors.New("not_authenticated")
	}
	return rec.UserID, nil
}

func writeAuthError(w http.ResponseWriter, err error) {
	_ = err
	writeJSON(w, http.StatusUnauthorized, response{Ok: false, Data: map[string]any{"error": "not_authenticated"}})
}

func writeCharacterError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		writeJSON(w, http.StatusNotFound, response{Ok: false, Data: map[string]any{"error": "not_found"}})
	case err.Error() == "not_authenticated":
		writeJSON(w, http.StatusUnauthorized, response{Ok: false, Data: map[string]any{"error": err.Error()}})
	case err.Error() == "forbidden", err.Error() == "director_authority_required", strings.Contains(err.Error(), "permission_required"), err.Error() == "target_must_be_cast_or_crew":
		writeJSON(w, http.StatusForbidden, response{Ok: false, Data: map[string]any{"error": err.Error()}})
	case strings.Contains(err.Error(), "required"), strings.Contains(err.Error(), "unknown_"), strings.Contains(err.Error(), "invalid_"), strings.Contains(err.Error(), "not_eligible"), strings.Contains(err.Error(), "not_found_"), strings.Contains(err.Error(), "_locked"):
		writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": err.Error()}})
	default:
		writeJSON(w, http.StatusInternalServerError, response{Ok: false, Data: map[string]any{"error": "character_kernel_failed"}})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
