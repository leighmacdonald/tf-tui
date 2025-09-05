package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/tf"
)

type ContentViewPortHeightMsg struct {
	contentViewPortHeight int
	height                int
	width                 int
}

func setContentViewPortHeight(viewport int, height int, width int) func() tea.Msg {
	return func() tea.Msg {
		return ContentViewPortHeightMsg{
			contentViewPortHeight: viewport,
			height:                height,
			width:                 width,
		}
	}
}

type SortPlayersMsg struct {
	sortColumn playerTableCol
	asc        bool
}

func selectPlayer(player Player) func() tea.Msg {
	return func() tea.Msg {
		return SelectedPlayerMsg{player: player}
	}
}

type SelectedPlayerMsg struct {
	player Player
	notes  string
}

func selectTeam(team tf.Team) func() tea.Msg {
	return func() tea.Msg {
		return team
	}
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

func setStatusMessage(msg string, err bool) tea.Cmd {
	return func() tea.Msg {
		return StatusMsg{Message: msg, Err: err}
	}
}

// SetViewMsg will Switch the currently displayed center content view.

func setContentView(view contentView) tea.Cmd {
	return func() tea.Msg {
		return view
	}
}

func setTab(tab tabView) tea.Cmd {
	return func() tea.Msg { return tab }
}

type ChatMsg struct {
	Message  string
	ChatType ChatType
}

func setConfig(config config.Config) tea.Cmd {
	return func() tea.Msg { return config }
}
