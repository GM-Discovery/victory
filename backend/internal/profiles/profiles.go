package profiles

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"victory/backend/internal/access"
)

type PublicFields struct {
	DisplayName         string         `json:"display_name"`
	StageName           string         `json:"stage_name"`
	Pronouns            string         `json:"pronouns"`
	HeadshotURL         string         `json:"headshot_url"`
	PerformanceAgeRange string         `json:"performance_age_range"`
	FavoriteFun         string         `json:"favorite_fun"`
	MostRelaxed         string         `json:"most_relaxed"`
	FavoriteColor       string         `json:"favorite_color"`
	FavoriteArtist      string         `json:"favorite_artist"`
	FavoriteFood        string         `json:"favorite_food"`
	FavoriteSong        string         `json:"favorite_song"`
	FavoritePlace       string         `json:"favorite_place"`
	FavoriteMovieOrShow string         `json:"favorite_movie_or_show"`
	HiddenTalent        string         `json:"hidden_talent"`
	IdealDay            string         `json:"ideal_day"`
	Bio                 string         `json:"bio"`
	Credits             string         `json:"credits"`
	Skills              string         `json:"skills"`
	Availability        string         `json:"availability"`
	PublicLinks         string         `json:"public_links"`
	History             []HistoryEntry `json:"history"`
}

type HistoryEntry struct {
	ProductionName string `json:"production_name"`
	Role           string `json:"role"`
	RunPeriod      string `json:"run_period"`
}

