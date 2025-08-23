package ui

import (
	"cmp"
	"fmt"
	"strings"

	"github.com/leighmacdonald/tf-tui/tf"
	"github.com/leighmacdonald/tf-tui/ui/styles"
	zone "github.com/lrstanley/bubblezone"
	"golang.org/x/exp/slices"
)

func NewTablePlayerData(parentZoneID string, playersUpdate Players, team tf.Team, cols ...playerTableCol) *TablePlayerData {
	data := TablePlayerData{
		zoneID:         parentZoneID,
		enabledColumns: []playerTableCol{ColMeta, ColName, ColScore, ColDeaths, ColPing},
	}

	if len(cols) > 0 {
		data.enabledColumns = cols
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

// TablePlayerData implements the table.Data interface to provide table data.
type TablePlayerData struct {
	players Players
	zoneID  string
	// Defines both the columns shown and the order they are rendered.
	enabledColumns []playerTableCol
	sortColumn     playerTableCol
	asc            bool
}

func (m *TablePlayerData) Headers() []string {
	var headers []string
	for _, col := range m.enabledColumns {
		switch col {
		case ColUID:
			headers = append(headers, zone.Mark(m.zoneID+"uid", "UID"))
		case ColName:
			headers = append(headers, zone.Mark(m.zoneID+"name", "Name"))
		case ColScore:
			headers = append(headers, zone.Mark(m.zoneID+"score", "Score"))
		case ColDeaths:
			headers = append(headers, zone.Mark(m.zoneID+"deaths", "Deaths"))
		case ColPing:
			headers = append(headers, zone.Mark(m.zoneID+"ping", "Ping"))
		case ColMeta:
			headers = append(headers, zone.Mark(m.zoneID+"meta", "Meta"))
		}
	}

	return headers
}

func (m *TablePlayerData) Sort(column playerTableCol, asc bool) {
	m.sortColumn = column
	m.asc = asc

	slices.SortStableFunc(m.players, func(a, b Player) int { //nolint:varnamelen
		switch m.sortColumn {
		case ColUID:
			return cmp.Compare(a.UserID, b.UserID)
		case ColName:
			return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		case ColScore:
			return cmp.Compare(a.Score, b.Score)
		case ColDeaths:
			return cmp.Compare(a.Deaths, b.Deaths)
		case ColPing:
			return cmp.Compare(a.Ping, b.Ping)
		case ColMeta:
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

func (m *TablePlayerData) At(row int, col int) string {
	if col > len(m.enabledColumns)-1 {
		return "oobcol"
	}
	if row > len(m.players)-1 {
		return "oobplr"
	}
	curCol := m.enabledColumns[col]
	player := m.players[row]
	switch curCol {
	case ColUID:
		return fmt.Sprintf("%d", player.UserID)
	case ColName:
		name := player.Name
		if name == "" {
			name = player.PersonaName
		}
		if name == "" {
			name = player.SteamID.String()
		}

		return zone.Mark(m.zoneID+player.SteamID.String(), name)
	case ColScore:
		return fmt.Sprintf("%d", player.Score)
	case ColDeaths:
		return fmt.Sprintf("%d", player.Deaths)
	case ColPing:
		return fmt.Sprintf("%d", player.Ping)
	case ColMeta:
		return m.metaColumn(player)
	}

	return "?"
}

func (m *TablePlayerData) Rows() int {
	return len(m.players)
}

func (m *TablePlayerData) Columns() int {
	return len(m.enabledColumns)
}

func (m *TablePlayerData) metaColumn(player Player) string {
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
