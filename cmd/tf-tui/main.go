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
	"sync"
	"time"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/fang"
	tftui "github.com/leighmacdonald/tf-tui/internal"
	"github.com/leighmacdonald/tf-tui/internal/config"
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", configPath,
		"Config file path")
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
func run(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

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

	loader := config.NewLoader(configUpdates)
	userConfig, errConfig := loader.Read()
	if errConfig != nil {
		return errors.Join(errConfig, errApp)
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
	cache, errCache := tftui.NewFilesystemCache()
	if errCache != nil {
		return errors.Join(errCache, errApp)
	}

	// Setup all the data sources responsible for fetching player data.
	httpClient := &http.Client{Timeout: config.DefaultHTTPTimeout}
	client, errClient := tfapi.NewClientWithResponses(userConfig.APIBaseURL, tfapi.WithHTTPClient(httpClient))
	if errClient != nil {
		return errors.Join(errClient, errApp)
	}

	// Setup a log source depending on the operating mode.
	router := events.NewRouter()
	metaFetcher := tftui.NewMetaFetcher(client, cache)
	bdFetcher := tftui.NewBDFetcher(httpClient, userConfig.BDLists, cache)
	states, errStates := tftui.NewStateTracker(router, userConfig, metaFetcher, bdFetcher)
	if errStates != nil {
		return errors.Join(errStates, errApp)
	}

	// Setup the sqlite database system.
	database, errDB := store.Open(ctx, config.Path(config.DefaultDBName), true)
	if errDB != nil {
		return errors.Join(errDB, errApp)
	}

	defer func() {
		if err := database.Close(); err != nil {
			slog.Error("Error closing database", slog.String("error", err.Error()))
		}
	}()

	logSources, errLogSource := openLogSources(ctx, userConfig)
	if errLogSource != nil {
		return errors.Join(errLogSource, errApp)
	}
	defer logSources.Close(ctx)

	go logSources.Start(ctx, router)

	if len(os.Getenv("DEBUG")) > 0 {
		consoleDebug := console.NewDebug("testdata/console.log")
		if errDebug := consoleDebug.Open(ctx); errDebug != nil {
			return errors.Join(errDebug, errApp)
		}
		go consoleDebug.Start(ctx, router)
		defer consoleDebug.Close(ctx)
	}

	done := make(chan any)

	app := NewApp(userConfig, states, database, router, logSources, configUpdates)

	go func() {
		if err := app.createUI(ctx, loader).Run(); err != nil {
			slog.Error("Failed to run UI", slog.String("error", err.Error()))
		}

		done <- "🫃"
	}()

	app.Start(ctx, done)

	return nil
}

func openLogSources(ctx context.Context, userConfig config.Config) ([]console.Source, error) {
	var logSources []console.Source
	if userConfig.ServerModeEnabled {
		listener, errListener := console.NewRemote(console.SRCDSListenerOpts{
			ExternalAddress: userConfig.ServerLogAddress,
			Secret:          userConfig.ServerLogSecret,
			ListenAddress:   userConfig.ServerListenAddress,
			RemoteAddress:   userConfig.Address,
			RemotePassword:  userConfig.Password,
		})
		if errListener != nil {
			return nil, errListener
		}
		logSources = append(logSources, listener)
	} else {
		logSources = append(logSources, console.NewLocal(userConfig.ConsoleLogPath))
	}

	waitGroup := &sync.WaitGroup{}
	for _, logSource := range logSources {
		waitGroup.Add(1)
		go func(source console.Source) {
			defer waitGroup.Done()
			if errOpen := source.Open(ctx); errOpen != nil {
				slog.Error("Failed to open log source", slog.String("error", errOpen.Error()))
			}
		}(logSource)
	}
	waitGroup.Wait()

	return logSources, nil
}
