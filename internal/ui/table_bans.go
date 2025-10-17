package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
)

type banTableCol int

const (
	colSite banTableCol = iota
	colDate
	colPerm
	colReason
)

type banTableSize = int

const (
	colSiteSize   banTableSize = 20
	colDateSize   banTableSize = 23
	colPermSize   banTableSize = 6
	colReasonSize banTableSize = -1
)

func newTableBansModel() tableBansModel {
	return tableBansModel{table: newUnstyledTable("Site", "Date", "Perm", "Reason")}
}

type tableBansModel struct {
	table                 *table.Table
	player                Player
	width                 int
	height                int
	ready                 bool
	contentViewPortHeight int
	viewport              viewport.Model
}

func (m tableBansModel) Init() tea.Cmd {
	return nil
}

func (m tableBansModel) Update(msg tea.Msg) (tableBansModel, tea.Cmd) {
	switch msg := msg.(type) {
	case contentViewPortHeightMsg:
		m.width = msg.width
		m.height = msg.height
		if !m.ready {
			m.viewport = viewport.New(msg.width, msg.contentViewPortHeight)
			m.ready = true
		} else {
			m.contentViewPortHeight = msg.contentViewPortHeight
			m.viewport.Height = msg.contentViewPortHeight
		}
	case selectedPlayerMsg:
		m.player = msg.player
		m.table.ClearRows()

		var rows [][]string
		if m.player.Bans != nil {
			for _, ban := range m.player.Bans {
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

func (m tableBansModel) Render(height int) string {
	m.viewport.Height = height
	var content string
	if len(m.player.Bans) == 0 {
		content = styles.InfoMessage.Width(m.width).Render("No bans found " + styles.IconNoBans)
	} else {
		content = m.table.StyleFunc(func(row, col int) lipgloss.Style {
			var width int
			switch banTableCol(col) {
			case colSite:
				width = colSiteSize
			case colDate:
				width = colDateSize
			case colPerm:
				width = colPermSize
			case colReason:
				width = m.width - colSiteSize - colDateSize - colPermSize - 2
			}
			switch {
			case row == table.HeaderRow:
				return styles.BanTableHeading.Width(width)
			case row%2 == 0:
				return styles.TableRowValuesEven.Width(width)
			default:
				return styles.TableRowValuesOdd.Width(width)
			}
		}).Render()
	}

	m.viewport.SetContent(lipgloss.JoinVertical(lipgloss.Left, renderTitleBar(m.width, "Bans"), content))

	return m.viewport.View()
}
