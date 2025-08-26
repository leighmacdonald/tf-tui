package ui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
)

func newHelpModel(buildVersion, buildDate, buildCommit string) helpModel {
	return helpModel{
		configPath:   config.PathConfig(config.DefaultConfigName),
		cachePath:    config.PathCache(config.CacheDirName),
		buildVersion: buildVersion,
		buildDate:    buildDate,
		buildCommit:  buildCommit,
	}
}

type helpModel struct {
	helpView     help.Model
	view         contentView
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
		case key.Matches(msg, DefaultKeyMap.back):
			// go back to main view
			if m.view == viewHelp {
				m.view = viewPlayerTables

				return m, setContentView(viewPlayerTables)
			}
		}
	case SetViewMsg:
		m.view = msg.view
	}

	return m, nil
}

func (m helpModel) View() string {
	left := m.helpView.FullHelpView([][]key.Binding{
		{
			DefaultKeyMap.config,
			DefaultKeyMap.start,
			DefaultKeyMap.stop,
			DefaultKeyMap.reset,
			DefaultKeyMap.quit,
			DefaultKeyMap.help,
			DefaultKeyMap.accept,
		},
	})

	middle := m.helpView.FullHelpView([][]key.Binding{
		{
			DefaultKeyMap.overview,
			DefaultKeyMap.bans,
			DefaultKeyMap.bd,
			DefaultKeyMap.comp,
			DefaultKeyMap.chat,
			DefaultKeyMap.console,
		},
	})

	right := m.helpView.FullHelpView([][]key.Binding{
		{
			DefaultKeyMap.nextTab,
			DefaultKeyMap.up,
			DefaultKeyMap.down,
			DefaultKeyMap.left,
			DefaultKeyMap.right,
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
