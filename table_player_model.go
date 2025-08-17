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
	colScoreSize  playerTableColSize = 7
	colDeathsSize playerTableColSize = 7
	colPingSize   playerTableColSize = 5
	colMetaSize   playerTableColSize = 8
)

func NewPlayerTableModel(team Team, selfSID steamid.SteamID) TablePlayerModel {
	zoneID := zone.NewPrefix()

	return TablePlayerModel{
		id:           zoneID,
		team:         team,
		selectedTeam: RED,
		data:         NewTablePlayerData(zoneID, Players{}, team),
		table:        NewUnstyledTable(),
		selfSteamID:  selfSID,
	}
}

type TablePlayerModel struct {
	id              string
	table           *table.Table
	data            *TablePlayerData
	team            Team
	selectedTeam    Team
	selectedSteamID steamid.SteamID
	height          int
	width           int
	selfSteamID     steamid.SteamID
}

func (m TablePlayerModel) Init() tea.Cmd {
	return nil
}

func (m TablePlayerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case Config:
		m.selfSteamID = msg.SteamID

		return m, nil
	case ContentViewPortHeightMsg:
		m.width = msg.width
		m.height = msg.height
		m.table.Width(msg.width / 2)

		return m, nil
	case SortPlayersMsg:
		m.data.Sort(msg.sortColumn, msg.asc)

		return m, nil
	case tea.MouseMsg:
		switch msg.Button { //nolint:exhaustive
		case tea.MouseButtonWheelUp:
			// return m.moveSelection(Up)
		case tea.MouseButtonWheelDown:
			// return m.moveSelection(Down)
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
		var cmd tea.Cmd
		switch {
		case key.Matches(msg, DefaultKeyMap.up):
			m, cmd = m.moveSelection(Up)

			return m, cmd
		case key.Matches(msg, DefaultKeyMap.down):
			m, cmd = m.moveSelection(Down)

			return m, cmd
		case key.Matches(msg, DefaultKeyMap.left):
			m, cmd = m.moveSelection(Left)

			return m, cmd
		case key.Matches(msg, DefaultKeyMap.right):
			m, cmd = m.moveSelection(Right)

			return m, cmd
		}

	case SelectedTableRowMsg:
		m.selectedTeam = msg.selectedTeam
		m.selectedSteamID = msg.selectedSteamID

		return m, nil
	case FullStateUpdateMsg:
		return m.updatePlayers(msg.players)
	}

	return m, nil
}

func (m TablePlayerModel) moveSelection(direction Direction) (TablePlayerModel, tea.Cmd) {
	currentRow := m.currentRowIndex()
	switch direction {
	case Up:
		if currentRow < 0 && len(m.data.players) > 0 {
			m.selectedSteamID = m.data.players[len(m.data.players)-1].SteamID

			break
		}
		if currentRow == 0 {
			break
		}
		if currentRow-1 >= 0 && max(0, len(m.data.players)-1) > currentRow-1 {
			m.selectedSteamID = m.data.players[currentRow-1].SteamID
		}
	case Down:
		if currentRow < 0 && len(m.data.players) > 0 {
			m.selectedSteamID = m.data.players[0].SteamID

			break
		}
		if currentRow >= len(m.data.players)-1 {
			break
		}
		m.selectedSteamID = m.data.players[currentRow+1].SteamID
	case Left:
		if m.team == RED {
			break
		}
		m.selectedTeam = RED
		if m.team == RED {
			m.selectedSteamID = m.data.players[len(m.data.players)-1].SteamID
		}
	case Right:
		if m.team == BLU {
			break
		}
		m.selectedTeam = BLU
		if m.team == BLU {
			m.selectedSteamID = m.data.players[len(m.data.players)-1].SteamID
		}
	}

	cmds := []tea.Cmd{func() tea.Msg {
		return SelectedTableRowMsg{selectedTeam: m.selectedTeam, selectedSteamID: m.selectedSteamID}
	}}

	if m.selectedTeam == m.team {
		if player, ok := m.currentPlayer(); ok {
			cmds = append(cmds, func() tea.Msg { return SelectedPlayerMsg{player: player} })
		}
	}

	return m, tea.Batch(cmds...)
}

func (m TablePlayerModel) currentPlayer() (Player, bool) {
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

func (m TablePlayerModel) currentRowIndex() int {
	for rowIdx, player := range m.data.players {
		if player.SteamID == m.selectedSteamID {
			return rowIdx
		}
	}

	return -1
}

func (m TablePlayerModel) updatePlayers(playersUpdate Players) (tea.Model, tea.Cmd) {
	m.data = NewTablePlayerData(m.id, playersUpdate, m.team)
	m.data.Sort(m.data.sortColumn, m.data.asc)
	m.table.Data(m.data)

	if m.selectedTeam == m.team {
		oldID := m.selectedSteamID
		if player, ok := m.currentPlayer(); !ok && len(m.data.players) > 0 {
			m.selectedSteamID = m.data.players[0].SteamID
		} else {
			m.selectedSteamID = player.SteamID
		}

		if !oldID.Equal(m.selectedSteamID) {
			return m, func() tea.Msg {
				return SelectedTableRowMsg{
					selectedTeam:    m.selectedTeam,
					selectedSteamID: m.selectedSteamID,
				}
			}
		}

		return m, nil
	}

	return m, nil
}

func (m TablePlayerModel) View() string {
	selectedRowIdx := m.currentRowIndex()

	return m.table.
		Headers(m.data.Headers()...).
		StyleFunc(func(row, col int) lipgloss.Style {
			isSelf := row >= 0 && len(m.data.players)-1 <= row && m.data.players[row].SteamID.Equal(m.selfSteamID)

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

			case playerTableCol(col) != colMeta && row == selectedRowIdx && m.team == m.selectedTeam:
				if m.team == RED {
					if playerTableCol(col) == colName {
						if isSelf {
							return styles.PlayerTableRowSelf.Width(int(width))
						}

						return styles.SelectedCellStyleNameRed.Width(int(width))
					}

					return styles.SelectedCellStyleRed.Width(int(width))
				}
				if playerTableCol(col) == colName {
					if isSelf {
						return styles.PlayerTableRowSelf.Width(int(width))
					}

					return styles.SelectedCellStyleNameBlu.Width(int(width))
				}

				return styles.SelectedCellStyleBlu.Width(int(width))
			case playerTableCol(col) == colName:
				return styles.PlayerTableRow.Width(int(width))
			case row%2 == 0:
				return styles.PlayerTableRow.Width(int(width))
			default:
				return styles.PlayerTableRowOdd.Width(int(width))
			}
		}).
		String()
}
