// Package tf handles interfacing with the game client.
package tf

import (
	"log/slog"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/leighmacdonald/steamid/v4/extra"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	// Max number of players supported by the game.
	MaxPlayerCount = 102
	// In game max message length.
	MaxMessageLength = 127
)

type Team int

const (
	UNASSIGNED Team = iota
	SPEC
	BLU
	RED
)

type KickReason string

const (
	KickReasonIdle     KickReason = "idle"
	KickReasonScamming KickReason = "scamming"
	KickReasonCheating KickReason = "cheating"
	KickReasonOther    KickReason = "other"
)

type ChatDest string

const (
	ChatDestAll   ChatDest = "all"
	ChatDestTeam  ChatDest = "team"
	ChatDestParty ChatDest = "party"
)

type ChatType int

const (
	AllChat ChatType = iota
	TeamChat
	PartyChat
)

// DumpPlayer holds the data returned from the `g15_dumpplayer` rcon command.
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

// Stats holds the data returned from the `stats` rcon command.
type Stats struct {
	CPU        float32
	InKBs      float32
	OutKBs     float32
	FPS        float32
	Uptime     int
	MapChanges int
	Players    int
	Connects   int
}

type Status struct {
	extra.Status
	Stats  Stats
	Region string
}

type GamePlugin struct {
	Index   int
	Name    string
	Version string
	Author  string
}

// [03] TF2 Tools (1.13.0.7251) by AlliedModders LLC
// 45 "NativeVotes MapChooser" (1.8.0 beta 1-ut) by AlliedModders LLC and Powerlord
var lineMatcher = regexp.MustCompile(`^\s+\[?(\d+)\]?\s?(.+?)\((.+?)\)\sby\s(.+?)$`) //noling:gochecknoglobals

// ParseGamePlugins transforms the output of the `sm plugins list` or `meta list` into a slice of GamePlugin.
func ParseGamePlugins(body string, sortName bool) []GamePlugin {
	var plugins []GamePlugin
	for line := range strings.Lines(body) {
		if line == "" {
			continue
		}

		match := lineMatcher.FindStringSubmatch(strings.TrimRight(line, "\n"))
		if len(match) != 5 {
			continue
		}

		index, errIndex := strconv.ParseUint(match[1], 10, 64)
		if errIndex != nil {
			slog.Error("Failed to parse index", slog.String("error", errIndex.Error()))

			continue
		}

		plugins = append(plugins, GamePlugin{
			Index:   int(index),
			Name:    strings.TrimSpace(strings.ReplaceAll(match[2], "\"", "")),
			Version: match[3],
			Author:  match[4],
		})
	}

	if sortName {
		slices.SortStableFunc(plugins, func(a, b GamePlugin) int {
			return strings.Compare(a.Name, b.Name)
		})
	}

	return plugins
}

type CVar struct {
	Name        string
	Cmd         bool
	Value       string
	Flags       []string
	Description string
}

type CVarList []CVar

func (c CVarList) Filter(prefix string) CVarList {
	var matching CVarList
	prefix = strings.ToLower(prefix)
	for _, cmd := range c {
		if prefix == "" || strings.HasPrefix(cmd.Name, prefix) {
			matching = append(matching, cmd)
		}
	}

	return matching
}

func (c CVarList) Names() []string {
	var names []string
	for _, name := range c {
		names = append(names, name.Name)
	}

	return names
}

func ParseCVars(lines string) CVarList {
	var cvars CVarList
	for line := range strings.Lines(lines) {
		var cvar CVar
		columns := strings.Split(line, ":")
		if len(columns) != 4 {
			continue
		}

		for idx, piece := range columns {
			piece = strings.TrimSpace(piece)
			switch idx {
			case 0:
				cvar.Name = piece
			case 1:
				if piece == "cmd" {
					cvar.Cmd = true
				} else {
					cvar.Value = piece
				}
			case 2:
				for key := range strings.SplitSeq(strings.ReplaceAll(piece, "\"", ""), ",") {
					tag := strings.TrimSpace(key)
					if tag == "" {
						continue
					}
					cvar.Flags = append(cvar.Flags, tag)
				}
			case 3:
				cvar.Description = piece
			}
		}

		cvars = append(cvars, cvar)
	}

	return cvars
}
