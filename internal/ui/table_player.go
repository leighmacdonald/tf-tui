package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
	zone "github.com/lrstanley/bubblezone"
)

// direction defines the cardinal directions the users can use in the UI.
type direction int

const (
	up direction = iota //nolint:varnamelen
	down
	left
	right
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

func newPlayerTableModel(team tf.Team, selfSID steamid.SteamID) *tablePlayerModel {
	zoneID := zone.NewPrefix()

	return &tablePlayerModel{
		id:           zoneID,
		team:         team,
		selectedTeam: tf.RED,
		data:         newTablePlayerData(zoneID, Players{}, team),
		table:        newUnstyledTable(),
		selfSteamID:  selfSID,
	}
}

type tablePlayerModel struct {
	id              string
	table           *table.Table
	data            *tablePlayerData
	team            tf.Team
	selectedTeam    tf.Team
	selectedSteamID steamid.SteamID
	height          int
	width           int
	selfSteamID     steamid.SteamID
}

func (m *tablePlayerModel) Init() tea.Cmd {
	return nil
}

func (m *tablePlayerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case config.Config:
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

					return m, tea.Sequence(
						selectTeam(m.team),
						selectPlayer(item))
				}
			}

			for _, markID := range []string{"name", "uid", "score", "meta", "deaths", "ping"} {
				if zone.Get(m.id + markID).InBounds(msg) {
					var col playerTableCol
					switch markID {
					case "uid":
						col = colUID
					case "score":
						col = colScore
					case "deaths":
						col = colDeaths
					case "ping":
						col = colPing
					case "meta":
						col = colMeta
					default:
						col = colName
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
			cmd = m.moveSelection(up)

			return m, cmd
		case key.Matches(msg, DefaultKeyMap.down):
			cmd = m.moveSelection(down)

			return m, cmd
		}

	case SelectedPlayerMsg:
		m.selectedSteamID = msg.player.SteamID

		return m, nil
	case SelectedTeamMsg:
		m.selectedTeam = msg.selectedTeam

		return m, m.selectClosestPlayer()
	case Players:
		return m.updatePlayers(msg)
	}

	return m, nil
}

func (m *tablePlayerModel) moveSelection(direction direction) tea.Cmd {
	currentRow := m.currentRowIndex()
	switch direction { //nolint:exhaustive
	case up:
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
	case down:
		if currentRow < 0 && len(m.data.players) > 0 {
			m.selectedSteamID = m.data.players[0].SteamID

			break
		}
		if currentRow >= len(m.data.players)-1 {
			break
		}
		m.selectedSteamID = m.data.players[currentRow+1].SteamID
	default:
		return nil
	}
	cmds := []tea.Cmd{}

	if m.selectedTeam == m.team {
		if player, ok := m.currentPlayer(); ok {
			cmds = append(cmds, selectPlayer(player))
		}
	}

	return tea.Batch(cmds...)
}

func (m *tablePlayerModel) currentPlayer() (Player, bool) {
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

func (m *tablePlayerModel) currentRowIndex() int {
	for rowIdx, player := range m.data.players {
		if player.SteamID == m.selectedSteamID {
			return rowIdx
		}
	}

	return -1
}

func (m *tablePlayerModel) selectClosestPlayer() tea.Cmd {
	var selectedPlayer Player
	if m.selectedTeam == m.team {
		oldID := m.selectedSteamID
		if player, ok := m.currentPlayer(); !ok && len(m.data.players) > 0 {
			m.selectedSteamID = m.data.players[0].SteamID
			selectedPlayer = m.data.players[0]
		} else {
			m.selectedSteamID = player.SteamID
			selectedPlayer = player
		}

		if !oldID.Equal(m.selectedSteamID) {
			return tea.Sequence(selectTeam(m.selectedTeam), selectPlayer(selectedPlayer))
		}
	}

	return nil
}

func (m *tablePlayerModel) updatePlayers(playersUpdate Players) (tea.Model, tea.Cmd) {
	m.data = newTablePlayerData(m.id, playersUpdate, m.team)
	m.data.Sort(m.data.sortColumn, m.data.asc)
	m.table.Data(m.data)

	return m, m.selectClosestPlayer()
}

func (m *tablePlayerModel) View() string {
	selectedRowIdx := m.currentRowIndex()

	return m.table.
		Headers(m.data.Headers()...).
		StyleFunc(func(row, col int) lipgloss.Style {
			isSelf := row >= 0 && len(m.data.players)-1 <= row && m.data.players[row].SteamID.Equal(m.selfSteamID)

			mappedCol := m.data.enabledColumns[col]
			width := colNameSize
			switch mappedCol {
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
				if m.team == tf.RED {
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
				if m.team == tf.RED {
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
