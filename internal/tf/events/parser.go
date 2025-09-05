package events

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/tf"
)

var (
	ErrNoMatch        = errors.New("no match found")
	ErrParseTimestamp = errors.New("failed to parse timestamp")
	ErrDuration       = errors.New("failed to parse connected duration")
)

type EventType int

const (
	Any EventType = iota - 1
	Kill
	Msg
	Connect
	Disconnect
	StatusID
	Hostname
	Map
	Tags
	Address
	Lobby
	Stats
)

type Event struct {
	Type            EventType
	Player          string
	PlayerPing      int
	PlayerConnected time.Duration
	Team            tf.Team
	UserID          int
	PlayerSID       steamid.SteamID
	Victim          string
	VictimSID       steamid.SteamID
	Message         string
	Timestamp       time.Time
	MetaData        string
	Dead            bool
	TeamOnly        bool
	Raw             string
}

func (e *Event) ApplyTimestamp(tsString string) error {
	ts, errTS := parseTimestamp(tsString)
	if errTS != nil {
		return errTS
	}

	e.Timestamp = ts

	return nil
}

type parser struct {
	evtChan     chan Event
	ReadChannel chan string
	rx          []*regexp.Regexp
	logger      *slog.Logger
}

const (
	teamPrefix     = "(TEAM) "
	deadPrefix     = "*DEAD* "
	deadTeamPrefix = "*DEAD*(TEAM) "
	// coachPrefix    = "*COACH* ".
)
const logTimestampFormat = "01/02/2006 - 15:04:05"

// parseTimestamp will convert the source formatted log timestamps into a time.Time value.
func parseTimestamp(timestamp string) (time.Time, error) {
	parsedTime, errParse := time.Parse(logTimestampFormat, timestamp)
	if errParse != nil {
		return time.Time{}, errors.Join(errParse, ErrParseTimestamp)
	}

	return parsedTime, nil
}

type updateType int

const (
	updateKill updateType = iota
	updateStatus
	updateLobby
	updateMap
	updateHostname
	updateTags
	changeMap
	updateTeam
	updateStats
	updateTestPlayer = 1000
)

func (ut updateType) String() string {
	switch ut {
	case updateKill:
		return "kill"
	case updateStatus:
		return "status"
	case updateLobby:
		return "lobby"
	case updateMap:
		return "map_name"
	case updateHostname:
		return "hostname"
	case updateTags:
		return "tags"
	case changeMap:
		return "change_map"
	case updateTeam:
		return "team"
	case updateTestPlayer:
		return "test_player"
	case updateStats:
		return "stats"
	default:
		return "unknown"
	}
}

func newParser() *parser {
	return &parser{
		rx: []*regexp.Regexp{
			// 08/16/2025 - 01:25:53: Completed demo, recording time 369.4, game frames 23494.?
			regexp.MustCompile(`^(?P<dt>[01]\d/[0123]\d/20\d{2}\s-\s\d{2}:\d{2}:\d{2}):\s(.+?)\skilled\s(.+?)\swith\s(.+)(\.|\. \(crit\))$`),
			regexp.MustCompile(`^(?P<dt>[01]\d/[0123]\d/20\d{2}\s-\s\d{2}:\d{2}:\d{2}):\s(?P<name>.+?)\s:\s{2}(?P<message>.+?)$`),
			regexp.MustCompile(`^(?P<dt>[01]\d/[0123]\d/20\d{2}\s-\s\d{2}:\d{2}:\d{2}):\s(.+?)\sconnected$`),
			regexp.MustCompile(`^(?P<dt>[01]\d/[0123]\d/20\d{2}\s-\s\d{2}:\d{2}:\d{2}):\s(Connecting to|Differing lobby received.).+?$`),
			regexp.MustCompile(`^(?P<dt>[01]\d/[0123]\d/20\d{2}\s-\s\d{2}:\d{2}:\d{2}):\s#\s{1,6}(?P<id>\d{1,6})\s"(?P<name>.+?)"\s+(?P<sid>\[U:\d:\d{1,10}])\s{1,8}(?P<time>\d{1,3}:\d{2}(:\d{2})?)\s+(?P<ping>\d{1,4})\s{1,8}(?P<loss>\d{1,3})\s(spawning|active)$`),
			regexp.MustCompile(`^(?P<dt>[01]\d/[0123]\d/20\d{2}\s-\s\d{2}:\d{2}:\d{2}):\shostname:\s(.+?)$`),
			regexp.MustCompile(`^(?P<dt>[01]\d/[0123]\d/20\d{2}\s-\s\d{2}:\d{2}:\d{2}):\smap\s{5}:\s(.+?)\sat.+?$`),
			regexp.MustCompile(`^(?P<dt>[01]\d/[0123]\d/20\d{2}\s-\s\d{2}:\d{2}:\d{2}):\stags\s{4}:\s(.+?)$`),
			regexp.MustCompile(`^(?P<dt>[01]\d/[0123]\d/20\d{2}\s-\s\d{2}:\d{2}:\d{2}):\sudp/ip\s{2}:\s(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d{1,5})$`),
			regexp.MustCompile(`^\s{2}(Member|Pending)\[\d+]\s+(?P<sid>\[.+?]).+?TF_GC_TEAM_(?P<team>(DEFENDERS|INVADERS))\s{2}type\s=\sMATCH_PLAYER$`),
			// CPU    In_(KB/s)  Out_(KB/s)  Uptime  Map_changes  FPS      Players  Connects
			// 0.00   82.99      619.13      287     14           66.67    64       900
			regexp.MustCompile(`^(\d+)\.(\d{1,2})\s+(\d+)\.(\d{1,2})\s+(\d+)\.(\d{1,2})\s+(\d+)\s+(\d+)\s+(\d+)\.(\d{1,2})\s+(\d+)\s+(\d+)$`),
		},
	}
}

