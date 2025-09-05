package ui

import (
	"cmp"
	"fmt"
	"strconv"
	"strings"

	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
	zone "github.com/lrstanley/bubblezone"
	"golang.org/x/exp/slices"
)

var (
	defaultLocalColumns  = []playerTableCol{colMeta, colName, colScore, colDeaths, colPing}
	defaultServerColumns = []playerTableCol{colMeta, colName, colLoss, colPing, colAddress}
)

func newTablePlayerData(parentZoneID string, serverMode bool, playersUpdate Players, team tf.Team, cols ...playerTableCol) *tablePlayerData {
	var enabledCols []playerTableCol
	switch {
	case len(cols) > 0:
		enabledCols = cols
	case serverMode:
		enabledCols = defaultServerColumns
	default:
		enabledCols = defaultLocalColumns
	}

	data := tablePlayerData{
		zoneID:         parentZoneID,
		sortColumn:     colScore,
		asc:            true,
		serverMode:     serverMode,
		players:        Players{},
		enabledColumns: enabledCols,
	}

	for _, player := range playersUpdate {
		if !player.SteamID.Valid() {
			continue
		}
		if player.Team != team {
			continue
		}

		data.players = append(data.players, player)
	}

	return &data
}

// tablePlayerData implements the table.Data interface to provide table data.
type tablePlayerData struct {
	players Players
	zoneID  string
	// Defines both the columns shown and the order they are rendered.
	enabledColumns []playerTableCol
	sortColumn     playerTableCol
	asc            bool
	serverMode     bool
}

func (m *tablePlayerData) Headers() []string {
	var headers []string
	for _, col := range m.enabledColumns {
		switch col {
		case colUID:
			headers = append(headers, zone.Mark(m.zoneID+"uid", "UID"))
		case colName:
			headers = append(headers, zone.Mark(m.zoneID+"name", "Name"))
		case colScore:
			headers = append(headers, zone.Mark(m.zoneID+"score", "Score"))
		case colDeaths:
			headers = append(headers, zone.Mark(m.zoneID+"deaths", "Deaths"))
		case colPing:
			headers = append(headers, zone.Mark(m.zoneID+"ping", "Ping"))
		case colMeta:
			headers = append(headers, zone.Mark(m.zoneID+"meta", "Meta"))
		case colAddress:
			headers = append(headers, zone.Mark(m.zoneID+"address", "Address"))
		case colLoss:
			headers = append(headers, zone.Mark(m.zoneID+"loss", "Loss"))
		case colTime:
			headers = append(headers, zone.Mark(m.zoneID+"time", "Time"))
		}
	}

	return headers
}

func (m *tablePlayerData) Sort(column playerTableCol, asc bool) {
	m.sortColumn = column
	m.asc = asc

	slices.SortStableFunc(m.players, func(a, b Player) int { //nolint:varnamelen
		switch m.sortColumn {
		case colUID:
			return cmp.Compare(a.UserID, b.UserID)
		case colName:
			return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		case colScore:
			return cmp.Compare(a.Score, b.Score)
		case colDeaths:
			return cmp.Compare(a.Deaths, b.Deaths)
		case colPing:
			return cmp.Compare(a.Ping, b.Ping)
		case colAddress:
			// TODO proper IP numerical compare
			return strings.Compare(a.Address, b.Address)
		case colLoss:
			return cmp.Compare(a.Loss, b.Loss)
		case colTime:
			return cmp.Compare(a.Time, b.Time)
		case colMeta:
			av := len(a.Bans) + int(a.NumberOfVacBans)
			bv := len(b.Bans) + int(b.NumberOfVacBans)

			return cmp.Compare(bv, av)
		default:
			return 0
		}
	})

	if m.asc {
		slices.Reverse(m.players)
	}
}

func (m *tablePlayerData) At(row int, col int) string {
	if col > len(m.enabledColumns)-1 {
		return "oobcol"
	}
	if row > len(m.players)-1 {
		return "oobplr"
	}
	curCol := m.enabledColumns[col]
	player := m.players[row]
	switch curCol {
	case colUID:
		return fmt.Sprintf("%d", player.UserID)
	case colName:
		name := player.Name
		if name == "" {
			name = player.PersonaName
		}
		if name == "" {
			name = player.SteamID.String()
		}

		return zone.Mark(m.zoneID+player.SteamID.String(), name)
	case colScore:
		return fmt.Sprintf("%d", player.Score)
	case colDeaths:
		return fmt.Sprintf("%d", player.Deaths)
	case colPing:
		return fmt.Sprintf("%d", player.Ping)
	case colMeta:
		return m.metaColumn(player)
	case colAddress:
		parts := strings.Split(player.Address, ":")
		if len(parts) > 0 {
			return parts[0]
		}

		return ""
	case colLoss:
		return strconv.Itoa(player.Loss)
	case colTime:
		return strconv.Itoa(player.Time)
	}

	return "?"
}

func (m *tablePlayerData) Rows() int {
	return len(m.players)
}

func (m *tablePlayerData) Columns() int {
	return len(m.enabledColumns)
}

func (m *tablePlayerData) metaColumn(player Player) string {
	var afflictions []string
	if len(player.Bans) > 0 {
		afflictions = append(afflictions, styles.IconBans)
	}

	if player.NumberOfVacBans > 0 {
		afflictions = append(afflictions, styles.IconVac)
	}

	// if len(afflictions) == 0 {
	//	afflictions = append(afflictions, styles.IconCheck)
	//}

	if len(player.CompetitiveTeams) > 0 {
		afflictions = append(afflictions, styles.IconComp)
	}

	// FIXME
	// if len(player.BDMatches) > 0 {
	// 	afflictions = append(afflictions, styles.IconBD)
	// }

	return strings.Join(afflictions, " ")
}
