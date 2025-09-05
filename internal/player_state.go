package internal

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/tf/events"
	"github.com/leighmacdonald/tf-tui/internal/tfapi"
)

type Player struct {
	SteamID       steamid.SteamID
	Name          string
	Ping          int
	Loss          int
	Address       string
	Time          int
	Score         int
	Deaths        int
	Connected     bool
	Team          tf.Team
	Alive         bool
	Health        int
	Valid         bool
	UserID        int
	BDMatches     []BDMatch
	Meta          tfapi.MetaProfile
	MetaUpdatedOn time.Time
	G15UpdatedOn  time.Time
}

type Players []Player

func NewPlayerStates(router *events.Router, conf config.Config, metaFetcher *MetaFetcher, bdFetcher *BDFetcher) *PlayerStates {
	incomingEvents := make(chan events.Event)
	router.ListenFor(events.StatusID, incomingEvents)

	return &PlayerStates{
		mu:             &sync.RWMutex{},
		players:        Players{},
		expiration:     time.Second * 30,
		checkInterval:  time.Second,
		metaFetcher:    metaFetcher,
		bdFetcher:      bdFetcher,
		dumpFetcher:    tf.NewDumpFetcher(conf.Address, conf.Password, conf.ServerModeEnabled),
		incomingEvents: incomingEvents,
	}
}

type PlayerStates struct {
	mu             *sync.RWMutex
	players        Players
	expiration     time.Duration
	checkInterval  time.Duration
	incomingEvents chan events.Event
	metaInFlight   atomic.Bool
	metaFetcher    *MetaFetcher
	dumpFetcher    tf.DumpFetcher
	bdFetcher      *BDFetcher
}

func (s *PlayerStates) Start(ctx context.Context) {
	// Load the bot detector lists
	go s.bdFetcher.Update(ctx)

	removeTicker := time.NewTicker(s.checkInterval)
	dumpTicker := time.NewTicker(time.Second * 2)

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

func (s *PlayerStates) onDumpTick(ctx context.Context) {
	if s.metaInFlight.Load() {
		return
	}
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(3)

	go func() {
		defer waitGroup.Done()
		s.updateMetaProfile(ctx)
	}()

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

func (s *PlayerStates) onIncomingEvent(event events.Event) error {
	switch event.Type { //nolint:exhaustive,gocritic
	case events.StatusID:
		player, errPlayer := s.Player(event.PlayerSID)
		if errPlayer != nil {
			if !errors.Is(errPlayer, errPlayerNotFound) {
				return errPlayer
			}

			player = Player{SteamID: event.PlayerSID, Meta: tfapi.MetaProfile{Bans: []tfapi.Ban{}}}
		}

		player.Name = event.Player

		s.SetPlayer(player)
	}

	return nil
}

func (s *PlayerStates) UpdateDumpPlayer(stats tf.DumpPlayer) {
	var players Players //nolint:prealloc
	for idx := range tf.MaxPlayerCount {
		sid := stats.SteamID[idx]
		if !sid.Valid() {
			// TODO verify this is ok, however i think g15 is filled sequentially.
			continue
		}

		player, playerErr := s.Player(sid)
		if playerErr != nil {
			if !errors.Is(playerErr, errPlayerNotFound) {
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
		player.Team = tf.Team(stats.Team[idx])
		player.UserID = stats.UserID[idx]
		player.G15UpdatedOn = time.Now()
		players = append(players, player)
	}

	s.SetPlayer(players...)
}

func (s *PlayerStates) SetPlayer(updates ...Player) {
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

func (s *PlayerStates) UpdateMetaProfile(metaProfiles ...tfapi.MetaProfile) {
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

func (s *PlayerStates) Player(steamID steamid.SteamID) (Player, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, player := range s.players {
		if steamID.Equal(player.SteamID) {
			return player, nil
		}
	}

	return Player{}, errPlayerNotFound
}

func (s *PlayerStates) Players() Players {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.players
}

func (s *PlayerStates) removeExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	var valid Players //nolint:prealloc
	for _, player := range s.players {
		if time.Since(player.G15UpdatedOn) > s.expiration {
			continue
		}

		valid = append(valid, player)
	}

	s.players = valid
}

func (s *PlayerStates) updateBD() {
	var (
		players = s.Players()
		updates = make(Players, len(players))
	)

	for idx, player := range players {
		player.BDMatches = s.bdFetcher.Search(player.SteamID)
		updates[idx] = player
	}

	s.SetPlayer(updates...)
}

func (s *PlayerStates) updateDump(ctx context.Context) {
	dump, errDump := s.dumpFetcher.Fetch(ctx)
	if errDump != nil {
		// s.uiUpdates <- ui.StatusMsg{
		// 	Err:     true,
		// 	Message: errDump.Error(),
		// }
		//
		// An error result will return a copy of the last successful dump still.
		slog.Error("Failed to fetch player dump", slog.String("error", errDump.Error()))
	}

	s.UpdateDumpPlayer(dump)
}

func (s *PlayerStates) updateMetaProfile(ctx context.Context) {
	s.metaInFlight.Store(true)
	defer s.metaInFlight.Store(false)

	var expires steamid.Collection
	for _, player := range s.Players() {
		if time.Since(player.MetaUpdatedOn) > time.Hour*24 {
			expires = append(expires, player.SteamID)
		}
	}

	if len(expires) == 0 {
		return
	}

	mProfiles, errProfiles := s.metaFetcher.MetaProfiles(ctx, expires)
	if errProfiles != nil {
		slog.Error("Failed to fetch meta profiles", slog.String("error", errProfiles.Error()))

		return
	}

	for _, profile := range mProfiles {
		s.UpdateMetaProfile(profile)
	}
}
