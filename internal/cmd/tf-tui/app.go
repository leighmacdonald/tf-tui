package main

import (
	"context"
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/state"
	"github.com/leighmacdonald/tf-tui/internal/store"
	"github.com/leighmacdonald/tf-tui/internal/tf/events"
	"github.com/leighmacdonald/tf-tui/internal/ui"
)

type UI interface {
	Send(msg tea.Msg)
	Run() error
}

// App is the main application container. Very little logic is contained within this struct. Its mostly
// responsible for routing messages between different systems.
type App struct {
	ui            UI
	config        config.Config
	state         *state.Manager
	uiUpdates     chan any
	configUpdates chan config.Config
	router        *events.Router
	database      store.DBTX
}

// NewApp returns a new application instance. To actually start the app you must call
// Start().
func NewApp(conf config.Config, states *state.Manager, database store.DBTX, router *events.Router,
	configUpdates chan config.Config,
) *App {
	app := &App{
		config:        conf,
		state:         states,
		configUpdates: configUpdates,
		uiUpdates:     make(chan any),
		router:        router,
		database:      database,
	}

	return app
}

// Start brings up all the background goroutines and starts the main event processing loop.
func (app *App) Start(ctx context.Context, done <-chan any) {
	// Start collecting state updates.
	go func() {
		if err := app.state.Start(ctx, app.router); err != nil {
			slog.Error("Failed to start state collector", slog.String("error", err.Error()))
		}
	}()

	// Start sending game state updates to the UI.
	go app.stateSyncer(ctx)

	// Start routing log events to the UI.
	go app.logEventUpdater(ctx)

	// Start sending UI updates to the UI.
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
	eventChan := make(chan events.Event, 10)
	app.router.ListenFor(-1, eventChan, events.Any)
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
			app.updateUIState()
		case <-ctx.Done():
			return
		}
	}
}

func (app *App) updateUIState() {
	if app.ui == nil {
		return
	}

	snapshots := app.state.Snapshots()
	uiSnaps := make([]ui.Snapshot, len(snapshots))
	for idx, snap := range snapshots {
		uiSnapsnot := ui.Snapshot{LogSecret: snap.LogSecret, Stats: snap.Stats}
		for _, player := range snap.Players {
			uiSnapsnot.Server.Players = append(uiSnapsnot.Server.Players, ui.Player{
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
		uiSnaps[idx] = uiSnapsnot
	}

	app.ui.Send(uiSnaps)
	if len(uiSnaps) > 0 {
		app.ui.Send(uiSnaps[0])
	}
}

func (app *App) createUI(ctx context.Context, loader ui.ConfigWriter) UI {
	if app.ui == nil {
		app.ui = ui.New(ctx, app.config, false,
			BuildVersion, BuildDate, BuildCommit,
			loader, config.PathCache(config.CacheDirName))
	}

	return app.ui
}
