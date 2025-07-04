package main

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/leighmacdonald/tf-tui/styles"
)

func newPlayerTable(rows [][]string, isRed bool, selectedRow int, selectionActive bool) *table.Table {
	border := styles.Blu
	if isRed {
		border = styles.Red
	}

	return table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(border)).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				if isRed == true {
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
		Headers("Name", "Score", "Deaths", "Ping").
		Rows(rows...)
}
