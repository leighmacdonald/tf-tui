package component

import (
	"cmp"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/leighmacdonald/tf-tui/internal/ui/model"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
	zone "github.com/lrstanley/bubblezone"
	"golang.org/x/exp/slices"
)

func newTableServerData(parentZoneID string, data []model.Snapshot, cols ...serverTableCol) *serverTableData {
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
	servers        []model.Snapshot
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
			headers = append(headers, zone.Mark(m.zoneID+"cc", "CC"))
		case colServerMap:
			headers = append(headers, zone.Mark(m.zoneID+"map", "Map"))
		case colServerPlayers:
			headers = append(headers, zone.Mark(m.zoneID+"players", "Players"))
		case colServerPing:
			headers = append(headers, zone.Mark(m.zoneID+"ping", "AvgPing"))
		case colServerUptime:
			headers = append(headers, zone.Mark(m.zoneID+"up", "Up"))
		case colServerFPS:
			headers = append(headers, zone.Mark(m.zoneID+"fps", "FPS"))
		case colServerCPU:
			headers = append(headers, zone.Mark(m.zoneID+"cpu", "CPU %"))
		case colServerInRate:
			headers = append(headers, zone.Mark(m.zoneID+"rate_in", "In KB/s"))
		case colServerOutRate:
			headers = append(headers, zone.Mark(m.zoneID+"rate_out", "Out KB/s"))
		case colServerConnects:
			headers = append(headers, zone.Mark(m.zoneID+"conns", "Conns"))
		}
	}

	return headers
}

func (m *serverTableData) Sort(column serverTableCol, asc bool) {
	m.sortColumn = column
	m.asc = asc

	slices.SortStableFunc(m.servers, func(a model.Snapshot, b model.Snapshot) int { //nolint:varnamelen
		switch m.sortColumn {
		case colServerName:
			return strings.Compare(strings.ToLower(a.Status.ServerName), strings.ToLower(b.Status.ServerName))
		case colServerRegion:
			return strings.Compare(strings.ToLower(a.Server.Region), strings.ToLower(b.Server.Region))
		case colServerMap:
			return strings.Compare(strings.ToLower(a.Status.Map), strings.ToLower(b.Status.Map))
		case colServerPlayers:
			return cmp.Compare(len(a.Server.Players), len(b.Status.Players))
		case colServerPing:
			return cmp.Compare(a.Server.Ping, b.Server.Ping)
		case colServerUptime:
			return cmp.Compare(a.Status.Stats.Uptime, b.Status.Stats.Uptime)
		case colServerFPS:
			return cmp.Compare(a.Status.Stats.FPS, b.Status.Stats.FPS)
		case colServerCPU:
			return cmp.Compare(a.Status.Stats.CPU, b.Status.Stats.CPU)
		case colServerInRate:
			return cmp.Compare(a.Status.Stats.InKBs, b.Status.Stats.InKBs)
		case colServerOutRate:
			return cmp.Compare(a.Status.Stats.OutKBs, b.Status.Stats.OutKBs)
		case colServerConnects:
			return cmp.Compare(a.Status.Stats.Connects, b.Status.Stats.Connects)
		default:
			return 0
		}
	})

	if m.asc {
		slices.Reverse(m.servers)
	}
}

func normalizeMapName(input string) string {
	if !strings.HasPrefix(input, "workshop/") {
		return input
	}

	input, _ = strings.CutPrefix(input, "workshop/")
	parts := strings.Split(input, ".")
	return parts[0]
}

func (m *serverTableData) At(row int, col int) string {
	if col > len(m.enabledColumns)-1 {
		return "oobcol"
	}
	if row > len(m.servers)-1 {
		return "oobrow"
	}
	curCol := m.enabledColumns[col]
	snapshot := m.servers[row]
	switch curCol {
	case colServerName:
		return zone.Mark(m.zoneID+snapshot.HostPort, snapshot.Status.ServerName)
	case colServerRegion:
		cc := snapshot.Server.Region
		if cc == "none" {
			cc = ""
		}
		return zone.Mark(m.zoneID+snapshot.Server.Region, styles.Flag(cc))
	case colServerMap:
		return normalizeMapName(snapshot.Status.Map)
	case colServerPlayers:
		return fmt.Sprintf("%d/%d", snapshot.Status.PlayersCount, snapshot.Status.PlayersMax)
	case colServerFPS:
		if snapshot.Status.Stats.FPS > 0 {
			return fmt.Sprintf("%2.2f", snapshot.Status.Stats.FPS)
		}
		return ""
	case colServerCPU:
		if snapshot.Status.Stats.CPU > 0 {
			return fmt.Sprintf("%2.2f", snapshot.Status.Stats.CPU)
		}
		return ""
	case colServerInRate:
		return fmt.Sprintf("%.2f", snapshot.Status.Stats.InKBs)
	case colServerOutRate:
		return fmt.Sprintf("%.2f", snapshot.Status.Stats.OutKBs)
	case colServerConnects:
		return fmt.Sprintf("%d", snapshot.Status.Stats.Connects)
	case colServerPing:
		avg := snapshot.AvgPing()
		if math.IsNaN(avg) || avg == 0 {
			return ""
		}

		return fmt.Sprintf("%.0fms", avg)
	case colServerUptime:
		uptime := time.Duration(snapshot.Status.Stats.Uptime) * time.Second

		return uptime.String()
	}

	return "?"
}

func (m *serverTableData) Rows() int {
	return len(m.servers)
}

func (m *serverTableData) Columns() int {
	return len(m.enabledColumns)
}
