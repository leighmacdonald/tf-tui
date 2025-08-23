package main

import (
	"context"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
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
	configUpdates := make(chan config.Config)
	go config.NotifyChanged(ctx, config.DefaultConfigName, configUpdates)

	if app.config.ConsoleLogPath != "" {
		if err := app.console.Read(app.config.ConsoleLogPath); err != nil {
			slog.Error("Failed to read console file", slog.String("error", err.Error()),
				slog.String("path", app.config.ConsoleLogPath))
		}
	}

	for {
		select {
		case conf := <-configUpdates:
			app.ui.Send(conf)
		case <-ctx.Done():
			return nil
		}
	}
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
