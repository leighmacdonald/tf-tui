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

// Direction defines the cardinal directions the users can use in the UI.
type Direction int

const (
	Up Direction = iota //nolint:varnamelen
	Down
	Left
	Right
)

// playerTableCol defines all available columns for the player table.
type playerTableCol int

const (
	colUID playerTableCol = iota
	colName
	colScore
	colDeaths
	colPing
	colMeta
)

// playerTableColSize defines the sizes of the player columns.
type playerTableColSize int

const (
	colUIDSize    playerTableColSize = 6
	colNameSize   playerTableColSize = 0
	colScoreSize  playerTableColSize = 5
	colDeathsSize playerTableColSize = 5
	colPingSize   playerTableColSize = 5
	colMetaSize   playerTableColSize = 8
)

// PlayerTablesModel is a higher level component that manages the two player tables as children.
type PlayerTablesModel struct {
	selectedTeam Team // red = 3, blu = 4
	redTable     tea.Model
	bluTable     tea.Model
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
	return &PlayerTablesModel{
		selectedTeam: RED,
		redTable:     NewPlayerTableModel(RED),
		bluTable:     NewPlayerTableModel(BLU),
	}
}

func (m PlayerTablesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 2)
	m.redTable, cmds[0] = m.redTable.Update(msg)
	m.bluTable, cmds[1] = m.bluTable.Update(msg)

	return m, tea.Batch(cmds...)
}

func (m PlayerTablesModel) selectedColumnPlayerCount() int {
	var (
		model PlayerTableModel
		ok    bool
	)

	switch m.selectedTeam {
	case RED:
		model, ok = m.redTable.(PlayerTableModel)
	case BLU:
		model, ok = m.bluTable.(PlayerTableModel)
	default:
		return 0
	}

	if !ok {
		return 0
	}

	return model.data.Rows()
}

func (m PlayerTablesModel) View() string {
	return lipgloss.JoinHorizontal(lipgloss.Top, m.redTable.View(), m.bluTable.View())
}

func NewPlayerTableModel(team Team) *PlayerTableModel {
	zoneID := zone.NewPrefix()
	foreground := styles.Red
	if team == BLU {
		foreground = styles.Blu
	}
	data := NewTablePlayerData(zoneID, []Player{}, team)

	return &PlayerTableModel{
		id:           zoneID,
		team:         team,
		selectedTeam: RED,
		data:         &data,
		table: table.New().
			BorderStyle(lipgloss.NewStyle().Foreground(foreground)).
			BorderHeader(false),
	}
}

type PlayerTableModel struct {
	id              string
	table           *table.Table
	data            *TablePlayerData
	team            Team
	selectedTeam    Team
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
			return m.moveSelection(Up)
		case tea.MouseButtonWheelDown:
			return m.moveSelection(Down)
		default:
			if msg.Action != tea.MouseActionRelease || msg.Button != tea.MouseButtonLeft {
				return m, nil
			}

			for _, item := range m.data.players {
				// Check each item to see if it's in bounds.
				if zone.Get(m.id + item.SteamID.String()).InBounds(msg) {
					m.selectedSteamID = item.SteamID
					m.selectedTeam = m.team

					return m, tea.Batch(func() tea.Msg {
						return SelectedTableRowMsg{selectedTeam: m.selectedTeam, selectedSteamID: m.selectedSteamID}
					}, func() tea.Msg {
						return SelectedPlayerMsg{player: item}
					})
				}
			}

			for _, markID := range []string{"name", "uid", "score", "meta", "deaths", "ping"} {
				if zone.Get(m.id + markID).InBounds(msg) {
					var col playerTableColumn
					switch markID {
					case "uid":
						col = playerUID
					case "score":
						col = playerScore
					case "deaths":
						col = playerDeaths
					case "ping":
						col = playerPing
					case "meta":
						col = playerMeta
					default:
						col = playerName
					}
					m.data.Sort(col, !m.data.asc)

					return m, nil
				}
			}

			return m, nil
		}
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
		m.selectedSteamID = msg.selectedSteamID
	case FullStateUpdateMsg:
		return m.updatePlayers(msg.players)
	}

	return m, nil
}

