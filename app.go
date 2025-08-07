package main

import (
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
	viewHelp
)

type AppModel struct {
	currentView           contentView
	previousView          contentView
	quitting              bool
	height                int
	width                 int
	activeTab             tabView
	scripting             *Scripting
	consoleView           *ConsoleModel
	detailPanel           *DetailPanelModel
	banTable              TableBansModel
	playerTables          tea.Model
	compTable             *TableCompModel
	configModel           tea.Model
	helpModel             tea.Model
	notesModel            tea.Model
	tabsModel             tea.Model
	statusModel           tea.Model
	chatModel             *ChatModel
	playerDataModel       tea.Model
	config                Config
	contentViewPortHeight int
	ftrHeight             int
	hdrHeight             int
	rendered              string
}

func New(config Config, doSetup bool, scripting *Scripting, client *ClientWithResponses) *AppModel {
	app := &AppModel{
		config:                config,
		currentView:           viewPlayerTables,
		previousView:          viewPlayerTables,
		activeTab:             TabOverview,
		scripting:             scripting,
		helpModel:             NewHelpModel(),
		playerTables:          NewTablePlayersModel(),
		banTable:              NewTableBansModel(),
		configModel:           NewConfigModal(config),
		compTable:             NewTableCompModel(),
		tabsModel:             NewTabsModel(),
		notesModel:            NewNotesModel(),
		detailPanel:           NewDetailPanelModel(config.Links),
		consoleView:           NewConsoleModel(config.ConsoleLogPath),
		statusModel:           NewStatusBarModel(BuildVersion),
		chatModel:             NewChatModel(),
		playerDataModel:       NewPlayerDataModel(client, config),
		contentViewPortHeight: 10,
		hdrHeight:             1,
		ftrHeight:             1,
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
		m.tabsModel.Init(),
		m.notesModel.Init(),
		m.consoleView.Init(),
		m.statusModel.Init(),
		m.chatModel.Init(),
		m.playerDataModel.Init(),

		func() tea.Msg {
			lists, err := downloadUserLists(m.config.BDLists)
			if err != nil {
				return err
			}

			return lists
		})
}

func (m AppModel) Update(inMsg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.isInitialized() {
		if _, ok := inMsg.(tea.WindowSizeMsg); !ok {
			return m, nil // return m.propagate(func() tea.Msg {
			//	return ContentViewPortHeightMsg{contentViewPortHeight: m.contentViewPortHeight, height: msg.Height, width: msg.Width}
			// })
		}
	}

	switch msg := inMsg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		m.contentViewPortHeight = m.height - m.hdrHeight - m.ftrHeight

		m2, cmd2 := m.propagate(inMsg)

		return m2, tea.Batch(func() tea.Msg {
			return ContentViewPortHeightMsg{
				contentViewPortHeight: m.contentViewPortHeight,
				height:                msg.Height,
				width:                 msg.Width,
			}
		}, cmd2)
	case TabChangeMsg:
		m.activeTab = tabView(msg)
	// Is it a key press?
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.quit):
			return m, tea.Quit
		case key.Matches(msg, DefaultKeyMap.help):
			if m.currentView == viewHelp {
				m.currentView = m.previousView
			} else {
				m.previousView = m.currentView
				m.currentView = viewHelp
			}
		case key.Matches(msg, DefaultKeyMap.config):
			if m.currentView == viewConfig {
				m.currentView = m.previousView
			} else {
				m.previousView = m.currentView
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

	// Early so we can use their size info
	footer = styles.FooterContainerStyle.
		Width(m.width).
		Render(lipgloss.JoinVertical(lipgloss.Top, m.statusModel.View()))
	header = m.tabsModel.View()
	hdr := styles.HeaderContainerStyle.Width(m.width).Render(header)
	_, hdrHeight := lipgloss.Size(hdr)
	// m.hdrHeight = hdrHeight

	ftr := styles.FooterContainerStyle.Width(m.width).Render(footer)
	_, ftrHeight := lipgloss.Size(ftr)
	// m.ftrHeight = ftrHeight

	contentViewPortHeight := m.height - hdrHeight - ftrHeight
	switch m.currentView {
	case viewConfig:
		content = m.configModel.View()
	case viewHelp:
		content = m.helpModel.View()
	case viewPlayerTables:
		tables := m.playerTables.View()
		playerHeight := m.height - lipgloss.Height(tables) - 5
		lowerPanelViewportHeight := contentViewPortHeight - lipgloss.Height(tables) - 2
		var ptContent string
		switch m.activeTab {
		case TabOverview:
			ptContent = m.detailPanel.View(lowerPanelViewportHeight)
		case TabBans:
			ptContent = m.banTable.View(lowerPanelViewportHeight)
		case TabComp:
			ptContent = m.compTable.View(lowerPanelViewportHeight)
		case TabNotes:
			ptContent = "Notes..."
		case TabChat:
			ptContent = m.chatModel.View(lowerPanelViewportHeight)
		case TabConsole:
			ptContent = m.consoleView.View(lowerPanelViewportHeight)
		}

		content = lipgloss.JoinVertical(
			lipgloss.Top,
			tables,
			lipgloss.NewStyle().
				Width(m.width-2).
				Height(playerHeight).
				Render(ptContent))
	}

	ctr := styles.ContentContainerStyle.Height(contentViewPortHeight).Render(content)

	return zone.Scan(lipgloss.JoinVertical(lipgloss.Center, hdr, ctr, ftr))
}

func (m AppModel) isInitialized() bool {
	return m.height != 0 && m.width != 0
}

func (m *AppModel) propagate(msg tea.Msg, cmd ...tea.Cmd) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 12)
	m.configModel, cmds[0] = m.configModel.Update(msg)
	m.playerTables, cmds[1] = m.playerTables.Update(msg)
	m.banTable, cmds[2] = m.banTable.Update(msg)
	m.helpModel, cmds[3] = m.helpModel.Update(msg)
	m.detailPanel, cmds[4] = m.detailPanel.Update(msg)
	m.tabsModel, cmds[5] = m.tabsModel.Update(msg)
	m.notesModel, cmds[6] = m.notesModel.Update(msg)
	m.compTable, cmds[7] = m.compTable.Update(msg)
	m.consoleView, cmds[8] = m.consoleView.Update(msg)
	m.statusModel, cmds[9] = m.statusModel.Update(msg)
	m.chatModel, cmds[10] = m.chatModel.Update(msg)
	m.playerDataModel, cmds[11] = m.playerDataModel.Update(msg)

	cmds = append(cmds, cmd...) //nolint:makezero

	return m, tea.Batch(cmds...)
}
