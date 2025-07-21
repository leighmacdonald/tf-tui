package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var errFetchMetaProfile = errors.New("failed to fetch meta profile")

func NewAPIs(client *ClientWithResponses) APIs {
	return APIs{client: client}
}

type APIs struct {
	client *ClientWithResponses
}

func (a APIs) getMetaProfiles(ctx context.Context, steamIDs steamid.Collection) ([]MetaProfile, error) {
	if len(steamIDs) == 0 {
		return nil, nil
	}

	resp, errResp := a.client.MetaProfile(ctx, &MetaProfileParams{Steamids: steamIDs.ToStringSlice()})
	if errResp != nil {
		return nil, errors.Join(errResp, errFetchMetaProfile)
	}
	defer resp.Body.Close()

	parsed, errParse := ParseMetaProfileResponse(resp)
	if errParse != nil {
		return nil, errors.Join(errResp, errParse)
	}

	return *parsed.JSON200, nil
}

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
		for playerIdx := range 24 {
			data.SteamID[playerIdx] = steamid.New(76561197960265730 + playerIdx)

			data.UserID[playerIdx] = playerIdx + 1
			data.Score[playerIdx] = playerIdx
			data.Ping[playerIdx] = playerIdx
			data.Deaths[playerIdx] = playerIdx
			if playerIdx%2 == 0 {
				data.Team[playerIdx] = BLU
			} else {
				data.Team[playerIdx] = RED
			}
			if playerIdx == 0 {
				data.SteamID[0] = steamid.New(76561197960265730)
			}
			if playerIdx == 5 {
				data.SteamID[5] = steamid.New(76561197970669109)
			}
			if playerIdx == 6 {
				data.SteamID[playerIdx] = steamid.New(76561198044497183)
			}
			data.Names[playerIdx] = data.SteamID[playerIdx].String()
		}

		return data, nil
		// return G15PlayerState{}, errExec
	}

	dump := parsePlayerState(strings.NewReader(response))
	lastUpdate = dump

	return dump, nil
}

// parsePlayerState provides the ability to parse the output of the `g15_dumpplayer` command into a PlayerState struct.
// This functionality requires the `-g15` launch parameter for TF2 to be set.
func parsePlayerState(reader io.Reader) G15PlayerState {
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
		tea.Printf("Error scanning g15 response: %v", err)

		return data
	}

	return data
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

var (
	errRCONParse = errors.New("RCON parse error")
	errRCON      = errors.New("errors making rcon request")
)

type rconConnection struct {
	addr     string
	password string
	timeout  time.Duration
}

func newRconConnection(addr string, password string) rconConnection {
	return rconConnection{
		addr:     addr,
		password: password,
		timeout:  time.Second,
	}
}

func (r rconConnection) exec(ctx context.Context, cmd string, large bool) (string, error) {
	conn, errConn := rcon.Dial(ctx, r.addr, r.password, r.timeout)
	if errConn != nil {
		return "", errors.Join(errConn, fmt.Errorf("%w: %s", errRCON, r.addr))
	}
	defer conn.Close()

	if large {
		return r.rconLarge(conn, cmd)
	}

	return r.rcon(conn, cmd)
}

func (r rconConnection) rcon(conn *rcon.RemoteConsole, cmd string) (string, error) {
	cmdID, errWrite := conn.Write(cmd)
	if errWrite != nil {
		return "", errors.Join(errWrite, errRCON)
	}

	resp, respID, errRead := conn.Read()
	if errRead != nil {
		return "", errors.Join(errRead, errRCON)
	}

	if respID != cmdID {
		slog.Warn("Mismatched command response ID", slog.Int("req", cmdID), slog.Int("resp", respID))
	}

	return resp, nil
}

// rconLarge is used for rcon responses that exceed the size of a single rcon packet (g15_dumpplayer).
func (r rconConnection) rconLarge(conn *rcon.RemoteConsole, cmd string) (string, error) {
	cmdID, errWrite := conn.Write(cmd)
	if errWrite != nil {
		return "", errors.Join(errWrite, errRCON)
	}

	var response string

	for {
		resp, respID, errRead := conn.Read()
		if errRead != nil {
			return "", errors.Join(errRead, errRCON)
		}

		if cmdID == respID {
			s := len(resp)
			response += resp

			if s < 4000 {
				break
			}
		}
	}

	return response, nil
}
