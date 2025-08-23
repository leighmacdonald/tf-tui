package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/config"
	"github.com/leighmacdonald/tf-tui/tf"
	"github.com/leighmacdonald/tf-tui/tfapi"
	"github.com/leighmacdonald/tf-tui/ui"
)

const (
	maxQueueSize = 100

	// How long we wait until a player should be ejected from our tracking.
	// This should be long enough to last through map changes without dropping the
	// known players.
	playerExpiration = time.Second * 30
)

var (
	errPlayerNotFound   = errors.New("player not found")
	errFetchMetaProfile = errors.New("failed to fetch meta profile")
	errDecodeJSON       = errors.New("failed to decode JSON")
)

func UnmarshalJSON[T any](reader io.Reader) (T, error) {
	var value T
	if err := json.NewDecoder(reader).Decode(&value); err != nil {
		return value, errors.Join(err, errDecodeJSON)
	}

	return value, nil
}

type Player struct {
	SteamID       steamid.SteamID
	Name          string
	Ping          int
	Score         int
	Deaths        int
	Connected     bool
	Team          tf.Team
	Alive         bool
	Health        int
	Valid         bool
	UserID        int
	Meta          tfapi.MetaProfile
	MetaUpdatedOn time.Time
	G15UpdatedOn  time.Time
}

func (p Player) Expired() bool {
	return time.Since(p.G15UpdatedOn) > playerExpiration
}

type Players []Player

func NewPlayerDataModel(config config.Config) *PlayerDataModel {
	return &PlayerDataModel{
		mu:          &sync.RWMutex{},
		players:     make(map[steamid.SteamID]*Player),
		updateQueue: make(chan steamid.SteamID, maxQueueSize),
		config:      config,
	}
}

type PlayerDataModel struct {
	config      config.Config
	players     map[steamid.SteamID]*Player
	mu          *sync.RWMutex
	updateQueue chan steamid.SteamID
	lists       []tfapi.BDSchema
}

func (m *PlayerDataModel) Start(ctx context.Context) {
	// TODO convert into Tick msg
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	var queue steamid.Collection
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-m.updateQueue:
			if !update.Valid() {
				continue
			}
			if slices.Contains(queue, update) {
				continue
			}
			queue = append(queue, update)
		case <-ticker.C:
			if len(queue) == 0 {
				continue
			}

			m.updateMeta(ctx, queue)
			m.updateUserListMatches()
			queue = nil
		}
	}
}

func (m *PlayerDataModel) Get(steamID steamid.SteamID) (Player, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	player, found := m.players[steamID]
	if !found {
		return Player{}, fmt.Errorf("%w: %s", errPlayerNotFound, steamID.String())
	}

	return *player, nil
}

func (m *PlayerDataModel) All() (Players, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var profiles Players //nolint:prealloc
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

func (m *PlayerDataModel) updateMeta(ctx context.Context, steamIDs steamid.Collection) {
	profiles, errProfiles := m.MetaProfiles(ctx, steamIDs)
	if errProfiles != nil {
		slog.Error("Failed to update meta profiles", slog.String("error", errProfiles.Error()))

		return
	}

	m.setProfiles(profiles...)
}

func (m *PlayerDataModel) updateUserLists() []tfapi.BDSchema {
	waitGroup := sync.WaitGroup{}
	mutex := sync.Mutex{}
	// There is no context passed down to children in tea apps... :(
	ctx := context.Background()
	var lists []tfapi.BDSchema
	for _, userList := range m.config.BDLists {
		waitGroup.Add(1)

		go func(list config.UserList) {
			defer waitGroup.Done()

			reqContext, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()

			req, errReq := http.NewRequestWithContext(reqContext, http.MethodGet, list.URL, nil)
			if errReq != nil {
				slog.Error("Failed to create request", slog.String("error", errReq.Error()))

				return
			}

			resp, errResp := http.DefaultClient.Do(req) //nolint:bodyclose
			if errResp != nil {
				slog.Error("Failed to get response", slog.String("error", errResp.Error()))

				return
			}

			defer func(body io.ReadCloser) {
				if err := body.Close(); err != nil {
					slog.Error("Failed to close response body", slog.String("error", err.Error()))
				}
			}(resp.Body)

			if resp.StatusCode != http.StatusOK {
				slog.Error("Failed to get response", slog.Int("status_code", resp.StatusCode))

				return
			}

			bdList, errUnmarshal := UnmarshalJSON[tfapi.BDSchema](resp.Body)
			if errUnmarshal != nil {
				slog.Error("Failed to unmarshal", slog.String("error", errUnmarshal.Error()))

				return
			}

			if len(os.Getenv("DEBUG")) > 0 {
				bdList.Players = append(bdList.Players, tfapi.BDPlayer{
					Attributes: []string{"cheater", "liar"},
					LastSeen: tfapi.BDLastSeen{
						PlayerName: "Evil Player",
						Time:       time.Now().Unix(),
					},
					Proof: []string{
						"Some proof that can easily be manipulated.",
						"Some more nonsense",
					},
					Steamid: steamid.New("76561197960265749"),
				})
			}

			mutex.Lock()
			lists = append(lists, bdList)
			mutex.Unlock()
		}(userList)
	}

	waitGroup.Wait()

	return lists
}

func (m *PlayerDataModel) updateUserListMatches() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// for _, player := range m.players {
	// FIXME
	// player.BDMatches = m.findBDPlayerMatches(player.SteamID)
	// }
}

func (m *PlayerDataModel) findBDPlayerMatches(steamID steamid.SteamID) []ui.MatchedBDPlayer {
	var matched []ui.MatchedBDPlayer
	for _, list := range m.lists {
		for _, player := range list.Players {
			var sid steamid.SteamID
			switch value := player.Steamid.(type) {
			case string:
				sid = steamid.New(value)
			case int64:
				sid = steamid.New(value)
			case steamid.SteamID:
				sid = value
			default:
				sid = steamid.New(value)
			}
			if !sid.Valid() {
				continue
			}
			if steamID.Equal(sid) {
				matched = append(matched, ui.MatchedBDPlayer{
					Player:   player,
					ListName: list.FileInfo.Title,
				})

				break
			}
		}
	}

	return matched
}
