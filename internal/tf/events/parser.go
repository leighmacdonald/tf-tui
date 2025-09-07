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
)

const (
	teamPrefix     = "(TEAM) "
	deadPrefix     = "*DEAD* "
	deadTeamPrefix = "*DEAD*(TEAM) "
	// coachPrefix    = "*COACH* ".
	logTimestampFormat = "01/02/2006 - 15:04:05"
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
	Stats
)

type Event struct {
	Type      EventType
	Timestamp time.Time
	Raw       string
	Data      any
}

func (e *Event) ApplyTimestamp(tsString string) error {
	ts, errTS := parseTimestamp(tsString)
	if errTS != nil {
		return errTS
	}

	e.Timestamp = ts

	return nil
}

type AnyEvent struct {
	Raw string
}

type ConnectEvent struct {
	Player string
}

type DisconnectEvent struct {
	Player string
}

type StatusIDEvent struct {
	UserID    int
	Player    string
	PlayerSID steamid.SteamID
	Connected int
	Ping      int
	Loss      int
	State     string
	Address   string
}

type HostnameEvent struct {
	Hostname string
}

type MapEvent struct {
	MapName string
}

type TagsEvent struct {
	Tags []string
}

type AddressEvent struct {
	Address string
}

type LobbyEvent struct {
	LobbyID string
}

type MsgEvent struct {
	Player    string
	PlayerSID steamid.SteamID
	Dead      bool
	TeamOnly  bool
	Message   string
}

type RawEvent struct {
	Raw string
}

type KillEvent struct {
	Event
	Player    string
	PlayerSID steamid.SteamID
	Victim    string
	VictimSID steamid.SteamID
	Weapon    string
	Crit      bool
}

type parser struct {
	evtChan     chan Event
	ReadChannel chan string
	rx          []*regexp.Regexp
	logger      *slog.Logger
}

// parseTimestamp will convert the source formatted log timestamps into a time.Time value.
func parseTimestamp(timestamp string) (time.Time, error) {
	parsedTime, errParse := time.Parse(logTimestampFormat, timestamp)
	if errParse != nil {
		return time.Time{}, errors.Join(errParse, ErrParseTimestamp)
	}

	return parsedTime, nil
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

		if errTS := outEvent.ApplyTimestamp(match[1]); errTS != nil {
			slog.Error("Failed to parse timestamp", slog.String("error", errTS.Error()))
		}

		switch outEvent.Type { //nolint:exhaustive
		case Connect:
			outEvent.Data = ConnectEvent{Player: match[2]}
		case Disconnect:
			outEvent.Data = DisconnectEvent{Player: match[2]}
		case Msg:
			outEvent.Data = parseMsg(match)
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

			loss, errLoss := strconv.ParseInt(match[8], 10, 32)
			if errLoss != nil {
				slog.Error("Failed to parse status loss", slog.String("error", errLoss.Error()))

				continue
			}

			dur, durErr := parseConnected(match[5])
			if durErr != nil {
				slog.Error("Failed to parse status duration", slog.String("error", durErr.Error()))

				continue
			}

			// TODO different data for server/client modes is avail
			outEvent.Data = StatusIDEvent{
				UserID:    int(userID),
				Player:    match[3],
				PlayerSID: steamid.New(match[4]),
				Connected: int(dur.Seconds()),
				Ping:      int(ping),
				Loss:      int(loss),
				State:     match[9],
				Address:   match[10],
			}
		case Kill:
			outEvent.Data = KillEvent{Player: match[2], Victim: match[3]}
		case Hostname:
			outEvent.Data = HostnameEvent{Hostname: match[2]}
		case Map:
			outEvent.Data = MapEvent{MapName: match[2]}
		case Tags:
			outEvent.Data = TagsEvent{Tags: strings.Split(match[2], ",")}
		case Address:
			outEvent.Data = AddressEvent{Address: match[2]}
		case Any:
			outEvent.Data = AnyEvent{Raw: msg}
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

func parseMsg(match []string) MsgEvent {
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

	return MsgEvent{
		Player:   name,
		Dead:     dead,
		TeamOnly: team,
		// PlayerSID: steamid.SteamID,
		Message: match[3],
	}
}
