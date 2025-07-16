package main

import (
	"cmp"
	"slices"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/leighmacdonald/tf-tui/styles"
)

func findCurrentUID(selected int, rows [][]string) int {
	for idx, row := range rows {
		if idx == selected {
			uid, errUID := strconv.ParseInt(row[0], 10, 32)
			if errUID != nil {
				tea.Printf(errUID.Error())
				continue
			}

			return int(uid)
		}
	}

	return -1
}

type TableDetailModel struct {
	table  *table.Table
	player Player
}

func (m TableDetailModel) Init() tea.Cmd {
	return nil
}

func (m TableDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SelectedPlayerMsg:
		m.player = msg.player
		m.table.ClearRows()

		var rows [][]string
		if m.player.meta.Bans != nil {
			for _, ban := range *m.player.meta.Bans {
				perm := "✅"
				if !ban.Permanent {
					perm = ""
				}
				rows = append(rows, []string{
					ban.SiteName,
					ban.CreatedOn.Format(time.DateTime),
					perm,
					ban.Reason,
				})
			}
		}

		m.table.Rows(rows...)
	}
	return m, nil
}

func (m TableDetailModel) View() string {
	return m.table.StyleFunc(func(row, col int) lipgloss.Style {
		switch col {
		case 0:
			return styles.CellStyle.Width(20)
		case 1:
			return styles.CellStyle.Width(21)
		case 2:
			return styles.CellStyle.Width(4)
		default:
			return styles.CellStyle.Width(60)
		}
	}).Render()
}

func NewTableDetailModel() tea.Model {
	return &TableDetailModel{table: newTableDetails()}
}

func newTableDetails() *table.Table {
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
		Headers("Site", "Date", "Perm", "Reason")
	return t
}

func sortTableRows(rows [][]string, col int) {
	slices.SortStableFunc(rows, func(a, b []string) int {
		av, _ := strconv.Atoi(a[col])
		bv, _ := strconv.Atoi(b[col])
		return cmp.Compare(bv, av)
	})
}
