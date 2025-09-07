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
	serverAddress string
	steamID       steamid.SteamID
	profile       tfapi.MetaProfile
	store         store.DBTX
}

func NewManager(router *events.Router, conf config.Config, metaFetcher *meta.MetaFetcher, bdFetcher *bd.BDFetcher, dbConn store.DBTX) (*Manager, error) {
	isLocalServer := func(address string) bool {
		return strings.HasPrefix(address, "127.0.0.1") || strings.HasPrefix(address, "localhost")
	}

	var servers []*serverState
	if conf.ServerModeEnabled {
		var servers []*serverState
		for _, server := range conf.Servers {
			if isLocalServer(server.Address) {
				continue
			}
			state, errState := newServerState(conf, server, router, bdFetcher, dbConn)
			if errState != nil {
				return nil, errState
			}
			servers = append(servers, state)
		}
	} else {
		for _, server := range conf.Servers {
			if isLocalServer(server.Address) {
				state, errState := newServerState(conf, conf.Servers[0], router, bdFetcher, dbConn)
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
		mu:           &sync.RWMutex{},
		serverStates: []*serverState{},
		metaFetcher:  metaFetcher,
		bdFetcher:    bdFetcher,
		config:       conf,
	}, nil
}

// Manager is a struct that tracks the state of servers and handles incoming events, routing them
// to the appropriate serverState.
//
// This is also responsible for fetching metaProfile updates. These are quite expensive calls, so they are
// queued and processed synchronously.
type Manager struct {
	mu *sync.RWMutex
	// serverStates contains the current state of each server. When in server mode, this will contain the state of all servers.
	// When in client mode, this will contain the state of the single local server.
	serverStates   []*serverState
	incomingEvents chan events.Event
	metaFetcher    *meta.MetaFetcher
	bdFetcher      *bd.BDFetcher
	metaQueue      chan serverMetaUpdate
	metaInFlight   atomic.Bool
	logSources     []console.Source
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
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(len(s.logSources))
	for _, logSource := range s.logSources {
		go func(source console.Source) {
			defer waitGroup.Done()
			source.Close(ctx)
		}(logSource)
	}
	waitGroup.Wait()
}

func (s *Manager) Start(ctx context.Context) error {
	logSources, errLogSource := s.openLogSources(ctx, s.config)
	if errLogSource != nil {
		return errLogSource
	}

	for _, logSource := range logSources {
		go logSource.Start(ctx, nil)
		panic("make this work")
	}

	return nil
}

func (s *Manager) openLogSources(ctx context.Context, userConfig config.Config) ([]console.Source, error) {
	var logSources []console.Source
	if userConfig.ServerModeEnabled {
		listener, errListener := console.NewRemote(console.SRCDSListenerOpts{
			ExternalAddress: userConfig.ServerLogAddress,
			Secret:          userConfig.ServerLogSecret,
			ListenAddress:   userConfig.ServerListenAddress,
			RemoteAddress:   userConfig.Address,
			RemotePassword:  userConfig.Password,
		})
		if errListener != nil {
			return nil, errListener
		}
		logSources = append(logSources, listener)
	} else {
		logSources = append(logSources, console.NewLocal(userConfig.ConsoleLogPath))
	}

	waitGroup := &sync.WaitGroup{}
	for _, logSource := range logSources {
		waitGroup.Add(1)
		go func(source console.Source) {
			defer waitGroup.Done()
			if errOpen := source.Open(ctx); errOpen != nil {
				slog.Error("Failed to open log source", slog.String("error", errOpen.Error()))
			}
		}(logSource)
	}
	waitGroup.Wait()

	return logSources, nil
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
