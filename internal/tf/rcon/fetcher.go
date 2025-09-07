package rcon

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
	"github.com/leighmacdonald/tf-tui/internal/tf"
)

var ErrDumpQuery = errors.New("failed to perform dump query")

func NewFetcher(address string, password string, serverMode bool) Fetcher {
	return Fetcher{
		Address:    address,
		Password:   password,
		serverMode: serverMode,
		g15re:      regexp.MustCompile(`^(m_szName|m_iPing|m_iScore|m_iDeaths|m_bConnected|m_iTeam|m_bAlive|m_iHealth|m_iAccountID|m_bValid|m_iUserID)\[(\d+)]\s(integer|bool|string)\s\((.+?)?\)$`),
		// CPU    In_(KB/s)  Out_(KB/s)  Uptime  Map_changes  FPS      Players  Connects
		// 0.00   82.99      619.13      287     14           66.67    64       900
		statsRe: regexp.MustCompile(`(\d+\.\d{1,2})\s+(\d+\.\d{1,2})\s+(\d+\.\d{1,2})\s+(\d+)\s+(\d+)\s+(\d+\.\d{1,2})\s+(\d+)\s+(\d+)`),
	}
}

type Fetcher struct {
	Address    string
	Password   string
	lastUpdate tf.DumpPlayer
	lastStats  tf.Stats
	serverMode bool
	g15re      *regexp.Regexp
	statsRe    *regexp.Regexp
}

func (f Fetcher) Fetch(ctx context.Context) (tf.DumpPlayer, tf.Stats, error) {
	command := "status"
	if f.serverMode {
		command = "stats;" + command
	} else {
		command = "g15_dumpplayer;" + command
	}

	response, errExec := New(f.Address, f.Password).Exec(ctx, command, true)
	if errExec != nil {
		if f.lastUpdate.SteamID[0].Valid() {
			return f.lastUpdate, f.lastStats, nil
		}
		if len(os.Getenv("DEBUG")) == 0 {
			return tf.DumpPlayer{}, tf.Stats{}, errors.Join(errExec, ErrDumpQuery)
		}
		// FIXME remove this test data generation eventually
		var data tf.DumpPlayer
		for playerIdx := range 24 {
			data.SteamID[playerIdx] = steamid.New(76561197960265730 + playerIdx)
			data.UserID[playerIdx] = playerIdx + 1
			data.Score[playerIdx] = playerIdx
			data.Ping[playerIdx] = playerIdx
			data.Deaths[playerIdx] = playerIdx
			if playerIdx%2 == 0 {
				data.Team[playerIdx] = tf.BLU
			} else {
				data.Team[playerIdx] = tf.RED
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

		return data, tf.Stats{}, nil
	}

	var (
		dump  tf.DumpPlayer
		stats tf.Stats
	)

	if f.serverMode {
		if match := f.statsRe.FindStringSubmatch(response); len(match) > 0 {
			stats = f.parseStats(match)
			f.lastStats = stats
		}

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
				dump.Team[idx] = tf.BLU
			} else {
				dump.Team[idx] = tf.RED
			}
		}
	} else {
		dump = f.parsePlayerState(strings.NewReader(response))
	}

	f.lastUpdate = dump

	return dump, stats, nil
}

// CPU    In_(KB/s)  Out_(KB/s)  Uptime  Map_changes  FPS      Players  Connects
// 49.76  80.38      1003.97     113     6            66.67    64       395.
func (f *Fetcher) parseStats(match []string) tf.Stats {
	cpu, errCPU := strconv.ParseFloat(match[1], 32)
	if errCPU != nil {
		slog.Error("Failed to parse CPU", slog.String("error", errCPU.Error()))
	}

	inKBs, errInKBs := strconv.ParseFloat(match[2], 32)
	if errInKBs != nil {
		slog.Error("Failed to parse in kbs", slog.String("error", errInKBs.Error()))
	}

	outKBs, errOutKBs := strconv.ParseFloat(match[3], 32)
	if errOutKBs != nil {
		slog.Error("Failed to parse out kbs", slog.String("error", errOutKBs.Error()))
	}

	uptime, errUptime := strconv.ParseInt(match[4], 10, 64)
	if errUptime != nil {
		slog.Error("Failed to parse uptime", slog.String("error", errUptime.Error()))
	}

	mapChanges, errMapChanges := strconv.ParseInt(match[5], 10, 64)
	if errMapChanges != nil {
		slog.Error("Failed to parse map changes", slog.String("error", errMapChanges.Error()))
	}

	fps, errFPS := strconv.ParseFloat(match[6], 32)
	if errFPS != nil {
		slog.Error("Failed to parse fps", slog.String("error", errFPS.Error()))
	}

	players, errPlayers := strconv.ParseInt(match[7], 10, 64)
	if errPlayers != nil {
		slog.Error("Failed to parse players", slog.String("error", errPlayers.Error()))
	}

	connects, errConnects := strconv.ParseInt(match[8], 10, 64)
	if errConnects != nil {
		slog.Error("Failed to parse connects", slog.String("error", errConnects.Error()))
	}

	return tf.Stats{
		CPU:        float32(cpu),
		InKBs:      float32(inKBs),
		OutKBs:     float32(outKBs),
		FPS:        float32(fps),
		Uptime:     int(uptime),
		MapChanges: int(mapChanges),
		Players:    int(players),
		Connects:   int(connects),
	}
}

// parsePlayerState provides the ability to parse the output of the `g15_dumpplayer` command into a PlayerState struct.
// This functionality requires the `-g15` launch parameter for TF2 to be set.
func (f *Fetcher) parsePlayerState(reader io.Reader) tf.DumpPlayer {
	var (
		data    tf.DumpPlayer
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
			data.Team[index] = tf.Team(parseInt(value, 0))
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
