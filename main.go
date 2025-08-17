package main

//go:generate go tool oapi-codegen -config .openapi.yaml https://tf-api.roto.lol/api/openapi/schema-3.0.json
//go:generate go tool sqlc generate -f .sqlc.yaml

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leighmacdonald/tf-tui/store"
	zone "github.com/lrstanley/bubblezone"
	_ "modernc.org/sqlite"
)

var (
	BuildVersion = "master"
	BuildCommit  = "00000000"
	BuildDate    = time.Now().Format("2006-01-02T15:04:05Z")
)

var errApp = errors.New("application error")

func main() {
	if err := Run(); err != nil {
		slog.Error("Exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func Run() error {
	ctx := context.Background()
	zone.NewGlobal()

	if len(os.Getenv("PROFILE")) > 0 {
		f, err := os.Create(os.Getenv("PROFILE"))
		if err != nil {
			return errors.Join(err, errApp)
		}

		if errStart := pprof.StartCPUProfile(f); errStart != nil {
			return errors.Join(errStart, errApp)
		}
		defer pprof.StopCPUProfile()
	}

	config, configFound := ConfigRead(defaultConfigName)
	logFile, errLogger := LoggerInit(defaultLogName, slog.LevelDebug)
	if errLogger != nil {
		return errors.Join(errLogger, errApp)
	}
	defer func(closer io.Closer) {
		if err := closer.Close(); err != nil {
			slog.Error("Failed to close log file", slog.String("error", err.Error()))
		}
	}(logFile)

	slog.Info("Starting tf-tui", slog.String("version", BuildVersion),
		slog.String("commit", BuildCommit), slog.String("date", BuildDate),
		slog.String("go", runtime.Version()))

	client, errClient := NewClientWithResponses(config.APIBaseURL, WithHTTPClient(&http.Client{
		Timeout: defaultHTTPTimeout,
	}))
	if errClient != nil {
		return errors.Join(errClient, errApp)
	}

	db, errDB := store.Connect(ctx, ConfigPath(defaultDBName))
	if errDB != nil {
		return errors.Join(errDB, errApp)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("Error closing database", slog.String("error", err.Error()))
		}
	}()

	cache, errCache := NewFilesystemCache()
	if errCache != nil {
		return errors.Join(errCache, errApp)
	}

	program := tea.NewProgram(New(config, !configFound, client, cache),
		tea.WithMouseCellMotion(), tea.WithAltScreen())

	go ConfigWatcher(ctx, program, defaultConfigName)

	if _, err := program.Run(); err != nil {
		return errors.Join(err, errApp)
	}

	return nil
}
