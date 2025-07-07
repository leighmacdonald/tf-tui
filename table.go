package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/leighmacdonald/tf-tui/styles"
)

type tableModel struct {
	selectedRow  int
	selectedTeam Team // red = 3, blu = 4
	redTable     *table.Table
	bluTable     *table.Table
	dump         *DumpPlayer
}

func newTableModel() *tableModel {
	return &tableModel{
		redTable: defaultTable(RED),
		bluTable: defaultTable(BLU),
	}
}

func (m tableModel) render() string {
	var (
		redRows [][]string
		bluRows [][]string
	)

	if m.dump != nil {
		for nameIdx := range maxDataSize {
			if !m.dump.SteamID[nameIdx].Valid() {
				continue
			}

			row := []string{
				m.dump.Names[nameIdx],
				fmt.Sprintf("%d", m.dump.Score[nameIdx]),
				fmt.Sprintf("%d", m.dump.Deaths[nameIdx]),
				fmt.Sprintf("%d", m.dump.Ping[nameIdx]),
			}

			switch m.dump.Team[nameIdx] {
			case 2:
				redRows = append(redRows, row)
			case 3:
				bluRows = append(bluRows, row)
			}
		}
	}

	srt(redRows)
	srt(bluRows)

	m.redTable.ClearRows()
	m.redTable.Rows(redRows...)

	m.bluTable.ClearRows()
	m.bluTable.Rows(bluRows...)

	return lipgloss.JoinHorizontal(lipgloss.Top, m.redTable.Render(), m.bluTable.Render())
}

func defaultTable(team Team) *table.Table {
	border := styles.Blu
	if team == RED {
		border = styles.Red
	}

	return table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(border)).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				if team == RED {
					if col == 0 {
						return styles.HeaderStyleRed.Width(30)
					}
					return styles.HeaderStyleRed
				}
				if col == 0 {
					return styles.HeaderStyleBlu.Width(30)
				}
				return styles.HeaderStyleBlu
			case col == 0:
				return styles.OddRowStyleName
			case row%2 == 0:
				return styles.EvenRowStyle

			default:
				return styles.OddRowStyle
			}
		}).
		Headers("Name", "Score", "Deaths", "Ping")
}
