package main

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/styles"
	zone "github.com/lrstanley/bubblezone"
)

type tabView int

const (
	TabOverview tabView = iota
	TabBans
	TabBD
	TabComp
	TabChat
	TabConsole
)

type TabLabel struct {
	label string
	tab   tabView
	id    string
}

func NewTabsModel() tea.Model {
	return &TabsModel{
		tabs: []TabLabel{
			{
				label: styles.IconInfo + " Overview",
				tab:   TabOverview,
				id:    zone.NewPrefix(),
			},
			{
				label: styles.IconBans + " Bans",
				tab:   TabBans,
				id:    zone.NewPrefix(),
			},
			{
				label: styles.IconBD + " Bot Det.",
				tab:   TabBD,
				id:    zone.NewPrefix(),
			},
			{
				label: styles.IconComp + " Comp",
				tab:   TabComp,
				id:    zone.NewPrefix(),
			},
			{
				label: styles.IconChat + " Chat",
				tab:   TabChat,
				id:    zone.NewPrefix(),
			},
			{
				label: styles.IconConsole + " Console",
				tab:   TabConsole,
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
			if zone.Get(m.id + item.label).InBounds(msg) {
				m.selectedTab = item.tab

				return m, func() tea.Msg { return TabChangeMsg(m.selectedTab) }
			}
		}

		return m, nil
	case ContentViewPortHeightMsg:
		m.width = msg.width

		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.nextTab):
			m.selectedTab++
			if m.selectedTab > TabConsole {
				m.selectedTab = TabOverview
			}
			changed = true
		case key.Matches(msg, DefaultKeyMap.prevTab):
			m.selectedTab--
			if m.selectedTab < TabOverview {
				m.selectedTab = TabConsole
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
		case key.Matches(msg, DefaultKeyMap.chat):
			m.selectedTab = TabChat
			changed = true
		}
	}

	if changed {
		return m, func() tea.Msg {
			return TabChangeMsg(m.selectedTab)
		}
	}

	return m, nil
}

func (m TabsModel) View() string {
	var tabs []string

	for _, tab := range m.tabs {
		if tab.tab == m.selectedTab {
			tabs = append(tabs, zone.Mark(m.id+tab.label, styles.TabsActive.Render(tab.label)))
		} else {
			tabs = append(tabs, zone.Mark(m.id+tab.label, styles.TabsInactive.Render(tab.label)))
		}
	}

	return styles.TabContainer.Width(m.width).Render(lipgloss.JoinHorizontal(lipgloss.Top, tabs...))
}