func (m PlayerTableModel) moveSelection(direction Direction) (tea.Model, tea.Cmd) {
	currentRow := m.currentRowIndex()
	switch direction {
	case Up:
		if currentRow < 0 && len(m.data.players) > 0 {
			m.selectedSteamID = m.data.players[len(m.data.players)-1].SteamID

			break
		}
		if currentRow == 0 {
			return m, nil
		}
		m.selectedSteamID = m.data.players[currentRow-1].SteamID
	case Down:
		if currentRow < 0 && len(m.data.players) > 0 {
			m.selectedSteamID = m.data.players[0].SteamID

			break
		}
		if currentRow >= len(m.data.players)-1 {
			return m, nil
		}
		m.selectedSteamID = m.data.players[currentRow+1].SteamID
	case Left:
		if m.team == RED {
			break
		}
		m.selectedTeam = RED
		if currentRow < 0 {
			currentRow = 0
		}
		if m.team == RED {
			m.selectedSteamID = m.data.players[len(m.data.players)-1].SteamID
		}
	case Right:
		if m.team == BLU {
			return m, nil
		}
		m.selectedTeam = BLU
		if currentRow < 0 {
			currentRow = 0
		}
		if m.team == BLU {
			m.selectedSteamID = m.data.players[len(m.data.players)-1].SteamID
		}
	}

	var cmd tea.Cmd
	if m.selectedTeam == m.team {
		if player, ok := m.currentPlayer(); ok {
			cmd = func() tea.Msg { return SelectedPlayerMsg{player: player} }
		}
	}

	return m, tea.Batch(cmd, func() tea.Msg {
		return SelectedTableRowMsg{selectedTeam: m.selectedTeam, selectedSteamID: m.selectedSteamID}
	})
}

func (m PlayerTableModel) currentPlayer() (Player, bool) {
	if m.selectedTeam != m.team {
		return Player{}, false
	}
	for _, player := range m.data.players {
		if player.SteamID == m.selectedSteamID {
			return player, true
		}
	}

	return Player{}, false
}

func (m PlayerTableModel) currentRowIndex() int {
	for rowIdx, player := range m.data.players {
		if player.SteamID == m.selectedSteamID {
			return rowIdx
		}
	}

	return -1
}

func (m PlayerTableModel) updatePlayers(playersUpdate []Player) (tea.Model, tea.Cmd) {
	data := NewTablePlayerData(m.id, playersUpdate, m.team)

	m.data = &data
	m.data.Sort(m.data.sortColumn, m.data.asc)
	m.table.Data(&data)

	if m.selectedTeam == m.team {
		if player, ok := m.currentPlayer(); !ok && len(data.players) > 0 {
			m.selectedSteamID = data.players[0].SteamID
		} else {
			m.selectedSteamID = player.SteamID
		}

		return m, func() tea.Msg {
			return SelectedTableRowMsg{
				selectedTeam:    m.selectedTeam,
				selectedSteamID: m.selectedSteamID,
			}
		}
	}

	return m, nil
}

func (m PlayerTableModel) View() string {
	selectedRowIdx := m.currentRowIndex()

	return m.table.
		Headers(m.data.Headers()...).
		StyleFunc(func(row, col int) lipgloss.Style {
			mappedCol := m.data.enabledColumns[col]
			width := colNameSize
			switch playerTableCol(mappedCol) {
			case colUID:
				width = colUIDSize
			case colName:
				width = colNameSize
			case colScore:
				width = colScoreSize
			case colDeaths:
				width = colDeathsSize
			case colPing:
				width = colPingSize
			case colMeta:
				width = colMetaSize
			}
			switch {
			case row == table.HeaderRow:
				if m.team == RED {
					if playerTableCol(col) == colName {
						return styles.HeaderStyleRed.Width(int(width))
					}

					return styles.HeaderStyleRed.Width(int(width))
				}
				if col == 1 {
					return styles.HeaderStyleBlu.Width(int(width))
				}

				return styles.HeaderStyleBlu
			case col != 5 && row == selectedRowIdx && m.team == m.selectedTeam:
				if m.team == RED {
					if playerTableCol(col) == colName {
						return styles.SelectedCellStyleNameRed.Width(int(width))
					}

					return styles.SelectedCellStyleRed.Width(int(width))
				}
				if playerTableCol(col) == colName {
					return styles.SelectedCellStyleNameBlu.Width(int(width))
				}

				return styles.SelectedCellStyleBlu.Width(int(width))
			case col == 1:
				return styles.EvenRowStyle.Width(int(width))
			case row%2 == 0:
				return styles.EvenRowStyle.Width(int(width))
			default:
				return styles.OddRowStyle.Width(int(width))
			}
		}).
		String()
}
