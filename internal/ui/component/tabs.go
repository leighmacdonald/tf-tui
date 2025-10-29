package component

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/internal/ui/command"
	"github.com/leighmacdonald/tf-tui/internal/ui/input"
	"github.com/leighmacdonald/tf-tui/internal/ui/model"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
	zone "github.com/lrstanley/bubblezone"
)

type TabLabel struct {
	label  string
	tab    model.Section
	zoneID string
}

func NewTabsModel() tea.Model {
	return &TabsModel{
		tabs: []TabLabel{
			{
				label:  "Servers",
				tab:    model.SectionServers,
				zoneID: zone.NewPrefix(),
			},
			{
				label:  "Players",
				tab:    model.SectionPlayers,
				zoneID: zone.NewPrefix(),
			},
			{
				label:  "Bans",
				tab:    model.SectionBans,
				zoneID: zone.NewPrefix(),
			},
			{
				label:  "Bot Det.",
				tab:    model.SectionBD,
				zoneID: zone.NewPrefix(),
			},
			{
				label:  "Comp",
				tab:    model.SectionComp,
				zoneID: zone.NewPrefix(),
			},
			{
				label:  "Chat",
				tab:    model.SectionChat,
				zoneID: zone.NewPrefix(),
			},
			{
				label:  "Console",
				tab:    model.SectionConsole,
				zoneID: zone.NewPrefix(),
			},
		},
		viewState: model.ViewState{Section: model.SectionServers},
	}
}

type TabsModel struct {
	tabs      []TabLabel
	viewState model.ViewState
	id        string
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
				vs := m.viewState
				vs.Section = item.tab

				return m, command.SetViewState(vs)
			}
		}

		return m, nil
	case model.ViewState:
		m.viewState = msg

		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, input.Default.NextTab):
			m.viewState.Section++
			if m.viewState.Section > model.SectionConsole {
				m.viewState.Section = model.SectionServers
			}
			changed = true
		case key.Matches(msg, input.Default.PrevTab):
			m.viewState.Section--
			if m.viewState.Section < model.SectionServers {
				m.viewState.Section = model.SectionConsole
			}
			changed = true
		case key.Matches(msg, input.Default.Overview):
			m.viewState.Section = model.SectionServers
			changed = true
		case key.Matches(msg, input.Default.Bans):
			m.viewState.Section = model.SectionBans
			changed = true
		case key.Matches(msg, input.Default.Comp):
			m.viewState.Section = model.SectionComp
			changed = true
		case key.Matches(msg, input.Default.Chat):
			m.viewState.Section = model.SectionChat
			changed = true
		}
	}

	if changed {
		return m, command.SetViewState(m.viewState)
	}

	return m, nil
}

func (m TabsModel) View() string {
	if m.viewState.Width == 0 {
		return ""
	}
	var tabs []string

	for _, tab := range m.tabs {
		if tab.tab == m.viewState.Section {
			tabs = append(tabs, zone.Mark(m.id+tab.label, styles.TabsActive.Render(tab.label)))
		} else {
			tabs = append(tabs, zone.Mark(m.id+tab.label, styles.TabsInactive.Render(tab.label)))
		}
	}

	return styles.WrapX(m.viewState.Width, styles.TabContainer.Width(m.viewState.Width).Render(lipgloss.JoinHorizontal(lipgloss.Top, tabs...)), "x")
}
