package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/config"
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
