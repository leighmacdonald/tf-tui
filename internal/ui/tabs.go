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
	tabServers tabView = iota
	tabPlayers
	tabBans
	tabBD
	tabComp
	tabChat
	tabConsole
)

type tabLabel struct {
	label  string
	tab    tabView
	zoneID string
}

func newTabsModel() tea.Model {
	return &tabsModel{
		tabs: []tabLabel{
			{
				label:  "Servers",
				tab:    tabServers,
				zoneID: zone.NewPrefix(),
			},
			{
				label:  "Players",
				tab:    tabPlayers,
				zoneID: zone.NewPrefix(),
			},
			{
				label:  "Bans",
				tab:    tabBans,
				zoneID: zone.NewPrefix(),
			},
			{
				label:  "Bot Det.",
				tab:    tabBD,
				zoneID: zone.NewPrefix(),
			},
			{
				label:  "Comp",
				tab:    tabComp,
				zoneID: zone.NewPrefix(),
			},
			{
				label:  "Chat",
				tab:    tabChat,
				zoneID: zone.NewPrefix(),
			},
			{
				label:  "Console",
				tab:    tabConsole,
				zoneID: zone.NewPrefix(),
			},
		},
		selectedTab: tabServers,
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
	case viewPortSizeMsg:
		m.width = msg.width

		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, defaultKeyMap.nextTab):
			m.selectedTab++
			if m.selectedTab > tabConsole {
				m.selectedTab = tabServers
			}
			changed = true
		case key.Matches(msg, defaultKeyMap.prevTab):
			m.selectedTab--
			if m.selectedTab < tabServers {
				m.selectedTab = tabConsole
			}
			changed = true
		case key.Matches(msg, defaultKeyMap.overview):
			m.selectedTab = tabServers
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
	if m.width == 0 {
		return ""
	}
	var tabs []string

	for _, tab := range m.tabs {
		if tab.tab == m.selectedTab {
			tabs = append(tabs, zone.Mark(m.id+tab.label, styles.TabsActive.Render(tab.label)))
		} else {
			tabs = append(tabs, zone.Mark(m.id+tab.label, styles.TabsInactive.Render(tab.label)))
		}
	}

	return styles.WrapX(m.width, styles.TabContainer.Width(m.width).Render(lipgloss.JoinHorizontal(lipgloss.Top, tabs...)), "x")
}
