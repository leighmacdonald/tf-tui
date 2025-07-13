package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/leighmacdonald/tf-tui/shared"
	"github.com/leighmacdonald/tf-tui/styles"
)

type tableModel struct {
	selectedRow  int
	selectedTeam Team // red = 3, blu = 4
	redTable     *table.Table
	bluTable     *table.Table
	redRows      int
	bluRows      int
	dump         shared.PlayerState
}

func newTableModel() *tableModel {
	model := &tableModel{selectedRow: 0, selectedTeam: RED}

	model.redTable = defaultTable(RED, model)
	model.bluTable = defaultTable(BLU, model)

	return model
}

type Direction int

const (
	Up Direction = iota
	Down
	Left
	Right
)

func (m *tableModel) moveSelection(direction Direction) {
	switch direction {
	case Up:
		if m.selectedRow > 0 {
			m.selectedRow--
		}
	case Down:
		if m.selectedRow < m.selectedColumnPlayerCount()-1 {
			m.selectedRow++
		}
	case Left:
		if m.selectedTeam != RED {
			m.selectedTeam = RED
			m.selectedRow = min(m.selectedColumnPlayerCount()-1, m.selectedRow)
		}
	case Right:
		if m.selectedTeam != BLU {
			m.selectedTeam = BLU
			m.selectedRow = min(m.selectedColumnPlayerCount()-1, m.selectedRow)
		}
	}
}

func (m *tableModel) selectedColumnPlayerCount() int {
	if m.selectedTeam == RED {
		return m.redRows
	}

	return m.bluRows
}

func (m *tableModel) View() string {
	var (
		redRows [][]string
		bluRows [][]string
	)

	for nameIdx := range shared.MaxDataSize {
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

	srt(redRows)
	srt(bluRows)

	m.redTable.ClearRows()
	m.redTable.Rows(redRows...)
	m.redRows = len(redRows)

	m.bluTable.ClearRows()
	m.bluTable.Rows(bluRows...)
	m.bluRows = len(bluRows)

	return lipgloss.JoinHorizontal(lipgloss.Top, m.redTable.Render(), m.bluTable.Render())
}

func defaultTable(team Team, parent *tableModel) *table.Table {
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
			case col == 0 && row == parent.selectedRow && team == parent.selectedTeam:
				if parent.selectedTeam == RED {
					return styles.SelectedRowStyleNameRed
				}
				return styles.SelectedRowStyleNameBlu
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

type BanTableModel struct {
	table *table.Table
}

func NewBanTableModel() *BanTableModel {
	return &BanTableModel{table: banTable()}
}

func banTable() *table.Table {
	t := table.New().
		Border(lipgloss.NormalBorder()).
		Height(20).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.Gray)).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return styles.HeaderStyleRed
			case row%2 == 0:
				return styles.EvenRowStyle
			default:
				return styles.OddRowStyle
			}
		}).
		Headers("Site", "Date", "Reason", "Perm")
	return t
}
