package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/tf"
)

type ContentViewPortHeightMsg struct {
	contentViewPortHeight int
	height                int
	width                 int
}

type SortPlayersMsg struct {
	sortColumn playerTableCol
	asc        bool
}

type SelectedPlayerMsg struct {
	player Player
	notes  string
}

type SelectedTableRowMsg struct {
	selectedTeam    tf.Team
	selectedSteamID steamid.SteamID
}

type clearStatusMessageMsg struct{}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearStatusMessageMsg{}
	})
}

type StatusMsg struct {
	Message string
	Err     bool
}

// SetViewMsg will Switch the currently displayed center content view.
type SetViewMsg struct {
	view contentView
}

func setContentView(view contentView) tea.Cmd {
	return func() tea.Msg {
		return SetViewMsg{view: view}
	}
}

type TabChangeMsg tabView

type ChatMsg struct {
	Message  string
	ChatType ChatType
}
