package internal

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/tf/events"
	"github.com/leighmacdonald/tf-tui/internal/tfapi"
)

const (
	playerTimeout  = time.Second * 30
	checkInterval  = time.Second * 2
	removeInterval = time.Second
)

var (
	errNoServersFound = errors.New("no servers configured")
)

type serverMetaUpdate struct {
	serverAddress string
	steamID       steamid.SteamID
	profile       tfapi.MetaProfile
}

func NewStateTracker(router *events.Router, conf config.Config, metaFetcher *MetaFetcher, bdFetcher *BDFetcher) (*StateTracker, error) {
	incomingEvents := make(chan events.Event)
	router.ListenFor(events.StatusID, incomingEvents)

	isLocal := func(address string) bool {
		return strings.HasPrefix(address, "127.0.0.1") || strings.HasPrefix(address, "localhost")
	}

	var servers []*serverState
	if conf.ServerModeEnabled {
		servers = make([]*serverState, len(conf.Servers))
		for i, server := range conf.Servers {
			if isLocal(server.Address) {
				continue
			}
			servers[i] = newServerState(server)
		}
	} else {
		for _, server := range conf.Servers {
			if isLocal(server.Address) {
				servers = []*serverState{newServerState(conf.Servers[0])}

				break
			}
		}
	}

	if len(servers) == 0 {
		return nil, errNoServersFound
	}

	return &StateTracker{
		mu:             &sync.RWMutex{},
		serverStates:   []*serverState{},
		metaFetcher:    metaFetcher,
		bdFetcher:      bdFetcher,
		dumpFetcher:    tf.NewDumpFetcher(conf.Address, conf.Password, conf.ServerModeEnabled),
		incomingEvents: incomingEvents,
	}, nil
}

// StateTracker is a struct that tracks the state of servers and handles incoming events, routing them
// to the appropriate serverState.
//
// This is also responsible for fetching metaProfile updates. These are quite expensive calls, so they are
// queued and processed synchronously.
type StateTracker struct {
	mu *sync.RWMutex
	// serverStates contains the current state of each server. When in server mode, this will contain the state of all servers.
	// When in client mode, this will contain the state of the single local server.
	serverStates   []*serverState
	incomingEvents chan events.Event
	metaFetcher    *MetaFetcher
	dumpFetcher    tf.DumpFetcher
	bdFetcher      *BDFetcher
	stats          events.StatsEvent
	metaQueue      chan serverMetaUpdate
	metaInFlight   atomic.Bool
}

func (s *StateTracker) Start(ctx context.Context) {
	// Load the bot detector lists
	go s.bdFetcher.Update(ctx)

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
			for _, server := range s.serverStates {
				server.removeExpired()
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *StateTracker) metaUpdater(ctx context.Context) {
	var queue []serverMetaUpdate

	go func() {
		updateTicker := time.NewTicker(time.Second)
		for {
			select {
			case <-updateTicker.C:

			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case update := <-s.metaQueue:
			queue = append(queue, update)
		case <-ctx.Done():
			return
		}
	}
}

func (s *StateTracker) onDumpTick(ctx context.Context) {
	waitGroup := &sync.WaitGroup{}
	for _, server := range s.serverStates {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			server.onDumpTick(ctx)
		}()
	}

	waitGroup.Wait()
}

func (s *StateTracker) onIncomingEvent(event events.Event) error {
	switch data := event.Data.(type) { //nolint:gocritic
	case events.StatusIDEvent:
		player, errPlayer := s.Player(data.PlayerSID)
		if errPlayer != nil {
			if !errors.Is(errPlayer, errPlayerNotFound) {
				return errPlayer
			}

			player = Player{SteamID: data.PlayerSID, Meta: tfapi.MetaProfile{Bans: []tfapi.Ban{}}}
		}

		player.Name = data.Player

		s.SetPlayer(player)
	}

	return nil
}

func (s *StateTracker) updateBD() {
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

func (s *StateTracker) Stats() events.StatsEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.stats
}

func (s *StateTracker) UpdateStats(stats events.StatsEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats = stats
}

func (s *StateTracker) updateMetaProfile(ctx context.Context) {
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
