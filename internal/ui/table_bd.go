package ui

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/table"
	"github.com/leighmacdonald/tf-tui/internal/tfapi"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
)

type bdTableCol int

const (
	colBDListName bdTableCol = iota
	colBDAttributes
	colBDLastName
	colBDLastSeen
	colBDProof
)

type bdTableSize int

const (
	colBDListNameSize   bdTableSize = 20
	colBDLastNameSize   bdTableSize = 20
	colBDLastSeenSize   bdTableSize = 10
	colBDAttributesSize bdTableSize = 30
	colBDProofSize      bdTableSize = -1
)

type MatchedBDPlayer struct {
	Player   tfapi.BDPlayer
	ListName string
}

func newTableBDModel() tableBDModel {
	return tableBDModel{
		table: newUnstyledTable("List Name", "Last Name", "Last Seen", "Attributes", "Proof"),
	}
}

type tableBDModel struct {
	table   *table.Table
	matched []MatchedBDPlayer
	width   int
}

func (m tableBDModel) Init() tea.Cmd {
	return nil
}

func (m tableBDModel) Update(msg tea.Msg) (tableBDModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ContentViewPortHeightMsg:
		m.width = msg.width
	case SelectedPlayerMsg:
		var rows [][]string
		// FIXME
		// for _, match := range msg.player.BDMatches {
		// 	lastSeen := time.Unix(match.Player.LastSeen.Time, 0)
		// 	rows = append(rows, []string{
		// 		match.ListName,
		// 		match.Player.LastSeen.PlayerName,
		// 		lastSeen.Format("2006-01-02"),
		// 		strings.Join(match.Player.Attributes, ", "),
		// 		strings.Join(match.Player.Proof, "\n"),
		// 	})
		// }
		m.table.ClearRows()
		m.table.Rows(rows...)
		m.table.Height(len(m.matched))
	}

	return m, nil
}

func (m tableBDModel) Render(height int) string {
	titleBar := renderTitleBar(m.width, "Bot Detector Matches")
	renderedTable := m.table.Height(height).StyleFunc(func(row, col int) lipgloss.Style {
		var width int
		switch bdTableCol(col) {
		case colBDListName:
			width = int(colBDListNameSize)
		case colBDLastSeen:
			width = int(colBDLastSeenSize)
		case colBDLastName:
			width = int(colBDLastNameSize)
		case colBDAttributes:
			width = int(colBDAttributesSize)
		case colBDProof:
			width = m.width - int(colBDListNameSize) - int(colBDLastSeenSize) - int(colBDAttributesSize) - int(colBDLastNameSize) - 2
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

	return lipgloss.JoinVertical(lipgloss.Top, titleBar, renderedTable)
}
