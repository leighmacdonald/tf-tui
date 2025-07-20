package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leighmacdonald/tf-tui/styles"
)

type tabView int

const (
	TabOverview tabView = iota
	TabBans
	TabComp
	TabNotes
)

type TabLabel struct {
	label string
	tab   tabView
}

func newTabsModel() tea.Model {
	return TabsModel{
		tabs: []TabLabel{
			{
				label: "[o]verview",
				tab:   TabOverview,
			},
			{
				label: "[b]ans",
				tab:   TabBans,
			},
			{
				label: "[c]omp",
				tab:   TabComp,
			},
			{
				label: "[n]otes",
				tab:   TabNotes,
			},
		},
		selectedTab: TabOverview,
	}
}

type TabsModel struct {
	tabs        []TabLabel
	selectedTab tabView
}

func (m TabsModel) Init() tea.Cmd {
	return nil
}

func (m TabsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	changed := false
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.nextTab):
			m.selectedTab++
			if m.selectedTab > TabNotes {
				m.selectedTab = TabOverview
			}
			changed = true
		case key.Matches(msg, DefaultKeyMap.overview):
			m.selectedTab = TabOverview
			changed = true
		case key.Matches(msg, DefaultKeyMap.bans):
			m.selectedTab = TabBans
			changed = true
		case key.Matches(msg, DefaultKeyMap.comp):
			m.selectedTab = TabComp
			changed = true
		case key.Matches(msg, DefaultKeyMap.notes):
			m.selectedTab = TabNotes
			changed = true
		}
	}

	if changed {
		return m, func() tea.Msg { return TabChangeMsg(m.selectedTab) }
	}

	return m, nil
}

func (m TabsModel) View() string {
	var builder strings.Builder
	builder.WriteString("\n")

	for _, tab := range m.tabs {
		if tab.tab == m.selectedTab {
			builder.WriteString(styles.TabsActive.Render(tab.label))
		} else {
			builder.WriteString(styles.TabsInactive.Render(tab.label))
		}
	}

	return builder.String()
}
