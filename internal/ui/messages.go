package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/tf"
)

type contentViewPortHeightMsg struct {
	contentViewPortHeight int
	height                int
	width                 int
}

func setContentViewPortHeight(viewport int, height int, width int) func() tea.Msg {
	return func() tea.Msg {
		return contentViewPortHeightMsg{
			contentViewPortHeight: viewport,
			height:                height,
			width:                 width,
		}
	}
}

type sortPlayersMsg struct {
	sortColumn playerTableCol
	asc        bool
}

func selectPlayer(player Player) func() tea.Msg {
	return func() tea.Msg {
		return selectedPlayerMsg{player: player}
	}
}

type selectedPlayerMsg struct {
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

type statusMsg struct {
	Message string
	Err     bool
}

func setStatusMessage(msg string, err bool) tea.Cmd {
	return func() tea.Msg {
		return statusMsg{Message: msg, Err: err}
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

type chatMsg struct {
	Message  string
	ChatType ChatType
}

func setConfig(config config.Config) tea.Cmd {
	return func() tea.Msg { return config }
}

// Used to differentiate from a plain Snapshot which are braodcast for all servers.
type selectServerMsg struct {
	server Snapshot
}

func setServer(server Snapshot) tea.Cmd {
	return func() tea.Msg { return selectServerMsg{server: server} }
}
