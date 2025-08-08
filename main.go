package main

//go:generate go tool oapi-codegen -config .openapi.yaml https://tf-api.roto.lol/api/openapi/schema-3.0.json
//go:generate go tool sqlc generate -f .sqlc.yaml

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leighmacdonald/tf-tui/store"
	zone "github.com/lrstanley/bubblezone"
	_ "modernc.org/sqlite"
)

var (
	BuildVersion = "v0.0.0"
	BuildCommit  = "none"
	BuildDate    = "unknown"
	BuildMode    = "production"
)

var errApp = errors.New("application error")

func main() {
	if err := run(); err != nil {
		tea.Println(err.Error())
		fmt.Println(err.Error()) //nolint:forbidigo
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	zone.NewGlobal()

	config, configFound := ConfigRead(defaultConfigName)
	logFile, errLogger := LoggerInit("", slog.LevelDebug)
	if errLogger != nil {
		return errLogger
	}
	defer logFile.Close()

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
			tea.Println(err.Error())
		}
	}()

	scripting, errScripting := NewScripting()
	if errScripting != nil {
		return errors.Join(errScripting, errApp)
	}

	// if errScripts := scripting.LoadDir("scripts"); errScripts != nil {
	//	fmt.Println("fatal:", errScripts.Error())
	//	os.Exit(1)
	//}

	cache, errCache := NewFilesystemCache()
	if errCache != nil {
		return errors.Join(errCache, errApp)
	}

	program := tea.NewProgram(New(config, !configFound, scripting, client, cache),
		tea.WithMouseCellMotion(), tea.WithAltScreen())

	go ConfigWatcher(ctx, program, defaultConfigName)

	if _, err := program.Run(); err != nil {
		return errors.Join(err, errApp)
	}

	return nil
}
