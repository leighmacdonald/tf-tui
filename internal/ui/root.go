package ui

import (
	"context"
	"errors"
	"log/slog"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/tf/events"
	"github.com/leighmacdonald/tf-tui/internal/ui/command"
	"github.com/leighmacdonald/tf-tui/internal/ui/component"
	"github.com/leighmacdonald/tf-tui/internal/ui/input"
	"github.com/leighmacdonald/tf-tui/internal/ui/model"
	"github.com/leighmacdonald/tf-tui/internal/ui/pages"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
	zone "github.com/lrstanley/bubblezone"
)

var ErrUIExit = errors.New("ui error returned")

type ConfigWriter interface {
	Write(config.Config) error
	Path() string
}

type UI struct {
	program *tea.Program
}

func New(ctx context.Context, config config.Config, doSetup bool, buildVersion string, buildDate string, buildCommit string,
	loader config.Writer, cachePath string, parentCtx chan any) *UI {
	zone.NewGlobal()

	return &UI{
		program: tea.NewProgram(
			NewRootModel(
				config,
				doSetup,
				buildVersion,
				buildDate,
				buildCommit,
				loader,
				cachePath,
				parentCtx),
			tea.WithMouseCellMotion(),
			tea.WithAltScreen(),
			tea.WithMouseAllMotion(),
			tea.WithContext(ctx),
			tea.WithFPS(30)),
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

// RootModel is the top level model for the ui side of the app.
type RootModel struct {
	viewState             model.ViewState
	viewStatePreviousView model.ViewState
	mainPage              tea.Model
	configPage            tea.Model
	helpPage              tea.Model
	statusBar             tea.Model
	serverMode            bool
}

func NewRootModel(userConfig config.Config, doSetup bool, buildVersion string, buildDate string, buildCommit string, loader config.Writer, cachePath string, parentChan chan any) *RootModel {
	var view model.ViewState
	if userConfig.ServerModeEnabled {
		view = model.ViewState{Page: model.PageMain, Section: model.SectionServers, KeyZone: model.KZserverTable}
	} else {
		view = model.ViewState{Page: model.PageMain, Section: model.SectionPlayers, KeyZone: model.KZplayerTableRED}
	}

	app := &RootModel{
		viewState:             view,
		viewStatePreviousView: view,
		helpPage:              pages.NewHelp(buildVersion, buildDate, buildCommit, loader.Path(), cachePath),
		configPage:            pages.NewConfig(userConfig, loader),
		mainPage:              pages.NewMain(userConfig),
		serverMode:            userConfig.ServerModeEnabled,
		statusBar:             component.NewStatusBarModel(buildVersion, userConfig.ServerModeEnabled),
	}

	if doSetup {
		app.viewState.Page = model.PageConfig
	}

	return app
}

func (m RootModel) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("tf-tui"),
		m.configPage.Init(),
		m.helpPage.Init(),
		m.mainPage.Init(),
		m.statusBar.Init(),
		textinput.Blink,
		command.SelectTeam(tf.RED),
	)
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
		_, ftrHeight := lipgloss.Size(m.renderFooter())
		vs := m.viewState
		vs.Height = msg.Height
		vs.Width = msg.Width
		vs.Upper = (msg.Height - ftrHeight) / 2
		vs.Lower = vs.Upper
		if vs.Upper%2 != 0 {
			vs.Lower -= 1
		}
		m.viewState = vs

		return m, command.SetViewState(vs)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, input.Default.Quit):
			if m.viewState.Page != model.PageMain {
				break
			}

			return m, tea.Quit
		case key.Matches(msg, input.Default.Help):
			if m.viewState.Page == model.PageHelp {
				m.viewState.Page = m.viewStatePreviousView.Page
			} else {
				m.viewStatePreviousView.Page = m.viewState.Page
				m.viewState.Page = model.PageHelp
			}
		case key.Matches(msg, input.Default.Config):
			if m.viewState.Page == model.PageConfig {
				m.viewState.Page = m.viewStatePreviousView.Page
			} else {
				m.viewStatePreviousView.Page = m.viewState.Page
				m.viewState.Page = model.PageConfig
			}
		case key.Matches(msg, input.Default.NextTab):
			return m, command.SetNextZone(m.viewState.Section, m.viewState.KeyZone, input.Right)
		case key.Matches(msg, input.Default.PrevTab):
			return m, command.SetNextZone(m.viewState.Section, m.viewState.KeyZone, input.Left)
		}
	case model.ViewState:
		m.viewState = msg
	}

	return m.propagate(inMsg)
}

func (m RootModel) propagate(msg tea.Msg, _ ...tea.Cmd) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 4)
	m.helpPage, cmds[0] = m.helpPage.Update(msg)
	m.configPage, cmds[1] = m.configPage.Update(msg)
	m.mainPage, cmds[2] = m.mainPage.Update(msg)
	m.statusBar, cmds[3] = m.statusBar.Update(msg)

	return m, tea.Batch(cmds...)
}

func (m RootModel) View() string {
	var content string

	switch m.viewState.Page {
	case model.PageConfig:
		content = m.configPage.View()
	case model.PageHelp:
		content = m.helpPage.View()
	case model.PageMain:
		content = m.mainPage.View()
	}

	return zone.Scan(lipgloss.JoinVertical(lipgloss.Left, content, m.renderFooter()))
}

func (m RootModel) renderFooter() string {
	// Early so we can use their size info
	footer := styles.FooterContainerStyle.
		Width(m.viewState.Width).
		Render(lipgloss.JoinVertical(lipgloss.Top, m.statusBar.View()))

	return styles.FooterContainerStyle.Width(m.viewState.Width).Render(footer)
}

func (m RootModel) isInitialized() bool {
	return m.viewState.Height != 0 && m.viewState.Width != 0
}

// logMsg is useful for debugging events. Tail the log file ~/.config/tf-tui/tf-tui.log
func logMsg(inMsg tea.Msg) {
	// Filter out very noisy stuff
	switch inMsg.(type) {
	case command.SelectServerSnapshotMsg:
	case component.LogRow:
		break
	case events.Event:
		break
	case model.Players:
		break
	case model.Snapshot:
		break
	case []model.Snapshot:
		break
	default:
		slog.Debug("tea.Msg", slog.Any("msg", inMsg))
	}
}
