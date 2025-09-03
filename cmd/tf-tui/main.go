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
	tftui "github.com/leighmacdonald/tf-tui/internal"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/store"
	"github.com/leighmacdonald/tf-tui/internal/tf"
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
		Use:   "version",
		Short: "Print version information",
		Long:  "Print detailed version information about tf-tui",
		Run:   version,
	}
)

var errApp = errors.New("application error")

func main() {
	configPath := config.PathConfig(config.DefaultConfigName)
	// cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", configPath,
		"Config file path")
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
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

	// Try and load the users config, uses default config on error.
	userConfig, errConfig := config.Read(cfgFile)
	if errConfig != nil {
		slog.Error("Failed to load config", slog.String("error", errConfig.Error()))
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
	metaFetcher := tftui.NewMetaFetcher(client, cache)
	bdFetcher := tftui.NewBDFetcher(httpClient, userConfig.BDLists, cache)

	// Setup the sqlite database system.
	database, errDB := store.Open(ctx, config.PathConfig(config.DefaultDBName), true)
	if errDB != nil {
		return errors.Join(errDB, errApp)
	}

	defer func() {
		if err := database.Close(); err != nil {
			slog.Error("Error closing database", slog.String("error", err.Error()))
		}
	}()

	// Setup a log source depending on the operating mode.
	logBroadcater := tf.NewLogBroadcaster()
	var logSource LogSource

	if userConfig.ServerModeEnabled {
		listener, errListener := tf.NewSRCDSListener(logBroadcater, tf.SRCDSListenerOpts{})
		if errListener != nil {
			return errListener
		}
		logSource = listener
	} else {
		logSource = tf.NewConsoleLog(logBroadcater)
	}

	done := make(chan any)

	app := NewApp(userConfig, metaFetcher, bdFetcher, database, logBroadcater, logSource)

	go func() {
		if err := app.createUI(ctx).Run(); err != nil {
			slog.Error("Failed to run UI", slog.String("error", err.Error()))
		}

		done <- "🫃"
	}()

	app.Start(ctx, done)

	return nil
}
