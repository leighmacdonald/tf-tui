package internal

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
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/tfapi"
)

var (
	errPlayerNotFound   = errors.New("player not found")
	errFetchMetaProfile = errors.New("failed to fetch meta profile")
	errDecodeJSON       = errors.New("failed to decode JSON")
)

func unmarshalJSON[T any](reader io.Reader) (T, error) {
	var value T
	if err := json.NewDecoder(reader).Decode(&value); err != nil {
		return value, errors.Join(err, errDecodeJSON)
	}

	return value, nil
}

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type BDMatch struct {
	Player   tfapi.BDPlayer
	ListName string
}

func NewBDFetcher(httpClient HTTPDoer, userLists []config.UserList, cache Cache) *BDFetcher {
	return &BDFetcher{
		mu:         &sync.RWMutex{},
		configured: userLists,
		httpClient: httpClient,
		lists:      []tfapi.BDSchema{},
		cache:      cache,
	}
}

type BDFetcher struct {
	configured []config.UserList
	mu         *sync.RWMutex
	lists      []tfapi.BDSchema
	httpClient HTTPDoer
	cache      Cache
}

func (m *BDFetcher) Update(ctx context.Context) {
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

			bdList, errUnmarshal := unmarshalJSON[tfapi.BDSchema](resp.Body)
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

func (m *BDFetcher) Search(steamID steamid.SteamID) []BDMatch {
	matched := []BDMatch{}
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
				matched = append(matched, BDMatch{
					Player:   player,
					ListName: list.FileInfo.Title,
				})

				break
			}
		}
	}

	return matched
}
