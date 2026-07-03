package characters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkbookPageField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Value       string `json:"value"`
	InputType   string `json:"input_type"`
	Placeholder string `json:"placeholder,omitempty"`
	Editable    bool   `json:"editable"`
}

type CharacterWorkbookEntry struct {
	ID               string         `json:"id"`
	CharacterCardID  string         `json:"character_card_id"`
	ModuleInstanceID string         `json:"module_instance_id,omitempty"`
	AuthorUserID     string         `json:"author_user_id"`
	PageKey          string         `json:"page_key"`
	EntryType        string         `json:"entry_type"`
	Title            string         `json:"title"`
	Body             string         `json:"body"`
	Payload          map[string]any `json:"payload,omitempty"`
	StageNumber      int            `json:"stage_number,omitempty"`
	SortOrder        int            `json:"sort_order"`
	CreatedAt        string         `json:"created_at"`
	UpdatedAt        string         `json:"updated_at"`
}

type CharacterWorkbookPage struct {
	Key      string                   `json:"key"`
	Title    string                   `json:"title"`
	Kind     string                   `json:"kind"`
	Summary  string                   `json:"summary,omitempty"`
	Editable bool                     `json:"editable"`
	Fields   []WorkbookPageField      `json:"fields,omitempty"`
	Entries  []CharacterWorkbookEntry `json:"entries,omitempty"`
	Journals []CharacterJournalEntry  `json:"journals,omitempty"`
}

type CharacterWorkbookView struct {
	Character      CharacterCard           `json:"character"`
	Module         map[string]any          `json:"module,omitempty"`
	Pages          []CharacterWorkbookPage `json:"pages"`
	ActivePage     string                  `json:"active_page,omitempty"`
	WorkbookStatus string                  `json:"workbook_status,omitempty"`
	ResumeHint     string                  `json:"resume_hint,omitempty"`
}

type WorkbookEventInput struct {
	PageKey     string         `json:"page_key"`
	EntryType   string         `json:"entry_type"`
	Title       string         `json:"title"`
	Body        string         `json:"body"`
	Payload     map[string]any `json:"payload"`
	StageNumber int            `json:"stage_number"`
	SortOrder   int            `json:"sort_order"`
}

type WorkbookEventRequest struct {
	CharacterCardID string               `json:"character_card_id"`
	ModuleKey       string               `json:"module_key"`
	ModuleStatus    string               `json:"module_status"`
	CurrentStage    int                  `json:"current_stage"`
	CurrentEvent    string               `json:"current_event"`
	WorkbookStatus  string               `json:"workbook_status"`
	WorkbookContext map[string]any       `json:"workbook_context"`
	Entries         []WorkbookEventInput `json:"entries"`
}

func HandleCharacterWorkbookByID(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireUser(ctx, pool, r)
		if err != nil {
			writeAuthError(w, err)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/api/character-workbooks/")
		if strings.HasSuffix(path, "/events") {
			if r.Method != http.MethodPost {
				writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false, Data: map[string]any{"error": "method_not_allowed"}})
				return
			}
			cardID := strings.TrimSpace(strings.TrimSuffix(path, "/events"))
			if cardID == "" {
				writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": "character_card_id_required"}})
				return
			}

			var input WorkbookEventRequest
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": "invalid_json"}})
				return
			}
			if strings.TrimSpace(input.CharacterCardID) == "" {
				input.CharacterCardID = cardID
			}
			if strings.TrimSpace(input.CharacterCardID) != cardID {
				writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": "character_card_id_mismatch"}})
				return
			}

			out, err := RecordWorkbookEvents(ctx, pool, userID, input)
			if err != nil {
				writeCharacterError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, response{Ok: true, Data: out})
			return
		}

		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false, Data: map[string]any{"error": "method_not_allowed"}})
			return
		}

		cardID := strings.TrimSpace(path)
		if cardID == "" {
			writeJSON(w, http.StatusBadRequest, response{Ok: false, Data: map[string]any{"error": "character_card_id_required"}})
			return
		}

		view, err := LoadCharacterWorkbookView(ctx, pool, userID, cardID)
		if err != nil {
			writeCharacterError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, response{Ok: true, Data: view})
	}
}

func LoadCharacterWorkbookView(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID string) (CharacterWorkbookView, error) {
	allowed, err := CanEditCard(ctx, pool, actorUserID, cardID)
	if err != nil {
		return CharacterWorkbookView{}, err
	}
	if !allowed {
		return CharacterWorkbookView{}, errors.New("forbidden")
	}

	card, module, entries, journals, err := loadWorkbookState(ctx, pool, actorUserID, cardID)
	if err != nil {
		return CharacterWorkbookView{}, err
	}

	pages := buildWorkbookPages(card, module, entries, journals)
	resumeHint := "Open Face to edit identity, History for event logs, Mechanics for the module, and Journal for private notes."
	if len(entries) > 0 {
		resumeHint = strings.TrimSpace(entries[len(entries)-1].Title)
	}

	return CharacterWorkbookView{
		Character:      card,
		Module:         module,
		Pages:          pages,
		ActivePage:     "face",
		WorkbookStatus: card.WorkbookStatus,
		ResumeHint:     resumeHint,
	}, nil
}

