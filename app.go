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
	selectedSteamID steamid.SteamID
	statusMsg       string
	statusError     bool
	activeTab       tabView
	scripting       *Scripting
	listManager     *UserListManager
	helpView        help.Model
	detailPanel     tea.Model
	banTable        tea.Model
	playerTables    tea.Model
	compTable       tea.Model
	configModel     tea.Model
	notesTextArea   tea.Model
	tabs            tea.Model
}

func New(config Config, doSetup bool, scripting *Scripting, cache *PlayerData) *AppModel {
	app := &AppModel{
		cache:         cache,
		config:        config,
		helpView:      help.New(),
		scripting:     scripting,
		playerTables:  NewTablePlayersModel(),
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
		parts := []string{m.playerTables.View()}

		switch m.activeTab {
		case TabOverview:
			parts = append(parts, m.detailPanel.View())
		case TabBans:
			parts = append(parts, m.banTable.View())
		case TabComp:
			parts = append(parts, m.compTable.View())
		case TabNotes:
			parts = append(parts, "Notes...")
		}
		content = lipgloss.JoinVertical(lipgloss.Top, parts...)
	}

	footer = styles.FooterContainerStyle.
		Width(m.width).
		Render(lipgloss.JoinVertical(lipgloss.Top, m.renderHelp(), m.renderDebug()))
	header = m.tabs.View()
	hdr := styles.HeaderContainerStyle.Width(m.width).Render(header)
	_, hdrHeight := lipgloss.Size(hdr)
	ftr := styles.FooterContainerStyle.Width(m.width).Render(footer)
	_, ftrHeight := lipgloss.Size(ftr)
	contentViewPortHeight := m.height - hdrHeight - ftrHeight
	// Send the UI for rendering
	return zone.Scan(lipgloss.JoinVertical(lipgloss.Center,
		hdr,
		styles.ContentContainerStyle.Height(contentViewPortHeight).Render(content),
		ftr))
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
	m.playerTables, cmds[1] = m.playerTables.Update(msg)
	m.banTable, cmds[2] = m.banTable.Update(msg)
	m.helpView, cmds[3] = m.helpView.Update(msg)
	m.detailPanel, cmds[4] = m.detailPanel.Update(msg)
	m.tabs, cmds[5] = m.tabs.Update(msg)
	m.notesTextArea, cmds[6] = m.notesTextArea.Update(msg)
	m.compTable, cmds[7] = m.compTable.Update(msg)

	return m, tea.Batch(cmds...)
}

func (m AppModel) title() string {
	return lipgloss.NewStyle().Bold(true).
		Align(lipgloss.Center).
		Render(fmt.Sprintf("t: %d s: %d",
			m.selectedTeam, m.selectedSteamID.Int64()))
}

func (m AppModel) status() string {
	if m.statusError {
		return styles.Status.Foreground(styles.Red).Render(m.statusMsg)
	}

	return styles.Status.Render(m.statusMsg)
}

func (m AppModel) renderDebug() string {
	return lipgloss.JoinHorizontal(lipgloss.Center, m.title(), m.status())
}
