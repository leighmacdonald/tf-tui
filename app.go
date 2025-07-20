package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/styles"
)

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
	height       int
	width        int
	selectedTeam Team
	selectedRow  int
	selectedUID  int
	statusMsg    string
	statusError  bool
	activeTab    tabView
	scripting    *Scripting
	helpView     help.Model
	detailPanel  tea.Model
	banTable     tea.Model
	playerTable  tea.Model
	configModel  tea.Model
	tabs         tea.Model
}

func newAppModel(config Config, doSetup bool, scripting *Scripting, cache *PlayerData) *AppModel {
	app := &AppModel{
		cache:       cache,
		altScreen:   config.FullScreen,
		config:      config,
		helpView:    help.New(),
		scripting:   scripting,
		playerTable: newTableModel(),
		banTable:    newTableDetailModel(),
		configModel: newConfigModal(config),
		tabs:        newTabsModel(),
		detailPanel: DetailPanel{},
	}

	if doSetup {
		app.currentView = viewConfig
	}

	return app
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(tea.SetWindowTitle("tf-tui"), m.tickEvery(), m.configModel.Init(), textinput.Blink, m.tabs.Init())
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
	case StatusMsg:
		m.statusMsg = msg.message
		m.statusError = msg.error

		return m, clearErrorAfter(time.Second * 5)
	case G15Msg:
		return m.onPlayerStateMsg(msg)
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width

		return m.propagate(inMsg)
	case SelectedTableRowMsg:
		m.selectedUID = msg.selectedUID
		m.selectedRow = msg.selectedRow
		m.selectedTeam = msg.selectedTeam
	case TabChangeMsg:
		m.activeTab = tabView(msg)
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
	case clearStatusMessageMsg:
		m.statusError = false
		m.statusMsg = ""

		return m, nil
	case SetViewMsg:
		m.currentView = msg.view
	}

	return m.propagate(inMsg)
}

func (m AppModel) View() string {
	var builder strings.Builder

	switch m.currentView {
	case viewConfig:
		builder.WriteString(m.configModel.View())
	case viewPlayerTables:
		builder.WriteString(m.playerTable.View())
		builder.WriteString(m.tabs.View())
		builder.WriteString("\n")
		switch m.activeTab {
		case TabOverview:
			builder.WriteString(m.detailPanel.View())
		case TabBans:
			builder.WriteString(m.banTable.View())
		}
	}

	builder.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, m.renderHelp(), m.renderHeading()))
	// Send the UI for rendering
	return builder.String()
}

func (m AppModel) isInitialized() bool {
	return m.height != 0 && m.width != 0
}

func (m AppModel) SelectedPlayer() (Player, bool) {
	if m.selectedUID < 0 {
		return Player{}, false
	}

	return m.cache.ByUID(m.selectedUID)
}

func (m AppModel) renderHelp() string {
	helpView := help.New()
	var builder strings.Builder
	builder.WriteString("\n")

	switch m.currentView {
	case viewConfig:
		builder.WriteString(helpView.ShortHelpView([]key.Binding{
			DefaultKeyMap.quit,
			DefaultKeyMap.accept,
		}))
	case viewPlayerTables:
		builder.WriteString(m.helpView.ShortHelpView([]key.Binding{
			DefaultKeyMap.quit,
			DefaultKeyMap.config,
			DefaultKeyMap.fs,
			DefaultKeyMap.up,
			DefaultKeyMap.down,
			DefaultKeyMap.left,
			DefaultKeyMap.right,
			DefaultKeyMap.nextTab,
		}))
	case viewConfigFiles:
		k := filepicker.DefaultKeyMap()
		builder.WriteString(helpView.ShortHelpView([]key.Binding{
			k.Down, k.Up, k.Open, k.Select, k.Back, k.GoToLast, k.GoToTop, k.PageDown, k.PageUp,
		}))
	}

	return builder.String()
}

func (m AppModel) tickEvery() tea.Cmd {
	return tea.Tick(time.Second, func(lastTime time.Time) tea.Msg {
		dump, errDump := fetchPlayerState(context.Background(), m.config.Address, m.config.Password)
		if errDump != nil {
			return G15Msg{err: errDump, t: lastTime}
		}

		return G15Msg{t: lastTime, dump: dump}
	})
}

func (m AppModel) onPlayerStateMsg(msg G15Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	if msg.err != nil {
		cmds = append(cmds, func() tea.Msg {
			return StatusMsg{
				message: msg.err.Error(),
				error:   true,
			}
		})
	}

	// if m.selectedRow > m.playerTable.selectedColumnPlayerCount()-1 {
	//	m.selectedRow = max(m.playerTable.selectedColumnPlayerCount()-1, 0)
	//}

	m.cache.SetStats(msg.dump)

	players, errPlayers := m.cache.All()
	if errPlayers != nil {
		return m, tea.Batch(m.tickEvery())
	}

	cmds = append(cmds, m.tickEvery(), func() tea.Msg {
		return FullStateUpdateMsg{
			players:     players,
			selectedUID: 0,
		}
	})

	return m, tea.Batch(cmds...)
}

func (m AppModel) propagate(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Propagate to all children.
	// m.tabs, _ = m.tabs.Update(msg)
	cmds := make([]tea.Cmd, 6)
	m.configModel, cmds[0] = m.configModel.Update(msg)
	m.playerTable, cmds[1] = m.playerTable.Update(msg)
	m.banTable, cmds[2] = m.banTable.Update(msg)
	m.helpView, cmds[3] = m.helpView.Update(msg)
	m.detailPanel, cmds[4] = m.detailPanel.Update(msg)
	m.tabs, cmds[5] = m.tabs.Update(msg)

	return m, tea.Batch(cmds...)
}

func (m AppModel) title() string {
	return styles.Title.
		Width(m.width / 2).
		Render(fmt.Sprintf("c: %d r: %d u: %d",
			m.selectedTeam, m.selectedRow, m.selectedUID))
}

func (m AppModel) status() string {
	if m.statusError {
		return styles.Status.Foreground(styles.Red).Width(m.width / 2).Render(m.statusMsg)
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
