package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/config"
	"github.com/leighmacdonald/tf-tui/tf"
	"github.com/leighmacdonald/tf-tui/tfapi"
	"github.com/leighmacdonald/tf-tui/ui"
)

const (
	maxQueueSize   = 100
	g15PlayerCount = 102

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

type DumpPlayer struct {
	Names     [g15PlayerCount]string
	Ping      [g15PlayerCount]int
	Score     [g15PlayerCount]int
	Deaths    [g15PlayerCount]int
	Connected [g15PlayerCount]bool
	Team      [g15PlayerCount]int
	Alive     [g15PlayerCount]bool
	Health    [g15PlayerCount]int
	SteamID   [g15PlayerCount]steamid.SteamID
	Valid     [g15PlayerCount]bool
	UserID    [g15PlayerCount]int
}

func NewPlayerDataModel(client *tfapi.ClientWithResponses, config config.Config, cache Cache) *PlayerDataModel {
	return &PlayerDataModel{
		mu:          &sync.RWMutex{},
		players:     make(map[steamid.SteamID]*Player),
		updateQueue: make(chan steamid.SteamID, maxQueueSize),
		client:      client,
		config:      config,
		g15re:       regexp.MustCompile(`^(m_szName|m_iPing|m_iScore|m_iDeaths|m_bConnected|m_iTeam|m_bAlive|m_iHealth|m_iAccountID|m_bValid|m_iUserID)\[(\d+)]\s(integer|bool|string)\s\((.+?)?\)$`),
		cache:       cache,
	}
}

type PlayerDataModel struct {
	client      *tfapi.ClientWithResponses
	lastUpdate  DumpPlayer
	g15re       *regexp.Regexp
	config      config.Config
	players     map[steamid.SteamID]*Player
	mu          *sync.RWMutex
	updateQueue chan steamid.SteamID
	cache       Cache
	lists       []tfapi.BDSchema
}

// MetaProfiles handles loading player MetaProfiles. It first attempts to load from a local filesystem cache
// and if any are missing or expired, they will be fetched from the api, and subsequently cached.
func (m *PlayerDataModel) MetaProfiles(ctx context.Context, steamIDs steamid.Collection) ([]tfapi.MetaProfile, error) {
	if len(steamIDs) == 0 {
		return nil, nil
	}

	profiles, missing, errCached := m.cachedMetaProfiles(steamIDs)
	if errCached != nil {
		return nil, errCached
	}

	if len(missing) == 0 {
		return profiles, nil
	}

	updates, errUpdates := m.fetchMetaProfiles(ctx, missing)
	if errUpdates != nil {
		return profiles, errUpdates
	}

	return append(profiles, updates...), nil
}

func (m *PlayerDataModel) cachedMetaProfiles(steamIDs steamid.Collection) ([]tfapi.MetaProfile, steamid.Collection, error) {
	var profiles []tfapi.MetaProfile //nolint:prealloc
	var missing steamid.Collection
	for _, steamID := range steamIDs {
		body, errGet := m.cache.Get(steamID, CacheMetaProfile)
		if errGet != nil {
			if !errors.Is(errGet, errCacheMiss) {
				return nil, nil, errors.Join(errGet, errFetchMetaProfile)
			}

			missing = append(missing, steamID)

			continue
		}

		cached, err := UnmarshalJSON[tfapi.MetaProfile](bytes.NewReader(body))
		if err != nil {
			missing = append(missing, steamID)

			continue
		}

		profiles = append(profiles, cached)
	}

	return profiles, missing, nil
}

func (m *PlayerDataModel) fetchMetaProfiles(ctx context.Context, steamIDs steamid.Collection) ([]tfapi.MetaProfile, error) {
	var profiles []tfapi.MetaProfile //nolint:prealloc
	resp, errResp := m.client.MetaProfile(ctx, &tfapi.MetaProfileParams{Steamids: strings.Join(steamIDs.ToStringSlice(), ",")})
	if errResp != nil {
		return nil, errors.Join(errResp, errFetchMetaProfile)
	}
	defer func(closer io.Closer) {
		if err := closer.Close(); err != nil {
			slog.Error("Failed to close response body", slog.String("error", err.Error()))
		}
	}(resp.Body)

	parsed, errParse := tfapi.ParseMetaProfileResponse(resp)
	if errParse != nil {
		return nil, errors.Join(errParse, errFetchMetaProfile)
	}

	for _, profile := range *parsed.JSON200 {
		var buf bytes.Buffer
		if errBody := json.NewEncoder(&buf).Encode(profile); errBody != nil {
			return nil, errors.Join(errBody, errFetchMetaProfile)
		}
		if errSet := m.cache.Set(steamid.New(profile.SteamId), CacheMetaProfile, buf.Bytes()); errSet != nil {
			return nil, errors.Join(errSet, errFetchMetaProfile)
		}

		profiles = append(profiles, profile)
	}

	return profiles, nil
}

func (m *PlayerDataModel) fetchPlayerState(ctx context.Context, address string, password string) (DumpPlayer, error) {
	conn := newRconConnection(address, password)
	response, errExec := conn.exec(ctx, "status;g15_dumpplayer", true)
	if errExec != nil {
		if m.lastUpdate.SteamID[0].Valid() {
			return m.lastUpdate, nil
		}
		if len(os.Getenv("DEBUG")) == 0 {
			return DumpPlayer{}, errExec
		}
		// FIXME remove this test data generation eventually
		var data DumpPlayer
		for playerIdx := range 24 {
			data.SteamID[playerIdx] = steamid.New(76561197960265730 + playerIdx)
			data.UserID[playerIdx] = playerIdx + 1
			data.Score[playerIdx] = playerIdx
			data.Ping[playerIdx] = playerIdx
			data.Deaths[playerIdx] = playerIdx
			if playerIdx%2 == 0 {
				data.Team[playerIdx] = tf.BLU
			} else {
				data.Team[playerIdx] = tf.RED
			}
			if playerIdx == 0 {
				data.SteamID[0] = steamid.New(76561197960265730)
			}
			if playerIdx == 5 {
				data.SteamID[5] = steamid.New(76561197970669109)
			}
			if playerIdx == 6 {
				data.SteamID[playerIdx] = steamid.New(76561198044497183)
			}
			if playerIdx == 1 {
				data.SteamID[playerIdx] = steamid.New(76561198084134025)
			}
		}

		return data, nil
	}

	dump := m.parsePlayerState(strings.NewReader(response))
	m.lastUpdate = dump

	return dump, nil
}

// parsePlayerState provides the ability to parse the output of the `g15_dumpplayer` command into a PlayerState struct.
// This functionality requires the `-g15` launch parameter for TF2 to be set.
func (m *PlayerDataModel) parsePlayerState(reader io.Reader) DumpPlayer {
	var (
		data    DumpPlayer
		scanner = bufio.NewScanner(reader)
	)

	for scanner.Scan() {
		matches := m.g15re.FindStringSubmatch(strings.Trim(scanner.Text(), "\r"))
		if len(matches) == 0 {
			continue
		}

		index := parseInt(matches[2], -1)
		if index < 0 {
			continue
		}

		value := ""
		if len(matches) == 5 {
			value = matches[4]
		}

		switch matches[1] {
		case "m_szName":
			data.Names[index] = value
		case "m_iPing":
			data.Ping[index] = parseInt(value, 0)
		case "m_iScore":
			data.Score[index] = parseInt(value, 0)
		case "m_iDeaths":
			data.Deaths[index] = parseInt(value, 0)
		case "m_bConnected":
			data.Connected[index] = parseBool(value)
		case "m_iTeam":
			data.Team[index] = parseInt(value, 0)
		case "m_bAlive":
			data.Alive[index] = parseBool(value)
		case "m_iHealth":
			data.Health[index] = parseInt(value, 0)
		case "m_iAccountID":
			data.SteamID[index] = steamid.New(parseInt(value, 0))
		case "m_bValid":
			data.Valid[index] = parseBool(value)
		case "m_iUserID":
			data.UserID[index] = parseInt(value, -1)
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error("Error scanning g15 response", slog.String("error", err.Error()))

		return data
	}

	return data
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

func (m *PlayerDataModel) setProfiles(profiles ...tfapi.MetaProfile) {
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

		player.Meta = profile
		player.MetaUpdatedOn = time.Now()

		m.players[sid] = player
	}
}

func (m *PlayerDataModel) SetStats(stats DumpPlayer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for idx := range g15PlayerCount {
		sid := stats.SteamID[idx]
		if !sid.Valid() {
			// TODO verify this is ok, however i think g15 is filled sequentially.
			continue
		}

		player, found := m.players[sid]
		if !found {
			player = &Player{SteamID: sid, Meta: tfapi.MetaProfile{Bans: []tfapi.Ban{}}}
			m.players[sid] = player
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
		player.Team = tf.Team(stats.Team[idx])
		player.UserID = stats.UserID[idx]
		player.G15UpdatedOn = time.Now()

		if !found || time.Since(player.MetaUpdatedOn) > time.Hour*24 {
			// Queue for a meta profile update
			select {
			case m.updateQueue <- sid:
			default:
			}
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

	var profiles Players
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
