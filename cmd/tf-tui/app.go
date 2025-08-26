package main

import (
	"context"
	"database/sql"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/store"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/ui"
)

// App is the main application container.
type App struct {
	ui            *ui.UI
	config        config.Config
	console       *tf.ConsoleLog
	metaFetcher   *internal.MetaFetcher
	dumpFetcher   tf.DumpFetcher
	bdFetcher     *internal.BDFetcher
	players       *internal.PlayerStates
	metaInFlight  atomic.Bool
	blackBox      *internal.BlackBox
	uiUpdates     chan any
	configUpdates chan config.Config
}

// New returns a new application instance. To actually start the app you must call
// Start().
func New(conf config.Config, metaFetcher *internal.MetaFetcher, bdFetcher *internal.BDFetcher, db *sql.DB) *App {
	return &App{
		config:        conf,
		metaFetcher:   metaFetcher,
		bdFetcher:     bdFetcher,
		console:       tf.NewConsoleLog(),
		players:       internal.NewPlayerStates(),
		dumpFetcher:   tf.NewDumpFetcher(conf.Address, conf.Password),
		configUpdates: make(chan config.Config),
		uiUpdates:     make(chan any),
		blackBox:      internal.NewBlackBox(store.New(db)),
	}
}

// Start brings up all the background goroutines and starts the main event processing loop.
func (app *App) Start(ctx context.Context, done <-chan any) {
	// Watch the config for writes
	go config.Notify(ctx, config.DefaultConfigName, app.configUpdates)

	// Handle removing expired players from the active player states
	go app.players.Cleaner(ctx)

	// Start sending player state updates to the UI.
	go app.stateSyncer(ctx)

	// Load the bot detector lists
	go app.bdFetcher.Update(ctx)

	// Open the console log and start processing events.
	if err := app.console.Open(app.config.ConsoleLogPath); err != nil {
		slog.Warn("Failed to open console file", slog.String("error", err.Error()),
			slog.String("path", app.config.ConsoleLogPath))
	}

	go app.logEventUpdater(ctx)
	go app.uiSender(ctx)

	dumpTicker := time.NewTicker(time.Duration(app.config.UpdateFreqMs) * time.Millisecond)
	for {
		select {
		case <-dumpTicker.C:
			if app.metaInFlight.Load() {
				continue
			}
			app.updateMetaProfile(ctx)
			app.updateBD()
			app.updateDump(ctx)
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
	eventChan := make(chan tf.LogEvent)
	app.console.RegisterHandler(tf.EvtAny, eventChan)
	app.console.RegisterHandler(tf.EvtAny, eventChan)
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
	for _, player := range app.players.Players() {
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

func (app *App) updateBD() {
	var updates internal.Players
	for _, player := range app.players.Players() {
		player.BDMatches = app.bdFetcher.Search(player.SteamID)
		updates = append(updates, player)
	}

	app.players.SetPlayer(updates...)
}

func (app *App) updateDump(ctx context.Context) {
	dump, errDump := app.dumpFetcher.Fetch(ctx)
	if errDump != nil {
		app.uiUpdates <- ui.StatusMsg{
			Err:     true,
			Message: errDump.Error(),
		}
	}

	app.players.UpdateDumpPlayer(dump)
}

func (app *App) updateMetaProfile(ctx context.Context) {
	app.metaInFlight.Store(true)
	defer app.metaInFlight.Store(false)

	var expires steamid.Collection
	for _, player := range app.players.Players() {
		if time.Since(player.MetaUpdatedOn) > time.Hour*24 {
			expires = append(expires, player.SteamID)
		}
	}

	if len(expires) == 0 {
		return
	}

	mProfiles, errProfiles := app.metaFetcher.MetaProfiles(ctx, expires)
	if errProfiles != nil {
		slog.Error("Failed to fetch meta profiles", slog.String("error", errProfiles.Error()))

		return
	}

	for _, profile := range mProfiles {
		app.players.UpdateMetaProfile(profile)
	}
}

func (app *App) createUI(ctx context.Context) *ui.UI {
	if app.ui == nil {
		app.ui = ui.New(ctx, app.config, false, BuildVersion, BuildDate, BuildCommit)
	}

	return app.ui
}
