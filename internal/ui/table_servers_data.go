package ui

import (
	"cmp"
	"fmt"
	"strings"

	zone "github.com/lrstanley/bubblezone"
	"golang.org/x/exp/slices"
)

func newTableServerData(parentZoneID string, data []Snapshot, cols ...serverTableCol) *serverTableData {
	var enabledCols []serverTableCol
	switch {
	case len(cols) > 0:
		enabledCols = cols
	default:
		enabledCols = defaultServerTableColumns
	}

	return &serverTableData{
		zoneID:         parentZoneID,
		servers:        data,
		enabledColumns: enabledCols,
		sortColumn:     colServerName,
		asc:            true,
	}
}

type serverTableData struct {
	zoneID         string
	servers        []Snapshot
	enabledColumns []serverTableCol
	sortColumn     serverTableCol
	asc            bool
}

func (m *serverTableData) Headers() []string {
	var headers []string
	for _, col := range m.enabledColumns {
		switch col {
		case colServerName:
			headers = append(headers, zone.Mark(m.zoneID+"server", "Server"))
		case colServerRegion:
			headers = append(headers, zone.Mark(m.zoneID+"reg", "Reg"))
		case colServerMap:
			headers = append(headers, zone.Mark(m.zoneID+"map", "Map"))
		case colServerPlayers:
			headers = append(headers, zone.Mark(m.zoneID+"players", "Players"))
		case colServerPing:
			headers = append(headers, zone.Mark(m.zoneID+"ping", "Ping"))
		}
	}

	return headers
}

func (m *serverTableData) Sort(column serverTableCol, asc bool) {
	m.sortColumn = column
	m.asc = asc

	slices.SortStableFunc(m.servers, func(a Snapshot, b Snapshot) int { //nolint:varnamelen
		switch m.sortColumn {
		case colServerName:
			return strings.Compare(strings.ToLower(a.Server.Hostname), strings.ToLower(b.Server.Hostname))
		case colServerRegion:
			return strings.Compare(strings.ToLower(a.Server.Region), strings.ToLower(b.Server.Region))
		case colServerMap:
			return strings.Compare(strings.ToLower(a.Server.Map), strings.ToLower(b.Server.Map))
		case colServerPlayers:
			return cmp.Compare(len(a.Server.Players), len(b.Server.Players))
		case colServerPing:
			return cmp.Compare(a.Server.Ping, b.Server.Ping)
		default:
			return 0
		}
	})

	if m.asc {
		slices.Reverse(m.servers)
	}
}

func (m *serverTableData) At(row int, col int) string {
	if col > len(m.enabledColumns)-1 {
		return "oobcol"
	}
	if row > len(m.servers)-1 {
		return "oobrow"
	}
	curCol := m.enabledColumns[col]
	server := m.servers[row]
	switch curCol {
	case colServerName:
		return zone.Mark(m.zoneID+server.Server.Hostname, server.Server.Hostname)
	case colServerRegion:
		return zone.Mark(m.zoneID+server.Server.Region, server.Server.Region)
	case colServerMap:
		return server.Server.Map
	case colServerPlayers:
		return fmt.Sprintf("%d", len(server.Server.Players))
	case colServerPing:
		return fmt.Sprintf("%d", int(server.Server.Ping))
	}

	return "?"
}

func (m *serverTableData) Rows() int {
	return len(m.servers)
}

func (m *serverTableData) Columns() int {
	return len(m.enabledColumns)
}
