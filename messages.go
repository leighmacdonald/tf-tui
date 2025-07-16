package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type errMsg error

type PlayerStateMsg struct {
	err  error
	t    time.Time
	dump G15PlayerState
}

type SelectedPlayerMsg struct {
	player Player
}

type SelectedTableRowMsg struct {
	selectedTeam Team
	selectedRow  int
	selectedUID  int
}

type clearErrorMsg struct{}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}
