package command

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/ui/input"
	"github.com/leighmacdonald/tf-tui/internal/ui/model"
)

func SetViewState(state model.ViewState) tea.Cmd {
	return func() tea.Msg { return state }
}

func SetNextZone(view model.Section, currentZone model.KeyZone, dir input.Direction) tea.Cmd {
	switch view {
	case model.SectionServers:
		return SetKeyZone(model.ServerZones.Next(currentZone, dir))
	case model.SectionPlayers:
		return SetKeyZone(model.PlayerZones.Next(currentZone, dir))
	case model.SectionBans:
		return SetKeyZone(model.BanZones.Next(currentZone, dir))
	case model.SectionBD:
		return SetKeyZone(model.BDZones.Next(currentZone, dir))
	case model.SectionComp:
		return SetKeyZone(model.CompZones.Next(currentZone, dir))
	case model.SectionChat:
		return SetKeyZone(model.ChatZones.Next(currentZone, dir))
	case model.SectionConsole:
		return SetKeyZone(model.ConsoleZones.Next(currentZone, dir))
	default:
		return nil
	}
}

func SetKeyZone(zone model.KeyZone) tea.Cmd {
	return func() tea.Msg { return zone }
}

const ClearMessageTimeout = time.Second * 10

type ClearStatusMessageMsg struct{}

func ClearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return ClearStatusMessageMsg{}
	})
}

func SelectTeam(team tf.Team) func() tea.Msg {
	return func() tea.Msg {
		return team
	}
}

type SelectedPlayerMsg struct {
	Player model.Player
	Notes  string
}

func SelectPlayer(player model.Player) func() tea.Msg {
	return func() tea.Msg {
		return SelectedPlayerMsg{Player: player}
	}
}

type SortMsg[T any] struct {
	SortColumn T
	Asc        bool
}

type StatusMsg struct {
	Message string
	Err     bool
}

func SetStatusMessage(msg string, err bool) tea.Cmd {
	return func() tea.Msg {
		return StatusMsg{Message: msg, Err: err}
	}
}

func SetConfig(config config.Config) tea.Cmd {
	return func() tea.Msg { return config }
}

// Used to differentiate from a plain Snapshot which are braodcast for all servers.
type SelectServerSnapshotMsg struct {
	Server model.Snapshot
}

func SetServer(server model.Snapshot) tea.Cmd {
	return func() tea.Msg { return SelectServerSnapshotMsg{Server: server} }
}

type ServerCVarList struct {
	HostPort string
	List     tf.CVarList
}

func SetServerCVarList(hostPort string, cvars tf.CVarList) tea.Cmd {
	return func() tea.Msg { return ServerCVarList{HostPort: hostPort, List: cvars} }
}

type RCONCommand struct {
	HostPort string
	Command  string
}

func SendRCONCommand(hostPort string, command string) tea.Cmd {
	return func() tea.Msg { return RCONCommand{HostPort: hostPort, Command: command} }
}
