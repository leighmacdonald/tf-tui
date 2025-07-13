package main

import (
	"errors"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	errRCONExec  = errors.New("RCON exec error")
	errRCONParse = errors.New("RCON parse error")
	errRCON      = errors.New("errors making rcon request")
)

type clearErrorMsg struct{}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}
