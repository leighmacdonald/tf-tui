package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
	zone "github.com/lrstanley/bubblezone"
)

type tabView int

const (
	tabOverview tabView = iota
	tabBans
	tabBD
	tabComp
	tabChat
	tabConsole
)

type tabLabel struct {
	label string
	tab   tabView
	id    string
}

func newTabsModel() tea.Model {
	return &tabsModel{
		tabs: []tabLabel{
			{
				label: styles.IconInfo + " Overview",
				tab:   tabOverview,
				id:    zone.NewPrefix(),
			},
			{
				label: styles.IconBans + " Bans",
				tab:   tabBans,
				id:    zone.NewPrefix(),
			},
			{
				label: styles.IconBD + " Bot Det.",
				tab:   tabBD,
				id:    zone.NewPrefix(),
			},
			{
				label: styles.IconComp + " Comp",
				tab:   tabComp,
				id:    zone.NewPrefix(),
			},
			{
				label: styles.IconChat + " Chat",
				tab:   tabChat,
				id:    zone.NewPrefix(),
			},
			{
				label: styles.IconConsole + " Console",
				tab:   tabConsole,
				id:    zone.NewPrefix(),
			},
		},
		selectedTab: tabOverview,
	}
}

type tabsModel struct {
	tabs        []tabLabel
	selectedTab tabView
	width       int
	id          string
}

func (m tabsModel) Init() tea.Cmd {
	return nil
}

func (m tabsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

				return m, setTab(m.selectedTab)
			}
		}

		return m, nil
	case contentViewPortHeightMsg:
		m.width = msg.width

		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, defaultKeyMap.nextTab):
			m.selectedTab++
			if m.selectedTab > tabConsole {
				m.selectedTab = tabOverview
			}
			changed = true
		case key.Matches(msg, defaultKeyMap.prevTab):
			m.selectedTab--
			if m.selectedTab < tabOverview {
				m.selectedTab = tabConsole
			}
			changed = true
		case key.Matches(msg, defaultKeyMap.overview):
			m.selectedTab = tabOverview
			changed = true
		case key.Matches(msg, defaultKeyMap.bans):
			m.selectedTab = tabBans
			changed = true
		case key.Matches(msg, defaultKeyMap.comp):
			m.selectedTab = tabComp
			changed = true
		case key.Matches(msg, defaultKeyMap.chat):
			m.selectedTab = tabChat
			changed = true
		}
	}

	if changed {
		return m, setTab(m.selectedTab)
	}

	return m, nil
}

func (m tabsModel) View() string {
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
