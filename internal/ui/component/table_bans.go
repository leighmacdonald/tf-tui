package component

import (
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/leighmacdonald/tf-tui/internal/ui/command"
	"github.com/leighmacdonald/tf-tui/internal/ui/model"
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

func NewTableBansModel() TableBansModel {
	return TableBansModel{table: NewUnstyledTable("Site", "Date", "Perm", "Reason")}
}

type TableBansModel struct {
	table     *table.Table
	player    model.Player
	ready     bool
	viewport  viewport.Model
	viewState model.ViewState
}

func (m TableBansModel) Init() tea.Cmd {
	return nil
}

func (m TableBansModel) Update(msg tea.Msg) (TableBansModel, tea.Cmd) {
	switch msg := msg.(type) {
	case model.ViewState:
		m.viewState = msg
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Lower-2)
			m.ready = true
		} else {
			m.viewport.Height = msg.Lower - 2
		}
	case command.SelectedPlayerMsg:
		m.player = msg.Player
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

func (m TableBansModel) Render(height int) string {
	m.viewport.Height = height - 2
	var content string
	if len(m.player.Bans) == 0 {
		content = styles.InfoMessage.Width(m.viewState.Width).Render("No bans found " + styles.IconNoBans)
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
				width = m.viewState.Width - colSiteSize - colDateSize - colPermSize - 2
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

	m.viewport.SetContent(content)

	return Container("Bans", m.viewState.Width, height, m.viewport.View(), false)
}
