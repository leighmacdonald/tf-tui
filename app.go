package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/styles"
	zone "github.com/lrstanley/bubblezone"
)

type contentView int

const (
	viewPlayerTables contentView = iota
	viewConfig
)

type AppModel struct {
	currentView     contentView
	quitting        bool
	height          int
	width           int
	activeTab       tabView
	scripting       *Scripting
	listManager     *UserListManager
	helpView        help.Model
	consoleView     tea.Model
	detailPanel     tea.Model
	banTable        tea.Model
	playerTables    tea.Model
	compTable       tea.Model
	configModel     tea.Model
	notesTextArea   tea.Model
	tabs            tea.Model
	statusView      tea.Model
	chatView        tea.Model
	playerDataModel tea.Model
}

func New(config Config, doSetup bool, scripting *Scripting, client *ClientWithResponses) *AppModel {
	app := &AppModel{
		currentView:     viewPlayerTables,
		activeTab:       TabOverview,
		scripting:       scripting,
		helpView:        help.New(),
		playerTables:    NewTablePlayersModel(),
		banTable:        NewTableBansModel(),
		configModel:     NewConfigModal(config),
		compTable:       NewTableCompModel(),
		tabs:            NewTabsModel(),
		notesTextArea:   NewNotesModel(),
		detailPanel:     NewDetailPanelModel(config.Links),
		listManager:     NewUserListManager(config.BDLists),
		consoleView:     NewConsoleModel(config.ConsoleLogPath),
		statusView:      NewStatusBarModel(),
		chatView:        NewChatModel(),
		playerDataModel: NewPlayerDataModel(client, config),
	}

	if doSetup {
		app.currentView = viewConfig
	}

	return app
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("tf-tui"),
		m.configModel.Init(),
		textinput.Blink,
		m.tabs.Init(),
		m.notesTextArea.Init(),
		m.consoleView.Init(),
		m.statusView.Init(),
		m.chatView.Init(),
		m.playerDataModel.Init(),
		func() tea.Msg {
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
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
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
	case viewConfig:
		content = m.configModel.View()
	case viewPlayerTables:
		tables := m.playerTables.View()
		var ptContent string
		switch m.activeTab {
		case TabOverview:
			ptContent = m.detailPanel.View()
		case TabBans:
			ptContent = m.banTable.View()
		case TabComp:
			ptContent = m.compTable.View()
		case TabNotes:
			ptContent = "Notes..."
		case TabChat:
			ptContent = m.chatView.View()
		case TabConsole:
			ptContent = m.consoleView.View()
		}

		playerHeight := m.height - lipgloss.Height(tables) - 5

		content = lipgloss.JoinVertical(
			lipgloss.Top,
			tables,
			lipgloss.NewStyle().Width(m.width-2).Height(playerHeight).
				Border(lipgloss.NormalBorder()).
				BorderForeground(styles.Gray).
				Render(ptContent))
	}

	footer = styles.FooterContainerStyle.
		Width(m.width).
		Render(lipgloss.JoinVertical(lipgloss.Top, m.renderHelp(), m.statusView.View()))
	header = m.tabs.View()
	hdr := styles.HeaderContainerStyle.Width(m.width).Render(header)
	_, hdrHeight := lipgloss.Size(hdr)
	ftr := styles.FooterContainerStyle.Width(m.width).Render(footer)
	_, ftrHeight := lipgloss.Size(ftr)
	contentViewPortHeight := m.height - hdrHeight - ftrHeight
	ctr := styles.ContentContainerStyle.Height(contentViewPortHeight).Render(content)

	return zone.Scan(lipgloss.JoinVertical(lipgloss.Center, hdr, ctr, ftr))
}

func (m AppModel) isInitialized() bool {
	return m.height != 0 && m.width != 0
}

// func (m AppModel) SelectedPlayer() (Player, bool) {
//	if !m.selectedSteamID.Valid() {
//		return Player{}, false
//	}
//
//	player, err := m.cache.Get(m.selectedSteamID)
//	if err != nil {
//		return Player{}, false
//	}
//
//	return player, true
// }

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
			DefaultKeyMap.console,
			DefaultKeyMap.chat,
		}))
	}

	return builder.String()
}

func (m AppModel) propagate(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 12)
	m.configModel, cmds[0] = m.configModel.Update(msg)
	m.playerTables, cmds[1] = m.playerTables.Update(msg)
	m.banTable, cmds[2] = m.banTable.Update(msg)
	m.helpView, cmds[3] = m.helpView.Update(msg)
	m.detailPanel, cmds[4] = m.detailPanel.Update(msg)
	m.tabs, cmds[5] = m.tabs.Update(msg)
	m.notesTextArea, cmds[6] = m.notesTextArea.Update(msg)
	m.compTable, cmds[7] = m.compTable.Update(msg)
	m.consoleView, cmds[8] = m.consoleView.Update(msg)
	m.statusView, cmds[9] = m.statusView.Update(msg)
	m.chatView, cmds[10] = m.chatView.Update(msg)
	m.playerDataModel, cmds[11] = m.playerDataModel.Update(msg)

	return m, tea.Batch(cmds...)
}
