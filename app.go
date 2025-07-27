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
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/styles"
	zone "github.com/lrstanley/bubblezone"
)

type contentView int

const (
	viewPlayerTables contentView = iota
	viewConfig
	viewConfigFiles
)

type AppModel struct {
	config          Config
	cache           *PlayerData
	api             APIs
	currentView     contentView
	titleState      string
	quitting        bool
	height          int
	width           int
	selectedTeam    Team
	selectedRow     int
	selectedSteamID steamid.SteamID
	statusMsg       string
	statusError     bool
	activeTab       tabView
	scripting       *Scripting
	listManager     *UserListManager
	helpView        help.Model
	detailPanel     tea.Model
	banTable        tea.Model
	playerTable     tea.Model
	compTable       tea.Model
	configModel     tea.Model
	notesTextArea   tea.Model
	tabs            tea.Model
}

func New(config Config, doSetup bool, scripting *Scripting, cache *PlayerData) *AppModel {
	helpView := help.New()

	app := &AppModel{
		cache:         cache,
		config:        config,
		helpView:      helpView,
		scripting:     scripting,
		playerTable:   NewTablePlayersModel(),
		banTable:      NewTableBansModel(),
		configModel:   NewConfigModal(config),
		compTable:     NewTableCompModel(),
		tabs:          NewTabsModel(),
		notesTextArea: NewTextAreaNotes(),
		detailPanel:   DetailPanel{links: config.Links},
		listManager:   NewUserListManager(config.BDLists),
	}

	if doSetup {
		app.currentView = viewConfig
	}

	return app
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(tea.SetWindowTitle("tf-tui"), m.tickEvery(), m.configModel.Init(),
		textinput.Blink, m.tabs.Init(), m.notesTextArea.Init(), func() tea.Msg {
			m.listManager.Sync()

			return nil
		})
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
		m.selectedSteamID = msg.selectedSteamID
		m.selectedRow = msg.selectedRow
		m.selectedTeam = msg.selectedTeam
	case TabChangeMsg:
		m.activeTab = tabView(msg)
	// Is it a key press?
	case tea.KeyMsg:
		switch {
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
	var (
		header  string
		content string
		footer  string
	)

	switch m.currentView {
	case viewConfigFiles:
		fallthrough
	case viewConfig:
		content = m.configModel.View()
	case viewPlayerTables:
		var builder strings.Builder
		builder.WriteString(m.playerTable.View())

		builder.WriteString("\n")
		switch m.activeTab {
		case TabOverview:
			builder.WriteString(m.detailPanel.View())
		case TabBans:
			builder.WriteString(m.banTable.View())
		case TabComp:
			builder.WriteString(m.compTable.View())
		case TabNotes:
			builder.WriteString("Notes...")
		}
		content = builder.String()
	}

	footer = styles.FooterContainerStyle.
		Width(m.width).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, m.renderHelp(), m.renderHeading()))
	header = m.tabs.View()
	// Send the UI for rendering
	return zone.Scan(lipgloss.JoinVertical(lipgloss.Top,
		styles.HeaderContainerStyle.Width(m.width).Render(header),
		styles.ContentContainerStyle.Height(m.height-4).Render(content),
		styles.FooterContainerStyle.Width(m.width).Render(footer)))
}

func (m AppModel) isInitialized() bool {
	return m.height != 0 && m.width != 0
}

func (m AppModel) SelectedPlayer() (Player, bool) {
	if !m.selectedSteamID.Valid() {
		return Player{}, false
	}

	player, err := m.cache.Get(m.selectedSteamID)
	if err != nil {
		return Player{}, false
	}

	return player, true
}

func (m AppModel) renderHelp() string {
	var builder strings.Builder

	switch m.currentView {
	case viewConfig:
		builder.WriteString(m.helpView.ShortHelpView([]key.Binding{
			DefaultKeyMap.quit,
			DefaultKeyMap.accept,
		}))
	case viewPlayerTables:
		builder.WriteString(m.helpView.ShortHelpView([]key.Binding{
			DefaultKeyMap.quit,
			DefaultKeyMap.config,
			DefaultKeyMap.up,
			DefaultKeyMap.down,
			DefaultKeyMap.left,
			DefaultKeyMap.right,
			DefaultKeyMap.nextTab,
		}))
	case viewConfigFiles:
		k := filepicker.DefaultKeyMap()
		builder.WriteString(m.helpView.ShortHelpView([]key.Binding{
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
	cmds := make([]tea.Cmd, 8)
	m.configModel, cmds[0] = m.configModel.Update(msg)
	m.playerTable, cmds[1] = m.playerTable.Update(msg)
	m.banTable, cmds[2] = m.banTable.Update(msg)
	m.helpView, cmds[3] = m.helpView.Update(msg)
	m.detailPanel, cmds[4] = m.detailPanel.Update(msg)
	m.tabs, cmds[5] = m.tabs.Update(msg)
	m.notesTextArea, cmds[6] = m.notesTextArea.Update(msg)
	m.compTable, cmds[7] = m.compTable.Update(msg)

	return m, tea.Batch(cmds...)
}

func (m AppModel) title() string {
	return styles.Title.
		Width(m.width / 2).
		Render(fmt.Sprintf("c: %d r: %d u: %d",
			m.selectedTeam, m.selectedRow, m.selectedSteamID.Int64()))
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
