package main

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/styles"
	zone "github.com/lrstanley/bubblezone"
)

type Direction int

const (
	Up Direction = iota //nolint:varnamelen
	Down
	Left
	Right
)

type PlayerTablesModel struct {
	id           string
	selectedRow  int
	selectedTeam Team // red = 3, blu = 4
	redTable     tea.Model
	bluTable     tea.Model
	redRows      [][]string
	bluRows      [][]string
	selectedUID  int
	height       int
	width        int
}

func (m PlayerTablesModel) Init() tea.Cmd {
	return tea.Batch(
		m.redTable.Init(),
		m.bluTable.Init(),
		func() tea.Msg {
			return SelectedTableRowMsg{selectedTeam: RED}
		})
}

func NewTablePlayersModel() *PlayerTablesModel {
	model := &PlayerTablesModel{
		selectedRow:  0,
		selectedTeam: RED,
		id:           zone.NewPrefix(),
		redTable:     NewPlayerTableModel(RED),
		bluTable:     NewPlayerTableModel(BLU),
	}

	return model
}

func (m PlayerTablesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 2)
	m.redTable, cmds[0] = m.redTable.Update(msg)
	m.bluTable, cmds[1] = m.bluTable.Update(msg)

	return m, tea.Batch(cmds...)
}

func (m PlayerTablesModel) selectedColumnPlayerCount() int {
	if m.selectedTeam == RED {
		if tm, ok := m.redTable.(PlayerTableModel); ok {
			return tm.data.Rows()
		}

		return 0
	}

	if tm, ok := m.bluTable.(PlayerTableModel); ok {
		return tm.data.Rows()
	}

	return 0
}

func (m PlayerTablesModel) View() string {
	return lipgloss.JoinHorizontal(lipgloss.Top, m.redTable.View(), m.bluTable.View())
}

func NewPlayerTableModel(team Team) *PlayerTableModel {
	foreground := styles.Red
	if team == BLU {
		foreground = styles.Blu
	}
	data := NewPlayerTableData([]Player{}, team)

	return &PlayerTableModel{
		id:   zone.NewPrefix(),
		team: team,
		data: &data,
		table: table.New().
			BorderStyle(lipgloss.NewStyle().Foreground(foreground)).
			BorderHeader(false),
	}
}

type PlayerTableModel struct {
	id              string
	table           *table.Table
	data            *PlayerTableData
	players         []Player
	team            Team
	rows            [][]string
	selectedTeam    Team
	selectedRow     int
	selectedSteamID steamid.SteamID
	height          int
	width           int
}

func (m PlayerTableModel) Init() tea.Cmd {
	return nil
}

func (m PlayerTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.Width(msg.Width / 2)

		return m, nil
	case SortPlayersMsg:
		m.data.Sort(msg.sortColumn, msg.asc)

		return m, nil
	case tea.MouseMsg:
		switch msg.Button { //nolint:exhaustive
		case tea.MouseButtonWheelUp:
			if m.selectedRow > 0 {
				m.selectedRow--
			}
		case tea.MouseButtonWheelDown:
			if m.selectedRow < len(m.rows)-1 {
				m.selectedRow++
			}
		default:
		}

		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.up):
			return m.moveSelection(Up)
		case key.Matches(msg, DefaultKeyMap.down):
			return m.moveSelection(Down)
		case key.Matches(msg, DefaultKeyMap.left):
			return m.moveSelection(Left)
		case key.Matches(msg, DefaultKeyMap.right):
			return m.moveSelection(Right)
		}

		return m, nil
	case SelectedTableRowMsg:
		m.selectedTeam = msg.selectedTeam
		m.selectedRow = msg.selectedRow
		m.selectedSteamID = msg.selectedSteamID
	case FullStateUpdateMsg:
		return m.updatePlayers(msg.players)
	}

	return m, nil
}

func (m PlayerTableModel) moveSelection(direction Direction) (tea.Model, tea.Cmd) {
	switch direction {
	case Up:
		if m.selectedRow > 0 {
			m.selectedRow--
		} else {
			return m, nil
		}
	case Down:
		if m.selectedRow < len(m.rows)-1 {
			m.selectedRow++
		} else {
			return m, nil
		}
	case Left:
		if m.selectedTeam != RED {
			m.selectedTeam = RED
			m.selectedRow = min(len(m.rows)-1, m.selectedRow)
			if m.selectedRow < 0 {
				m.selectedRow = 0
			}
		} else {
			return m, nil
		}
	case Right:
		if m.selectedTeam != BLU {
			m.selectedTeam = BLU
			m.selectedRow = min(len(m.rows)-1, m.selectedRow)
			if m.selectedRow < 0 {
				m.selectedRow = 0
			}
		} else {
			return m, nil
		}
	}

	var cmd tea.Cmd
	if m.selectedTeam == m.team {
		row := m.data.players[m.selectedRow]
		m.selectedSteamID = row.SteamID
		cmd = func() tea.Msg { return SelectedPlayerMsg{player: row} }
	}

	return m, tea.Batch(
		cmd, func() tea.Msg {
			return SelectedTableRowMsg{selectedTeam: m.selectedTeam, selectedRow: m.selectedRow, selectedSteamID: m.selectedSteamID}
		})
}

func (m PlayerTableModel) updatePlayers(playersUpdate []Player) (tea.Model, tea.Cmd) {
	data := NewPlayerTableData(playersUpdate, m.team)

	m.data = &data
	m.data.Sort(m.data.sortColumn, m.data.asc)
	m.table.Data(&data)

	if m.selectedTeam == m.team {
		row := m.data.players[m.selectedRow]
		m.selectedSteamID = row.SteamID
	}

	return m, func() tea.Msg {
		return SelectedTableRowMsg{
			selectedTeam:    m.selectedTeam,
			selectedRow:     m.selectedRow,
			selectedSteamID: m.selectedSteamID,
		}
	}
}

type playerTableCol int

const (
	colUID playerTableCol = iota
	colName
	colScore
	colDeaths
	colPing
	colMeta
)

func (m PlayerTableModel) View() string {
	return m.table.
		Headers(m.data.Headers()...).
		StyleFunc(func(row, col int) lipgloss.Style {
			mappedCol := m.data.enabledColumns[col]
			width := 10
			switch playerTableCol(mappedCol) {
			case colUID:
				width = 6
			case colName:
				width = 0
			case colScore:
				width = 5
			case colDeaths:
				width = 5
			case colPing:
				width = 5
			case colMeta:
				width = 10
			}
			switch {
			case row == table.HeaderRow:
				if m.team == RED {
					if playerTableCol(col) == colName {
						return styles.HeaderStyleRed.Width(width)
					}

					return styles.HeaderStyleRed.Width(width)
				}
				if col == 1 {
					return styles.HeaderStyleBlu.Width(width)
				}

				return styles.HeaderStyleBlu
			case col != 5 && row == m.selectedRow && m.team == m.selectedTeam:
				if m.team == RED {
					if playerTableCol(col) == colName {
						return styles.SelectedCellStyleNameRed.Width(width)
					}

					return styles.SelectedCellStyleRed.Width(width)
				}
				if playerTableCol(col) == colName {
					return styles.SelectedCellStyleNameBlu.Width(width)
				}

				return styles.SelectedCellStyleBlu.Width(width)
			case col == 1:
				return styles.EvenRowStyle.Width(width)
			case row%2 == 0:
				return styles.EvenRowStyle.Width(width)
			default:
				return styles.OddRowStyle.Width(width)
			}
		}).
		String()
}
