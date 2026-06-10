package identity

import (
	"sort"
	"strings"
	"sync"
	"time"
)

type DiscordAudioPresenceState struct {
	GuildID       string
	ChannelID     string
	UserID        string
	Username      string
	GlobalName    string
	Nick          string
	AvatarHash    string
	Discriminator string
	SelfMute      bool
	SelfDeaf      bool
	Mute          bool
	Deaf          bool
	LastSeenAt    time.Time
}

type DiscordAudioPresenceStore struct {
	mu     sync.RWMutex
	guilds map[string]map[string]DiscordAudioPresenceState
}

func NewDiscordAudioPresenceStore() *DiscordAudioPresenceStore {
	return &DiscordAudioPresenceStore{
		guilds: map[string]map[string]DiscordAudioPresenceState{},
	}
}

func (s *DiscordAudioPresenceStore) Upsert(state DiscordAudioPresenceState) {
	if s == nil {
		return
	}
	guildID := strings.TrimSpace(state.GuildID)
	userID := strings.TrimSpace(state.UserID)
	if guildID == "" || userID == "" {
		return
	}
	state.GuildID = guildID
	state.UserID = userID
	if state.LastSeenAt.IsZero() {
		state.LastSeenAt = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.guilds == nil {
		s.guilds = map[string]map[string]DiscordAudioPresenceState{}
	}
	if _, ok := s.guilds[guildID]; !ok {
		s.guilds[guildID] = map[string]DiscordAudioPresenceState{}
	}
	s.guilds[guildID][userID] = state
}

func (s *DiscordAudioPresenceStore) Remove(guildID, userID string) {
	if s == nil {
		return
	}
	guildID = strings.TrimSpace(guildID)
	userID = strings.TrimSpace(userID)
	if guildID == "" || userID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if guilds := s.guilds; guilds != nil {
		if entries, ok := guilds[guildID]; ok {
			delete(entries, userID)
			if len(entries) == 0 {
				delete(guilds, guildID)
			}
		}
	}
}

func (s *DiscordAudioPresenceStore) ReplaceGuildVoiceStates(guildID string, states []DiscordAudioPresenceState) {
	if s == nil {
		return
	}
	guildID = strings.TrimSpace(guildID)
	if guildID == "" {
		return
	}

	next := make(map[string]DiscordAudioPresenceState, len(states))
	for _, state := range states {
		if strings.TrimSpace(state.UserID) == "" {
			continue
		}
		state.GuildID = guildID
		if state.LastSeenAt.IsZero() {
			state.LastSeenAt = time.Now().UTC()
		}
		next[strings.TrimSpace(state.UserID)] = state
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.guilds == nil {
		s.guilds = map[string]map[string]DiscordAudioPresenceState{}
	}
	if len(next) == 0 {
		delete(s.guilds, guildID)
		return
	}
	s.guilds[guildID] = next
}

func (s *DiscordAudioPresenceStore) Snapshot(guildID, channelID string) []DiscordAudioPresenceState {
	if s == nil {
		return nil
	}
	guildID = strings.TrimSpace(guildID)
	channelID = strings.TrimSpace(channelID)
	if guildID == "" || channelID == "" {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	guildStates := s.guilds[guildID]
	if len(guildStates) == 0 {
		return []DiscordAudioPresenceState{}
	}

	items := make([]DiscordAudioPresenceState, 0, len(guildStates))
	for _, state := range guildStates {
		if strings.TrimSpace(state.ChannelID) != channelID {
			continue
		}
		items = append(items, state)
	}

	sort.SliceStable(items, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(displayNameForPresenceState(items[i])))
		right := strings.ToLower(strings.TrimSpace(displayNameForPresenceState(items[j])))
		if left == right {
			return strings.ToLower(strings.TrimSpace(items[i].UserID)) < strings.ToLower(strings.TrimSpace(items[j].UserID))
		}
		return left < right
	})

	return items
}

func displayNameForPresenceState(state DiscordAudioPresenceState) string {
	for _, value := range []string{state.Nick, state.GlobalName, state.Username, state.UserID} {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return state.UserID
}
