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
				tea.Printf("%s", errUID.Error())

				continue
			}

			return int(uid)
		}
	}

	return -1
}

type TableBansModel struct {
	table  *table.Table
	player Player
	width  int
	height int
}

func (m TableBansModel) Init() tea.Cmd {
	return nil
}

func (m TableBansModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case SelectedPlayerMsg:
		m.player = msg.player
		m.table.ClearRows()

		var rows [][]string
		if m.player.meta.Bans != nil {
			for _, ban := range *m.player.meta.Bans {
				perm := styles.IconCheck
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

func (m TableBansModel) View() string {
	if m.player.meta.Bans == nil || len(*m.player.meta.Bans) == 0 {
		return "No bans found " + styles.IconDrCool
	}
	return m.table.StyleFunc(func(_, col int) lipgloss.Style {
		switch col {
		case 0:
			return styles.CellStyle.Width(20)
		case 1:
			return styles.CellStyle.Width(21)
		case 2:
			return styles.CellStyle.Width(4)
		default:
			return styles.CellStyle.Width(40)
		}
	}).Width(m.width).Render()
}

func NewTableBansModel() tea.Model {
	return &TableBansModel{table: newTableDetails()}
}

func newTableDetails() *table.Table {
	return table.New().
		// Border(lipgloss.NormalBorder()).
		Height(20).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.Gray)).
		StyleFunc(func(row, _ int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return styles.HeaderStyleRed.Padding(0)
			case row%2 == 0:
				return styles.EvenRowStyle
			default:
				return styles.OddRowStyle
			}
		}).
		Headers("Site", "Date", "Perm", "Reason")
}

func sortTableRows(rows [][]string, col int) {
	slices.SortStableFunc(rows, func(a, b []string) int {
		av, _ := strconv.Atoi(a[col])
		bv, _ := strconv.Atoi(b[col])

		return cmp.Compare(bv, av)
	})
}
