package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/leighmacdonald/tf-tui/styles"
	zone "github.com/lrstanley/bubblezone"
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
		return len(m.redRows)
	}

	return len(m.bluRows)
}

func (m PlayerTablesModel) View() string {
	return lipgloss.JoinHorizontal(lipgloss.Top, m.redTable.View(), m.bluTable.View())
}

func NewPlayerTableModel(team Team) *PlayerTableModel {
	foreground := styles.Red
	if team == BLU {
		foreground = styles.Blu
	}
	return &PlayerTableModel{
		id:   zone.NewPrefix(),
		team: team,
		table: table.New().
			BorderStyle(lipgloss.NewStyle().Foreground(foreground)).
			Headers("UID", "Name", "Score", "Deaths", "Ping", "Meta").
			BorderHeader(false)}
}

type PlayerTableModel struct {
	id           string
	table        *table.Table
	players      []Player
	team         Team
	rows         [][]string
	selectedTeam Team
	selectedRow  int
	selectedUID  int
	height       int
	width        int
}

func (m PlayerTableModel) Init() tea.Cmd {
	return nil
}

func (m PlayerTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.Width((msg.Width / 2) - 1)
		return m, nil
	case tea.MouseMsg:
		// FIXME
		switch msg.Button {
		case tea.MouseButtonWheelDown:
			if m.selectedRow > 0 {
				m.selectedRow--
			}
		case tea.MouseButtonWheelUp:
			if m.selectedRow < len(m.rows)-1 {
				m.selectedRow++
			}
		}
		return m, nil
	case tea.KeyMsg:
		updated := false
		switch {
		case key.Matches(msg, DefaultKeyMap.up):
			if m.selectedRow > 0 {
				m.selectedRow--
				updated = true
			}
		case key.Matches(msg, DefaultKeyMap.down):
			if m.selectedRow < len(m.rows)-1 {
				m.selectedRow++
				updated = true
			}
		case key.Matches(msg, DefaultKeyMap.left):
			if m.selectedTeam != RED {
				m.selectedTeam = RED
				m.selectedRow = min(len(m.rows)-1, m.selectedRow)
				updated = true
			}
		case key.Matches(msg, DefaultKeyMap.right):
			if m.selectedTeam != BLU {
				m.selectedTeam = BLU
				m.selectedRow = min(len(m.rows)-1, m.selectedRow)
				updated = true
			}
		}
		if updated {
			var cmd tea.Cmd
			if m.selectedTeam == m.team {
				m.selectedUID = findCurrentUID(m.selectedRow, m.rows)
				for _, p := range m.players {
					if p.UserID == m.selectedUID {
						cmd = func() tea.Msg { return SelectedPlayerMsg{player: p} }

						break
					}
				}
			}

			return m, tea.Batch(cmd, func() tea.Msg {
				return SelectedTableRowMsg{
					selectedTeam: m.selectedTeam,
					selectedRow:  m.selectedRow,
					selectedUID:  m.selectedUID,
				}
			})
		}

		return m, nil
	case SelectedTableRowMsg:
		m.selectedTeam = msg.selectedTeam
		m.selectedRow = msg.selectedRow
		m.selectedUID = msg.selectedUID
	case FullStateUpdateMsg:
		m.players = msg.players
		// m.selectedUID = msg.selectedUID
		var (
			rows [][]string
		)

		for _, player := range m.players {
			if !player.SteamID.Valid() {
				continue
			}

			var afflictions []string
			if len(*player.meta.Bans) > 0 {
				afflictions = append(afflictions, styles.IconBans)
			}

			if player.meta.NumberOfVacBans > 0 {
				afflictions = append(afflictions, styles.IconVac)
			}

			if len(afflictions) == 0 {
				afflictions = append(afflictions, styles.IconCheck)
			}

			name := player.Name
			if name == "" {
				name = player.SteamID.String()
			}

			row := []string{
				fmt.Sprintf("%d", player.UserID),
				name,
				fmt.Sprintf("%d", player.Score),
				fmt.Sprintf("%d", player.Deaths),
				fmt.Sprintf("%d", player.Ping),
				strings.Join(afflictions, " "),
			}

			switch player.Team {
			case m.team:
				rows = append(rows, row)
			}
		}

		sortTableRows(rows, 0)

		m.table.ClearRows()
		m.table.Rows(rows...)
		m.rows = rows

		if m.selectedTeam == m.team {
			m.selectedUID = findCurrentUID(m.selectedRow, m.rows)
		}

		return m, func() tea.Msg {
			return SelectedTableRowMsg{
				selectedTeam: m.selectedTeam,
				selectedRow:  m.selectedRow,
				selectedUID:  m.selectedUID,
			}
		}
	}

	return m, nil
}

func (m PlayerTableModel) View() string {
	return m.table.StyleFunc(func(row, col int) lipgloss.Style {
		switch {
		case row == table.HeaderRow:
			if m.team == RED {
				if col == 1 {
					return styles.HeaderStyleRed.Width(30)
				}

				return styles.HeaderStyleRed
			}
			if col == 1 {
				return styles.HeaderStyleBlu.Width(30)
			}

			return styles.HeaderStyleBlu
		case col != 5 && row == m.selectedRow && m.team == m.selectedTeam:
			if m.team == RED {
				if col == 1 {
					return styles.SelectedCellStyleNameRed
				}

				return styles.SelectedCellStyleRed
			}
			if col == 1 {
				return styles.SelectedCellStyleNameBlu
			}

			return styles.SelectedCellStyleBlu
		case col == 1:
			return styles.EvenRowStyle
		case row%2 == 0:
			return styles.EvenRowStyle
		default:
			return styles.OddRowStyle
		}
	}).String()
}
