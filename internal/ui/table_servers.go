package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
	zone "github.com/lrstanley/bubblezone"
)

type serverTableCol int

const (
	colServerName serverTableCol = iota
	colServerMap
	colServerRegion
	colServerPlayers
	colServerPing
	colServerUptime
	colServerCPU
	colServerFPS
	colServerInRate
	colServerOutRate
	colServerConnects
)

type serverTableColSize int

const (
	colServerNameSize     serverTableColSize = 0
	colServerRegionSize   serverTableColSize = 4
	colServerMapSize      serverTableColSize = 25
	colServerPlayersSize  serverTableColSize = 8
	colServerPingSize     serverTableColSize = 10
	colServerUptimeSize   serverTableColSize = 10
	colServerCPUSize      serverTableColSize = 6
	colServerFPSSize      serverTableColSize = 6
	colServerInRateSize   serverTableColSize = 12
	colServerOutRateSize  serverTableColSize = 12
	colServerConnectsSize serverTableColSize = 6
)

var defaultServerTableColumns = []serverTableCol{
	colServerRegion,
	colServerName,
	colServerMap,
	colServerPlayers,
	colServerPing,
	colServerUptime,
	colServerCPU,
	colServerFPS,
	colServerInRate,
	colServerOutRate,
	colServerConnects,
}

func newServerTableModel() *serverTableModel {
	zoneID := zone.NewPrefix()
	return &serverTableModel{
		zoneID:   zoneID,
		table:    newUnstyledTable(),
		data:     newTableServerData(zoneID, nil, defaultServerTableColumns...),
		viewport: viewport.New(1, 1),
	}
}

type serverTableModel struct {
	zoneID          string
	viewport        viewport.Model
	table           *table.Table
	data            *serverTableData
	selectedsServer string
	width           int
	contentHeight   int
	inputActive     bool
}

func (m *serverTableModel) Init() tea.Cmd {
	return nil
}

func (m *serverTableModel) Update(msg tea.Msg) (*serverTableModel, tea.Cmd) {
	switch msg := msg.(type) {
	case inputZoneChangeMsg:
		m.inputActive = msg.zone == zoneServers
	case contentViewPortHeightMsg:
		m.width = msg.width
		m.contentHeight = min(msg.contentViewPortHeight, msg.height/2)
		m.viewport.Width = msg.width
		m.viewport.Height = min(msg.contentViewPortHeight, msg.height/2) - 2
	case []Snapshot:
		m.data = newTableServerData(m.zoneID, msg)
		m.data.Sort(m.data.sortColumn, m.data.asc)
		m.table.Data(m.data)
		// Send a snapshot for the currently selected server.
		if len(m.data.servers) > 0 {
			if m.selectedsServer == "" {
				m.selectedsServer = m.data.servers[0].HostPort
			}
			for _, snaps := range m.data.servers {
				if m.selectedsServer == snaps.HostPort {
					return m, setServer(snaps)
				}
			}

		}
	case selectServerSnapshotMsg:
		m.selectedsServer = msg.server.HostPort
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionRelease || msg.Button != tea.MouseButtonLeft {
			return m, nil
		}

		for _, item := range m.data.servers {
			if zone.Get(m.zoneID + item.HostPort).InBounds(msg) {
				return m, tea.Batch(setServer(item), setInputZone(zoneServers))
			}
		}

		for _, markID := range []string{"n", "m", "r", "pl", "pi", "u", "cp", "f", "i", "o", "co"} {
			if zone.Get(m.zoneID + markID).InBounds(msg) {
				var col serverTableCol
				switch markID {
				case "n":
					col = colServerName
				case "m":
					col = colServerMap
				case "r":
					col = colServerRegion
				case "pl":
					col = colServerPlayers
				case "pi":
					col = colServerPing
				case "u":
					col = colServerUptime
				case "cp":
					col = colServerCPU
				case "f":
					col = colServerFPS
				case "i":
					col = colServerInRate
				case "o":
					col = colServerOutRate
				case "co":
					col = colServerConnects
				}

				m.data.Sort(col, !m.data.asc)

				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

func (m *serverTableModel) View() string {
	currentIdx := m.currentRowIndex()

	content := m.table.
		Width(m.width).
		Headers(m.data.Headers()...).
		StyleFunc(func(row, col int) lipgloss.Style {
			mappedCol := m.data.enabledColumns[col]
			width := colServerNameSize
			switch mappedCol {
			case colServerName:
				width = colServerNameSize
			case colServerRegion:
				width = colServerRegionSize
			case colServerMap:
				width = colServerMapSize
			case colServerPlayers:
				width = colServerPlayersSize
			case colServerPing:
				width = colServerPingSize
			case colServerUptime:
				width = colServerUptimeSize
			case colServerFPS:
				width = colServerFPSSize
			case colServerCPU:
				width = colServerCPUSize
			case colServerInRate:
				width = colServerInRateSize
			case colServerOutRate:
				width = colServerOutRateSize
			case colServerConnects:
				width = colServerConnectsSize
			}

			switch {
			case row == table.HeaderRow:
				return styles.HeaderStyleBlu
			case currentIdx == row && col != 0:
				return styles.SelectedCellStyleNameBlu.Width(int(width))
			case row%2 == 0:
				return styles.PlayerTableRow.Width(int(width))
			default:
				return styles.PlayerTableRowOdd.Width(int(width))
			}
		}).
		String()

	m.viewport.SetContent(content)
	return m.viewport.View()
}

func (m *serverTableModel) currentRowIndex() int {
	for rowIdx, server := range m.data.servers {
		if server.HostPort == m.selectedsServer {
			return rowIdx
		}
	}

	return 0
}
