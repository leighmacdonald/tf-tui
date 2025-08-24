package main

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/config"
	"github.com/leighmacdonald/tf-tui/tf"
	"github.com/leighmacdonald/tf-tui/ui"
)

type App struct {
	ui            *ui.UI
	config        config.Config
	console       *tf.ConsoleLog
	metaFetcher   *MetaFetcher
	bdFetcher     *BDFetcher
	players       *PlayerStates
	metaInFlight  atomic.Bool
	configUpdates chan config.Config
}

func New(conf config.Config, metaFetcher *MetaFetcher, bdFetcheer *BDFetcher) *App {
	return &App{
		console:       tf.NewConsoleLog(),
		config:        conf,
		metaFetcher:   metaFetcher,
		bdFetcher:     bdFetcheer,
		players:       NewPlayerStates(),
		configUpdates: make(chan config.Config),
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

func (app *App) Start(ctx context.Context, done <-chan any) {
	dumpTicker := time.NewTicker(time.Duration(app.config.UpdateFreqMs) * time.Millisecond)

	go config.Notify(ctx, config.DefaultConfigName, app.configUpdates)
	go app.players.cleaner(ctx)
	go app.stateSyncer(ctx)
	go app.bdFetcher.Update(ctx)

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
			app.updateBD()
		case conf := <-app.configUpdates:
			app.ui.Send(conf)
		case <-ctx.Done():
			return
		case <-done:
			return
		}
	}
}

func (app *App) updateBD() {
	var updates Players
	for _, player := range app.players.Players() {
		player.BDMatches = app.bdFetcher.Search(player.SteamID)
		updates = append(updates, player)
	}

	app.players.SetPlayer(updates...)
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
