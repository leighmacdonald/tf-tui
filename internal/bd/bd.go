// Package bd handles fetching and querying the TF2 Bot Detector list schema.
package bd

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/cache"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/network/encoding"
	"github.com/leighmacdonald/tf-tui/internal/tfapi"
)

// HTTPDoer defines a common interface for HTTP clients.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// Match represents a matching bot detector player entry.
type Match struct {
	Player   tfapi.BDPlayer
	ListName string
}

// New creates a new bot detector client.
func New(httpClient HTTPDoer, userLists []config.UserList, cache cache.Cache) *Fetcher {
	return &Fetcher{
		mu:         &sync.RWMutex{},
		configured: userLists,
		httpClient: httpClient,
		lists:      []tfapi.BDSchema{},
		cache:      cache,
	}
}

// Fetcher tracks the current known bot detector list state, updating periodically.
type Fetcher struct {
	configured []config.UserList
	mu         *sync.RWMutex
	lists      []tfapi.BDSchema
	httpClient HTTPDoer
	cache      cache.Cache
}

// Update downloads all configured bot lists concurrently.
func (m *Fetcher) Update(ctx context.Context) {
	var (
		waitGroup = sync.WaitGroup{}
		lists     = make([]tfapi.BDSchema, len(m.configured))
		updates   = make(chan tfapi.BDSchema)
	)

	for _, userList := range m.configured {
		waitGroup.Add(1)

		go func(list config.UserList) {
			defer waitGroup.Done()

			req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, list.URL, nil)
			if errReq != nil {
				slog.Error("Failed to create request", slog.String("error", errReq.Error()))

				return
			}

			resp, errResp := m.httpClient.Do(req) //nolint:bodyclose
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

			bdList, errUnmarshal := encoding.UnmarshalJSON[tfapi.BDSchema](resp.Body)
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

			// TODO save cached copy
			// m.cache.Set(...)

			updates <- bdList
			slog.Debug("Downloaded bd list", slog.String("name", bdList.FileInfo.Title))
		}(userList)
	}

	go func() {
		waitGroup.Wait()
		close(updates)
	}()

	for update := range updates {
		lists = append(lists, update)
	}

	m.mu.Lock()
	m.lists = lists
	m.mu.Unlock()
}

// Search through all bot lists. Can return multiple matched results.
func (m *Fetcher) Search(steamID steamid.SteamID) []Match {
	matched := []Match{}
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
				matched = append(matched, Match{
					Player:   player,
					ListName: list.FileInfo.Title,
				})

				break
			}
		}
	}

	return matched
}
