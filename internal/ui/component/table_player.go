package component

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/ui/command"
	"github.com/leighmacdonald/tf-tui/internal/ui/input"
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

func NewPlayerTableModel(team tf.Team, selfSID steamid.SteamID, serverMode bool) *TablePlayerModel {
	zoneID := zone.NewPrefix()

	return &TablePlayerModel{
		id:           zoneID,
		team:         team,
		selectedTeam: tf.RED,
		table:        NewUnstyledTable(),
		selfSteamID:  selfSID,
		serverMode:   serverMode,
		serverData:   map[string]*tablePlayerData{},
	}
}

type TablePlayerModel struct {
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
	viewState       model.ViewState
}

func (m *TablePlayerModel) Init() tea.Cmd {
	return nil
}

func (m *TablePlayerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case config.Config:
		m.selfSteamID = msg.SteamID

		return m, nil
	case model.ViewState:
		m.viewState = msg
		// When the screen with is a odd number, increase the size of the right player table
		// to ensure that it fills the screen fully.
		half := msg.Width / 2
		if msg.Width%2 != 0 && m.team == tf.BLU {
			m.table.Width(half + 1)
		} else {
			m.table.Width(half)
		}

		return m, nil
	case command.SelectServerSnapshotMsg:
		m.selectedServer = msg.Server.HostPort
		if _, ok := m.serverData[msg.Server.HostPort]; !ok {
			m.serverData[msg.Server.HostPort] = NewTablePlayerData(m.id, m.serverMode, nil, m.team)
		}
	case command.SortMsg[playerTableCol]:
		if data, ok := m.serverData[m.selectedServer]; ok {
			data.Sort(msg.SortColumn, msg.Asc)
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
						command.SelectTeam(m.team),
						command.SelectPlayer(item))
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
		case key.Matches(msg, input.Default.Up):
			cmd = m.moveSelection(input.Up)

			return m, cmd
		case key.Matches(msg, input.Default.Down):
			cmd = m.moveSelection(input.Down)

			return m, cmd
		}

	case command.SelectedPlayerMsg:
		m.selectedSteamID = msg.Player.SteamID

		return m, nil
	case tf.Team:
		m.selectedTeam = msg

		return m, m.selectClosestPlayer()
	case model.Snapshot:
		return m.updatePlayers(msg)
	}

	return m, nil
}

func (m *TablePlayerModel) isActiveZone() bool {
	return (m.team == tf.RED && m.viewState.KeyZone == model.KZplayerTableRED) || (m.team == tf.BLU && m.viewState.KeyZone == model.KZplayerTableBLU)
}

func (m *TablePlayerModel) moveSelection(dir input.Direction) tea.Cmd {
	data, ok := m.serverData[m.selectedServer]
	if !ok {
		return nil
	}

	currentRow := m.currentRowIndex()
	switch dir { //nolint:exhaustive
	case input.Up:
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
	case input.Down:
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
			cmds = append(cmds, command.SelectPlayer(player))
		}
	}

	return tea.Batch(cmds...)
}

func (m *TablePlayerModel) currentPlayer() (model.Player, bool) {
	data, ok := m.serverData[m.selectedServer]
	if !ok {
		return model.Player{}, false
	}

	if m.selectedTeam != m.team {
		return model.Player{}, false
	}
	for _, player := range data.players {
		if player.SteamID == m.selectedSteamID {
			return player, true
		}
	}

	return model.Player{}, false
}

func (m *TablePlayerModel) currentRowIndex() int {
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

func (m *TablePlayerModel) selectClosestPlayer() tea.Cmd {
	data, ok := m.serverData[m.selectedServer]
	if !ok {
		return nil
	}

	var selectedPlayer model.Player
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
			return tea.Sequence(command.SelectTeam(m.selectedTeam), command.SelectPlayer(selectedPlayer))
		}
	}

	return nil
}

func (m *TablePlayerModel) updatePlayers(snapshot model.Snapshot) (tea.Model, tea.Cmd) {
	data := NewTablePlayerData(m.id, m.serverMode, snapshot.Server.Players, m.team)
	data.Sort(data.sortColumn, data.asc)
	m.table.Data(data)
	m.serverData[snapshot.HostPort] = data

	return m, m.selectClosestPlayer()
}

func (m *TablePlayerModel) View() string {
	data, ok := m.serverData[m.selectedServer]
	if !ok {
		data = NewTablePlayerData(m.id, m.serverMode, nil, m.team)
	}
	selectedRowIdx := m.currentRowIndex()

	title := "RED"
	if m.team == tf.BLU {
		title = "BLU"
	}

	return Container(title, calcPct(m.viewState.Width, 50), m.viewState.Height/2, m.table.
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
