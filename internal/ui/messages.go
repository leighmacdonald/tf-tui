package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/tf"
)

// type viewPortSizeMsg struct {
// 	upperSize int
// 	lowerSize int
// 	height    int
// 	width     int
// }

// func setViewPortSizeMsg(upper int, lower int, height int, width int) func() tea.Msg {
// 	return func() tea.Msg {
// 		return viewPortSizeMsg{
// 			upperSize: upper,
// 			lowerSize: lower,
// 			height:    height,
// 			width:     width,
// 		}
// 	}
// }

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

// func setContentView(view page) tea.Cmd {
// 	return func() tea.Msg {
// 		return view
// 	}
// }

// func setTab(tab section) tea.Cmd {
// 	var zone keyZone
// 	if tab == tabServers {
// 		zone = serverTable
// 	} else {
// 		zone = playerTableRED
// 	}
// 	return tea.Batch(func() tea.Msg { return tab }, func() tea.Msg { return zone })
// }

type chatMsg struct {
	Message  string
	ChatType ChatType
}

func setConfig(config config.Config) tea.Cmd {
	return func() tea.Msg { return config }
}

// Used to differentiate from a plain Snapshot which are braodcast for all servers.
type selectServerSnapshotMsg struct {
	server Snapshot
}

func setServer(server Snapshot) tea.Cmd {
	return func() tea.Msg { return selectServerSnapshotMsg{server: server} }
}

type serverCVarList struct {
	HostPort string
	List     tf.CVarList
}

func setServerCVarList(hostPort string, cvars tf.CVarList) tea.Cmd {
	return func() tea.Msg { return serverCVarList{HostPort: hostPort, List: cvars} }
}

type RCONCommand struct {
	HostPort string
	Command  string
}

func sendRCONCommand(hostPort string, command string) tea.Cmd {
	return func() tea.Msg { return RCONCommand{HostPort: hostPort, Command: command} }
}

type viewState struct {
	section section
	page    page
	keyZone keyZone

	upperSize int
	lowerSize int
	height    int
	width     int
}

func setViewState(state viewState) tea.Cmd {
	return func() tea.Msg { return state }
}
