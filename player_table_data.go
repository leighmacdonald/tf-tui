package main

import (
	"cmp"
	"fmt"
	"strconv"
	"strings"

	"github.com/leighmacdonald/tf-tui/styles"
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

func NewPlayerTableData(playersUpdate []Player, team Team, cols ...playerTableColumn) PlayerTableData {
	data := PlayerTableData{
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

	return data
}

// PlayerTableData implements the table.Data interface to provide table data.
type PlayerTableData struct {
	players []Player
	// Defines both the columns show and the order they are rendered.
	enabledColumns []playerTableColumn
	sortColumn     playerTableColumn
	asc            bool
}

func (m *PlayerTableData) Headers() []string {
	var headers []string
	for _, col := range m.enabledColumns {
		switch col {
		case playerUID:
			headers = append(headers, "UID")
		case playerName:
			headers = append(headers, "Name")
		case playerScore:
			headers = append(headers, "Score")
		case playerDeaths:
			headers = append(headers, "Deaths")
		case playerPing:
			headers = append(headers, "Ping")
		case playerMeta:
			headers = append(headers, "Meta")
		}
	}

	return headers
}

func (m *PlayerTableData) Sort(column playerTableColumn, asc bool) {
	m.sortColumn = column
	m.asc = asc

	slices.SortStableFunc(m.players, func(a, b Player) int {
		switch m.sortColumn {
		case playerUID:
			return cmp.Compare(a.UserID, b.UserID)
		case playerName:
			av, _ := strconv.Atoi(a.Name)
			bv, _ := strconv.Atoi(b.Name)
			return cmp.Compare(av, bv)
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

func (m *PlayerTableData) At(row int, col int) string {
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
		return player.Name
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

func (m *PlayerTableData) Rows() int {
	return len(m.players)
}

func (m *PlayerTableData) Columns() int {
	return len(m.enabledColumns)
}

func (m *PlayerTableData) metaColumn(player Player) string {
	var afflictions []string
	if len(player.meta.Bans) > 0 {
		afflictions = append(afflictions, styles.IconBans)
	}

	if player.meta.NumberOfVacBans > 0 {
		afflictions = append(afflictions, styles.IconVac)
	}

	if len(afflictions) == 0 {
		afflictions = append(afflictions, styles.IconCheck)
	}

	return strings.Join(afflictions, " ")
}
