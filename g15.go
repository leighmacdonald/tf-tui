package main

import (
	"bufio"
	"context"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

var g15re = regexp.MustCompile(`^(m_szName|m_iPing|m_iScore|m_iDeaths|m_bConnected|m_iTeam|m_bAlive|m_iHealth|m_iAccountID|m_bValid|m_iUserID)\[(\d+)]\s(integer|bool|string)\s\((.+?)?\)$`)

const MaxDataSize = 102

type G15PlayerState struct {
	Names     [MaxDataSize]string
	Ping      [MaxDataSize]int
	Score     [MaxDataSize]int
	Deaths    [MaxDataSize]int
	Connected [MaxDataSize]bool
	Team      [MaxDataSize]int
	Alive     [MaxDataSize]bool
	Health    [MaxDataSize]int
	SteamID   [MaxDataSize]steamid.SteamID
	Valid     [MaxDataSize]bool
	UserID    [MaxDataSize]int
}

var lastUpdate G15PlayerState

func fetchPlayerState(ctx context.Context, address string, password string) (G15PlayerState, error) {
	conn := newRconConnection(address, password)
	response, errExec := conn.exec(ctx, "g15_dumpplayer", true)
	if errExec != nil {
		if lastUpdate.SteamID[0].Valid() {
			return lastUpdate, nil
		}
		var data G15PlayerState
		for i := range 23 {
			data.SteamID[i] = steamid.New(76561197960265730 + i)
			data.Names[i] = data.SteamID[i].String()
			data.UserID[i] = i + 1
			data.Score[i] = i
			data.Ping[i] = i
			data.Deaths[i] = i
			if i%2 == 0 {
				data.Team[i] = BLU
			} else {
				data.Team[i] = RED
			}
			if i == 0 {
				data.SteamID[0] = steamid.New(76561197960265730)
			}
		}
		return data, nil
		//return G15PlayerState{}, errExec
	}

	dump, err := parsePlayerState(strings.NewReader(response))
	if err != nil {
		return G15PlayerState{}, err
	}

	lastUpdate = dump

	return dump, nil
}

// parsePlayerState provides the ability to parse the output of the `g15_dumpplayer` command into a PlayerState struct.
// This functionality requires the `-g15` launch parameter for TF2 to be set.
func parsePlayerState(reader io.Reader) (G15PlayerState, error) {
	var (
		data    G15PlayerState
		scanner = bufio.NewScanner(reader)
	)

	for scanner.Scan() {
		matches := g15re.FindStringSubmatch(strings.Trim(scanner.Text(), "\r"))
		if len(matches) == 0 {
			continue
		}

		index := parseInt(matches[2], -1)
		if index < 0 {
			continue
		}

		value := ""
		if len(matches) == 5 {
			value = matches[4]
		}

		switch matches[1] {
		case "m_szName":
			data.Names[index] = value
		case "m_iPing":
			data.Ping[index] = parseInt(value, 0)
		case "m_iScore":
			data.Score[index] = parseInt(value, 0)
		case "m_iDeaths":
			data.Deaths[index] = parseInt(value, 0)
		case "m_bConnected":
			data.Connected[index] = parseBool(value)
		case "m_iTeam":
			data.Team[index] = parseInt(value, 0)
		case "m_bAlive":
			data.Alive[index] = parseBool(value)
		case "m_iHealth":
			data.Health[index] = parseInt(value, 0)
		case "m_iAccountID":
			data.SteamID[index] = steamid.New(int32(parseInt(value, 0)))
		case "m_bValid":
			data.Valid[index] = parseBool(value)
		case "m_iUserID":
			data.UserID[index] = parseInt(value, -1)
		}
	}

	if err := scanner.Err(); err != nil {
		return data, err
	}

	return data, nil
}

func parseInt(s string, def int) int {
	index, errIndex := strconv.ParseInt(s, 10, 32)
	if errIndex != nil {
		return def
	}

	return int(index)
}

func parseBool(s string) bool {
	val, errParse := strconv.ParseBool(s)
	if errParse != nil {
		return false
	}

	return val
}
