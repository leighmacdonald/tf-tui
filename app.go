package main

import (
	"context"
	"io"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
	"github.com/leighmacdonald/tf-tui/config"
	"github.com/leighmacdonald/tf-tui/tf"
	"github.com/leighmacdonald/tf-tui/tfapi"
	"github.com/leighmacdonald/tf-tui/ui"
)

type App struct {
	ui              *tea.Program
	config          config.Config
	client          *tfapi.ClientWithResponses
	console         *tf.ConsoleLog
	playerDataModel *PlayerDataModel
}

func NewApp(config config.Config, cache Cache, client *tfapi.ClientWithResponses) *App {
	return &App{
		client:          client,
		playerDataModel: NewPlayerDataModel(client, config, cache),
		console:         tf.NewConsoleLog(),
		config:          config,
	}
}

func (app *App) Start(ctx context.Context) error {
	go app.configWatcher(ctx, config.DefaultConfigName)

	if app.config.ConsoleLogPath != "" {
		if err := app.console.Read(app.config.ConsoleLogPath); err != nil {
			slog.Error("Failed to read console file", slog.String("error", err.Error()),
				slog.String("path", app.config.ConsoleLogPath))
		}
	}

	<-ctx.Done()

	return nil
}

func (app *App) createUI(ctx context.Context) (*tea.Program, error) {
	if app.ui != nil {
		return nil, nil
	}

	program := tea.NewProgram(ui.New(app.config, false, BuildVersion, BuildDate, BuildCommit),
		tea.WithMouseCellMotion(), tea.WithAltScreen(), tea.WithContext(ctx))
	app.ui = program

	return program, nil
}

// configWatcher is responsible for monitoring the config file for external changes and
// subsequently sending the new Config to the *tea.Program to broadcast the changed Config.
func (app *App) configWatcher(ctx context.Context, name string) {
	watcher, errWatcher := fsnotify.NewWatcher()
	if errWatcher != nil {
		return
	}
	defer func(closer io.Closer) {
		if err := closer.Close(); err != nil {
			slog.Error("watcher close error", slog.String("err", err.Error()))
		}
	}(watcher)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-watcher.Events:
				if event.Op != fsnotify.Rename && event.Op != fsnotify.Write {
					continue
				}

				conf, errRead := config.Read(name)
				if errRead != nil {
					slog.Error("Failed to read config", slog.String("error", errRead.Error()))
					continue
				}
				if app.ui != nil {
					app.ui.Send(conf)
				}
			}
		}
	}()

	configPath := config.PathConfig(name)
	if err := watcher.Add(configPath); err != nil {
		slog.Error("Error adding watch for config", slog.String("error", err.Error()))
	}

	<-ctx.Done()
}
