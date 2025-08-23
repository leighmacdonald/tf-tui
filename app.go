package main

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/config"
	"github.com/leighmacdonald/tf-tui/tf"
	"github.com/leighmacdonald/tf-tui/tfapi"
	"github.com/leighmacdonald/tf-tui/ui"
)

type App struct {
	ui           *tea.Program
	config       config.Config
	console      *tf.ConsoleLog
	fetcher      *MetaFetcher
	metaInFlight atomic.Bool
	playerStates []Player
	playerMutex  *sync.RWMutex
}

func New(config config.Config, fetcher *MetaFetcher) *App {
	return &App{
		console:     tf.NewConsoleLog(),
		config:      config,
		fetcher:     fetcher,
		playerMutex: &sync.RWMutex{},
	}
}

func (app *App) Start(ctx context.Context) {
	dumpTicker := time.NewTicker(time.Second * 2)
	configUpdates := make(chan config.Config)
	go config.Notify(ctx, config.DefaultConfigName, configUpdates)

	if app.config.ConsoleLogPath != "" {
		if err := app.console.Open(app.config.ConsoleLogPath); err != nil {
			slog.Warn("Failed to open console file", slog.String("error", err.Error()),
				slog.String("path", app.config.ConsoleLogPath))
		}
	}

	for {
		select {
		case <-dumpTicker.C:
			if app.metaInFlight.Load() {
				continue
			}
			app.updateMetaProfile(ctx)
		case conf := <-configUpdates:
			app.ui.Send(conf)
		case <-ctx.Done():
			return
		}
	}
}

func (app *App) updateMetaProfile(ctx context.Context) {
	app.metaInFlight.Store(true)
	defer app.metaInFlight.Store(false)

	var expires steamid.Collection
	for _, player := range app.playerStates {
		if time.Since(player.MetaUpdatedOn) > time.Hour*24 {
			expires = append(expires, player.SteamID)
		}
	}

	app.fetcher.MetaProfiles(ctx, expires)
}

func (app *App) createUI(ctx context.Context) *tea.Program {
	program := tea.NewProgram(ui.New(app.config, false, BuildVersion, BuildDate, BuildCommit),
		tea.WithMouseCellMotion(), tea.WithAltScreen(), tea.WithContext(ctx))
	app.ui = program

	return program
}

func (app *App) setProfiles(profiles ...tfapi.MetaProfile) {
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

func (app *App) SetStats(stats tf.DumpPlayer) {
	app.playerMutex.Lock()
	defer app.playerMutex.Unlock()

	for idx := range tf.MaxPlayerCount {
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
