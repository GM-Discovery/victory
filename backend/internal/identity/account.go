package identity

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"victory/backend/internal/access"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccountSummary struct {
	User        AccountUser         `json:"user"`
	Auth        AccountAuth         `json:"auth"`
	Memberships []AccountMembership `json:"memberships"`
	Authority   AccountAuthority    `json:"authority"`
}

type AccountUser struct {
	ID            string `json:"id"`
	Handle        string `json:"handle"`
	DisplayName   string `json:"display_name"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	HasPassword   bool   `json:"has_password"`
}

type AccountAuth struct {
	DiscordLinked bool                `json:"discord_linked"`
	Discord       *AccountDiscordLink `json:"discord,omitempty"`
}

type AccountDiscordLink struct {
	DiscordUserID string `json:"discord_user_id"`
	Username      string `json:"username"`
	GlobalName    string `json:"global_name"`
}

type AccountMembership struct {
	LocationID   string `json:"location_id"`
	LocationSlug string `json:"location_slug"`
	LocationName string `json:"location_name"`
	Role         string `json:"role"`
}

type AccountAuthority struct {
	IsProducer        bool     `json:"is_producer"`
	ProducerLocations []string `json:"producer_locations"`
	CurrentRole       string   `json:"current_role"`
	IsOperator        bool     `json:"is_operator"`
}

func HandleAccountMe(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"ok":    false,
				"error": "method_not_allowed",
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		userID, err := currentUserID(ctx, pool, r)
		if err != nil || strings.TrimSpace(userID) == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"ok":    false,
				"error": "not_authenticated",
			})
			return
		}

		account, err := loadAccountSummary(ctx, pool, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, errAccountNotFound) {
				writeJSON(w, http.StatusUnauthorized, map[string]any{
					"ok":    false,
					"error": "not_authenticated",
				})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok":    false,
				"error": "failed_to_load_account",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"data": account,
		})
	}
}

var errAccountNotFound = errors.New("account_not_found")

func loadAccountSummary(ctx context.Context, pool *pgxpool.Pool, userID string) (AccountSummary, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return AccountSummary{}, errAccountNotFound
	}

	var account AccountSummary
	if err := pool.QueryRow(ctx, `
		SELECT
			u.id::text,
			COALESCE(NULLIF(u.handle, ''), LEFT(u.id::text, 8), 'Unknown Participant'),
			COALESCE(NULLIF(u.display_name, ''), NULLIF(u.handle, ''), LEFT(u.id::text, 8), 'Unknown Participant'),
			COALESCE(u.email, ''),
			u.email_verified_at IS NOT NULL,
			EXISTS (SELECT 1 FROM auth.password_credentials pc WHERE pc.user_id = u.id)
		FROM users u
		WHERE u.id = $1
		LIMIT 1
	`, userID).Scan(&account.User.ID, &account.User.Handle, &account.User.DisplayName, &account.User.Email, &account.User.EmailVerified, &account.User.HasPassword); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AccountSummary{}, errAccountNotFound
		}
		return AccountSummary{}, err
	}

	var discordUserID, discordUsername, discordGlobalName string
	if err := pool.QueryRow(ctx, `
		SELECT
			discord_user_id,
			COALESCE(NULLIF(username, ''), ''),
			COALESCE(NULLIF(global_name, ''), '')
		FROM auth.discord_identities
		WHERE user_id = $1
		LIMIT 1
	`, userID).Scan(&discordUserID, &discordUsername, &discordGlobalName); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return AccountSummary{}, err
		}
	} else {
		account.Auth.DiscordLinked = true
		account.Auth.Discord = &AccountDiscordLink{
			DiscordUserID: discordUserID,
			Username:      discordUsername,
			GlobalName:    discordGlobalName,
		}
	}

	rows, err := pool.Query(ctx, `
		SELECT
			lm.location_id::text,
			COALESCE(NULLIF(l.slug, ''), ''),
			COALESCE(NULLIF(l.name, ''), ''),
			lm.role::text
		FROM location_memberships lm
		JOIN locations l ON l.id = lm.location_id
		WHERE lm.user_id = $1
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
		  l.slug ASC,
		  lm.created_at ASC
	`, userID)
	if err != nil {
		return AccountSummary{}, err
	}
	defer rows.Close()

	producerLocations := make([]string, 0, 2)
	membershipSeen := map[string]struct{}{}
	for rows.Next() {
		var membership AccountMembership
		if err := rows.Scan(&membership.LocationID, &membership.LocationSlug, &membership.LocationName, &membership.Role); err != nil {
			return AccountSummary{}, err
		}

		key := membership.LocationID + "|" + membership.Role
		if _, ok := membershipSeen[key]; ok {
			continue
		}
		membershipSeen[key] = struct{}{}

		account.Memberships = append(account.Memberships, membership)
		if membership.Role == "producer" {
			producerLocations = append(producerLocations, membership.LocationSlug)
		}
	}
	if err := rows.Err(); err != nil {
		return AccountSummary{}, err
	}

	currentRole, err := access.CurrentDefaultLocationRole(ctx, pool, userID)
	if err != nil || strings.TrimSpace(currentRole) == "" {
		currentRole = "audience"
	}

	isOperator, err := access.IsOperatorUser(ctx, pool, userID)
	if err != nil {
		isOperator = false
	}

	account.Authority = AccountAuthority{
		IsProducer:        len(producerLocations) > 0,
		ProducerLocations: dedupeStrings(producerLocations),
		CurrentRole:       currentRole,
		IsOperator:        isOperator,
	}

	return account, nil
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
