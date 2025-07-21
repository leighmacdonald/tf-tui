package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leighmacdonald/tf-tui/styles"
	zone "github.com/lrstanley/bubblezone"
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
	id    string
}

func NewTabsModel() tea.Model {
	return &TabsModel{
		//id: zone.NewPrefix(),
		tabs: []TabLabel{
			{
				label: "[o]verview",
				tab:   TabOverview,
				id:    zone.NewPrefix(),
			},
			{
				label: "[b]ans",
				tab:   TabBans,
				id:    zone.NewPrefix(),
			},
			{
				label: "[c]omp",
				tab:   TabComp,
				id:    zone.NewPrefix(),
			},
			{
				label: "[n]otes",
				tab:   TabNotes,
				id:    zone.NewPrefix(),
			},
		},
		selectedTab: TabOverview,
	}
}

type TabsModel struct {
	tabs        []TabLabel
	selectedTab tabView
	width       int
	id          string
}

func (m TabsModel) Init() tea.Cmd {
	return nil
}

func (m TabsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	changed := false
	switch msg := msg.(type) {
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionRelease || msg.Button != tea.MouseButtonLeft {
			return m, nil
		}
		for _, item := range m.tabs {
			// Check each item to see if it's in bounds.
			if zone.Get(item.id).InBounds(msg) {
				m.selectedTab = item.tab

				break
			}
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
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
			builder.WriteString(zone.Mark(tab.id, styles.TabsActive.Render(tab.label)))
		} else {
			builder.WriteString(zone.Mark(tab.id, styles.TabsInactive.Render(tab.label)))
		}
	}

	return styles.TabContainer.Width(m.width).Render(builder.String())
}