type Identity struct {
	UserID      string `json:"user_id"`
	Handle      string `json:"handle"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Persona     any    `json:"persona"`
}

type Profile struct {
	User        Identity     `json:"user"`
	IsPublished bool         `json:"is_published"`
	PublishedAt string       `json:"published_at"`
	State       string       `json:"state"`
	Draft       PublicFields `json:"draft"`
	Published   PublicFields `json:"published"`
	Public      PublicFields `json:"public"`
}

type ProfileView struct {
	Owner   Identity `json:"owner"`
	Profile Profile  `json:"profile"`
}

type response struct {
	Ok   bool        `json:"ok"`
	Data interface{} `json:"data,omitempty"`
}

type profileRequest struct {
	UserID              string         `json:"user_id"`
	DisplayName         string         `json:"display_name"`
	StageName           string         `json:"stage_name"`
	Pronouns            string         `json:"pronouns"`
	HeadshotURL         string         `json:"headshot_url"`
	PerformanceAgeRange string         `json:"performance_age_range"`
	FavoriteFun         string         `json:"favorite_fun"`
	MostRelaxed         string         `json:"most_relaxed"`
	FavoriteColor       string         `json:"favorite_color"`
	FavoriteArtist      string         `json:"favorite_artist"`
	FavoriteFood        string         `json:"favorite_food"`
	FavoriteSong        string         `json:"favorite_song"`
	FavoritePlace       string         `json:"favorite_place"`
	FavoriteMovieOrShow string         `json:"favorite_movie_or_show"`
	HiddenTalent        string         `json:"hidden_talent"`
	IdealDay            string         `json:"ideal_day"`
	Bio                 string         `json:"bio"`
	Credits             string         `json:"credits"`
	Skills              string         `json:"skills"`
	Availability        string         `json:"availability"`
	PublicLinks         string         `json:"public_links"`
	History             []HistoryEntry `json:"history"`
}

func EnsureKernel9ProfileSurface(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS performer_profiles (
		  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,

		  draft_display_name TEXT NOT NULL DEFAULT '',
		  draft_stage_name TEXT NOT NULL DEFAULT '',
		  draft_pronouns TEXT NOT NULL DEFAULT '',
		  draft_headshot_url TEXT NOT NULL DEFAULT '',
		  draft_performance_age_range TEXT NOT NULL DEFAULT '',
		  draft_favorite_fun TEXT NOT NULL DEFAULT '',
		  draft_most_relaxed TEXT NOT NULL DEFAULT '',
		  draft_favorite_color TEXT NOT NULL DEFAULT '',
		  draft_favorite_artist TEXT NOT NULL DEFAULT '',
		  draft_bio TEXT NOT NULL DEFAULT '',
		  draft_credits TEXT NOT NULL DEFAULT '',
		  draft_skills TEXT NOT NULL DEFAULT '',
		  draft_availability TEXT NOT NULL DEFAULT '',
		  draft_public_links TEXT NOT NULL DEFAULT '',
		  draft_history TEXT NOT NULL DEFAULT '[]',

		  published_display_name TEXT NOT NULL DEFAULT '',
		  published_stage_name TEXT NOT NULL DEFAULT '',
		  published_pronouns TEXT NOT NULL DEFAULT '',
		  published_headshot_url TEXT NOT NULL DEFAULT '',
		  published_performance_age_range TEXT NOT NULL DEFAULT '',
		  published_favorite_fun TEXT NOT NULL DEFAULT '',
		  published_most_relaxed TEXT NOT NULL DEFAULT '',
		  published_favorite_color TEXT NOT NULL DEFAULT '',
		  published_favorite_artist TEXT NOT NULL DEFAULT '',
		  published_bio TEXT NOT NULL DEFAULT '',
		  published_credits TEXT NOT NULL DEFAULT '',
		  published_skills TEXT NOT NULL DEFAULT '',
		  published_availability TEXT NOT NULL DEFAULT '',
		  published_public_links TEXT NOT NULL DEFAULT '',
		  published_history TEXT NOT NULL DEFAULT '[]',

		  is_published BOOLEAN NOT NULL DEFAULT FALSE,
		  published_at TIMESTAMPTZ,
		  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_performer_profiles_is_published
		  ON performer_profiles(is_published);

		ALTER TABLE performer_profiles
		  ADD COLUMN IF NOT EXISTS draft_favorite_fun TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS draft_most_relaxed TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS draft_favorite_color TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS draft_favorite_artist TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS draft_favorite_food TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS draft_favorite_song TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS draft_favorite_place TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS draft_favorite_movie_or_show TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS draft_hidden_talent TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS draft_ideal_day TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS draft_display_name TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS draft_history TEXT NOT NULL DEFAULT '[]',
		  ADD COLUMN IF NOT EXISTS published_favorite_fun TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS published_most_relaxed TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS published_favorite_color TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS published_favorite_artist TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS published_favorite_food TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS published_favorite_song TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS published_favorite_place TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS published_favorite_movie_or_show TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS published_hidden_talent TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS published_ideal_day TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS published_display_name TEXT NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS published_history TEXT NOT NULL DEFAULT '[]';
	`)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		WITH location_row AS (
		  SELECT id FROM locations WHERE slug = 'amurray-family' LIMIT 1
		),
		lot_row AS (
		  SELECT id FROM lots
		  WHERE location_id = (SELECT id FROM location_row)
		    AND slug = 'main-lot'
		  LIMIT 1
		)
		INSERT INTO venues (lot_id, name, slug, kind, config, is_public, is_workshop)
		SELECT
		  (SELECT id FROM lot_row),
		  'The Greenroom',
		  'greenroom',
		  'profile',
		  '{
			"surface":"identity",
			"public_profile":true,
			"edit_link":"/venues/trailers/"
		  }'::jsonb,
		  FALSE,
		  FALSE
		WHERE NOT EXISTS (
		  SELECT 1 FROM venues
		  WHERE lot_id = (SELECT id FROM lot_row)
		    AND slug = 'greenroom'
		);

		WITH location_row AS (
		  SELECT id FROM locations WHERE slug = 'amurray-family' LIMIT 1
		),
		lot_row AS (
		  SELECT id FROM lots
		  WHERE location_id = (SELECT id FROM location_row)
		    AND slug = 'main-lot'
		  LIMIT 1
		)
		INSERT INTO venues (lot_id, name, slug, kind, config, is_public, is_workshop)
		SELECT
		  (SELECT id FROM lot_row),
		  'Trailers',
		  'trailers',
		  'profile',
		  '{
			"surface":"identity",
			"drafting":true,
			"public_profile":false
		  }'::jsonb,
		  FALSE,
		  FALSE
		WHERE NOT EXISTS (
		  SELECT 1 FROM venues
		  WHERE lot_id = (SELECT id FROM lot_row)
		    AND slug = 'trailers'
		);
	`)
	if err != nil {
		return err
	}

	return nil
}

func HandleGetMyProfile(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeForbiddenJSON(w, err)
			return
		}

		row, err := loadProfile(ctx, pool, userID)
		if err != nil {
			writeErrorJSON(w, err)
			return
		}

		writeJSON(w, http.StatusOK, response{Ok: true, Data: row})
	}
}

func HandleGetPublicProfile(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeForbiddenJSON(w, err)
			return
		}

		targetUserID := strings.TrimSpace(r.URL.Query().Get("user_id"))
		if targetUserID == "" {
			targetUserID = userID
		}

		row, err := loadProfile(ctx, pool, targetUserID)
		if err != nil {
			writeErrorJSON(w, err)
			return
		}

		row.Profile = row.Profile.publicProjection()
		row.Profile.Draft = PublicFields{}
		row.Profile.Published = PublicFields{}
		writeJSON(w, http.StatusOK, response{Ok: true, Data: row})
	}
}

func HandleSaveMyProfile(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch && r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeForbiddenJSON(w, err)
			return
		}

		var input profileRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeErrorJSON(w, errors.New("invalid_json"))
			return
		}

		input.UserID = userID
		row, err := upsertDraftProfile(ctx, pool, input)
		if err != nil {
			writeErrorJSON(w, err)
			return
		}

		writeJSON(w, http.StatusOK, response{Ok: true, Data: row})
	}
}

func HandlePublishMyProfile(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeForbiddenJSON(w, err)
			return
		}

		row, err := publishProfile(ctx, pool, userID)
		if err != nil {
			writeErrorJSON(w, err)
			return
		}

		writeJSON(w, http.StatusOK, response{Ok: true, Data: row})
	}
}

func HandleAdminSaveProfile(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch && r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeForbiddenJSON(w, err)
			return
		}

		if ok, err := hasProducerOverride(ctx, pool, userID); err != nil {
			writeErrorJSON(w, err)
			return
		} else if !ok {
			writeForbiddenJSON(w, errors.New("producer_membership_required"))
			return
		}

		var input profileRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeErrorJSON(w, errors.New("invalid_json"))
			return
		}

		if strings.TrimSpace(input.UserID) == "" {
			writeErrorJSON(w, errors.New("user_id is required"))
			return
		}

		row, err := upsertDraftProfile(ctx, pool, input)
		if err != nil {
			writeErrorJSON(w, err)
			return
		}

		writeJSON(w, http.StatusOK, response{Ok: true, Data: row})
	}
}

func HandleAdminPublishProfile(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Ok: false})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID, err := requireAuthenticatedUser(ctx, pool, r)
		if err != nil {
			writeForbiddenJSON(w, err)
			return
		}

		if ok, err := hasProducerOverride(ctx, pool, userID); err != nil {
			writeErrorJSON(w, err)
			return
		} else if !ok {
			writeForbiddenJSON(w, errors.New("producer_membership_required"))
			return
		}

		var input profileRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeErrorJSON(w, errors.New("invalid_json"))
			return
		}

		if strings.TrimSpace(input.UserID) == "" {
			writeErrorJSON(w, errors.New("user_id is required"))
			return
		}

		row, err := publishProfile(ctx, pool, input.UserID)
		if err != nil {
			writeErrorJSON(w, err)
			return
		}

		writeJSON(w, http.StatusOK, response{Ok: true, Data: row})
	}
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

func loadProfile(ctx context.Context, pool *pgxpool.Pool, userID string) (ProfileView, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ProfileView{}, errors.New("user_id is required")
	}

	var row ProfileView
	var publishedAt string
	var draft, published PublicFields
	var draftHistoryJSON, publishedHistoryJSON string

	err := pool.QueryRow(ctx, `
		SELECT
			u.id::text,
			COALESCE(NULLIF(u.handle, ''), LEFT(u.id::text, 8), 'Unknown Participant'),
			COALESCE(NULLIF(u.display_name, ''), NULLIF(u.handle, ''), LEFT(u.id::text, 8), 'Unknown Participant'),
			COALESCE((
				SELECT lm.role::text
				FROM location_memberships lm
				WHERE lm.user_id = u.id
				  AND lm.active = TRUE
				ORDER BY
				  CASE lm.role
					WHEN 'producer' THEN 1
					WHEN 'director' THEN 2
					WHEN 'cast' THEN 3
					WHEN 'crew' THEN 4
					WHEN 'audience' THEN 5
					ELSE 99
				  END,
				  lm.created_at ASC
				LIMIT 1
				), 'audience'),
				COALESCE(pp.is_published, FALSE),
				COALESCE(pp.published_at::text, ''),
				COALESCE(pp.draft_display_name, ''),
				COALESCE(pp.draft_stage_name, ''),
				COALESCE(pp.draft_pronouns, ''),
				COALESCE(pp.draft_headshot_url, ''),
				COALESCE(pp.draft_performance_age_range, ''),
			COALESCE(pp.draft_favorite_fun, ''),
			COALESCE(pp.draft_most_relaxed, ''),
			COALESCE(pp.draft_favorite_color, ''),
			COALESCE(pp.draft_favorite_artist, ''),
			COALESCE(pp.draft_favorite_food, ''),
			COALESCE(pp.draft_favorite_song, ''),
			COALESCE(pp.draft_favorite_place, ''),
			COALESCE(pp.draft_favorite_movie_or_show, ''),
			COALESCE(pp.draft_hidden_talent, ''),
			COALESCE(pp.draft_ideal_day, ''),
				COALESCE(pp.draft_bio, ''),
				COALESCE(pp.draft_credits, ''),
				COALESCE(pp.draft_skills, ''),
				COALESCE(pp.draft_availability, ''),
				COALESCE(pp.draft_public_links, ''),
				COALESCE(pp.draft_history, '[]'),
				COALESCE(pp.published_display_name, ''),
				COALESCE(pp.published_stage_name, ''),
				COALESCE(pp.published_pronouns, ''),
				COALESCE(pp.published_headshot_url, ''),
			COALESCE(pp.published_performance_age_range, ''),
			COALESCE(pp.published_favorite_fun, ''),
			COALESCE(pp.published_most_relaxed, ''),
			COALESCE(pp.published_favorite_color, ''),
			COALESCE(pp.published_favorite_artist, ''),
			COALESCE(pp.published_favorite_food, ''),
			COALESCE(pp.published_favorite_song, ''),
			COALESCE(pp.published_favorite_place, ''),
			COALESCE(pp.published_favorite_movie_or_show, ''),
			COALESCE(pp.published_hidden_talent, ''),
			COALESCE(pp.published_ideal_day, ''),
				COALESCE(pp.published_bio, ''),
				COALESCE(pp.published_credits, ''),
				COALESCE(pp.published_skills, ''),
				COALESCE(pp.published_availability, ''),
				COALESCE(pp.published_public_links, ''),
				COALESCE(pp.published_history, '[]')
			FROM users u
			LEFT JOIN performer_profiles pp ON pp.user_id = u.id
			WHERE u.id = $1
			LIMIT 1
		`, userID).Scan(
		&row.Owner.UserID,
		&row.Owner.Handle,
		&row.Owner.DisplayName,
		&row.Owner.Role,
		&row.Profile.IsPublished,
		&publishedAt,
		&draft.DisplayName,
		&draft.StageName,
		&draft.Pronouns,
		&draft.HeadshotURL,
		&draft.PerformanceAgeRange,
		&draft.FavoriteFun,
		&draft.MostRelaxed,
		&draft.FavoriteColor,
		&draft.FavoriteArtist,
		&draft.FavoriteFood,
		&draft.FavoriteSong,
		&draft.FavoritePlace,
		&draft.FavoriteMovieOrShow,
		&draft.HiddenTalent,
		&draft.IdealDay,
		&draft.Bio,
		&draft.Credits,
		&draft.Skills,
		&draft.Availability,
		&draft.PublicLinks,
		&draftHistoryJSON,
		&published.DisplayName,
		&published.StageName,
		&published.Pronouns,
		&published.HeadshotURL,
		&published.PerformanceAgeRange,
		&published.FavoriteFun,
		&published.MostRelaxed,
		&published.FavoriteColor,
		&published.FavoriteArtist,
		&published.FavoriteFood,
		&published.FavoriteSong,
		&published.FavoritePlace,
		&published.FavoriteMovieOrShow,
		&published.HiddenTalent,
		&published.IdealDay,
		&published.Bio,
		&published.Credits,
		&published.Skills,
		&published.Availability,
		&published.PublicLinks,
		&publishedHistoryJSON,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProfileView{}, errors.New("user_not_found")
		}
		return ProfileView{}, err
	}

	row.Owner.Persona = nil
	row.Profile.User = row.Owner
	row.Profile.PublishedAt = publishedAt
	draft.History = decodeHistoryEntries(draftHistoryJSON)
	published.History = decodeHistoryEntries(publishedHistoryJSON)
	row.Profile.Draft = draft
	row.Profile.Published = published
	row.Profile.Public = publishedProjection(row.Profile)
	row.Profile.State = profileState(row.Profile)
	return row, nil
}

func upsertDraftProfile(ctx context.Context, pool *pgxpool.Pool, input profileRequest) (ProfileView, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	if input.UserID == "" {
		return ProfileView{}, errors.New("user_id is required")
	}

	sanitized := sanitizeProfileRequest(input)

	if sanitized.DisplayName != "" {
		if _, err := pool.Exec(ctx, `
			UPDATE users
			SET display_name = $2,
				updated_at = NOW()
			WHERE id = $1
		`, sanitized.UserID, sanitized.DisplayName); err != nil {
			return ProfileView{}, err
		}
	}

	draftHistoryJSON, err := encodeHistoryEntries(sanitized.History)
	if err != nil {
		return ProfileView{}, err
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO performer_profiles (
			user_id,
			draft_display_name,
			draft_stage_name,
			draft_pronouns,
			draft_headshot_url,
			draft_performance_age_range,
			draft_favorite_fun,
			draft_most_relaxed,
			draft_favorite_color,
			draft_favorite_artist,
			draft_favorite_food,
			draft_favorite_song,
			draft_favorite_place,
			draft_favorite_movie_or_show,
			draft_hidden_talent,
			draft_ideal_day,
			draft_bio,
			draft_credits,
			draft_skills,
			draft_availability,
			draft_public_links,
			draft_history,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, NOW())
		ON CONFLICT (user_id) DO UPDATE
		SET
			draft_display_name = EXCLUDED.draft_display_name,
			draft_stage_name = EXCLUDED.draft_stage_name,
			draft_pronouns = EXCLUDED.draft_pronouns,
			draft_headshot_url = EXCLUDED.draft_headshot_url,
			draft_performance_age_range = EXCLUDED.draft_performance_age_range,
			draft_favorite_fun = EXCLUDED.draft_favorite_fun,
			draft_most_relaxed = EXCLUDED.draft_most_relaxed,
			draft_favorite_color = EXCLUDED.draft_favorite_color,
			draft_favorite_artist = EXCLUDED.draft_favorite_artist,
			draft_favorite_food = EXCLUDED.draft_favorite_food,
			draft_favorite_song = EXCLUDED.draft_favorite_song,
			draft_favorite_place = EXCLUDED.draft_favorite_place,
			draft_favorite_movie_or_show = EXCLUDED.draft_favorite_movie_or_show,
			draft_hidden_talent = EXCLUDED.draft_hidden_talent,
			draft_ideal_day = EXCLUDED.draft_ideal_day,
			draft_bio = EXCLUDED.draft_bio,
			draft_credits = EXCLUDED.draft_credits,
			draft_skills = EXCLUDED.draft_skills,
			draft_availability = EXCLUDED.draft_availability,
			draft_public_links = EXCLUDED.draft_public_links,
			draft_history = EXCLUDED.draft_history,
			updated_at = NOW()
		`, sanitized.UserID, sanitized.DisplayName, sanitized.StageName, sanitized.Pronouns, sanitized.HeadshotURL, sanitized.PerformanceAgeRange, sanitized.FavoriteFun, sanitized.MostRelaxed, sanitized.FavoriteColor, sanitized.FavoriteArtist, sanitized.FavoriteFood, sanitized.FavoriteSong, sanitized.FavoritePlace, sanitized.FavoriteMovieOrShow, sanitized.HiddenTalent, sanitized.IdealDay, sanitized.Bio, sanitized.Credits, sanitized.Skills, sanitized.Availability, sanitized.PublicLinks, draftHistoryJSON)
	if err != nil {
		return ProfileView{}, err
	}

	return loadProfile(ctx, pool, sanitized.UserID)
}

