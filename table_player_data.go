package main

import (
	"cmp"
	"fmt"
	"strings"

	"github.com/leighmacdonald/tf-tui/styles"
	zone "github.com/lrstanley/bubblezone"
	"golang.org/x/exp/slices"
)

type playerTableColumn int

const (
	playerUID playerTableColumn = iota
	playerName
	playerScore
	playerDeaths
	playerPing
	playerMeta
)

func NewTablePlayerData(parentZoneID string, playersUpdate Players, team Team, cols ...playerTableColumn) *TablePlayerData {
	data := TablePlayerData{
		zoneID:         parentZoneID,
		enabledColumns: []playerTableColumn{playerMeta, playerName, playerScore, playerDeaths, playerPing},
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
	enabledColumns []playerTableColumn
	sortColumn     playerTableColumn
	asc            bool
}

func (m *TablePlayerData) Headers() []string {
	var headers []string
	for _, col := range m.enabledColumns {
		switch col {
		case playerUID:
			headers = append(headers, zone.Mark(m.zoneID+"uid", "UID"))
		case playerName:
			headers = append(headers, zone.Mark(m.zoneID+"name", "Name"))
		case playerScore:
			headers = append(headers, zone.Mark(m.zoneID+"score", "Score"))
		case playerDeaths:
			headers = append(headers, zone.Mark(m.zoneID+"deaths", "Deaths"))
		case playerPing:
			headers = append(headers, zone.Mark(m.zoneID+"ping", "Ping"))
		case playerMeta:
			headers = append(headers, zone.Mark(m.zoneID+"meta", "Meta"))
		}
	}

	return headers
}

func (m *TablePlayerData) Sort(column playerTableColumn, asc bool) {
	m.sortColumn = column
	m.asc = asc

	slices.SortStableFunc(m.players, func(a, b Player) int { //nolint:varnamelen
		switch m.sortColumn {
		case playerUID:
			return cmp.Compare(a.UserID, b.UserID)
		case playerName:
			return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		case playerScore:
			return cmp.Compare(a.Score, b.Score)
		case playerDeaths:
			return cmp.Compare(a.Deaths, b.Deaths)
		case playerPing:
			return cmp.Compare(a.Ping, b.Ping)
		case playerMeta:
			av := len(a.meta.Bans) + int(a.meta.NumberOfVacBans)
			bv := len(b.meta.Bans) + int(b.meta.NumberOfVacBans)

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
	case playerUID:
		return fmt.Sprintf("%d", player.UserID)
	case playerName:
		name := player.Name
		if name == "" {
			name = player.meta.PersonaName
		}
		if name == "" {
			name = player.SteamID.String()
		}

		return zone.Mark(m.zoneID+player.SteamID.String(), name)
	case playerScore:
		return fmt.Sprintf("%d", player.Score)
	case playerDeaths:
		return fmt.Sprintf("%d", player.Deaths)
	case playerPing:
		return fmt.Sprintf("%d", player.Ping)
	case playerMeta:
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
	if len(player.meta.Bans) > 0 {
		afflictions = append(afflictions, styles.IconBans)
	}

	if player.meta.NumberOfVacBans > 0 {
		afflictions = append(afflictions, styles.IconVac)
	}

	// if len(afflictions) == 0 {
	//	afflictions = append(afflictions, styles.IconCheck)
	//}

	if len(player.meta.CompetitiveTeams) > 0 {
		afflictions = append(afflictions, styles.IconComp)
	}

	return strings.Join(afflictions, " ")
}
