package state

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/bd"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/meta"
	"github.com/leighmacdonald/tf-tui/internal/store"
	"github.com/leighmacdonald/tf-tui/internal/tf/events"
	"github.com/leighmacdonald/tf-tui/internal/tfapi"
)

const (
	playerTimeout  = time.Second * 30
	checkInterval  = time.Second * 2
	removeInterval = time.Second
)

var errNoServersFound = errors.New("no servers configured")

type serverMetaUpdate struct {
	logAddress int
	steamID    steamid.SteamID
	profile    tfapi.MetaProfile
}

func NewManager(router *events.Router, conf config.Config, metaFetcher *meta.MetaFetcher,
	bdFetcher *bd.BDFetcher, dbConn store.DBTX) (*Manager, error) {
	isLocalServer := func(address string) bool {
		// Fragile, probably better as a flag.
		return strings.HasPrefix(address, "127.0.0.1") || strings.HasPrefix(address, "localhost")
	}

	var servers []*serverState
	if conf.ServerModeEnabled {
		for _, server := range conf.Servers {
			if isLocalServer(server.Address) {
				continue
			}
			state, errState := newServerState(conf, server, router, bdFetcher, dbConn, false)
			if errState != nil {
				return nil, errState
			}

			servers = append(servers, state)
		}
	} else {
		for _, server := range conf.Servers {
			if isLocalServer(server.Address) {
				state, errState := newServerState(conf, conf.Servers[0], router, bdFetcher, dbConn, true)
				if errState != nil {
					return nil, errState
				}

				servers = []*serverState{state}

				break
			}
		}
	}

	if len(servers) == 0 {
		return nil, errNoServersFound
	}

	return &Manager{
		serverStates: servers,
		metaFetcher:  metaFetcher,
		config:       conf,
	}, nil
}

// Manager is a struct that tracks the state of servers and handles incoming events, routing them
// to the appropriate serverState.
//
// This is also responsible for fetching metaProfile updates. These are quite expensive calls, so they are
// queued and processed synchronously.
type Manager struct {
	// serverStates contains the current state of each server. When in server mode, this will contain the state of all servers.
	// When in client mode, this will contain the state of the single local server.
	serverStates   []*serverState
	incomingEvents chan events.Event
	metaFetcher    *meta.MetaFetcher
	metaQueue      chan serverMetaUpdate
	metaInFlight   atomic.Bool
	config         config.Config
}

func (s *Manager) Snapshots() []Snapshot {
	var snapshots []Snapshot
	for _, server := range s.serverStates {
		snapshots = append(snapshots, server.Snapshot())
	}

	return snapshots
}

func (s *Manager) Close(ctx context.Context) {
	localTimeout, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(len(s.serverStates))
	for _, logSource := range s.serverStates {
		go func(source *serverState) {
			defer waitGroup.Done()
			source.close(localTimeout)
		}(logSource)
	}
	waitGroup.Wait()
}

func (s *Manager) Start(ctx context.Context) {
	for _, server := range s.serverStates {
		go server.start(ctx)
	}
}

func (s *Manager) metaUpdater(ctx context.Context) {
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

func (s *Manager) onDumpTick(ctx context.Context) {
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

// metaplayer updates are handled at a higher level just to make it a bit simpler to schedule the requests
// as the api is quite expensive to call.
func (s *Manager) updateMetaProfile(ctx context.Context) {
	s.metaInFlight.Store(true)
	defer s.metaInFlight.Store(false)

	// var expires steamid.Collection
	// for _, player := range s.Players() {
	// 	if time.Since(player.MetaUpdatedOn) > time.Hour*24 {
	// 		expires = append(expires, player.SteamID)
	// 	}
	// }

	// if len(expires) == 0 {
	// 	return
	// }

	// mProfiles, errProfiles := s.metaFetcher.MetaProfiles(ctx, expires)
	// if errProfiles != nil {
	// 	slog.Error("Failed to fetch meta profiles", slog.String("error", errProfiles.Error()))

	// 	return
	// }

	// for _, profile := range mProfiles {
	// 	s.UpdateMetaProfile(profile)
	// }
}
