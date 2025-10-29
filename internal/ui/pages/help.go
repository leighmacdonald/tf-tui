package pages

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/internal/ui/command"
	"github.com/leighmacdonald/tf-tui/internal/ui/input"
	"github.com/leighmacdonald/tf-tui/internal/ui/model"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
)

func NewHelp(buildVersion, buildDate, buildCommit string, configPath string, cachePath string) Help {
	return Help{
		configPath:   configPath,
		cachePath:    cachePath,
		buildVersion: buildVersion,
		buildDate:    buildDate,
		buildCommit:  buildCommit,
	}
}

type Help struct {
	helpView     help.Model
	viewState    model.ViewState
	configPath   string
	cachePath    string
	buildVersion string
	buildDate    string
	buildCommit  string
}

func (m Help) Init() tea.Cmd {
	return nil
}

func (m Help) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch { //nolint:gocritic
		case key.Matches(msg, input.Default.Back):
			// go back to main view
			if m.viewState.Page == model.PageHelp {
				m.viewState.Page = model.PageMain

				return m, command.SetViewState(m.viewState)
			}
		}
	case model.Page:
		m.viewState.Page = msg
	}

	return m, nil
}

func (m Help) View() string {
	left := m.helpView.FullHelpView([][]key.Binding{
		{
			input.Default.Config,
			input.Default.Start,
			input.Default.Stop,
			input.Default.Reset,
			input.Default.Quit,
			input.Default.Help,
			input.Default.Accept,
		},
	})

	middle := m.helpView.FullHelpView([][]key.Binding{
		{
			input.Default.Overview,
			input.Default.Bans,
			input.Default.BD,
			input.Default.Comp,
			input.Default.Chat,
			input.Default.Console,
		},
	})

	right := m.helpView.FullHelpView([][]key.Binding{
		{
			input.Default.NextTab,
			input.Default.Up,
			input.Default.Down,
			input.Default.Left,
			input.Default.Right,
		},
	})

	helpContent := lipgloss.JoinHorizontal(lipgloss.Top,
		styles.HelpBox.Render(left), styles.HelpBox.Render(middle), styles.HelpBox.Render(right))

	commit := m.buildCommit
	//goland:noinspection GoBoolExpressions
	if len(commit) > 8 {
		commit = m.buildCommit[0:8]
	}

	content := lipgloss.JoinVertical(lipgloss.Center, helpContent,
		styles.DetailRow("Version", m.buildVersion),
		styles.DetailRow("Commit", commit),
		styles.DetailRow("Date", m.buildDate),
		styles.DetailRow("Config Path", m.configPath),
		styles.DetailRow("Cache Path", m.cachePath),
	)

	return lipgloss.Place(lipgloss.Width(content), lipgloss.Height(content),
		lipgloss.Center, lipgloss.Center, content)
}
