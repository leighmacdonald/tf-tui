package ui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
)

func newHelpModel(buildVersion, buildDate, buildCommit string, configPath string, cachePath string) helpModel {
	return helpModel{
		configPath:   configPath,
		cachePath:    cachePath,
		buildVersion: buildVersion,
		buildDate:    buildDate,
		buildCommit:  buildCommit,
	}
}

type helpModel struct {
	helpView     help.Model
	viewState    viewState
	configPath   string
	cachePath    string
	buildVersion string
	buildDate    string
	buildCommit  string
}

func (m helpModel) Init() tea.Cmd {
	return nil
}

func (m helpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch { //nolint:gocritic
		case key.Matches(msg, defaultKeyMap.back):
			// go back to main view
			if m.viewState.page == pageHelp {
				m.viewState.page = pageMain

				return m, setViewStateStruct(m.viewState)
			}
		}
	case page:
		m.viewState.page = msg
	}

	return m, nil
}

func (m helpModel) View() string {
	left := m.helpView.FullHelpView([][]key.Binding{
		{
			defaultKeyMap.config,
			defaultKeyMap.start,
			defaultKeyMap.stop,
			defaultKeyMap.reset,
			defaultKeyMap.quit,
			defaultKeyMap.help,
			defaultKeyMap.accept,
		},
	})

	middle := m.helpView.FullHelpView([][]key.Binding{
		{
			defaultKeyMap.overview,
			defaultKeyMap.bans,
			defaultKeyMap.bd,
			defaultKeyMap.comp,
			defaultKeyMap.chat,
			defaultKeyMap.console,
		},
	})

	right := m.helpView.FullHelpView([][]key.Binding{
		{
			defaultKeyMap.nextTab,
			defaultKeyMap.up,
			defaultKeyMap.down,
			defaultKeyMap.left,
			defaultKeyMap.right,
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
