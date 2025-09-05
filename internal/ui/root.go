package ui

import (
	"log/slog"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/tf/events"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
	zone "github.com/lrstanley/bubblezone"
)

// rootModel is the root model for the rootModel side of the app.
type rootModel struct {
	currentView           contentView
	previousView          contentView
	quitting              bool
	height                int
	width                 int
	activeTab             tabView
	consoleView           consoleModel
	detailPanel           detailPanelModel
	banTable              tableBansModel
	compTable             tableCompModel
	bdTable               tableBDModel
	configModel           tea.Model
	helpModel             tea.Model
	notesModel            notesModel
	tabsModel             tea.Model
	statusModel           tea.Model
	chatModel             chatModel
	redTable              tea.Model
	bluTable              tea.Model
	contentViewPortHeight int
	footerHeight          int
	headerHeight          int
	rendered              string
}

func newRootModel(config config.Config, doSetup bool, buildVersion string, buildDate string, buildCommit string) *rootModel {
	app := &rootModel{
		currentView:  viewPlayerTables,
		previousView: viewPlayerTables,
		activeTab:    tabOverview,
		helpModel:    newHelpModel(buildVersion, buildDate, buildCommit),
		redTable:     newPlayerTableModel(tf.RED, config.SteamID),
		bluTable:     newPlayerTableModel(tf.BLU, config.SteamID),
		banTable:     newTableBansModel(),
		configModel:  newConfigModal(config),
		compTable:    newTableCompModel(),
		bdTable:      newTableBDModel(),
		tabsModel:    newTabsModel(),
		notesModel:   newNotesModel(),
		detailPanel:  newDetailPanelModel(config.Links),
		consoleView:  newConsoleModel(),
		statusModel:  newStatusBarModel(buildVersion),
		chatModel:    newChatModel(),

		contentViewPortHeight: 10,
		headerHeight:          1,
		footerHeight:          1,
	}

	if doSetup {
		app.currentView = viewConfig
	}

	return app
}

func (m rootModel) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("tf-tui"),
		m.configModel.Init(),
		textinput.Blink,
		m.tabsModel.Init(),
		m.notesModel.Init(),
		m.consoleView.Init(),
		m.statusModel.Init(),
		m.chatModel.Init(),
		m.bdTable.Init(),
		m.redTable.Init(),
		m.bluTable.Init(),
		selectTeam(tf.RED),
	)
}

func logMsg(inMsg tea.Msg) {
	// Filter out very noisy stuff
	switch inMsg.(type) {
	case events.Event:
		break
	case Players:
		break
	default:
		slog.Debug("tea.Msg", slog.Any("msg", inMsg))
	}
}

func (m rootModel) Update(inMsg tea.Msg) (tea.Model, tea.Cmd) {
	logMsg(inMsg)

	if !m.isInitialized() {
		if _, ok := inMsg.(tea.WindowSizeMsg); !ok {
			return m, nil
		}
	}

	switch msg := inMsg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		m.contentViewPortHeight = m.height - m.headerHeight - m.footerHeight

		return m, setContentViewPortHeight(m.contentViewPortHeight, m.height, m.width)
	case TabChangeMsg:
		m.activeTab = tabView(msg)
	// Is it a key press?
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.quit):
			if m.currentView != viewPlayerTables {
				break
			}

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
		case key.Matches(msg, DefaultKeyMap.left):
			return m, selectTeam(tf.RED)

		case key.Matches(msg, DefaultKeyMap.right):
			return m, selectTeam(tf.BLU)
		}
	case SetViewMsg:
		m.currentView = msg.view
	}

	return m.propagate(inMsg)
}

func (m rootModel) View() string {
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
		playerTables := lipgloss.JoinHorizontal(lipgloss.Top, m.redTable.View(), m.bluTable.View())
		playerHeight := m.height - lipgloss.Height(playerTables) - 5
		lowerPanelViewportHeight := contentViewPortHeight - lipgloss.Height(playerTables) - 2
		var ptContent string
		switch m.activeTab {
		case tabOverview:
			ptContent = m.detailPanel.Render(lowerPanelViewportHeight)
		case tabBans:
			ptContent = m.banTable.Render(lowerPanelViewportHeight)
		case tabBD:
			ptContent = m.bdTable.Render(lowerPanelViewportHeight)
		case tabComp:
			ptContent = m.compTable.Render(lowerPanelViewportHeight)
		case tabChat:
			ptContent = m.chatModel.View(lowerPanelViewportHeight)
		case tabConsole:
			ptContent = m.consoleView.Render(lowerPanelViewportHeight)
		}

		content = lipgloss.JoinVertical(
			lipgloss.Top,
			playerTables,

			lipgloss.NewStyle().
				Width(m.width-2).
				Height(playerHeight).
				Render(ptContent))
	}

	ctr := styles.ContentContainerStyle.Height(contentViewPortHeight).Render(content)

	return zone.Scan(lipgloss.JoinVertical(lipgloss.Left, hdr, ctr, ftr))
}

func (m rootModel) isInitialized() bool {
	return m.height != 0 && m.width != 0
}

func (m rootModel) propagate(msg tea.Msg, _ ...tea.Cmd) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 14)
	m.redTable, cmds[1] = m.redTable.Update(msg)
	m.bluTable, cmds[2] = m.bluTable.Update(msg)
	m.banTable, cmds[3] = m.banTable.Update(msg)
	m.helpModel, cmds[4] = m.helpModel.Update(msg)
	m.detailPanel, cmds[5] = m.detailPanel.Update(msg)
	m.tabsModel, cmds[6] = m.tabsModel.Update(msg)
	m.notesModel, cmds[7] = m.notesModel.Update(msg)
	m.compTable, cmds[8] = m.compTable.Update(msg)
	m.consoleView, cmds[9] = m.consoleView.Update(msg)
	m.statusModel, cmds[10] = m.statusModel.Update(msg)
	m.chatModel, cmds[11] = m.chatModel.Update(msg)
	m.configModel, cmds[12] = m.configModel.Update(msg)
	m.bdTable, cmds[13] = m.bdTable.Update(msg)

	return m, tea.Batch(cmds...)
}
