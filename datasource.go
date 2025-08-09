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
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	// How long we wait until a player should be ejected from our tracking.
	// This should be long enough to last through map changes without dropping the
	// known players.
	playerExpiration = time.Second * 30
	maxQueueSize     = 100
	g15PlayerCount   = 102
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
	return time.Since(p.g15UpdatedOn) > playerExpiration
}

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

func NewPlayerDataModel(client *ClientWithResponses, config Config, cache Cache) *PlayerDataModel {
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
	client      *ClientWithResponses
	lastUpdate  DumpPlayer
	g15re       *regexp.Regexp
	config      Config
	players     map[steamid.SteamID]*Player
	mu          *sync.RWMutex
	updateQueue chan steamid.SteamID
	cache       Cache
}

func (m *PlayerDataModel) Init() tea.Cmd {
	go m.Start(context.Background())

	return tea.Batch(m.tickEvery())
}

func (m *PlayerDataModel) View() string {
	// Data only
	return ""
}

func (m *PlayerDataModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case Config:
		m.config = msg

		return m, nil
	case DumpPlayerMsg:
		return m.onPlayerStateMsg(msg)
	}

	return m, nil
}

func (m *PlayerDataModel) tickEvery() tea.Cmd {
	return tea.Tick(time.Second, func(lastTime time.Time) tea.Msg {
		dump, errDump := m.fetchPlayerState(context.Background(), m.config.Address, m.config.Password)
		if errDump != nil {
			return DumpPlayerMsg{err: errDump, t: lastTime}
		}

		return DumpPlayerMsg{t: lastTime, dump: dump}
	})
}

func (m *PlayerDataModel) onPlayerStateMsg(msg DumpPlayerMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	if msg.err != nil {
		cmds = append(cmds, func() tea.Msg {
			return StatusMsg{
				message: msg.err.Error(),
				error:   true,
			}
		})
	}

	m.SetStats(msg.dump)

	players, errPlayers := m.All()
	if errPlayers != nil {
		return m, tea.Batch(m.tickEvery())
	}

	cmds = append(cmds, m.tickEvery(), func() tea.Msg {
		return FullStateUpdateMsg{players: players}
	})

	return m, tea.Batch(cmds...)
}

func (m *PlayerDataModel) getMetaProfiles(ctx context.Context, steamIDs steamid.Collection) ([]MetaProfile, error) {
	if len(steamIDs) == 0 {
		return nil, nil
	}

	var missing steamid.Collection
	var profiles []MetaProfile // nolint:prealloc

	for _, steamID := range steamIDs {
		body, errGet := m.cache.Get(steamID, CacheMetaProfile)
		if errGet != nil {
			if !errors.Is(errGet, errCacheMiss) {
				return nil, errors.Join(errGet, errFetchMetaProfile)
			}

			missing = append(missing, steamID)

			continue
		}

		cached, err := UnmarshalJSON[MetaProfile](bytes.NewReader(body))
		if err != nil {
			missing = append(missing, steamID)

			continue
		}

		profiles = append(profiles, cached)
	}

	if len(missing) == 0 {
		return profiles, nil
	}

	resp, errResp := m.client.MetaProfile(ctx, &MetaProfileParams{Steamids: strings.Join(missing.ToStringSlice(), ",")})
	if errResp != nil {
		return nil, errors.Join(errResp, errFetchMetaProfile)
	}
	defer resp.Body.Close()

	parsed, errParse := ParseMetaProfileResponse(resp)
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
				data.Team[playerIdx] = BLU
			} else {
				data.Team[playerIdx] = RED
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

func (m *PlayerDataModel) setProfiles(profiles ...MetaProfile) {
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
			player = &Player{SteamID: sid, meta: MetaProfile{Bans: []Ban{}}}
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

func (m *PlayerDataModel) Get(steamID steamid.SteamID) (Player, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	player, found := m.players[steamID]
	if !found {
		return Player{}, fmt.Errorf("%w: %s", errPlayerNotFound, steamID.String())
	}

	return *player, nil
}

func (m *PlayerDataModel) All() ([]Player, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var profiles []Player //nolint:prealloc
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
	profiles, errProfiles := m.getMetaProfiles(ctx, steamIDs)
	if errProfiles != nil {
		slog.Error("Failed to update meta profiles", slog.String("error", errProfiles.Error()))

		return
	}

	m.setProfiles(profiles...)
}
