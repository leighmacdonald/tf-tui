package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type DumpPlayerMsg struct {
	err  error
	t    time.Time
	dump DumpPlayer
}

type SortPlayersMsg struct {
	sortColumn playerTableColumn
	asc        bool
}

type SelectedPlayerMsg struct {
	player Player
	notes  string
}

type SelectedTableRowMsg struct {
	selectedTeam    Team
	selectedSteamID steamid.SteamID
}

type clearStatusMessageMsg struct{}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearStatusMessageMsg{}
	})
}

type FullStateUpdateMsg struct {
	players []Player
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

type ConsoleLogMsg struct {
	logs []LogEvent
	t    time.Time
}

type ChatMsg struct {
	Message  string
	ChatType ChatType
}
