package main

import (
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/leighmacdonald/tf-tui/styles"
)

type TableBansModel struct {
	table                 *table.Table
	player                Player
	width                 int
	height                int
	ready                 bool
	contentViewPortHeight int
	viewport              viewport.Model
}

func (m TableBansModel) Init() tea.Cmd {
	return nil
}

func (m TableBansModel) Update(msg tea.Msg) (TableBansModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ContentViewPortHeightMsg:
		m.width = msg.width
		m.height = msg.height
		if !m.ready {
			m.viewport = viewport.New(msg.width, msg.contentViewPortHeight)
			m.ready = true
		} else {
			m.contentViewPortHeight = msg.contentViewPortHeight
			m.viewport.Height = msg.contentViewPortHeight
		}
	case SelectedPlayerMsg:
		m.player = msg.player
		m.table.ClearRows()

		var rows [][]string
		if m.player.meta.Bans != nil {
			for _, ban := range m.player.meta.Bans {
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

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

func (m TableBansModel) View(height int) string {
	m.viewport.Height = height
	var content string
	if len(m.player.meta.Bans) == 0 {
		content = "No bans found " + styles.IconDrCool
	} else {
		content = m.table.StyleFunc(func(_, col int) lipgloss.Style {
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

	m.viewport.SetContent(lipgloss.JoinVertical(lipgloss.Left, renderTitleBar(m.width, "Bans"), content))

	return m.viewport.View()
}

func NewTableBansModel() TableBansModel {
	return TableBansModel{table: newTableDetails()}
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
