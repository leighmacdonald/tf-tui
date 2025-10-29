package pages

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/ui/component"
	"github.com/leighmacdonald/tf-tui/internal/ui/model"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
)

func NewMain(config config.Config) *Main {
	return &Main{
		banTableModel:          component.NewTableBansModel(),
		compTableModel:         component.NewTableCompModel(),
		bdTableModel:           component.NewTableBDModel(),
		detailPanelModel:       component.NewDetailPanelModel(config.Links),
		consoleModel:           component.NewConsoleModel(),
		serversTableModel:      component.NewServerTableModel(),
		serverDetailPanelModel: component.NewServerDetailPanel(),
		tabsModel:              component.NewTabsModel(),
		notesModel:             component.NewNotesModel(),
		redTableModel:          component.NewPlayerTableModel(tf.RED, config.SteamID, config.ServerModeEnabled),
		bluTableModel:          component.NewPlayerTableModel(tf.BLU, config.SteamID, config.ServerModeEnabled),
		chatModel:              component.NewChatModel(),
	}
}

type Main struct {
	consoleModel           *component.ConsoleModel
	detailPanelModel       component.DetailPanelModel
	serverDetailPanelModel component.ServerDetailPanelModel
	banTableModel          component.TableBansModel
	compTableModel         component.TableCompModel
	bdTableModel           component.TableBDModel
	serversTableModel      *component.ServerTableModel
	notesModel             component.NotesModel
	tabsModel              tea.Model
	chatModel              component.ChatModel
	redTableModel          tea.Model
	bluTableModel          tea.Model
	serverMode             bool
	viewState              model.ViewState
}

func (m Main) Init() tea.Cmd {
	return tea.Batch(
		m.tabsModel.Init(),
		m.notesModel.Init(),
		m.consoleModel.Init(),
		m.chatModel.Init(),
		m.bdTableModel.Init(),
		m.redTableModel.Init(),
		m.bluTableModel.Init(),
		m.serversTableModel.Init(),
		m.serverDetailPanelModel.Init())
}

func (m Main) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case model.ViewState:
		m.viewState = msg
	}
	return m, nil
}

func (m Main) View() string {
	header := m.tabsModel.View()
	hdr := styles.HeaderContainerStyle.Width(m.viewState.Width).Render(header)
	//_, hdrHeight := lipgloss.Size(hdr)

	var upper string
	if m.serverMode && m.viewState.Section == model.SectionServers {
		upper = m.serversTableModel.View()
	} else {
		upper = lipgloss.JoinHorizontal(lipgloss.Top, m.redTableModel.View(), m.bluTableModel.View())
	}

	// topContentHeight := min(m.height-lipgloss.Height(upper)-5, 20)
	lowerPanelViewportHeight := m.viewState.Lower - lipgloss.Height(upper) - 2
	var lower string
	switch m.viewState.Section {
	case model.SectionServers:
		lower = m.serverDetailPanelModel.Render(lowerPanelViewportHeight)
	case model.SectionPlayers:
		lower = m.detailPanelModel.Render(lowerPanelViewportHeight)
	case model.SectionBans:
		lower = m.banTableModel.Render(lowerPanelViewportHeight)
	case model.SectionBD:
		lower = m.bdTableModel.Render(lowerPanelViewportHeight)
	case model.SectionComp:
		lower = m.compTableModel.Render(lowerPanelViewportHeight)
	case model.SectionChat:
		lower = m.chatModel.View(lowerPanelViewportHeight)
	case model.SectionConsole:
		lower = m.consoleModel.Render(lowerPanelViewportHeight)
	}

	return lipgloss.JoinVertical(lipgloss.Top, hdr, upper, lower)
}

func (m Main) propagate(msg tea.Msg, _ ...tea.Cmd) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 16)

	m.redTableModel, cmds[1] = m.redTableModel.Update(msg)
	m.bluTableModel, cmds[2] = m.bluTableModel.Update(msg)
	m.banTableModel, cmds[3] = m.banTableModel.Update(msg)

	m.detailPanelModel, cmds[5] = m.detailPanelModel.Update(msg)
	m.tabsModel, cmds[6] = m.tabsModel.Update(msg)
	m.notesModel, cmds[7] = m.notesModel.Update(msg)
	m.compTableModel, cmds[8] = m.compTableModel.Update(msg)
	m.consoleModel, cmds[9] = m.consoleModel.Update(msg)

	m.chatModel, cmds[11] = m.chatModel.Update(msg)

	m.bdTableModel, cmds[13] = m.bdTableModel.Update(msg)
	m.serversTableModel, cmds[14] = m.serversTableModel.Update(msg)
	m.serverDetailPanelModel, cmds[15] = m.serverDetailPanelModel.Update(msg)

	return m, tea.Batch(cmds...)
}
