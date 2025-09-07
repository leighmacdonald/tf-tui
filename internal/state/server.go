package state

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/bd"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/store"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/tf/console"
	"github.com/leighmacdonald/tf-tui/internal/tf/events"
	"github.com/leighmacdonald/tf-tui/internal/tfapi"
	"github.com/leighmacdonald/tf-tui/internal/ui"
)

var (
	ErrPlayerNotFound = errors.New("player not found")
)

type Snapshot struct {
	LogSecret int
	Players   Players
	Stats     events.StatsEvent
}

func newServerState(conf config.Config, server config.Server, router *events.Router, bdFetcher *bd.BDFetcher, dbConn store.DBTX) (*serverState, error) {
	logSource, errListener := console.NewRemote(console.SRCDSListenerOpts{
		ExternalAddress: conf.ServerLogAddress,
		Secret:          conf.ServerLogSecret,
		ListenAddress:   conf.ServerListenAddress,
		RemoteAddress:   conf.Address,
		RemotePassword:  conf.Password,
	})
	if errListener != nil {
		return nil, errListener
	}

	allEvent := make(chan events.Event, 10)
	router.ListenFor(server.LogSecret, allEvent, events.Any)
	blackbox := newBlackBox(store.New(dbConn), router, allEvent)

	serverEvents := make(chan events.Event)
	router.ListenFor(server.LogSecret, serverEvents, events.StatusID)

	dumpFetcher := tf.NewDumpFetcher(conf.Address, conf.Password, conf.ServerModeEnabled)

	return &serverState{
		mu:             &sync.RWMutex{},
		server:         server,
		blackbox:       blackbox,
		incomingEvents: serverEvents,
		bdFetcher:      bdFetcher,
		logSource:      logSource,
		dumpFetcher:    dumpFetcher,
	}, nil
}

type serverState struct {
	mu             *sync.RWMutex
	players        Players
	server         config.Server
	blackbox       *blackBox
	incomingEvents chan events.Event
	bdFetcher      *bd.BDFetcher
	logSource      console.Source
	dumpFetcher    tf.DumpFetcher
	stats          events.StatsEvent
}

func (s *serverState) Start(ctx context.Context) {
	// Start recording events.
	go s.blackbox.Start(ctx)

	removeTicker := time.NewTicker(removeInterval)
	dumpTicker := time.NewTicker(checkInterval)

	for {
		select {
		case event := <-s.incomingEvents:
			if err := s.onIncomingEvent(event); err != nil {
				slog.Error("failed handling incoming log event", slog.String("error", err.Error()))
			}
		case <-dumpTicker.C:
			s.onDumpTick(ctx)
		case <-removeTicker.C:
			s.removeExpired()
		case <-ctx.Done():
			return
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

	s.SetPlayer(updates...)
}

func (s *serverState) Stats() events.StatsEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.stats
}

func (s *serverState) UpdateStats(stats events.StatsEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats = stats
}

func (s *serverState) SetPlayer(updates ...Player) {
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

func (s *serverState) onIncomingEvent(event events.Event) error {
	switch data := event.Data.(type) { //nolint:gocritic
	case events.StatusIDEvent:
		player, errPlayer := s.Player(data.PlayerSID)
		if errPlayer != nil {
			if !errors.Is(errPlayer, ErrPlayerNotFound) {
				return errPlayer
			}

			player = Player{SteamID: data.PlayerSID, Meta: tfapi.MetaProfile{Bans: []tfapi.Ban{}}}
		}

		player.Name = data.Player

		s.SetPlayer(player)
	}

	return nil
}

func (s *serverState) onDumpTick(ctx context.Context) {
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(3)

	// go func() {
	// 	defer waitGroup.Done()
	// 	s.updateMetaProfile(ctx)
	// }()

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

func (s *serverState) Player(steamID steamid.SteamID) (Player, error) {
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

func (s *serverState) PlayersUI() ui.Players {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var players ui.Players


	return players
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
		player, err := s.Player(steamid.New(meta.SteamId))
		if err != nil {
			return
		}

		player.Meta = meta
		players[index] = player
	}

	s.SetPlayer(players...)
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

		player, playerErr := s.Player(sid)
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

	s.SetPlayer(players...)
}
