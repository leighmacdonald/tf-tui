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

type PlayerTableData struct {
	players        []Player
	enabledColumns []playerTableColumn
	sortColumn     playerTableColumn
	asc            bool
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
			av := len(a.meta.Bans) + int(b.meta.NumberOfVacBans)
			bv := len(a.meta.Bans) + int(b.meta.NumberOfVacBans)

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
	player := m.players[row]
	switch col {
	case 0:
		return fmt.Sprintf("%d", player.UserID)
	case 1:
		return player.Name
	case 2:
		return fmt.Sprintf("%d", player.Score)
	case 3:
		return fmt.Sprintf("%d", player.Deaths)
	case 4:
		return fmt.Sprintf("%d", player.Ping)
	case 5:
		return m.metaColumn(player)
	}
	return "cell"
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
