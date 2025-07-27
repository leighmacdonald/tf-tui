package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/leighmacdonald/tf-tui/styles"
)

type TableComp struct {
	player Player
	table  *table.Table
	width  int
	height int
}

func NewTableCompModel() tea.Model {
	return &TableComp{table: table.New().
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

func (m TableComp) Init() tea.Cmd {
	return nil
}

func (m TableComp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
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

func (m TableComp) View() string {
	if len(m.player.meta.CompetitiveTeams) == 0 {
		return "No league history found"
	}

	return m.table.Width(m.width).Render()
}
