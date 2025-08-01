package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/styles"
)

type StatusBarModel struct {
	width       int
	hostname    string
	mapName     string
	statusMsg   string
	statusError bool
	players     []Player
	redPlayers  int
	bluPlayers  int
}

func NewStatusBarModel() *StatusBarModel {
	return &StatusBarModel{}
}

func (m StatusBarModel) Init() tea.Cmd {
	return nil
}

func (m StatusBarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case FullStateUpdateMsg:
		var (
			red int
			blu int
		)

		for _, player := range msg.players {
			switch player.Team {
			case RED:
				red++
			case BLU:
				blu++
			}
		}
		m.players = msg.players
		m.redPlayers = red
		m.bluPlayers = blu
	case StatusMsg:
		m.statusMsg = msg.message
		m.statusError = msg.error

		return m, clearErrorAfter(time.Second * 10)
	case clearStatusMessageMsg:
		m.statusError = false
		m.statusMsg = ""
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case LogEvent:
		switch msg.Type {
		case EvtHostname:
			m.hostname = msg.MetaData
		case EvtMap:
			m.mapName = msg.MetaData
		}
	}

	return m, nil
}

func (m StatusBarModel) View() string {
	return lipgloss.NewStyle().Width(m.width).Background(styles.Black).Render(lipgloss.JoinHorizontal(lipgloss.Top,
		styles.StatusRedTeam.Render(fmt.Sprintf("%3d", m.redPlayers)),
		styles.StatusBluTeam.Render(fmt.Sprintf("%3d", m.bluPlayers)),
		styles.StatusHostname.Render(m.hostname),
		styles.StatusMap.Render(m.mapName)))
}

func (m StatusBarModel) status() string {
	if m.statusError {
		return styles.StatusError.Render(m.statusMsg)
	}

	return styles.StatusMessage.Render(m.statusMsg)
}
