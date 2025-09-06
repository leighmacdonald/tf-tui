package tf

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/leighmacdonald/steamid/v4/extra"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/tf/rcon"
)

type DumpPlayer struct {
	Names     [MaxPlayerCount]string
	Ping      [MaxPlayerCount]int
	Score     [MaxPlayerCount]int
	Deaths    [MaxPlayerCount]int
	Connected [MaxPlayerCount]bool
	Team      [MaxPlayerCount]Team
	Alive     [MaxPlayerCount]bool
	Health    [MaxPlayerCount]int
	SteamID   [MaxPlayerCount]steamid.SteamID
	Valid     [MaxPlayerCount]bool
	UserID    [MaxPlayerCount]int
	Loss      [MaxPlayerCount]int
	State     [MaxPlayerCount]string
	Address   [MaxPlayerCount]string
	Time      [MaxPlayerCount]int
}

var ErrDumpQuery = errors.New("failed to query g15_dumpplayer")

func NewDumpFetcher(address string, password string, serverMode bool) DumpFetcher {
	return DumpFetcher{
		Address:    address,
		Password:   password,
		serverMode: serverMode,
		g15re:      regexp.MustCompile(`^(m_szName|m_iPing|m_iScore|m_iDeaths|m_bConnected|m_iTeam|m_bAlive|m_iHealth|m_iAccountID|m_bValid|m_iUserID)\[(\d+)]\s(integer|bool|string)\s\((.+?)?\)$`),
	}
}

type DumpFetcher struct {
	Address    string
	Password   string
	lastUpdate DumpPlayer
	serverMode bool
	g15re      *regexp.Regexp
}

func (f DumpFetcher) Fetch(ctx context.Context) (DumpPlayer, error) {
	command := "status"
	if f.serverMode {
		command = "status"
	} else {
		command = "g15_dumpplayer;" + command
	}

	response, errExec := rcon.New(f.Address, f.Password).Exec(ctx, command, true)
	if errExec != nil {
		if f.lastUpdate.SteamID[0].Valid() {
			return f.lastUpdate, nil
		}
		if len(os.Getenv("DEBUG")) == 0 {
			return DumpPlayer{}, errors.Join(errExec, ErrDumpQuery)
		}
		// FIXME remove this test data generation eventually
		var data DumpPlayer
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
			if playerIdx == 1 {
				data.SteamID[playerIdx] = steamid.New(76561198084134025)
			}
		}

		return data, nil
	}

	var dump DumpPlayer

	if f.serverMode {
		status, errStatus := extra.ParseStatus(response, true)
		if errStatus != nil {
			slog.Error("failed to parse status", slog.String("error", errStatus.Error()))
		}

		slices.SortStableFunc(status.Players, func(a extra.Player, b extra.Player) int {
			if a.UserID > b.UserID {
				return 1
			} else if a.UserID < b.UserID {
				return -1
			}

			return 0
		})

		for idx, player := range status.Players {
			dump.Connected[idx] = player.State == "active"
			dump.Names[idx] = player.Name
			dump.Ping[idx] = player.Ping
			dump.SteamID[idx] = player.SID
			dump.Address[idx] = fmt.Sprintf("%s:%d", player.IP.String(), player.Port)
			dump.Loss[idx] = player.Loss
			dump.State[idx] = player.State
			dump.Time[idx] = int(player.ConnectedTime.Seconds())
			dump.UserID[idx] = player.UserID
			// We have no way of knowing their real teams in server mode.
			if idx%2 == 0 {
				dump.Team[idx] = BLU
			} else {
				dump.Team[idx] = RED
			}
		}
	} else {
		dump = f.parsePlayerState(strings.NewReader(response))
	}

	f.lastUpdate = dump

	return dump, nil
}

// parsePlayerState provides the ability to parse the output of the `g15_dumpplayer` command into a PlayerState struct.
// This functionality requires the `-g15` launch parameter for TF2 to be set.
func (f *DumpFetcher) parsePlayerState(reader io.Reader) DumpPlayer {
	var (
		data    DumpPlayer
		scanner = bufio.NewScanner(reader)
	)

	for scanner.Scan() {
		matches := f.g15re.FindStringSubmatch(strings.Trim(scanner.Text(), "\r"))
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
			data.Team[index] = Team(parseInt(value, 0))
		case "m_bAlive":
			data.Alive[index] = parseBool(value)
		case "m_iHealth":
			data.Health[index] = parseInt(value, 0)
		case "m_iAccountID":
			data.SteamID[index] = steamid.New(parseInt(value, 0))
		case "m_bValid":
			data.Valid[index] = parseBool(value)
		case "m_iUserID":
			data.UserID[index] = parseInt(value, -1)
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error("Error scanning g15 response", slog.String("error", err.Error()))

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
