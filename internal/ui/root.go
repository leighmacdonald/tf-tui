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
	viewState              viewState
	viewStatePreviousView  viewState
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
	var view viewState
	if userConfig.ServerModeEnabled {
		view = viewState{page: pageMain, section: tabServers, keyZone: serverTable}
	} else {
		view = viewState{page: pageMain, section: tabPlayers, keyZone: playerTableRED}
	}

	app := &rootModel{
		parentContextChan:      parentChan,
		viewState:              view,
		viewStatePreviousView:  view,
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
		app.viewState.page = pageConfig
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
		vs := m.viewState
		vs.height = msg.Height
		vs.width = msg.Width
		upper := (msg.Height - m.headerHeight - m.footerHeight) / 2
		lower := upper
		if upper%2 != 0 {
			lower -= 1
		}

		return m, setViewStateStruct(vs)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, defaultKeyMap.quit):
			if m.viewState.page != pageMain {
				break
			}

			return m, tea.Quit
		case key.Matches(msg, defaultKeyMap.help):
			if m.viewState.page == pageHelp {
				m.viewState.page = m.viewStatePreviousView.page
			} else {
				m.viewStatePreviousView.page = m.viewState.page
				m.viewState.page = pageHelp
			}
		case key.Matches(msg, defaultKeyMap.config):
			if m.viewState.page == pageConfig {
				m.viewState.page = m.viewStatePreviousView.page
			} else {
				m.viewStatePreviousView.page = m.viewState.page
				m.viewState.page = pageConfig
			}
		case key.Matches(msg, defaultKeyMap.nextTab):
			return m, setNextZone(m.viewState.section, m.viewState.keyZone, right)
		case key.Matches(msg, defaultKeyMap.prevTab):
			return m, setNextZone(m.viewState.section, m.viewState.keyZone, left)
		}
	case viewState:
		m.viewState = msg
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
		Width(m.viewState.width).
		Render(lipgloss.JoinVertical(lipgloss.Top, m.statusModel.View()))
	header = m.tabsModel.View()
	hdr := styles.HeaderContainerStyle.Width(m.viewState.width).Render(header)
	_, hdrHeight := lipgloss.Size(hdr)
	// m.hdrHeight = hdrHeight

	ftr := styles.FooterContainerStyle.Width(m.viewState.width).Render(footer)
	_, ftrHeight := lipgloss.Size(ftr)
	// m.ftrHeight = ftrHeight

	contentViewPortHeight := m.viewState.height - hdrHeight - ftrHeight
	switch m.viewState.page {
	case pageConfig:
		content = m.configModelModel.View()
	case pageHelp:
		content = m.helpModel.View()
	case pageMain:
		var upper string
		if m.serverMode && m.viewState.section == tabServers {
			upper = m.serversTableModel.View()
		} else {
			upper = lipgloss.JoinHorizontal(lipgloss.Top, m.redTableModel.View(), m.bluTableModel.View())
		}

		// topContentHeight := min(m.height-lipgloss.Height(upper)-5, 20)
		lowerPanelViewportHeight := contentViewPortHeight - lipgloss.Height(upper) - 2
		var lower string
		switch m.viewState.section {
		case tabServers:
			lower = m.serverDetailPanelModel.Render(lowerPanelViewportHeight)
		case tabPlayers:
			lower = m.detailPanelModel.Render(lowerPanelViewportHeight)
		case tabBans:
			lower = m.banTableModel.Render(lowerPanelViewportHeight)
		case tabBD:
			lower = m.bdTableModel.Render(lowerPanelViewportHeight)
		case tabComp:
			lower = m.compTableModel.Render(lowerPanelViewportHeight)
		case tabChat:
			lower = m.chatModel.View(lowerPanelViewportHeight)
		case tabConsole:
			lower = m.consoleModel.Render(lowerPanelViewportHeight)
		}

		content = lipgloss.JoinVertical(lipgloss.Top, upper, lower)
	}

	ctr := styles.ContentContainerStyle.Height(contentViewPortHeight).Render(content)

	return zone.Scan(lipgloss.JoinVertical(lipgloss.Left, hdr, ctr, ftr))
}

func (m rootModel) isInitialized() bool {
	return m.viewState.height != 0 && m.viewState.width != 0
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
