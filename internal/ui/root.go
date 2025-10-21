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

// rootModel is the top level model for the ui side of the app.
type rootModel struct {
	currentView            contentView
	previousView           contentView
	height                 int
	width                  int
	activeTab              tabView
	consoleModel           *consoleModel
	detailPanelModel       detailPanelModel
	serverDetailPanelModel serverDetailPanelModel
	banTableModel          tableBansModel
	compTableModel         tableCompModel
	bdTableModel           tableBDModel
	serversTableModel      *serverTableModel
	configModelModel       tea.Model
	helpModel              tea.Model
	notesModel             notesModel
	tabsModel              tea.Model
	statusModel            tea.Model
	chatModel              chatModel
	redTableModel          tea.Model
	bluTableModel          tea.Model
	footerHeight           int
	headerHeight           int
	serverMode             bool
	parentContextChan      chan any
}

func newRootModel(userConfig config.Config, doSetup bool, buildVersion string, buildDate string, buildCommit string, loader ConfigWriter, cachePath string, parentChan chan any) *rootModel {
	app := &rootModel{
		parentContextChan:      parentChan,
		currentView:            viewMain,
		previousView:           viewMain,
		activeTab:              tabServers,
		helpModel:              newHelpModel(buildVersion, buildDate, buildCommit, loader.Path(), cachePath),
		redTableModel:          newPlayerTableModel(tf.RED, userConfig.SteamID, userConfig.ServerModeEnabled),
		bluTableModel:          newPlayerTableModel(tf.BLU, userConfig.SteamID, userConfig.ServerModeEnabled),
		banTableModel:          newTableBansModel(),
		configModelModel:       newConfigModal(userConfig, loader),
		compTableModel:         newTableCompModel(),
		bdTableModel:           newTableBDModel(),
		tabsModel:              newTabsModel(),
		notesModel:             newNotesModel(),
		detailPanelModel:       newDetailPanelModel(userConfig.Links),
		consoleModel:           newConsoleModel(),
		serversTableModel:      newServerTableModel(),
		statusModel:            newStatusBarModel(buildVersion, userConfig.ServerModeEnabled),
		chatModel:              newChatModel(),
		serverDetailPanelModel: newServerDetailPanel(),
		serverMode:             userConfig.ServerModeEnabled,
		headerHeight:           1,
		footerHeight:           1,
	}

	if doSetup {
		app.currentView = viewConfig
	}

	return app
}

func (m rootModel) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("tf-tui"),
		m.configModelModel.Init(),
		textinput.Blink,
		m.tabsModel.Init(),
		m.notesModel.Init(),
		m.consoleModel.Init(),
		m.statusModel.Init(),
		m.chatModel.Init(),
		m.bdTableModel.Init(),
		m.redTableModel.Init(),
		m.bluTableModel.Init(),
		m.serversTableModel.Init(),
		m.serverDetailPanelModel.Init(),
		selectTeam(tf.RED),
	)
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
		contentViewPortHeight := m.height - m.headerHeight - m.footerHeight
		return m, setContentViewPortHeight(contentViewPortHeight, m.height, m.width)
	case tabView:
		m.activeTab = msg
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, defaultKeyMap.quit):
			if m.currentView != viewMain {
				break
			}

			return m, tea.Quit
		case key.Matches(msg, defaultKeyMap.help):
			if m.currentView == viewHelp {
				m.currentView = m.previousView
			} else {
				m.previousView = m.currentView
				m.currentView = viewHelp
			}
		case key.Matches(msg, defaultKeyMap.config):
			if m.currentView == viewConfig {
				m.currentView = m.previousView
			} else {
				m.previousView = m.currentView
				m.currentView = viewConfig
			}
		case key.Matches(msg, defaultKeyMap.left):
			return m, selectTeam(tf.RED)

		case key.Matches(msg, defaultKeyMap.right):
			return m, selectTeam(tf.BLU)
		}
	case contentView:
		m.currentView = msg
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
		content = m.configModelModel.View()
	case viewHelp:
		content = m.helpModel.View()
	case viewMain:
		var topContent string
		if m.serverMode {
			topContent = lipgloss.JoinHorizontal(lipgloss.Top, m.serversTableModel.View())
		} else {
			topContent = lipgloss.JoinHorizontal(lipgloss.Top, m.redTableModel.View(), m.bluTableModel.View())
		}

		topContentHeight := min(m.height-lipgloss.Height(topContent)-5, 20)
		lowerPanelViewportHeight := contentViewPortHeight - lipgloss.Height(topContent) - 2
		var ptContent string
		switch m.activeTab {
		case tabServers:
			ptContent = m.serverDetailPanelModel.Render(lowerPanelViewportHeight)
		case tabPlayers:
			ptContent = m.detailPanelModel.Render(lowerPanelViewportHeight)
		case tabBans:
			ptContent = m.banTableModel.Render(lowerPanelViewportHeight)
		case tabBD:
			ptContent = m.bdTableModel.Render(lowerPanelViewportHeight)
		case tabComp:
			ptContent = m.compTableModel.Render(lowerPanelViewportHeight)
		case tabChat:
			ptContent = m.chatModel.View(lowerPanelViewportHeight)
		case tabConsole:
			ptContent = m.consoleModel.Render(lowerPanelViewportHeight)
		}

		content = lipgloss.JoinVertical(
			lipgloss.Top,
			topContent,
			"",
			lipgloss.NewStyle().
				Width(m.width-2).
				Height(topContentHeight).
				Render(ptContent))
	}

	ctr := styles.ContentContainerStyle.Height(contentViewPortHeight).Render(content)

	return zone.Scan(lipgloss.JoinVertical(lipgloss.Left, hdr, ctr, ftr))
}

func (m rootModel) isInitialized() bool {
	return m.height != 0 && m.width != 0
}

func (m rootModel) propagate(msg tea.Msg, _ ...tea.Cmd) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 16)

	m.redTableModel, cmds[1] = m.redTableModel.Update(msg)
	m.bluTableModel, cmds[2] = m.bluTableModel.Update(msg)
	m.banTableModel, cmds[3] = m.banTableModel.Update(msg)
	m.helpModel, cmds[4] = m.helpModel.Update(msg)
	m.detailPanelModel, cmds[5] = m.detailPanelModel.Update(msg)
	m.tabsModel, cmds[6] = m.tabsModel.Update(msg)
	m.notesModel, cmds[7] = m.notesModel.Update(msg)
	m.compTableModel, cmds[8] = m.compTableModel.Update(msg)
	m.consoleModel, cmds[9] = m.consoleModel.Update(msg)
	m.statusModel, cmds[10] = m.statusModel.Update(msg)
	m.chatModel, cmds[11] = m.chatModel.Update(msg)
	m.configModelModel, cmds[12] = m.configModelModel.Update(msg)
	m.bdTableModel, cmds[13] = m.bdTableModel.Update(msg)
	m.serversTableModel, cmds[14] = m.serversTableModel.Update(msg)
	m.serverDetailPanelModel, cmds[15] = m.serverDetailPanelModel.Update(msg)

	return m, tea.Batch(cmds...)
}

// logMsg is useful for debugging events. Tail the log file ~/.config/tf-tui/tf-tui.log
func logMsg(inMsg tea.Msg) {
	// Filter out very noisy stuff
	switch inMsg.(type) {
	case selectServerSnapshotMsg:
	case LogRow:
		break
	case events.Event:
		break
	case Players:
		break
	case Snapshot:
		break
	case []Snapshot:
		break
	default:
		slog.Debug("tea.Msg", slog.Any("msg", inMsg))
	}
}