func (parser *parser) parse(msg string, outEvent *Event) error {
	// the index must match the index of the EventType const values
	for parserIdx, rxMatcher := range parser.rx {
		match := rxMatcher.FindStringSubmatch(msg)
		if match == nil {
			continue
		}
		outEvent.Raw = msg
		outEvent.Type = EventType(parserIdx)
		if outEvent.Type != Lobby {
			if errTS := outEvent.ApplyTimestamp(match[1]); errTS != nil {
				slog.Error("Failed to parse timestamp", slog.String("error", errTS.Error()))
			}
		}

		switch outEvent.Type {
		case Stats:
			// TODO
		case Connect:
			outEvent.Player = match[2]
		case Disconnect:
			outEvent.MetaData = match[2]
		case Msg:
			name := match[2]
			dead := false
			team := false

			if after, ok := strings.CutPrefix(name, teamPrefix); ok {
				name = after
				team = true
			}

			if after, ok := strings.CutPrefix(name, deadTeamPrefix); ok {
				name = after
				dead = true
				team = true
			} else if strings.HasPrefix(name, deadPrefix) {
				dead = true
				name = strings.TrimPrefix(name, deadPrefix)
			}

			outEvent.TeamOnly = team
			outEvent.Dead = dead
			outEvent.Player = name
			outEvent.Message = match[3]
		case StatusID:
			userID, errUserID := strconv.ParseInt(match[2], 10, 32)
			if errUserID != nil {
				slog.Error("Failed to parse status userid", slog.String("error", errUserID.Error()))

				continue
			}

			ping, errPing := strconv.ParseInt(match[7], 10, 32)
			if errPing != nil {
				slog.Error("Failed to parse status ping", slog.String("error", errPing.Error()))

				continue
			}

			dur, durErr := parseConnected(match[5])
			if durErr != nil {
				slog.Error("Failed to parse status duration", slog.String("error", durErr.Error()))

				continue
			}

			outEvent.UserID = int(userID)
			outEvent.Player = match[3]
			outEvent.PlayerSID = steamid.New(match[4])
			outEvent.PlayerConnected = dur
			outEvent.PlayerPing = int(ping)
		case Kill:
			outEvent.Player = match[2]
			outEvent.Victim = match[3]
		case Hostname:
			outEvent.MetaData = match[2]
		case Map:
			outEvent.MetaData = match[2]
		case Tags:
			outEvent.MetaData = match[2]
		case Address:
			outEvent.MetaData = match[2]
		case Lobby:
			outEvent.PlayerSID = steamid.New(match[2])
			if match[3] == "INVADERS" {
				outEvent.Team = tf.BLU
			} else {
				outEvent.Team = tf.RED
			}
		case Any:
		}

		return nil
	}

	return ErrNoMatch
}

func parseConnected(d string) (time.Duration, error) {
	var (
		pcs      = strings.Split(d, ":")
		dur      time.Duration
		parseErr error
	)

	switch len(pcs) {
	case 3:
		dur, parseErr = time.ParseDuration(fmt.Sprintf("%sh%sm%ss", pcs[0], pcs[1], pcs[2]))
	case 2:
		dur, parseErr = time.ParseDuration(fmt.Sprintf("%sm%ss", pcs[0], pcs[1]))
	case 1:
		dur, parseErr = time.ParseDuration(fmt.Sprintf("%ss", pcs[0]))
	default:
		dur = 0
	}

	if parseErr != nil {
		return 0, errors.Join(parseErr, ErrDuration)
	}

	return dur, nil
}
