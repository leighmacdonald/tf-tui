package ui

import (
	"context"
	"errors"
	"log/slog"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/config"
	"github.com/leighmacdonald/tf-tui/tf"
	"github.com/leighmacdonald/tf-tui/ui/styles"
	zone "github.com/lrstanley/bubblezone"
)

var ErrUIExit = errors.New("ui error returned")

type contentView int

const (
	viewPlayerTables contentView = iota
	viewConfig
	viewHelp
)

// RootModel is the root model for the RootModel side of the app.
type RootModel struct {
	currentView  contentView
	previousView contentView
	quitting     bool
	height       int
	width        int
	activeTab    tabView
	consoleView  ConsoleModel
	detailPanel  DetailPanelModel
	banTable     TableBansModel
	compTable    TableCompModel
	bdTable      TableBDModel
	configModel  tea.Model
	helpModel    tea.Model
	notesModel   NotesModel
	tabsModel    tea.Model
	statusModel  tea.Model
	chatModel    ChatModel
	redTable     tea.Model
	bluTable     tea.Model

	contentViewPortHeight int
	ftrHeight             int
	hdrHeight             int
	rendered              string
}

type UI struct {
	program *tea.Program
}

func New(ctx context.Context, config config.Config, doSetup bool, buildVersion string, buildDate string, buildCommit string) *UI {
	zone.NewGlobal()
	rootModel := NewRootModel(config, doSetup, buildVersion, buildDate, buildCommit)
	program := tea.NewProgram(rootModel, tea.WithMouseCellMotion(), tea.WithAltScreen(), tea.WithContext(ctx))

	return &UI{
		program: program,
	}
}

func (t UI) Run() error {
	if _, err := t.program.Run(); err != nil {
		return errors.Join(err, ErrUIExit)
	}

	return nil
}

func (t UI) Send(msg tea.Msg) {
	t.program.Send(msg)
}

func NewRootModel(config config.Config, doSetup bool, buildVersion string, buildDate string, buildCommit string) *RootModel {
	app := &RootModel{
		currentView:  viewPlayerTables,
		previousView: viewPlayerTables,
		activeTab:    TabOverview,
		helpModel:    NewHelpModel(buildVersion, buildDate, buildCommit),
		redTable:     NewPlayerTableModel(tf.RED, config.SteamID),
		bluTable:     NewPlayerTableModel(tf.BLU, config.SteamID),
		banTable:     NewTableBansModel(),
		configModel:  NewConfigModal(config),
		compTable:    NewTableCompModel(),
		bdTable:      NewTableBDModel(),
		tabsModel:    NewTabsModel(),
		notesModel:   NewNotesModel(),
		detailPanel:  NewDetailPanelModel(config.Links),
		consoleView:  NewConsoleModel(config.ConsoleLogPath),
		statusModel:  NewStatusBarModel(buildVersion),
		chatModel:    NewChatModel(),

		contentViewPortHeight: 10,
		hdrHeight:             1,
		ftrHeight:             1,
	}

	if doSetup {
		app.currentView = viewConfig
	}

	return app
}

func (m RootModel) Init() tea.Cmd {
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
		m.bluTable.Init(), func() tea.Msg {
			return SelectedTableRowMsg{selectedTeam: tf.RED}
		})
}

func logMsg(inMsg tea.Msg) {
	// Filter out very noisy stuff
	switch inMsg.(type) {
	case ConsoleLogMsg:
		break
	case FullStateUpdateMsg:
		break
	default:
		slog.Debug("tea.Msg", slog.Any("msg", inMsg))
	}
}

func (m RootModel) Update(inMsg tea.Msg) (tea.Model, tea.Cmd) {
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
		m.contentViewPortHeight = m.height - m.hdrHeight - m.ftrHeight

		return m, func() tea.Msg {
			return ContentViewPortHeightMsg{
				contentViewPortHeight: m.contentViewPortHeight,
				height:                msg.Height,
				width:                 msg.Width,
			}
		}
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
		}

	case SetViewMsg:
		m.currentView = msg.view
	}

	return m.propagate(inMsg)
}

func (m RootModel) View() string {
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
		case TabOverview:
			ptContent = m.detailPanel.Render(lowerPanelViewportHeight)
		case TabBans:
			ptContent = m.banTable.Render(lowerPanelViewportHeight)
		case TabBD:
			ptContent = m.bdTable.Render(lowerPanelViewportHeight)
		case TabComp:
			ptContent = m.compTable.Render(lowerPanelViewportHeight)
		case TabChat:
			ptContent = m.chatModel.View(lowerPanelViewportHeight)
		case TabConsole:
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

	return zone.Scan(lipgloss.JoinVertical(lipgloss.Center, hdr, ctr, ftr))
}

func (m RootModel) isInitialized() bool {
	return m.height != 0 && m.width != 0
}

func (m RootModel) propagate(msg tea.Msg, _ ...tea.Cmd) (tea.Model, tea.Cmd) {
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
