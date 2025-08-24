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

	"github.com/leighmacdonald/tf-tui/config"
	"github.com/leighmacdonald/tf-tui/store"
	"github.com/leighmacdonald/tf-tui/tfapi"
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

// Run is the main entry point of tf-tui.
func Run() error {
	ctx := context.Background()

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

	userConfig, errConfig := config.Read(config.DefaultConfigName)
	if errConfig != nil {
		slog.Error("Failed to load config", slog.String("error", errConfig.Error()))
	}

	logFile, errLogger := config.LoggerInit(config.DefaultLogName, slog.LevelDebug)
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

	httpClient := &http.Client{Timeout: config.DefaultHTTPTimeout}
	client, errClient := tfapi.NewClientWithResponses(userConfig.APIBaseURL, tfapi.WithHTTPClient(httpClient))
	if errClient != nil {
		return errors.Join(errClient, errApp)
	}

	database, errDB := store.Connect(ctx, config.PathConfig(config.DefaultDBName))
	if errDB != nil {
		return errors.Join(errDB, errApp)
	}

	defer func() {
		if err := database.Close(); err != nil {
			slog.Error("Error closing database", slog.String("error", err.Error()))
		}
	}()

	cache, errCache := NewFilesystemCache()
	if errCache != nil {
		return errors.Join(errCache, errApp)
	}

	app := New(userConfig,
		NewMetaFetcher(client, cache),
		NewBDFetcher(httpClient, userConfig.BDLists, cache))

	done := make(chan any)

	go func() {
		if err := app.createUI(ctx).Run(); err != nil {
			slog.Error("Failed to run UI", slog.String("error", err.Error()))
		}

		done <- "let me out"
	}()

	app.Start(ctx, done)

	return nil
}
