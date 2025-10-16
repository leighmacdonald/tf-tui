package ui

import (
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
	colServerName,
	colServerMap,
	colServerRegion,
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
	return &serverTableModel{
		zoneID: zone.NewPrefix(),
		table:  newUnstyledTable(),
		data:   newTableServerData("", nil, defaultServerTableColumns...),
	}
}

type serverTableModel struct {
	zoneID         string
	table          *table.Table
	data           *serverTableData
	selectedServer string
	width          int
	height         int
}

func (m *serverTableModel) Init() tea.Cmd {
	return nil
}

func (m *serverTableModel) Update(msg tea.Msg) (*serverTableModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ContentViewPortHeightMsg:
		m.width = msg.width
		m.height = msg.height
	case []Snapshot:
		m.data = newTableServerData(m.zoneID, msg)
		m.data.Sort(m.data.sortColumn, m.data.asc)
		m.table.Data(m.data)
	}

	return m, nil
}

func (m *serverTableModel) View() string {
	return m.table.
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
