package main

//go:generate go tool oapi-codegen -config config.yaml https://tf-api.roto.lol/api/openapi/schema-3.0.yaml

import (
	"context"
	"net/http"
	"os"
	"path"

	"github.com/adrg/xdg"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

type Team int

const (
	UNASSIGNED = iota
	SPEC
	BLU
	RED
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	zone.NewGlobal()

	if len(os.Getenv("DEBUG")) > 0 {
		logFile, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			return err
		}

		defer func(f *os.File) {
			_ = f.Close()
		}(logFile)
	}

	config, configFound := configRead(defaultConfigName)

	client, errClient := NewClientWithResponses(config.APIBaseURL, WithHTTPClient(&http.Client{
		Timeout: defaultHTTPTimeout,
	}))
	if errClient != nil {
		return errClient
	}

	apis := NewAPIs(client)

	if err := os.MkdirAll(path.Join(xdg.ConfigHome, "tf-tui"), 0o755); err != nil {
		return err
	}

	scripting, errScripting := NewScripting()
	if errScripting != nil {
		return errScripting
	}

	// if errScripts := scripting.LoadDir("scripts"); errScripts != nil {
	//	fmt.Println("fatal:", errScripts.Error())
	//	os.Exit(1)
	//}

	players := newPlayerStates(apis)
	go players.Start(ctx)

	model := New(config, !configFound, scripting, players)
	program := tea.NewProgram(model, tea.WithMouseCellMotion(), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		return err
	}

	return nil
}