func publishProfile(ctx context.Context, pool *pgxpool.Pool, userID string) (ProfileView, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ProfileView{}, errors.New("user_id is required")
	}

	_, err := pool.Exec(ctx, `
		INSERT INTO performer_profiles (user_id, updated_at)
		VALUES ($1, NOW())
		ON CONFLICT (user_id) DO NOTHING
	`, userID)
	if err != nil {
		return ProfileView{}, err
	}

	_, err = pool.Exec(ctx, `
		UPDATE performer_profiles
		SET
			published_display_name = draft_display_name,
			published_stage_name = draft_stage_name,
			published_pronouns = draft_pronouns,
			published_headshot_url = draft_headshot_url,
			published_performance_age_range = draft_performance_age_range,
			published_favorite_fun = draft_favorite_fun,
			published_most_relaxed = draft_most_relaxed,
			published_favorite_color = draft_favorite_color,
			published_favorite_artist = draft_favorite_artist,
			published_favorite_food = draft_favorite_food,
			published_favorite_song = draft_favorite_song,
			published_favorite_place = draft_favorite_place,
			published_favorite_movie_or_show = draft_favorite_movie_or_show,
			published_hidden_talent = draft_hidden_talent,
			published_ideal_day = draft_ideal_day,
			published_bio = draft_bio,
			published_credits = draft_credits,
			published_skills = draft_skills,
			published_availability = draft_availability,
			published_public_links = draft_public_links,
			published_history = draft_history,
			is_published = TRUE,
			published_at = NOW(),
			updated_at = NOW()
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return ProfileView{}, err
	}

	return loadProfile(ctx, pool, userID)
}

func hasProducerOverride(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, error) {
	if ok, err := access.IsOperatorUser(ctx, pool, userID); err == nil && ok {
		return true, nil
	}

	var ok bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM location_memberships lm
			WHERE lm.user_id = $1
			  AND lm.role = 'producer'
			  AND lm.active = TRUE
		)
	`, strings.TrimSpace(userID)).Scan(&ok)
	return ok, err
}

