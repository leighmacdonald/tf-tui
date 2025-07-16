package main

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config config.yaml ./openapi.yaml

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/adrg/xdg"
	tea "github.com/charmbracelet/bubbletea"
)

type Team int

const (
	UNASSIGNED = iota
	SPEC
	BLU
	RED
)

func main() {
	ctx := context.Background()
	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer func(f *os.File) {
			if errClose := f.Close(); errClose != nil {
				fmt.Println("error:", errClose.Error())
			}
		}(f)
	}

	client, errClient := NewClientWithResponses("http://localhost:8888/", WithHTTPClient(&http.Client{
		Timeout: defaultHTTPTimeout,
	}))
	if errClient != nil {
		fmt.Println("fatal:", errClient)
		os.Exit(1)
	}

	apis := NewAPIs(client)

	if err := os.MkdirAll(path.Join(xdg.ConfigHome, "tf-tui"), 0755); err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}

	config, configFound := configRead(defaultConfigName)

	scripting, errScripting := NewScripting()
	if errScripting != nil {
		fmt.Println("fatal:", errScripting.Error())
		os.Exit(1)
	}

	//if errScripts := scripting.LoadDir("scripts"); errScripts != nil {
	//	fmt.Println("fatal:", errScripts.Error())
	//	os.Exit(1)
	//}

	var opts []tea.ProgramOption
	if config.FullScreen {
		opts = append(opts, tea.WithAltScreen())
	}

	players := newPlayerStates(apis)
	go players.Start(ctx)

	model := newAppModel(config, !configFound, scripting, players)
	program := tea.NewProgram(model, opts...)
	if _, err := program.Run(); err != nil {
		fmt.Printf("There's been an error :( %v", err)
		os.Exit(1)
	}
}
