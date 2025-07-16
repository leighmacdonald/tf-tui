package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/styles"
)

var DefaultKeyMap = keymap{
	reset: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "reset"),
	),
	quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "Quit"),
	),
	config: key.NewBinding(
		key.WithKeys("E"),
		key.WithHelp("E", "Conf"),
	),
	up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑", "Up"),
	),
	down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓", "Down"),
	),
	left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←", "RED"),
	),
	right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→", "BLU"),
	),
	fs: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "Toggle View"),
	),
}

type contentView int

const (
	viewPlayerTables contentView = iota
	viewConfig
	viewConfigFiles
)

type AppModel struct {
	config       Config
	cache        *PlayerData
	api          APIs
	altScreen    bool
	currentView  contentView
	titleState   string
	quitting     bool
	err          errMsg
	messages     []string
	height       int
	width        int
	selectedTeam Team
	selectedRow  int
	selectedUID  int
	statusMsg    string
	scripting    *Scripting
	helpView     help.Model
	banTable     tea.Model
	playerTable  tea.Model
	configModel  tea.Model
	tabs         tea.Model
}

func newAppModel(config Config, doSetup bool, scripting *Scripting, cache *PlayerData) *AppModel {
	address := config.Address
	if address == "" {
		address = "127.0.0.1:27015"
	}
	app := &AppModel{
		cache:       cache,
		altScreen:   config.FullScreen,
		config:      config,
		helpView:    help.New(),
		scripting:   scripting,
		playerTable: newTableModel(),
		banTable:    NewTableDetailModel(),
		configModel: newConfigModal(config),
	}

	if doSetup {
		app.currentView = viewConfig
	}

	return app

}
func (m AppModel) isInitialized() bool {
	return m.height != 0 && m.width != 0
}
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(tea.SetWindowTitle("tf-tui"), m.tickEvery(), m.configModel.Init(), textinput.Blink)
}

func (m AppModel) SelectedPlayer() (Player, bool) {
	if m.selectedUID < 0 {
		return Player{}, false
	}

	return m.cache.ByUID(m.selectedUID)
}

func (m AppModel) View() string {
	var b strings.Builder
	b.WriteString(m.renderHeading())
	switch m.currentView {
	case viewConfig:
		b.WriteString(m.configModel.View())
	case viewPlayerTables:
		b.WriteString("\n" + m.playerTable.View())
		b.WriteString("\n")

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, m.banTable.View()))

		b.WriteString("\n")
		b.WriteString(m.helpView.ShortHelpView([]key.Binding{
			DefaultKeyMap.quit,
			DefaultKeyMap.config,
			DefaultKeyMap.fs,
			DefaultKeyMap.up,
			DefaultKeyMap.down,
			DefaultKeyMap.left,
			DefaultKeyMap.right,
		}))
	}

	// The footer
	b.WriteString(strings.Join(m.messages, "\n"))

	// Send the UI for rendering
	return b.String()
}

func (m AppModel) tickEvery() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		dump, errDump := fetchPlayerState(context.Background(), m.config.Address, m.config.Password)
		if errDump != nil {
			return PlayerStateMsg{err: errDump, t: t}
		}

		return PlayerStateMsg{t: t, dump: dump}
	})
}

type FullStateUpdateMsg struct {
	players     []Player
	selectedUID int
}

func (m AppModel) onPlayerStateMsg(msg PlayerStateMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
	}

	//if m.selectedRow > m.playerTable.selectedColumnPlayerCount()-1 {
	//	m.selectedRow = max(m.playerTable.selectedColumnPlayerCount()-1, 0)
	//}

	m.cache.SetStats(msg.dump)

	players, errPlayers := m.cache.All()
	if errPlayers != nil {
		return m, tea.Batch(m.tickEvery())
	}

	return m, tea.Batch(m.tickEvery(), func() tea.Msg {
		return FullStateUpdateMsg{
			players:     players,
			selectedUID: 0,
		}
	})
}

func (m AppModel) propagate(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Propagate to all children.
	//m.tabs, _ = m.tabs.Update(msg)
	if m.currentView == viewConfig {
		m.configModel, _ = m.configModel.Update(msg)
	}

	var cmd tea.Cmd
	m.playerTable, cmd = m.playerTable.Update(msg)
	m.banTable, _ = m.banTable.Update(msg)
	m.helpView, _ = m.helpView.Update(msg)

	return m, cmd
}
func (m AppModel) Update(inMsg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.isInitialized() {
		if _, ok := inMsg.(tea.WindowSizeMsg); !ok {
			return m, nil
		}
	}

	switch msg := inMsg.(type) {
	case Config:
		m.config = msg
	case PlayerStateMsg:
		return m.onPlayerStateMsg(msg)
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		return m.propagate(inMsg)
	case SelectedTableRowMsg:
		m.selectedUID = msg.selectedUID
		m.selectedRow = msg.selectedRow
		m.selectedTeam = msg.selectedTeam
	// Is it a key press?
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.fs):
			var cmd tea.Cmd
			if m.altScreen {
				cmd = tea.ExitAltScreen
			} else {
				cmd = tea.EnterAltScreen
			}
			m.altScreen = !m.altScreen
			return m, cmd
		case key.Matches(msg, DefaultKeyMap.quit):
			return m, tea.Quit
		case key.Matches(msg, DefaultKeyMap.config):
			if m.currentView == viewConfig {
				m.currentView = viewPlayerTables
			} else {
				m.currentView = viewConfig
			}
		}
		return m.propagate(inMsg)
	case clearErrorMsg:
		m.err = nil
		return m, nil
	case errMsg:
		m.err = msg
		return m, nil
	}

	return m.propagate(inMsg)
}

func (m AppModel) title() string {
	return styles.Title.
		Width(m.width / 2).
		Render(fmt.Sprintf("c: %d r: %d u: %d",
			m.selectedTeam, m.selectedRow, m.selectedUID))
}

func (m AppModel) status() string {
	if m.err != nil {
		return styles.Status.Width(m.width / 2).Render(m.err.Error())
	}
	return styles.Status.Width(m.width / 2).Render(m.statusMsg)
}

func (m AppModel) renderHeading() string {
	out := lipgloss.JoinHorizontal(lipgloss.Top, m.title(), m.status())

	if m.quitting {
		return out + "\n"
	}

	return out
}