func loadWorkbookState(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID string) (CharacterCard, map[string]any, []CharacterWorkbookEntry, []CharacterJournalEntry, error) {
	var card CharacterCard
	var createdAt, updatedAt time.Time
	var workbookContextRaw, sheetLinksRaw []byte
	err := pool.QueryRow(ctx, `
		SELECT id::text, owner_user_id::text, location_id::text, COALESCE(production_id::text, ''), workbook_status, workbook_context, name, pronouns, portrait_url, color, tagline, public_description, private_notes, sheet_links, created_at, updated_at
		FROM character_cards
		WHERE id = $1
		  AND is_deleted = FALSE
		LIMIT 1
	`, cardID).Scan(&card.ID, &card.OwnerUserID, &card.LocationID, &card.ProductionID, &card.WorkbookStatus, &workbookContextRaw, &card.Name, &card.Pronouns, &card.PortraitURL, &card.Color, &card.Tagline, &card.PublicDescription, &card.PrivateNotes, &sheetLinksRaw, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CharacterCard{}, nil, nil, nil, errors.New("character_card_not_found")
		}
		return CharacterCard{}, nil, nil, nil, err
	}
	card.WorkbookContext = decodeJSONMap(workbookContextRaw)
	if err := finishCharacterCard(&card, sheetLinksRaw, createdAt, updatedAt); err != nil {
		return CharacterCard{}, nil, nil, nil, err
	}

	module, err := loadWorkbookModule(ctx, pool, cardID)
	if err != nil {
		return CharacterCard{}, nil, nil, nil, err
	}

	entries, err := loadWorkbookEntries(ctx, pool, cardID)
	if err != nil {
		return CharacterCard{}, nil, nil, nil, err
	}

	journals, err := loadWorkbookJournals(ctx, pool, actorUserID, cardID)
	if err != nil {
		return CharacterCard{}, nil, nil, nil, err
	}

	return card, module, entries, journals, nil
}

