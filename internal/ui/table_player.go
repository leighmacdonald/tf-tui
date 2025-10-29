package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/ui/model"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
	zone "github.com/lrstanley/bubblezone"
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
	colAddress
	colLoss
	colTime
)

// playerTableColSize defines the sizes of the player columns.
type playerTableColSize int

const (
	colUIDSize     playerTableColSize = 6
	colNameSize    playerTableColSize = 0
	colScoreSize   playerTableColSize = 7
	colDeathsSize  playerTableColSize = 7
	colPingSize    playerTableColSize = 5
	colMetaSize    playerTableColSize = 8
	colAddressSize playerTableColSize = 15
	colLossSize    playerTableColSize = 5
	colTimeSize    playerTableColSize = 5
)

func newPlayerTableModel(team tf.Team, selfSID steamid.SteamID, serverMode bool) *tablePlayerModel {
	zoneID := zone.NewPrefix()

	return &tablePlayerModel{
		id:           zoneID,
		team:         team,
		selectedTeam: tf.RED,
		table:        newUnstyledTable(),
		selfSteamID:  selfSID,
		serverMode:   serverMode,
		serverData:   map[string]*tablePlayerData{},
	}
}

type tablePlayerModel struct {
	id         string
	serverMode bool
	table      *table.Table
	// data            *tablePlayerData
	team            tf.Team
	selectedTeam    tf.Team
	selectedSteamID steamid.SteamID
	selectedServer  string
	serverData      map[string]*tablePlayerData
	selfSteamID     steamid.SteamID
	viewState       viewState
}

func (m *tablePlayerModel) Init() tea.Cmd {
	return nil
}

func (m *tablePlayerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case config.Config:
		m.selfSteamID = msg.SteamID

		return m, nil
	case viewState:
		m.viewState = msg
		// When the screen with is a odd number, increase the size of the right player table
		// to ensure that it fills the screen fully.
		half := msg.width / 2
		if msg.width%2 != 0 && m.team == tf.BLU {
			m.table.Width(half + 1)
		} else {
			m.table.Width(half)
		}

		return m, nil
	case selectServerSnapshotMsg:
		m.selectedServer = msg.server.HostPort
		if _, ok := m.serverData[msg.server.HostPort]; !ok {
			m.serverData[msg.server.HostPort] = newTablePlayerData(m.id, m.serverMode, nil, m.team)
		}
	case sortPlayersMsg:
		if data, ok := m.serverData[m.selectedServer]; ok {
			data.Sort(msg.sortColumn, msg.asc)
			m.table.Data(data)
		}

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

			data, ok := m.serverData[m.selectedServer]
			if !ok {
				break
			}

			for _, item := range data.players {
				// Check each item to see if it's in bounds.
				if zone.Get(m.id + item.SteamID.String()).InBounds(msg) {
					m.selectedSteamID = item.SteamID
					m.selectedTeam = m.team

					return m, tea.Sequence(
						selectTeam(m.team),
						selectPlayer(item))
				}
			}

			for _, markID := range []string{"name", "uid", "score", "meta", "deaths", "ping", "address", "loss", "time"} {
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
					case "address":
						col = colAddress
					case "loss":
						col = colLoss
					case "time":
						col = colTime
					default:
						col = colName
					}
					data.Sort(col, !data.asc)

					return m, nil
				}
			}

			return m, nil
		}
	case tea.KeyMsg:
		if !m.isActiveZone() {
			break
		}
		var cmd tea.Cmd
		switch {
		case key.Matches(msg, defaultKeyMap.up):
			cmd = m.moveSelection(up)

			return m, cmd
		case key.Matches(msg, defaultKeyMap.down):
			cmd = m.moveSelection(down)

			return m, cmd
		}

	case selectedPlayerMsg:
		m.selectedSteamID = msg.player.SteamID

		return m, nil
	case tf.Team:
		m.selectedTeam = msg

		return m, m.selectClosestPlayer()
	case Snapshot:
		return m.updatePlayers(msg)
	}

	return m, nil
}

func (m *tablePlayerModel) isActiveZone() bool {
	return (m.team == tf.RED && m.viewState.keyZone == playerTableRED) || (m.team == tf.BLU && m.viewState.keyZone == playerTableBLU)
}

func (m *tablePlayerModel) moveSelection(direction direction) tea.Cmd {
	data, ok := m.serverData[m.selectedServer]
	if !ok {
		return nil
	}

	currentRow := m.currentRowIndex()
	switch direction { //nolint:exhaustive
	case up:
		if currentRow < 0 && len(data.players) > 0 {
			m.selectedSteamID = data.players[len(data.players)-1].SteamID

			break
		}
		if currentRow == 0 {
			break
		}
		if currentRow-1 >= 0 && max(0, len(data.players)-1) > currentRow-1 {
			m.selectedSteamID = data.players[currentRow-1].SteamID
		}
	case down:
		if currentRow < 0 && len(data.players) > 0 {
			m.selectedSteamID = data.players[0].SteamID

			break
		}
		if currentRow >= len(data.players)-1 {
			break
		}
		m.selectedSteamID = data.players[currentRow+1].SteamID
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
	data, ok := m.serverData[m.selectedServer]
	if !ok {
		return Player{}, false
	}

	if m.selectedTeam != m.team {
		return Player{}, false
	}
	for _, player := range data.players {
		if player.SteamID == m.selectedSteamID {
			return player, true
		}
	}

	return Player{}, false
}

func (m *tablePlayerModel) currentRowIndex() int {
	data, ok := m.serverData[m.selectedServer]
	if !ok {
		return -1
	}

	for rowIdx, player := range data.players {
		if player.SteamID == m.selectedSteamID {
			return rowIdx
		}
	}

	return -1
}

func (m *tablePlayerModel) selectClosestPlayer() tea.Cmd {
	data, ok := m.serverData[m.selectedServer]
	if !ok {
		return nil
	}

	var selectedPlayer Player
	if m.selectedTeam == m.team {
		oldID := m.selectedSteamID
		if player, ok := m.currentPlayer(); !ok && len(data.players) > 0 {
			m.selectedSteamID = data.players[0].SteamID
			selectedPlayer = data.players[0]
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

func (m *tablePlayerModel) updatePlayers(snapshot Snapshot) (tea.Model, tea.Cmd) {
	data := newTablePlayerData(m.id, m.serverMode, snapshot.Server.Players, m.team)
	data.Sort(data.sortColumn, data.asc)
	m.table.Data(data)
	m.serverData[snapshot.HostPort] = data

	return m, m.selectClosestPlayer()
}

func (m *tablePlayerModel) View() string {
	data, ok := m.serverData[m.selectedServer]
	if !ok {
		data = newTablePlayerData(m.id, m.serverMode, nil, m.team)
	}
	selectedRowIdx := m.currentRowIndex()

	title := "RED"
	if m.team == tf.BLU {
		title = "BLU"
	}

	return model.Container(title, calcPct(m.viewState.width, 50), m.viewState.height/2, m.table.
		Headers(data.Headers()...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if len(data.players) == 0 {
				return styles.HeaderStyleBlu
			}
			isSelf := row >= 0 && len(data.players)-1 <= row && data.players[row].SteamID.Equal(m.selfSteamID)

			mappedCol := data.enabledColumns[col]
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
			case colAddress:
				width = colAddressSize
			case colLoss:
				width = colLossSize
			case colTime:
				width = colTimeSize
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
		String(), m.isActiveZone())
}