func sanitizeProfileRequest(in profileRequest) profileRequest {
	in.UserID = strings.TrimSpace(in.UserID)
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.StageName = strings.TrimSpace(in.StageName)
	in.Pronouns = strings.TrimSpace(in.Pronouns)
	in.HeadshotURL = strings.TrimSpace(in.HeadshotURL)
	in.PerformanceAgeRange = strings.TrimSpace(in.PerformanceAgeRange)
	in.FavoriteFun = strings.TrimSpace(in.FavoriteFun)
	in.MostRelaxed = strings.TrimSpace(in.MostRelaxed)
	in.FavoriteColor = strings.TrimSpace(in.FavoriteColor)
	in.FavoriteArtist = strings.TrimSpace(in.FavoriteArtist)
	in.FavoriteFood = strings.TrimSpace(in.FavoriteFood)
	in.FavoriteSong = strings.TrimSpace(in.FavoriteSong)
	in.FavoritePlace = strings.TrimSpace(in.FavoritePlace)
	in.FavoriteMovieOrShow = strings.TrimSpace(in.FavoriteMovieOrShow)
	in.HiddenTalent = strings.TrimSpace(in.HiddenTalent)
	in.IdealDay = strings.TrimSpace(in.IdealDay)
	in.Bio = strings.TrimSpace(in.Bio)
	in.Credits = strings.TrimSpace(in.Credits)
	in.Skills = strings.TrimSpace(in.Skills)
	in.Availability = strings.TrimSpace(in.Availability)
	in.PublicLinks = strings.TrimSpace(in.PublicLinks)
	in.History = normalizeHistoryEntries(in.History)
	return in
}

