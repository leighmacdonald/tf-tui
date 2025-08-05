package main

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/leighmacdonald/tf-tui/styles"
)

type TableCompModel struct {
	player   Player
	table    *table.Table
	width    int
	height   int
	ready    bool
	viewport viewport.Model
}

func NewTableCompModel() *TableCompModel {
	return &TableCompModel{table: table.New().
		// Border(lipgloss.NormalBorder()).
		Height(20).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.Gray)).
		StyleFunc(func(row, col int) lipgloss.Style {
			width := 10
			switch col {
			case 1:
				width = 40
			case 2:
				width = 16
			case 3:
				width = 20
			case 4:
				width = 30
			}
			switch {
			case row == table.HeaderRow:
				return styles.HeaderStyleRed.Padding(0)
			case row%2 == 0:
				return styles.EvenRowStyle.Width(width)
			default:
				return styles.OddRowStyle.Width(width)
			}
		}).
		Headers("League", "Competition", "format", "Division", "Team Name")}
}

func (m *TableCompModel) Init() tea.Cmd {
	return nil
}

func (m *TableCompModel) Update(msg tea.Msg) (*TableCompModel, tea.Cmd) { //nolint:unparam
	switch msg := msg.(type) {
	case ContentViewPortHeightMsg:
		m.width = msg.width
		m.height = msg.height
		if !m.ready {
			m.viewport = viewport.New(msg.width, msg.contentViewPortHeight)
			m.ready = true
		} else {
			m.viewport.Height = msg.contentViewPortHeight
		}
	case SelectedPlayerMsg:
		m.player = msg.player
		m.table.ClearRows()

		var rows [][]string
		if m.player.meta.CompetitiveTeams != nil {
			for _, team := range m.player.meta.CompetitiveTeams {
				rows = append(rows, []string{
					team.League,
					team.SeasonName,
					team.Format,
					team.DivisionName,
					team.TeamName,
				})
			}
		}

		m.table.Rows(rows...)
	}

	return m, nil
}

func (m *TableCompModel) View(height int) string {
	titlebar := renderTitleBar(m.width, "League History")
	var content string
	if len(m.player.meta.CompetitiveTeams) == 0 {
		content = "No league history found"
	} else {
		m.table.Width(m.width).Render()
	}
	m.viewport.SetContent(content)
	m.viewport.Height = height - lipgloss.Height(titlebar)

	return lipgloss.JoinVertical(lipgloss.Left, titlebar, m.viewport.View())
}
