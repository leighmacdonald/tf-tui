package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type G15Msg struct {
	err  error
	t    time.Time
	dump G15PlayerState
}

type SelectedPlayerMsg struct {
	player Player
	notes  string
}

type SelectedTableRowMsg struct {
	selectedTeam Team
	selectedRow  int
	selectedUID  int
}

type clearStatusMessageMsg struct{}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearStatusMessageMsg{}
	})
}

type FullStateUpdateMsg struct {
	players     []Player
	selectedUID int
}

type StatusMsg struct {
	message string
	error   bool
}

// SetViewMsg will Switch the currently displayed center content view.
type SetViewMsg struct {
	view contentView
}

type TabChangeMsg tabView
