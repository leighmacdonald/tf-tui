package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/tf/events"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
)

type statusBarModel struct {
	width       int
	hostname    string
	mapName     string
	statusMsg   string
	statusError bool
	snapshot    Snapshot
	version     string
	serverMode  bool
}

func newStatusBarModel(version string, serverMode bool) *statusBarModel {
	return &statusBarModel{version: version, serverMode: serverMode}
}

func (m statusBarModel) Init() tea.Cmd {
	return nil
}

func (m statusBarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case selectServerSnapshotMsg:
		m.snapshot = msg.server
	case statusMsg:
		m.statusMsg = msg.Message
		m.statusError = msg.Err

		return m, clearErrorAfter(clearMessageTimeout)
	case clearStatusMessageMsg:
		m.statusError = false
		m.statusMsg = ""
	case contentViewPortHeightMsg:
		m.width = msg.width
	case events.Event:
		switch data := msg.Data.(type) {
		case events.HostnameEvent:
			m.hostname = data.Hostname
		case events.MapEvent:
			m.mapName = data.MapName
		}
	}

	return m, nil
}

func (m statusBarModel) View() string {
	var args []string
	if !m.serverMode {
		args = append(args,
			styles.StatusRedTeam.Render(fmt.Sprintf("%3d", m.snapshot.Server.Players.TeamCount(tf.RED))),
			styles.StatusBluTeam.Render(fmt.Sprintf("%3d", m.snapshot.Server.Players.TeamCount(tf.BLU))))
	} else {
		if m.snapshot.Status.Stats.FPS < 66 {
			args = append(args, styles.StatusError.Underline(true).Render(fmt.Sprintf("FPS %2.2f", m.snapshot.Status.Stats.FPS)))
		} else {
			args = append(args, styles.StatusBluTeam.Render(fmt.Sprintf("FPS %2.2f  ", m.snapshot.Status.Stats.FPS)))
		}
		args = append(args,
			styles.StatusRedTeam.Render(fmt.Sprintf("CPU %2.2f  ", m.snapshot.Status.Stats.CPU)),
			styles.StatusMessage.Render(fmt.Sprintf("In/Out kb/s %.2f/%.2f", m.snapshot.Status.Stats.InKBs, m.snapshot.Status.Stats.OutKBs)),
			styles.StatusRedTeam.Render(fmt.Sprintf("Up %d", m.snapshot.Status.Stats.Uptime)),
			styles.StatusMap.Render("steam://run/440//+connect%20"+m.snapshot.HostPort),
		)
	}
	args = append(args,
		styles.StatusVersion.Render(m.version),
		styles.StatusHelp.Render(fmt.Sprintf("%s %s", defaultKeyMap.help.Help().Key, defaultKeyMap.help.Help().Desc)),
		m.status(),
		styles.StatusMap.Render(m.mapName))

	return lipgloss.NewStyle().Width(m.width).Background(styles.Black).Render(lipgloss.JoinHorizontal(lipgloss.Top, args...))
}

func (m statusBarModel) status() string {
	if m.statusMsg != "" {
		if m.statusError {
			return styles.StatusError.Render(m.statusMsg)
		}

		return styles.StatusMessage.Render(m.statusMsg)
	}

	return styles.StatusHostname.Render(m.hostname)
}