func normalizeHistoryEntries(entries []HistoryEntry) []HistoryEntry {
	cleaned := make([]HistoryEntry, 0, len(entries))
	for _, entry := range entries {
		productionName := strings.TrimSpace(entry.ProductionName)
		role := strings.TrimSpace(entry.Role)
		runPeriod := strings.TrimSpace(entry.RunPeriod)
		if productionName == "" && role == "" && runPeriod == "" {
			continue
		}
		cleaned = append(cleaned, HistoryEntry{
			ProductionName: productionName,
			Role:           role,
			RunPeriod:      runPeriod,
		})
	}
	return cleaned
}

func encodeHistoryEntries(entries []HistoryEntry) (string, error) {
	if len(entries) == 0 {
		return "[]", nil
	}
	raw, err := json.Marshal(normalizeHistoryEntries(entries))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func decodeHistoryEntries(raw string) []HistoryEntry {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return []HistoryEntry{}
	}

	var entries []HistoryEntry
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return []HistoryEntry{}
	}

	return normalizeHistoryEntries(entries)
}

func profileState(profile Profile) string {
	if profile.IsPublished && hasAnyPublicProfileData(profile.Published) {
		return "published"
	}
	if hasAnyPublicProfileData(profile.Draft) {
		return "draft"
	}
	return "empty"
}

