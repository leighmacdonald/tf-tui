package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/fang"
	_ "github.com/joho/godotenv/autoload"
	"github.com/leighmacdonald/tf-tui/internal/bd"
	"github.com/leighmacdonald/tf-tui/internal/cache"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/meta"
	"github.com/leighmacdonald/tf-tui/internal/state"
	"github.com/leighmacdonald/tf-tui/internal/store"
	"github.com/leighmacdonald/tf-tui/internal/tf/console"
	"github.com/leighmacdonald/tf-tui/internal/tf/events"
	"github.com/leighmacdonald/tf-tui/internal/tfapi"
	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
)

var (
	BuildVersion   = "master"
	BuildCommit    = "00000000"
	BuildDate      = time.Now().Format("2006-01-02T15:04:05Z")
	BuildGoVersion = runtime.Version()
	cfgFile        string
	rootCmd        = &cobra.Command{
		Use:   "tf-api",
		Short: "TF2 companion TUI",
		Long:  `tf-tui - A real-time game analysis and information tool for Team Fortress 2`,
		RunE:  run,
	}

	versionCmd = &cobra.Command{
		Use:               "version",
		Short:             "Print version information",
		Long:              "Print detailed version information about tf-tui",
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		Run:               version,
	}
)

var errApp = errors.New("application error")

func main() {
	configPath := config.Path(config.DefaultConfigName)
	// cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", configPath, "Config file path")
	rootCmd.AddCommand(versionCmd)

	if err := fang.Execute(context.Background(), rootCmd); err != nil {
		slog.Error("Exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func version(_ *cobra.Command, _ []string) {
	fmt.Printf("tf-tui - TF2 Terminal UI\n\n")      //nolint:forbidigo
	fmt.Printf("  Version: %s\n", BuildVersion)     //nolint:forbidigo
	fmt.Printf("  Commit:  %s\n", BuildCommit)      //nolint:forbidigo
	fmt.Printf("  Built:   %s\n", BuildDate)        //nolint:forbidigo
	fmt.Printf("  Runtime: %s\n\n", BuildGoVersion) //nolint:forbidigo
}

// run is the main entry point of tf-tui.
func run(cmd *cobra.Command, _ []string) error {
	// If PROFILE is set, it will be used as the output file path for the profiler.
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

	// Make sure our config & data home exists.
	if err := os.MkdirAll(path.Join(xdg.ConfigHome, config.ConfigDirName), 0o750); err != nil {
		return errors.Join(err, errApp)
	}

	var userConfig config.Config
	configUpdates := make(chan config.Config)

	configLoader := config.NewLoader(configUpdates)
	userConfig, errConfig := configLoader.Read()
	if errConfig != nil {
		return errors.Join(errApp, errConfig)
	}
	// Setup file based logger. This is very useful for us as our console is taken over by the ui.
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

	// Setup the filesystem cache, creating any necessary directories.
	cache, errCache := cache.New()
	if errCache != nil {
		return errors.Join(errCache, errApp)
	}

	// Setup all the data sources responsible for fetching player data.
	httpClient := &http.Client{Timeout: config.DefaultHTTPTimeout}
	client, errClient := tfapi.NewClientWithResponses(userConfig.APIBaseURL, tfapi.WithHTTPClient(httpClient))
	if errClient != nil {
		return errors.Join(errClient, errApp)
	}

	// Setup the sqlite database system.
	database, errDB := store.Open(cmd.Context(), config.Path(config.DefaultDBName), true)
	if errDB != nil {
		return errors.Join(errDB, errApp)
	}

	defer func() {
		if err := database.Close(); err != nil {
			slog.Error("Error closing database", slog.String("error", err.Error()))
		}
	}()

	// Setup a log source depending on the operating mode.
	router := events.NewRouter()
	metaFetcher := meta.New(client, cache)
	bdFetcher := bd.New(httpClient, userConfig.BDLists, cache)
	// Download the lists.
	go bdFetcher.Update(cmd.Context())

	states, errStates := state.NewManager(router, userConfig, metaFetcher, bdFetcher, database)
	if errStates != nil {
		return errors.Join(errStates, errApp)
	}

	if userConfig.Debug {
		consoleDebug := console.NewDebug("testdata/console.log")
		if errDebug := consoleDebug.Open(); errDebug != nil {
			return errors.Join(errDebug, errApp)
		}
		go consoleDebug.Start(cmd.Context(), router)
		defer func() {
			if err := consoleDebug.Close(cmd.Context()); err != nil {
				slog.Error("Error closing console debug", slog.String("error", err.Error()))
			}
		}()
	}

	done := make(chan any)
	app := NewApp(userConfig, states, database, router, configUpdates)

	go func() {
		if err := app.createUI(cmd.Context(), configLoader).Run(); err != nil {
			slog.Error("Failed to run UI", slog.String("error", err.Error()))
		}

		done <- "🫃"
	}()

	app.Start(cmd.Context(), done)

	return nil
}
