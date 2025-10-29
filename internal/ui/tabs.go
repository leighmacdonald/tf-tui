package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
	zone "github.com/lrstanley/bubblezone"
)

type section int

const (
	tabServers section = iota
	tabPlayers
	tabBans
	tabBD
	tabComp
	tabChat
	tabConsole
)

type tabLabel struct {
	label  string
	tab    section
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
		viewState: viewState{section: tabServers},
	}
}

type tabsModel struct {
	tabs      []tabLabel
	viewState viewState
	id        string
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
				vs := m.viewState
				vs.section = item.tab

				return m, setViewState(vs)
			}
		}

		return m, nil
	case viewState:
		m.viewState = msg

		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, defaultKeyMap.nextTab):
			m.viewState.section++
			if m.viewState.section > tabConsole {
				m.viewState.section = tabServers
			}
			changed = true
		case key.Matches(msg, defaultKeyMap.prevTab):
			m.viewState.section--
			if m.viewState.section < tabServers {
				m.viewState.section = tabConsole
			}
			changed = true
		case key.Matches(msg, defaultKeyMap.overview):
			m.viewState.section = tabServers
			changed = true
		case key.Matches(msg, defaultKeyMap.bans):
			m.viewState.section = tabBans
			changed = true
		case key.Matches(msg, defaultKeyMap.comp):
			m.viewState.section = tabComp
			changed = true
		case key.Matches(msg, defaultKeyMap.chat):
			m.viewState.section = tabChat
			changed = true
		}
	}

	if changed {
		return m, setViewState(m.viewState)
	}

	return m, nil
}

func (m tabsModel) View() string {
	if m.viewState.width == 0 {
		return ""
	}
	var tabs []string

	for _, tab := range m.tabs {
		if tab.tab == m.viewState.section {
			tabs = append(tabs, zone.Mark(m.id+tab.label, styles.TabsActive.Render(tab.label)))
		} else {
			tabs = append(tabs, zone.Mark(m.id+tab.label, styles.TabsInactive.Render(tab.label)))
		}
	}

	return styles.WrapX(m.viewState.width, styles.TabContainer.Width(m.viewState.width).Render(lipgloss.JoinHorizontal(lipgloss.Top, tabs...)), "x")
}
