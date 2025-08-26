package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tf2 "github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
)

type statusBarModel struct {
	width       int
	hostname    string
	mapName     string
	statusMsg   string
	statusError bool
	players     Players
	redPlayers  int
	bluPlayers  int
	version     string
}

func newStatusBarModel(version string) *statusBarModel {
	return &statusBarModel{version: version}
}

func (m statusBarModel) Init() tea.Cmd {
	return nil
}

func (m statusBarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case Players:
		var (
			red int
			blu int
		)

		for _, player := range msg {
			switch player.Team {
			case tf2.RED:
				red++
			case tf2.BLU:
				blu++
			}
		}
		m.players = msg
		m.redPlayers = red
		m.bluPlayers = blu
	case StatusMsg:
		m.statusMsg = msg.Message
		m.statusError = msg.Err

		return m, clearErrorAfter(time.Second * 10)
	case clearStatusMessageMsg:
		m.statusError = false
		m.statusMsg = ""
	case ContentViewPortHeightMsg:
		m.width = msg.width
	case tf2.LogEvent:
		switch msg.Type {
		case tf2.EvtHostname:
			m.hostname = msg.MetaData
		case tf2.EvtMap:
			m.mapName = msg.MetaData
		}
	}

	return m, nil
}

func (m statusBarModel) View() string {
	return lipgloss.NewStyle().Width(m.width).Background(styles.Black).Render(lipgloss.JoinHorizontal(lipgloss.Top,
		styles.StatusRedTeam.Render(fmt.Sprintf("%3d", m.redPlayers)),
		styles.StatusBluTeam.Render(fmt.Sprintf("%3d", m.bluPlayers)),
		styles.StatusVersion.Render(m.version),
		styles.StatusHelp.Render(fmt.Sprintf("%s %s", DefaultKeyMap.help.Help().Key, DefaultKeyMap.help.Help().Desc)),
		m.status(),
		styles.StatusMap.Render(m.mapName)))
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
