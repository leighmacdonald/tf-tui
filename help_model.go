package main

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func NewHelpModel() HelpModel {
	return HelpModel{}
}

type HelpModel struct {
	helpView help.Model
	view     contentView
}

func (m HelpModel) Init() tea.Cmd {
	return nil
}

func (m HelpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) { //nolint:gocritic
	case SetViewMsg:
		m.view = msg.view
	}

	return m, nil
}

func (m HelpModel) View() string {
	content := m.helpView.FullHelpView([][]key.Binding{
		{DefaultKeyMap.start, DefaultKeyMap.stop, DefaultKeyMap.reset, DefaultKeyMap.quit},
		{DefaultKeyMap.help, DefaultKeyMap.accept, DefaultKeyMap.bans, DefaultKeyMap.reset},
		{DefaultKeyMap.config, DefaultKeyMap.up, DefaultKeyMap.down, DefaultKeyMap.left, DefaultKeyMap.right},
	})

	return lipgloss.Place(lipgloss.Width(content), lipgloss.Height(content), lipgloss.Center, lipgloss.Center, content)
}
