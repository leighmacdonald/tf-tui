package main

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/leighmacdonald/tf-tui/internal"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/store"
	"github.com/leighmacdonald/tf-tui/internal/tf/console"
	"github.com/leighmacdonald/tf-tui/internal/tf/events"
	"github.com/leighmacdonald/tf-tui/internal/ui"
)

// App is the main application container.
type App struct {
	ui            *ui.UI
	config        config.Config
	playerStates  *internal.PlayerStates
	blackBox      *internal.BlackBox
	uiUpdates     chan any
	configUpdates chan config.Config
	broadcaster   *events.Router
	logSource     console.Source
}

// NewApp returns a new application instance. To actually start the app you must call
// Start().
func NewApp(conf config.Config, states *internal.PlayerStates, database *sql.DB, broadcaster *events.Router, logSource console.Source,
) *App {
	app := &App{
		config:        conf,
		playerStates:  states,
		configUpdates: make(chan config.Config),
		uiUpdates:     make(chan any),
		blackBox:      internal.NewBlackBox(store.New(database)),
		broadcaster:   broadcaster,
		logSource:     logSource,
	}

	return app
}

// Start brings up all the background goroutines and starts the main event processing loop.
func (app *App) Start(ctx context.Context, done <-chan any) {
	// Watch the config for writes
	go config.Notify(ctx, config.DefaultConfigName, app.configUpdates)

	// Handle removing expired players from the active player states
	go app.playerStates.Start(ctx)

	// Start sending player state updates to the UI.
	go app.stateSyncer(ctx)

	// Open the console log and start processing events.
	if err := app.logSource.Open(ctx); err != nil {
		slog.Warn("Failed to open console file", slog.String("error", err.Error()),
			slog.String("path", app.config.ConsoleLogPath))
	}

	go app.logEventUpdater(ctx)
	go app.uiSender(ctx)

	for {
		select {
		case conf := <-app.configUpdates:
			app.uiUpdates <- conf
		case <-ctx.Done():
			return
		case <-done:
			return
		}
	}
}

// logEventUpdater sends console log events to the UI for display.
func (app *App) logEventUpdater(ctx context.Context) {
	eventChan := make(chan events.Event)
	app.broadcaster.ListenFor(events.Any, eventChan)
	for {
		select {
		case evt := <-eventChan:
			app.uiUpdates <- evt
		case <-ctx.Done():
			return
		}
	}
}

// uiSender handles forwarding all events to the UI.
func (app *App) uiSender(ctx context.Context) {
	for {
		select {
		case msg := <-app.uiUpdates:
			if app.ui != nil {
				app.ui.Send(msg)
			}
		case <-ctx.Done():
			return
		}
	}
}

// stateSyncer periodically sends the updated player states to the ui. The update rate of this
// can be controlled by the `update_freq_ms` config parameter, defaulting to 2000ms.
func (app *App) stateSyncer(ctx context.Context) {
	ticker := time.NewTicker(time.Millisecond * time.Duration(app.config.UpdateFreqMs))

	for {
		select {
		case <-ticker.C:
			app.sendPlayerStates()
		case <-ctx.Done():
			return
		}
	}
}

func (app *App) sendPlayerStates() {
	if app.ui == nil {
		return
	}

	var players ui.Players
	for _, player := range app.playerStates.Players() {
		players = append(players, ui.Player{
			SteamID:                  player.SteamID,
			Name:                     player.Name,
			Ping:                     player.Ping,
			Score:                    player.Score,
			Deaths:                   player.Deaths,
			Connected:                player.Connected,
			Team:                     player.Team,
			Alive:                    player.Alive,
			Valid:                    player.Valid,
			UserID:                   player.UserID,
			Health:                   player.Health,
			Address:                  player.Address,
			Time:                     player.Time,
			Loss:                     player.Loss,
			Bans:                     player.Meta.Bans,
			Friends:                  player.Meta.Friends,
			CommunityBanned:          player.Meta.CommunityBanned,
			CommunityVisibilityState: player.Meta.CommunityVisibilityState,
			CompetitiveTeams:         player.Meta.CompetitiveTeams,
			DaysSinceLastBan:         player.Meta.DaysSinceLastBan,
			EconomyBan:               player.Meta.EconomyBan,
			LogsCount:                player.Meta.LogsCount,
			NumberOfGameBans:         player.Meta.NumberOfGameBans,
			NumberOfVacBans:          player.Meta.NumberOfVacBans,
			PersonaName:              player.Meta.PersonaName,
			ProfileState:             player.Meta.ProfileState,
			RealName:                 player.Meta.RealName,
			TimeCreated:              player.Meta.TimeCreated,
		})
	}

	app.ui.Send(players)
}

func (app *App) createUI(ctx context.Context) *ui.UI {
	if app.ui == nil {
		app.ui = ui.New(ctx, app.config, false, BuildVersion, BuildDate, BuildCommit)
	}

	return app.ui
}
