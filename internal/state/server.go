package state

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/bd"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/geoip"
	"github.com/leighmacdonald/tf-tui/internal/store"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/tf/events"
	"github.com/leighmacdonald/tf-tui/internal/tf/rcon"
	"github.com/leighmacdonald/tf-tui/internal/tfapi"
)

var (
	ErrPlayerNotFound = errors.New("player not found")
	ErrRegistration   = errors.New("logaddress registration error")
	ErrUnregistration = errors.New("logaddress unregistration error")
)

type Snapshot struct {
	LogSecret int
	Players   Players
	Stats     tf.Stats
}

func newServerState(conf config.Config, server config.Server, router *events.Router, bdFetcher *bd.Fetcher,
	dbConn store.DBTX,
) *serverState {
	allEvent := make(chan events.Event, 10)
	router.ListenFor(server.LogSecret, allEvent, events.Any)
	blackbox := newBlackBox(store.New(dbConn), allEvent)

	serverEvents := make(chan events.Event)
	router.ListenFor(server.LogSecret, serverEvents, events.Any)

	dumpFetcher := rcon.NewFetcher(server.Address, server.Password, conf.ServerModeEnabled)

	return &serverState{
		mu:              &sync.RWMutex{},
		server:          server,
		blackbox:        blackbox,
		incomingEvents:  serverEvents,
		bdFetcher:       bdFetcher,
		dumpFetcher:     dumpFetcher,
		externalAddress: conf.ServerLogAddress,
	}
}

// serverState is responsible for keeping track of the server state.
type serverState struct {
	mu              *sync.RWMutex
	players         Players
	server          config.Server
	externalAddress string
	blackbox        *blackBox
	incomingEvents  chan events.Event
	bdFetcher       *bd.Fetcher
	dumpFetcher     rcon.Fetcher
	stats           tf.Stats
	countryCode     string
	address         string
	hostName        string
	mapName         string
	tags            []string
	eventCount      atomic.Int64
}

func (s *serverState) close(ctx context.Context) error {
	return s.unregisterAddress(ctx)
}

func (s *serverState) unregisterAddress(ctx context.Context) error {
	// Be cool and remove ourselves from the log address list.
	conn := rcon.New(s.server.Address, s.server.Password)
	if _, errExec := conn.Exec(ctx, "logaddress_del "+s.externalAddress, false); errExec != nil {
		return errors.Join(errExec, ErrUnregistration)
	}

	slog.Debug("Successfully unregistered logaddress", slog.String("address", s.externalAddress))

	return nil
}

func (s *serverState) registerAddress(ctx context.Context) error {
	conn := rcon.New(s.server.Address, s.server.Password)
	_, errExec := conn.Exec(ctx, "logaddress_add "+s.externalAddress, false)
	if errExec != nil {
		return errors.Join(errExec, ErrRegistration)
	}

	resp, err := conn.Exec(ctx, "logaddress_list", false)
	if err != nil {
		return errors.Join(err, ErrRegistration)
	}

	slog.Debug("Successfully registered logaddress", slog.String("address", s.externalAddress))

	if !strings.Contains(resp, s.externalAddress) {
		return ErrRegistration
	}

	return nil
}

func (s *serverState) start(ctx context.Context) error {
	if err := s.registerAddress(ctx); err != nil {
		return err
	}

	record, errRecord := geoip.Lookup(s.externalAddress)
	if errRecord != nil {
		slog.Error("failed to lookup server country code", slog.String("error", errRecord.Error()))
	} else {
		s.countryCode = strings.ToLower(record.Country.ISOCode)
	}

	// Start recording events.
	go s.blackbox.Start(ctx)

	removeTicker := time.NewTicker(removeInterval)
	dumpTicker := time.NewTicker(checkInterval)

	for {
		select {
		case event := <-s.incomingEvents:
			s.onIncomingEvent(event)
		case <-dumpTicker.C:
			s.onDumpTick(ctx)
		case <-removeTicker.C:
			s.removeExpired()
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *serverState) updateBD() {
	var (
		snapshot = s.Snapshot()
		updates  = make(Players, len(snapshot.Players))
	)

	for idx, player := range snapshot.Players {
		player.BDMatches = s.bdFetcher.Search(player.SteamID)
		updates[idx] = player
	}

	s.setPlayer(updates...)
}

func (s *serverState) Stats() tf.Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.stats
}

func (s *serverState) UpdateStats(stats tf.Stats) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats = stats
}