func publishedProjection(profile Profile) PublicFields {
	if profile.IsPublished && hasAnyPublicProfileData(profile.Published) {
		return profile.Published
	}
	return PublicFields{}
}

func (p Profile) publicProjection() Profile {
	p.Public = publishedProjection(p)
	p.State = profileState(p)
	return p
}

func hasAnyPublicProfileData(fields PublicFields) bool {
	return strings.TrimSpace(fields.DisplayName) != "" ||
		strings.TrimSpace(fields.StageName) != "" ||
		strings.TrimSpace(fields.Pronouns) != "" ||
		strings.TrimSpace(fields.HeadshotURL) != "" ||
		strings.TrimSpace(fields.PerformanceAgeRange) != "" ||
		strings.TrimSpace(fields.FavoriteFun) != "" ||
		strings.TrimSpace(fields.MostRelaxed) != "" ||
		strings.TrimSpace(fields.FavoriteColor) != "" ||
		strings.TrimSpace(fields.FavoriteArtist) != "" ||
		strings.TrimSpace(fields.FavoriteFood) != "" ||
		strings.TrimSpace(fields.FavoriteSong) != "" ||
		strings.TrimSpace(fields.FavoritePlace) != "" ||
		strings.TrimSpace(fields.FavoriteMovieOrShow) != "" ||
		strings.TrimSpace(fields.HiddenTalent) != "" ||
		strings.TrimSpace(fields.IdealDay) != "" ||
		strings.TrimSpace(fields.Bio) != "" ||
		strings.TrimSpace(fields.Credits) != "" ||
		strings.TrimSpace(fields.Skills) != "" ||
		strings.TrimSpace(fields.Availability) != "" ||
		strings.TrimSpace(fields.PublicLinks) != "" ||
		len(fields.History) > 0
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeErrorJSON(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	code := err.Error()

	switch code {
	case "not_authenticated":
		status = http.StatusUnauthorized
	case "producer_membership_required":
		status = http.StatusForbidden
	case "user_not_found":
		status = http.StatusNotFound
	}

	writeJSON(w, status, response{Ok: false, Data: map[string]any{
		"error": code,
	}})
}

func writeForbiddenJSON(w http.ResponseWriter, err error) {
	code := "forbidden"
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		code = err.Error()
	}

	status := http.StatusForbidden
	if code == "not_authenticated" {
		status = http.StatusUnauthorized
	}

	writeJSON(w, status, response{Ok: false, Data: map[string]any{
		"error": code,
	}})
}
