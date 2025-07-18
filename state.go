package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	// How long we wait until a player should be ejected from our tracking.
	// This should be long enough to last through map changes without dropping the
	// known players.
	playerExpiration = time.Second * 300
	maxQueueSize     = 100
)

var (
	errPlayerNotFound = errors.New("player not found")
)

type Player struct {
	SteamID       steamid.SteamID
	Name          string
	Ping          int
	Score         int
	Deaths        int
	Connected     bool
	Team          Team
	Alive         bool
	Health        int
	Valid         bool
	UserID        int
	meta          MetaProfile
	metaUpdatedOn time.Time
	g15UpdatedOn  time.Time
}

func (p Player) Expired() bool {
	return time.Since(p.g15UpdatedOn) > playerExpiration*2
}

type PlayerData struct {
	players     map[steamid.SteamID]*Player
	mu          *sync.RWMutex
	updateQueue chan steamid.SteamID
	apis        APIs
}

func newPlayerStates(apis APIs) *PlayerData {
	return &PlayerData{
		mu:          &sync.RWMutex{},
		players:     make(map[steamid.SteamID]*Player),
		updateQueue: make(chan steamid.SteamID, maxQueueSize),
		apis:        apis,
	}
}

func (m *PlayerData) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	var queue steamid.Collection
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-m.updateQueue:
			if slices.Contains(queue, update) {
				continue
			}
			queue = append(queue, update)
		case <-ticker.C:
			if len(queue) == 0 {
				continue
			}

			m.updateMeta(ctx, queue)
			queue = nil
		}
	}
}

func (m *PlayerData) setProfiles(profiles ...MetaProfile) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, profile := range profiles {
		sid := steamid.New(profile.SteamId)
		if !sid.Valid() {
			continue
		}

		player, found := m.players[sid]
		if !found {
			player = &Player{SteamID: sid}
		}

		player.meta = profile
		player.metaUpdatedOn = time.Now()

		m.players[sid] = player
	}
}

func (m *PlayerData) SetStats(stats G15PlayerState) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for idx := range MaxDataSize {
		sid := stats.SteamID[idx]
		if !sid.Valid() {
			// TODO verify this is ok, however i think g15 is filled sequentially.
			continue
		}

		player, found := m.players[sid]
		if !found {
			player = &Player{SteamID: sid}
		}

		player.Valid = stats.Valid[idx]
		player.Health = stats.Health[idx]
		player.Alive = stats.Alive[idx]
		player.Deaths = stats.Deaths[idx]
		player.Ping = stats.Ping[idx]
		player.Health = stats.Health[idx]
		player.Score = stats.Score[idx]
		player.Connected = stats.Connected[idx]
		player.Name = stats.Names[idx]
		player.Team = Team(stats.Team[idx])
		player.UserID = stats.UserID[idx]
		player.g15UpdatedOn = time.Now()

		if !found || time.Since(player.metaUpdatedOn) > time.Hour*24 {
			// Queue for a meta profile update
			select {
			case m.updateQueue <- sid:
			default:
			}
		}
	}
}

func (m *PlayerData) Get(steamID steamid.SteamID) (Player, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	player, found := m.players[steamID]
	if !found {
		return Player{}, fmt.Errorf("%w: %s", errPlayerNotFound, steamID.String())
	}

	return *player, nil
}

func (m *PlayerData) All() ([]Player, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var profiles []Player
	for _, player := range m.players {
		// Remove the expired player entries from the active player list
		if player.Expired() {
			delete(m.players, player.SteamID)

			continue
		}
		profiles = append(profiles, *player)
	}

	return profiles, nil
}

func (m *PlayerData) updateMeta(ctx context.Context, steamIDs steamid.Collection) {
	profiles, errProfiles := m.apis.getMetaProfiles(ctx, steamIDs)
	if errProfiles != nil {
		tea.Printf("errProfiles: %v\n", errProfiles)

		return
	}

	m.setProfiles(profiles...)
}

func (m *PlayerData) ByUID(uid int) (Player, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, player := range m.players {
		if player.UserID == uid {
			return *player, true
		}
	}

	return Player{}, false
}