func (s *serverState) setPlayer(updates ...Player) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var existing bool
	for _, player := range updates {
		for playerIdx := range s.players {
			if s.players[playerIdx].SteamID.Equal(player.SteamID) {
				s.players[playerIdx] = player
				existing = true

				continue
			}
		}
		if !existing {
			s.players = append(s.players, player)
		}
	}
}

func (s *serverState) onIncomingEvent(event events.Event) {
	switch data := event.Data.(type) {
	case events.AddressEvent:
		s.onAddress(data.Address)
	case events.ConnectEvent:
	case events.DisconnectEvent:
	case events.HostnameEvent:
		s.onHostName(data.Hostname)
	case events.KillEvent:
	case events.MapEvent:
		s.onMapName(data.MapName)
	case events.MsgEvent:
	case events.TagsEvent:
		s.onTags(data.Tags)
	case events.RawEvent:
		s.eventCount.Add(1)
	case events.StatusIDEvent:
		s.onStatusID(data)
	}
}

func (s *serverState) onStatusID(data events.StatusIDEvent) {
	player, errPlayer := s.player(data.PlayerSID)
	if errPlayer != nil {
		if !errors.Is(errPlayer, ErrPlayerNotFound) {
			return
		}

		player = Player{SteamID: data.PlayerSID, Meta: tfapi.MetaProfile{Bans: []tfapi.Ban{}}}
	}

	player.Name = data.Player

	s.setPlayer(player)
}

func (s *serverState) onAddress(address string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.address = address
}

func (s *serverState) onHostName(hostname string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.hostName = hostname
}

func (s *serverState) onMapName(mapName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.mapName = mapName
}

func (s *serverState) onTags(tags []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tags = tags
}

func (s *serverState) onDumpTick(ctx context.Context) {
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(2)

	go func() {
		defer waitGroup.Done()
		s.updateBD()
	}()

	go func() {
		defer waitGroup.Done()
		s.updateDump(ctx)
	}()

	waitGroup.Wait()
}

func (s *serverState) player(steamID steamid.SteamID) (Player, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, player := range s.players {
		if steamID.Equal(player.SteamID) {
			return player, nil
		}
	}

	return Player{}, ErrPlayerNotFound
}

func (s *serverState) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return Snapshot{LogSecret: s.server.LogSecret, Players: s.players, Stats: s.stats}
}

func (s *serverState) updateDump(ctx context.Context) {
	dump, stats, errDump := s.dumpFetcher.Fetch(ctx)
	if errDump != nil {
		// s.uiUpdates <- ui.StatusMsg{
		// 	Err:     true,
		// 	Message: errDump.Error(),
		// }
		//
		// An error result will return a copy of the last successful dump still.
		slog.Error("Failed to fetch player dump", slog.String("error", errDump.Error()))
	}

	s.UpdateStats(stats)
	s.UpdateDumpPlayer(dump)
}

func (s *serverState) UpdateMetaProfile(metaProfiles ...tfapi.MetaProfile) {
	players := make(Players, len(metaProfiles))
	for index, meta := range metaProfiles {
		player, err := s.player(steamid.New(meta.SteamId))
		if err != nil {
			return
		}

		player.Meta = meta
		players[index] = player
	}

	s.setPlayer(players...)
}

func (s *serverState) removeExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	var valid Players
	for _, player := range s.players {
		if time.Since(player.G15UpdatedOn) > playerTimeout {
			continue
		}

		valid = append(valid, player)
	}

	s.players = valid
}

func (s *serverState) UpdateDumpPlayer(stats tf.DumpPlayer) {
	var players Players
	for idx := range tf.MaxPlayerCount {
		sid := stats.SteamID[idx]
		if !sid.Valid() {
			// TODO verify this is ok, however i think g15 is filled sequentially.
			continue
		}

		player, playerErr := s.player(sid)
		if playerErr != nil {
			if !errors.Is(playerErr, ErrPlayerNotFound) {
				return
			}
			player = Player{SteamID: sid, Meta: tfapi.MetaProfile{Bans: []tfapi.Ban{}}}
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
		player.Loss = stats.Loss[idx]
		player.Address = stats.Address[idx]
		player.Time = stats.Time[idx]
		player.Team = stats.Team[idx]
		player.UserID = stats.UserID[idx]
		player.G15UpdatedOn = time.Now()
		players = append(players, player)
	}

	s.setPlayer(players...)
}
