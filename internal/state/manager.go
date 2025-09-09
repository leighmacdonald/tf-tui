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
	"github.com/leighmacdonald/tf-tui/internal/meta"
	"github.com/leighmacdonald/tf-tui/internal/store"
	"github.com/leighmacdonald/tf-tui/internal/tf/console"
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
	logSource  console.Source
}

func NewManager(router *events.Router, conf config.Config, metaFetcher *meta.Fetcher,
	bdFetcher *bd.Fetcher, dbConn store.DBTX,
) (*Manager, error) {
	isLocalServer := func(address string) bool {
		// Fragile, probably better as a flag.
		return strings.HasPrefix(address, "127.0.0.1") || strings.HasPrefix(address, "localhost")
	}

	var source console.Source
	if !conf.ServerModeEnabled {
		source = console.NewLocal(conf.ConsoleLogPath)
	} else {
		logSource, errListener := console.NewRemote(console.RemoteOpts{
			ListenAddress: conf.ServerBindAddress,
		})
		if errListener != nil {
			return nil, errListener
		}
		source = logSource
	}

	var servers []*serverState
	if conf.ServerModeEnabled {
		for _, server := range conf.Servers {
			if isLocalServer(server.Address) {
				continue
			}

			servers = append(servers, newServerState(conf, server, router, bdFetcher, dbConn))
		}
	} else {
		for _, server := range conf.Servers {
			if isLocalServer(server.Address) || server.LogSecret == 0 {
				servers = append(servers, newServerState(conf, conf.Servers[0], router, bdFetcher, dbConn))

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
		logSource:    source,
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
	logSource      console.Source
	metaFetcher    *meta.Fetcher
	metaQueue      chan serverMetaUpdate
	metaInFlight   atomic.Bool
	config         config.Config
}

func (s *Manager) Snapshots() []Snapshot {
	snapshots := make([]Snapshot, len(s.serverStates))
	for idx, server := range s.serverStates {
		snapshots[idx] = server.Snapshot()
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
		go func(srv *serverState) {
			if err := srv.start(ctx); err != nil {
				slog.Error("failed to start server state updater", slog.String("error", err.Error()))
			}
		}(server)
	}

	if errOpen := s.logSource.Open(); errOpen != nil {
		slog.Error("Failed to open log source", slog.String("error", errOpen.Error()))
	}
}

func (s *Manager) metaUpdater(ctx context.Context) {
	var queue []serverMetaUpdate

	go func() {
		updateTicker := time.NewTicker(time.Second)
		for {
			select {
			case <-updateTicker.C:
				for _, update := range queue {
					// TODO
					slog.Debug(update.steamID.String())
				}
				queue = nil
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
func (s *Manager) updateMetaProfile(_ context.Context) {
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