func loadWorkbookModule(ctx context.Context, pool *pgxpool.Pool, cardID string) (map[string]any, error) {
	var moduleID, rulesetKey, rulesetVersion, productionID, venueID, status, flowVersion, chartVersion, currentEvent string
	var currentStage int
	var moduleContextRaw []byte
	err := pool.QueryRow(ctx, `
		SELECT id::text, ruleset_key, ruleset_version, COALESCE(production_id::text, ''), COALESCE(venue_id::text, ''), module_status, creation_flow_version, parentage_chart_version, current_stage, current_event, module_context
		FROM character_workbook_modules
		WHERE character_card_id = $1
		ORDER BY created_at ASC
		LIMIT 1
	`, cardID).Scan(&moduleID, &rulesetKey, &rulesetVersion, &productionID, &venueID, &status, &flowVersion, &chartVersion, &currentStage, &currentEvent, &moduleContextRaw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return map[string]any{
		"id":                      moduleID,
		"ruleset_key":             rulesetKey,
		"ruleset_version":         rulesetVersion,
		"production_id":           productionID,
		"venue_id":                venueID,
		"module_status":           status,
		"creation_flow_version":   flowVersion,
		"parentage_chart_version": chartVersion,
		"current_stage":           currentStage,
		"current_event":           currentEvent,
		"module_context":          decodeJSONMap(moduleContextRaw),
	}, nil
}

func loadWorkbookEntries(ctx context.Context, pool *pgxpool.Pool, cardID string) ([]CharacterWorkbookEntry, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, character_card_id::text, COALESCE(module_instance_id::text, ''), author_user_id::text, page_key, entry_type, title, body, payload, COALESCE(stage_number, 0), sort_order, created_at, updated_at
		FROM character_workbook_entries
		WHERE character_card_id = $1
		ORDER BY sort_order ASC, created_at ASC
	`, cardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []CharacterWorkbookEntry{}
	for rows.Next() {
		var entry CharacterWorkbookEntry
		var payloadRaw []byte
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&entry.ID, &entry.CharacterCardID, &entry.ModuleInstanceID, &entry.AuthorUserID, &entry.PageKey, &entry.EntryType, &entry.Title, &entry.Body, &payloadRaw, &entry.StageNumber, &entry.SortOrder, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		if len(payloadRaw) > 0 {
			_ = json.Unmarshal(payloadRaw, &entry.Payload)
		}
		if entry.Payload == nil {
			entry.Payload = map[string]any{}
		}
		entry.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		entry.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		out = append(out, entry)
	}
	return out, rows.Err()
}

func loadWorkbookJournals(ctx context.Context, pool *pgxpool.Pool, actorUserID, cardID string) ([]CharacterJournalEntry, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, character_card_id::text, COALESCE(module_instance_id::text, ''), author_user_id::text, visibility, body, COALESCE(venue_id::text, ''), COALESCE(session_id::text, ''), COALESCE(showing_id::text, ''), archived_at, deleted_at, created_at, updated_at
		FROM character_journals
		WHERE character_card_id = $1
		  AND author_user_id = $2
		  AND deleted_at IS NULL
		ORDER BY created_at DESC
	`, cardID, actorUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []CharacterJournalEntry{}
	for rows.Next() {
		var entry CharacterJournalEntry
		var archivedAt, deletedAt *time.Time
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&entry.ID, &entry.CharacterCardID, &entry.ModuleInstanceID, &entry.AuthorUserID, &entry.Visibility, &entry.Body, &entry.VenueID, &entry.SessionID, &entry.ShowingID, &archivedAt, &deletedAt, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		if archivedAt != nil {
			entry.ArchivedAt = archivedAt.UTC().Format(time.RFC3339)
		}
		if deletedAt != nil {
			entry.DeletedAt = deletedAt.UTC().Format(time.RFC3339)
		}
		entry.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		entry.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		out = append(out, entry)
	}
	return out, rows.Err()
}

func buildWorkbookPages(card CharacterCard, module map[string]any, entries []CharacterWorkbookEntry, journals []CharacterJournalEntry) []CharacterWorkbookPage {
	faceFields := []WorkbookPageField{
		{Key: "name", Label: "Name", Value: card.Name, InputType: "text", Placeholder: "Enter a name", Editable: true},
		{Key: "pronouns", Label: "Pronouns", Value: card.Pronouns, InputType: "text", Placeholder: "they/them", Editable: true},
		{Key: "portrait_url", Label: "Portrait URL", Value: card.PortraitURL, InputType: "url", Placeholder: "https://", Editable: true},
		{Key: "color", Label: "Color", Value: card.Color, InputType: "color", Editable: true},
		{Key: "tagline", Label: "Tagline", Value: card.Tagline, InputType: "text", Placeholder: "A short line about the character", Editable: true},
		{Key: "public_description", Label: "Public Description", Value: card.PublicDescription, InputType: "textarea", Placeholder: "What the public should know", Editable: true},
		{Key: "private_notes", Label: "Private Notes", Value: card.PrivateNotes, InputType: "textarea", Placeholder: "Private notes for the player", Editable: true},
	}
	faceFields = append(faceFields, workbookRootSummaryFields(card)...)

	return []CharacterWorkbookPage{
		{
			Key:      "face",
			Title:    "Face",
			Kind:     "face",
			Summary:  "Identity fields and the workbook root.",
			Editable: true,
			Fields:   faceFields,
		},
		{
			Key:      "history",
			Title:    "History",
			Kind:     "history",
			Summary:  "Append-forward workbook events.",
			Editable: false,
			Entries:  filterWorkbookEntries(entries, "history", "creation_progress"),
		},
		{
			Key:      "mechanics",
			Title:    "Mechanics",
			Kind:     "mechanics",
			Summary:  "Ruleset and mechanical state derived from the module.",
			Editable: true,
			Fields: append([]WorkbookPageField{
				{Key: "workbook_status", Label: "Workbook Status", Value: card.WorkbookStatus, InputType: "text", Placeholder: "draft", Editable: true},
				{Key: "ruleset_key", Label: "Ruleset", Value: stringValue(module["ruleset_key"]), InputType: "text", Editable: false},
				{Key: "ruleset_version", Label: "Ruleset Version", Value: stringValue(module["ruleset_version"]), InputType: "text", Editable: false},
				{Key: "current_stage", Label: "Current Stage", Value: intValueString(module["current_stage"]), InputType: "text", Editable: false},
				{Key: "current_event", Label: "Current Event", Value: stringValue(module["current_event"]), InputType: "text", Editable: false},
			}, append(chapter3MechanicsFields(card.WorkbookContext), chapter4MechanicsFields(card.WorkbookContext)...)...),
		},
		{
			Key:      "journal",
			Title:    "Journal",
			Kind:     "journal",
			Summary:  "Private player notes.",
			Editable: true,
			Journals: journals,
		},
		{
			Key:      "module",
			Title:    "Socio Module",
			Kind:     "module_face",
			Summary:  "The active venue/ruleset face page.",
			Editable: false,
			Fields: []WorkbookPageField{
				{Key: "module_status", Label: "Module Status", Value: stringValue(module["module_status"]), InputType: "text", Editable: false},
				{Key: "creation_flow_version", Label: "Creation Flow", Value: stringValue(module["creation_flow_version"]), InputType: "text", Editable: false},
				{Key: "parentage_chart_version", Label: "Parentage Chart", Value: stringValue(module["parentage_chart_version"]), InputType: "text", Editable: false},
				{Key: "current_stage", Label: "Current Stage", Value: intValueString(module["current_stage"]), InputType: "text", Editable: false},
				{Key: "current_event", Label: "Current Event", Value: stringValue(module["current_event"]), InputType: "text", Editable: false},
				{Key: "module_context", Label: "Module Context", Value: jsonStringify(module["module_context"]), InputType: "textarea", Editable: false},
			},
		},
		{
			Key:      "creation_progress",
			Title:    "Creation Progress",
			Kind:     "progress",
			Summary:  "Stage milestones and autosave progress.",
			Editable: false,
			Entries:  filterWorkbookEntries(entries, "creation_progress"),
		},
	}
}

func filterWorkbookEntries(entries []CharacterWorkbookEntry, pageKeys ...string) []CharacterWorkbookEntry {
	if len(entries) == 0 || len(pageKeys) == 0 {
		return []CharacterWorkbookEntry{}
	}
	allowed := map[string]bool{}
	for _, key := range pageKeys {
		allowed[strings.TrimSpace(key)] = true
	}
	out := make([]CharacterWorkbookEntry, 0, len(entries))
	for _, entry := range entries {
		if allowed[strings.TrimSpace(entry.PageKey)] {
			out = append(out, entry)
		}
	}
	return out
}

func RecordWorkbookEvents(ctx context.Context, pool *pgxpool.Pool, actorUserID string, input WorkbookEventRequest) (map[string]any, error) {
	actorUserID = strings.TrimSpace(actorUserID)
	cardID := strings.TrimSpace(input.CharacterCardID)
	if actorUserID == "" {
		return nil, errors.New("not_authenticated")
	}
	if cardID == "" {
		return nil, errors.New("character_card_id_required")
	}

	allowed, err := CanEditCard(ctx, pool, actorUserID, cardID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errors.New("forbidden")
	}

	moduleID, err := ensureWorkbookModuleForCard(ctx, pool, cardID, input)
	if err != nil {
		return nil, err
	}

	entries := make([]CharacterWorkbookEntry, 0, len(input.Entries))
	for idx, item := range input.Entries {
		entry, err := insertWorkbookEntry(ctx, pool, cardID, moduleID, actorUserID, item, idx)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	if err := updateWorkbookContextAfterEvents(ctx, pool, cardID, input, entries); err != nil {
		return nil, err
	}

	return map[string]any{
		"character_card_id":  cardID,
		"module_instance_id": moduleID,
		"entries":            entries,
	}, nil
}

func ensureWorkbookModuleForCard(ctx context.Context, pool *pgxpool.Pool, cardID string, input WorkbookEventRequest) (string, error) {
	moduleKey := strings.TrimSpace(input.ModuleKey)
	if moduleKey == "" {
		moduleKey = "socio"
	}
	moduleStatus := strings.TrimSpace(input.ModuleStatus)
	if moduleStatus == "" {
		moduleStatus = "draft"
	}

	var moduleID string
	err := pool.QueryRow(ctx, `
		INSERT INTO character_workbook_modules (
			character_card_id,
			ruleset_key,
			ruleset_version,
			module_status,
			creation_flow_version,
			parentage_chart_version,
			current_stage,
			current_event,
			module_context
		)
		VALUES ($1, $2, '1.1', $3, 'v1', '1.1', COALESCE(NULLIF($4, 0), 1), COALESCE(NULLIF($5, ''), 'egg_donor_parentage'), COALESCE($6::jsonb, '{}'::jsonb))
		ON CONFLICT (character_card_id, ruleset_key)
		DO UPDATE SET
			module_status = EXCLUDED.module_status,
			current_stage = EXCLUDED.current_stage,
			current_event = EXCLUDED.current_event,
			module_context = COALESCE(EXCLUDED.module_context, character_workbook_modules.module_context),
			updated_at = NOW()
		RETURNING id::text
	`, cardID, moduleKey, moduleStatus, input.CurrentStage, input.CurrentEvent, jsonString(input.WorkbookContext)).Scan(&moduleID)
	if err != nil {
		return "", err
	}
	return moduleID, nil
}

func insertWorkbookEntry(ctx context.Context, pool *pgxpool.Pool, cardID, moduleID, authorUserID string, input WorkbookEventInput, index int) (CharacterWorkbookEntry, error) {
	pageKey := strings.TrimSpace(input.PageKey)
	if pageKey == "" {
		pageKey = "history"
	}
	entryType := strings.TrimSpace(input.EntryType)
	title := strings.TrimSpace(input.Title)
	body := strings.TrimSpace(input.Body)
	payload := input.Payload
	if payload == nil {
		payload = map[string]any{}
	}

	sortOrder := input.SortOrder
	if sortOrder <= 0 {
		sortOrder = index + 1
	}

	if existing, ok, err := findDuplicateWorkbookEntry(ctx, pool, cardID, entryType, title, body); err != nil {
		return CharacterWorkbookEntry{}, err
	} else if ok {
		return existing, nil
	}

	var entry CharacterWorkbookEntry
	var payloadRaw []byte
	var createdAt, updatedAt time.Time
	err := pool.QueryRow(ctx, `
		INSERT INTO character_workbook_entries (
			character_card_id,
			module_instance_id,
			author_user_id,
			page_key,
			entry_type,
			title,
			body,
			payload,
			stage_number,
			sort_order
		)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6, $7, $8::jsonb, NULLIF($9, 0), $10)
		RETURNING id::text, character_card_id::text, COALESCE(module_instance_id::text, ''), author_user_id::text, page_key, entry_type, title, body, payload, COALESCE(stage_number, 0), sort_order, created_at, updated_at
	`, cardID, moduleID, authorUserID, pageKey, entryType, title, body, jsonString(payload), input.StageNumber, sortOrder).
		Scan(&entry.ID, &entry.CharacterCardID, &entry.ModuleInstanceID, &entry.AuthorUserID, &entry.PageKey, &entry.EntryType, &entry.Title, &entry.Body, &payloadRaw, &entry.StageNumber, &entry.SortOrder, &createdAt, &updatedAt)
	if err != nil {
		return CharacterWorkbookEntry{}, err
	}
	if len(payloadRaw) > 0 {
		_ = json.Unmarshal(payloadRaw, &entry.Payload)
	}
	if entry.Payload == nil {
		entry.Payload = map[string]any{}
	}
	entry.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	entry.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return entry, nil
}

// findDuplicateWorkbookEntry guards the append-only history ledger against
// stacking identical entries when a client resubmits the same builder step
// (resume, double click, retry). It only matches the most recent entry with
// the same character/entry_type/title/body, so a genuine correction with
// different content still inserts a new row.
func findDuplicateWorkbookEntry(ctx context.Context, pool *pgxpool.Pool, cardID, entryType, title, body string) (CharacterWorkbookEntry, bool, error) {
	if entryType == "" {
		return CharacterWorkbookEntry{}, false, nil
	}

	var entry CharacterWorkbookEntry
	var payloadRaw []byte
	var createdAt, updatedAt time.Time
	err := pool.QueryRow(ctx, `
		SELECT id::text, character_card_id::text, COALESCE(module_instance_id::text, ''), author_user_id::text, page_key, entry_type, title, body, payload, COALESCE(stage_number, 0), sort_order, created_at, updated_at
		FROM character_workbook_entries
		WHERE character_card_id = $1 AND entry_type = $2 AND title = $3 AND body = $4
		ORDER BY created_at DESC
		LIMIT 1
	`, cardID, entryType, title, body).
		Scan(&entry.ID, &entry.CharacterCardID, &entry.ModuleInstanceID, &entry.AuthorUserID, &entry.PageKey, &entry.EntryType, &entry.Title, &entry.Body, &payloadRaw, &entry.StageNumber, &entry.SortOrder, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CharacterWorkbookEntry{}, false, nil
		}
		return CharacterWorkbookEntry{}, false, err
	}
	if len(payloadRaw) > 0 {
		_ = json.Unmarshal(payloadRaw, &entry.Payload)
	}
	if entry.Payload == nil {
		entry.Payload = map[string]any{}
	}
	entry.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	entry.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return entry, true, nil
}

func updateWorkbookContextAfterEvents(ctx context.Context, pool *pgxpool.Pool, cardID string, input WorkbookEventRequest, entries []CharacterWorkbookEntry) error {
	contextPatch := normalizeWorkbookContext(input.WorkbookContext)
	if len(contextPatch) == 0 && len(entries) == 0 && strings.TrimSpace(input.WorkbookStatus) == "" {
		return nil
	}

	var raw []byte
	if err := pool.QueryRow(ctx, `
		SELECT workbook_context
		FROM character_cards
		WHERE id = $1
		LIMIT 1
	`, cardID).Scan(&raw); err != nil {
		return err
	}

	current := decodeJSONMap(raw)
	if current == nil {
		current = map[string]any{}
	}

	if strings.TrimSpace(input.WorkbookStatus) != "" {
		current["workbook_status"] = strings.TrimSpace(input.WorkbookStatus)
	}
	if strings.TrimSpace(input.ModuleKey) != "" {
		current["active_module_key"] = strings.TrimSpace(input.ModuleKey)
	}
	if strings.TrimSpace(input.ModuleStatus) != "" {
		current["module_status"] = strings.TrimSpace(input.ModuleStatus)
	}
	if input.CurrentStage > 0 {
		current["current_stage"] = input.CurrentStage
	}
	if strings.TrimSpace(input.CurrentEvent) != "" {
		current["current_event"] = strings.TrimSpace(input.CurrentEvent)
	}
	if len(entries) > 0 {
		last := entries[len(entries)-1]
		current["last_workbook_entry"] = map[string]any{
			"page_key":     last.PageKey,
			"entry_type":   last.EntryType,
			"title":        last.Title,
			"created_at":   last.CreatedAt,
			"stage_number": last.StageNumber,
		}
	}
	for key, value := range contextPatch {
		current[key] = value
	}

	payloadJSON, err := json.Marshal(current)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		UPDATE character_cards
		SET workbook_context = $2::jsonb,
		    workbook_status = COALESCE(NULLIF($3, ''), workbook_status),
		    updated_at = NOW()
		WHERE id = $1
	`, cardID, string(payloadJSON), strings.TrimSpace(input.WorkbookStatus))
	return err
}

func workbookRootSummaryFields(card CharacterCard) []WorkbookPageField {
	context := card.WorkbookContext
	if len(context) == 0 {
		return nil
	}
	if strings.ToLower(strings.TrimSpace(stringValue(context["source"]))) != "catharsis" {
		return nil
	}

	effectiveRoll := catharsisEffectiveParentageRoll(context)
	combinedRoll := catharsisCombinedParentageRoll(context)
	effectiveEntry, ok := ParentageChartEntryForRoll(effectiveRoll)
	if !ok {
		effectiveEntry, _ = ParentageChartEntryForRoll(3)
	}
	startingCredit := titleizeContextValue(effectiveEntry.StartingCredit)
	if effectiveRoll == 0 {
		startingCredit = titleizeContextValue(context["socio_parentage_starting_credit"])
	}

	fields := []WorkbookPageField{
		{Key: "workbook_source", Label: "Workbook Source", Value: titleizeContextValue(context["source"]), InputType: "text", Editable: false},
		{Key: "socio_parentage_chart_version", Label: "Parentage Chart", Value: titleizeContextValue(context["socio_parentage_chart_version"]), InputType: "text", Editable: false},
		{Key: "socio_parentage_roll", Label: "Parentage Roll", Value: titleizeContextValue(effectiveRoll), InputType: "text", Editable: false},
		{Key: "socio_parentage_total_roll", Label: "Combined Roll", Value: titleizeContextValue(combinedRoll), InputType: "text", Editable: false},
		{Key: "socio_parentage_summary", Label: "Parentage Summary", Value: socioParentageSummary(context), InputType: "textarea", Editable: false},
		{Key: "socio_starting_wealth", Label: "Starting Wealth", Value: titleizeContextValue(context["socio_starting_wealth"]), InputType: "text", Editable: false},
		{Key: "socio_parentage_starting_credit", Label: "Resolved Starting Credit", Value: startingCredit, InputType: "text", Editable: false},
		{Key: "socio_wealth_inheritance", Label: "Wealth Inheritance", Value: socioInheritanceSummary(context), InputType: "textarea", Editable: false},
	}

	if stage := intValueString(context["current_stage"]); stage != "" {
		fields = append(fields, WorkbookPageField{Key: "current_stage_summary", Label: "Current Stage", Value: stage, InputType: "text", Editable: false})
	}
	if event := stringValue(context["current_event"]); strings.TrimSpace(event) != "" {
		fields = append(fields, WorkbookPageField{Key: "current_event_summary", Label: "Current Event", Value: event, InputType: "text", Editable: false})
	}
	fields = append(fields, chapter3FaceWidgetFields(context)...)
	fields = append(fields, chapter4FaceWidgetFields(context)...)
	return fields
}

// chapter3FaceWidgetFields renders the compact, permanent Chapter 3
// archetype widget for the Face page (title, motto/short description) once
// an archetype has been confirmed.
func chapter3FaceWidgetFields(context map[string]any) []WorkbookPageField {
	raw, ok := context["chapter3"].(map[string]any)
	if !ok {
		return nil
	}
	confirmed, _ := raw["confirmed"].(bool)
	if !confirmed {
		return nil
	}
	summary := stringValue(raw["custom_summary"])
	if strings.TrimSpace(summary) == "" {
		archetype, _ := Chapter3ArchetypeByKey(stringValue(raw["archetype_key"]))
		summary = archetype.Motto
		if summary == "" {
			summary = archetype.ShortDescription
		}
	}
	fields := []WorkbookPageField{
		{Key: "chapter3_archetype", Label: "Archetype", Value: stringValue(raw["archetype_title"]), InputType: "text", Editable: false},
		{Key: "chapter3_archetype_summary", Label: "Archetype Summary", Value: summary, InputType: "textarea", Editable: false},
	}
	if effect := stringValue(raw["mechanical_effect"]); strings.TrimSpace(effect) != "" {
		fields = append(fields, WorkbookPageField{Key: "chapter3_mechanical_effect", Label: "Mechanical Effect", Value: effect, InputType: "textarea", Editable: false})
	}
	return fields
}

// chapter3MechanicsFields silently projects the confirmed archetype's
// Mechanics dependencies (primary/secondary attribute, key skill) needed by
// Kernel 56's Chapter 4 skill-group selection.
func chapter3MechanicsFields(context map[string]any) []WorkbookPageField {
	raw, ok := context["chapter3"].(map[string]any)
	if !ok {
		return nil
	}
	confirmed, _ := raw["confirmed"].(bool)
	if !confirmed {
		return nil
	}
	fields := []WorkbookPageField{
		{Key: "chapter3_archetype_stable_id", Label: "Archetype ID", Value: stringValue(raw["archetype_stable_id"]), InputType: "text", Editable: false},
		{Key: "chapter3_primary_attribute", Label: "Archetype Primary Attribute", Value: stringValue(raw["primary_attribute"]), InputType: "text", Editable: false},
		{Key: "chapter3_secondary_attribute", Label: "Archetype Secondary Attribute", Value: stringValue(raw["secondary_attribute"]), InputType: "text", Editable: false},
		{Key: "chapter3_key_skill", Label: "Archetype Key Skill", Value: stringValue(raw["key_skill"]), InputType: "text", Editable: false},
	}
	if strings.EqualFold(stringValue(raw["archetype_key"]), "custom") {
		fields = append(fields,
			WorkbookPageField{Key: "chapter3_custom_summary", Label: "Custom Summary", Value: stringValue(raw["custom_summary"]), InputType: "textarea", Editable: false},
			WorkbookPageField{Key: "chapter3_mechanical_effect", Label: "Mechanical Effect", Value: stringValue(raw["mechanical_effect"]), InputType: "textarea", Editable: false},
		)
	}
	return fields
}

// chapter4FaceWidgetFields renders the compact, permanent Chapter 4 first
// trained skill widget for the Face page (name, attribute, d6) once a first
// skill has been confirmed. Per Kernel 56, after onboarding completes the
// Face page prioritizes archetype + first skill over the ten creation
// attributes.
func chapter4FaceWidgetFields(context map[string]any) []WorkbookPageField {
	raw, ok := context["chapter4"].(map[string]any)
	if !ok {
		return nil
	}
	confirmed, _ := raw["confirmed"].(bool)
	if !confirmed {
		return nil
	}
	fields := []WorkbookPageField{
		{Key: "chapter4_first_skill", Label: "First Trained Skill", Value: stringValue(raw["skill_name"]), InputType: "text", Editable: false},
		{Key: "chapter4_first_skill_die", Label: "Training Die", Value: stringValue(raw["die_size"]), InputType: "text", Editable: false},
	}
	if desc := stringValue(raw["skill_description"]); strings.TrimSpace(desc) != "" {
		fields = append(fields, WorkbookPageField{Key: "chapter4_first_skill_description", Label: "Skill Description", Value: desc, InputType: "textarea", Editable: false})
	}
	return fields
}

// chapter4MechanicsFields silently projects the confirmed first-skill fact
// (stable ID, attribute, training state, die size, helpers) so later dice
// actions can act on it by stable skill ID rather than parsing display text.
func chapter4MechanicsFields(context map[string]any) []WorkbookPageField {
	raw, ok := context["chapter4"].(map[string]any)
	if !ok {
		return nil
	}
	confirmed, _ := raw["confirmed"].(bool)
	if !confirmed {
		return nil
	}
	return []WorkbookPageField{
		{Key: "chapter4_skill_stable_id", Label: "First Skill ID", Value: stringValue(raw["skill_stable_id"]), InputType: "text", Editable: false},
		{Key: "chapter4_skill_name", Label: "First Skill", Value: stringValue(raw["skill_name"]), InputType: "text", Editable: false},
		{Key: "chapter4_skill_attribute", Label: "First Skill Attribute", Value: stringValue(raw["attribute_name"]), InputType: "text", Editable: false},
		{Key: "chapter4_training_state", Label: "Training State", Value: stringValue(raw["training_state"]), InputType: "text", Editable: false},
		{Key: "chapter4_die_size", Label: "Training Die", Value: stringValue(raw["die_size"]), InputType: "text", Editable: false},
	}
}

func socioParentageSummary(context map[string]any) string {
	parents := anySlice(context["socio_parentage_parents"])
	lines := make([]string, 0, len(parents)+2)
	effectiveRoll := catharsisEffectiveParentageRoll(context)
	effectiveEntry, ok := ParentageChartEntryForRoll(effectiveRoll)
	if !ok {
		effectiveEntry = ParentageChartEntry{}
	}
	if effectiveRoll > 0 {
		lines = append(lines, fmt.Sprintf("Effective roll: %d", effectiveRoll))
		if effectiveEntry.SocialClass != "" {
			lines = append(lines, fmt.Sprintf("Resolved class: %s", effectiveEntry.SocialClass))
		}
		if effectiveEntry.StartingCredit > 0 {
			lines = append(lines, fmt.Sprintf("Starting credit: %d", effectiveEntry.StartingCredit))
		}
	}
	if combinedRoll := catharsisCombinedParentageRoll(context); combinedRoll > 0 && combinedRoll != effectiveRoll {
		lines = append(lines, fmt.Sprintf("Combined roll: %d", combinedRoll))
	}
	if len(parents) == 0 {
		if class := titleizeContextValue(context["socio_parentage_class"]); class != "" {
			lines = append(lines, fmt.Sprintf("Resolved class: %s", class))
		}
		if credit := titleizeContextValue(context["socio_parentage_starting_credit"]); credit != "" {
			lines = append(lines, fmt.Sprintf("Starting credit: %s", credit))
		}
		return strings.Join(lines, "\n")
	}

	for _, raw := range parents {
		parent, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		index := intValueString(parent["parent_index"])
		if index == "" {
			index = "?"
		}
		roll := titleizeContextValue(parent["roll_total"])
		class := titleizeContextValue(parent["social_class"])
		credit := titleizeContextValue(parent["starting_credit"])
		flip := titleizeContextValue(parent["coin_flip_result"])
		if flip == "" {
			flip = titleizeContextValue(parent["coin_flip"])
		}
		wealth := titleizeContextValue(parent["inherited_wealth"])
		if wealth == "" {
			wealth = "0"
		}

		lineParts := []string{fmt.Sprintf("Parent %s", index)}
		if roll != "" {
			lineParts = append(lineParts, fmt.Sprintf("roll %s", roll))
		}
		if class != "" {
			lineParts = append(lineParts, class)
		}
		if credit != "" {
			lineParts = append(lineParts, fmt.Sprintf("credit %s", credit))
		}
		if flip != "" {
			lineParts = append(lineParts, fmt.Sprintf("coin flip %s", flip))
		}
		lineParts = append(lineParts, fmt.Sprintf("inheritance %s", wealth))
		lines = append(lines, strings.Join(lineParts, " · "))
	}

	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}

func catharsisEffectiveParentageRoll(context map[string]any) int {
	if rows := normalizeCatharsisParentageRows(context["socio_parentage_parents"]); len(rows) > 0 {
		return effectiveParentageRoll(rows)
	}
	if roll, ok := parseCatharsisRoll(context["socio_parentage_roll"]); ok {
		return roll
	}
	return 0
}

func catharsisCombinedParentageRoll(context map[string]any) int {
	if roll, ok := parseCatharsisRoll(context["socio_parentage_total_roll"]); ok {
		return roll
	}
	return totalParentageRoll(normalizeCatharsisParentageRows(context["socio_parentage_parents"]))
}

func socioInheritanceSummary(context map[string]any) string {
	lines := []string{}
	if wealth := titleizeContextValue(context["socio_starting_wealth"]); wealth != "" {
		lines = append(lines, fmt.Sprintf("Starting wealth: %s", wealth))
	}
	if sourceIndex := titleizeContextValue(context["socio_starting_wealth_source_parent_index"]); sourceIndex != "" {
		lines = append(lines, fmt.Sprintf("Inherited from parent %s", sourceIndex))
	}
	if sourceRoll := titleizeContextValue(context["socio_starting_wealth_source_roll"]); sourceRoll != "" {
		lines = append(lines, fmt.Sprintf("Source roll: %s", sourceRoll))
	}
	if sourceClass := titleizeContextValue(context["socio_starting_wealth_source_class"]); sourceClass != "" {
		lines = append(lines, fmt.Sprintf("Source class: %s", sourceClass))
	}
	if eligible := titleizeContextValue(context["socio_wealth_eligible_parents"]); eligible != "" {
		lines = append(lines, fmt.Sprintf("Eligible parents: %s", eligible))
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}

func titleizeContextValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case []byte:
		return strings.TrimSpace(string(v))
	case int:
		return strconv.Itoa(v)
	case int32:
		return strconv.Itoa(int(v))
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.Itoa(int(v))
	case float32:
		return strconv.Itoa(int(v))
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return jsonStringify(v)
	}
}

func anySlice(value any) []any {
	switch v := value.(type) {
	case []any:
		return v
	case []map[string]any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, item)
		}
		return out
	default:
		return nil
	}
}

func jsonString(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func jsonStringify(value any) string {
	if value == nil {
		return "{}"
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func intValueString(value any) string {
	switch v := value.(type) {
	case int:
		return strconv.Itoa(v)
	case int32:
		return strconv.Itoa(int(v))
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.Itoa(int(v))
	default:
		return ""
	}
}
