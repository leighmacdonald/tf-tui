package events

import (
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
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
	Version
)

type Event struct {
	// How we identify the owner of this event.
	LogSecret int
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
type VersionEvent struct {
	Version int
	Secure  bool
}
type TagsEvent struct {
	Tags []string
}

type AddressEvent struct {
	Address netip.Addr
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

type UDPIPEvent struct {
	Address string
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
	rx          []regexPair
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

type regexPair struct {
	regex     *regexp.Regexp
	eventType EventType
}

func newParser() *parser {
	return &parser{
		rx: []regexPair{
			// 08/16/2025 - 01:25:53: Completed demo, recording time 369.4, game frames 23494.?
			{eventType: Kill, regex: regexp.MustCompile(`^(?:[01]\d/[0123]\d/20\d{2}\s-\s\d{2}:\d{2}:\d{2}:\s+)?(.+?)\skilled\s(.+?)\swith\s(.+)(\.|\. \(crit\))$`)},
			{eventType: Msg, regex: regexp.MustCompile(`(?P<name>.+?)\s:\s{2}(?P<message>.+?)$`)},
			{eventType: Connect, regex: regexp.MustCompile(`(.+?)\sconnected$`)},
			{eventType: Disconnect, regex: regexp.MustCompile(`(Connecting to|Differing lobby received.).+?$`)},
			{eventType: StatusID, regex: regexp.MustCompile(`#\s+(?P<id>\d{1,6})\s"(?P<name>.+?)"\s+(?P<sid>\[U:\d:\d{1,10}])\s{1,8}(?P<time>\d{1,3}:\d{2}(?::\d{2})?)\s+(?P<ping>\d{1,4})\s{1,8}(?P<loss>\d{1,3})\s(spawning|active)(?P<ip>\s+.+?)?$`)},
			{eventType: Hostname, regex: regexp.MustCompile(`hostname:\s(.+?)$`)},
			{eventType: Map, regex: regexp.MustCompile(`map\s{5}:\s(.+?)\sat.+?$`)},
			{eventType: Tags, regex: regexp.MustCompile(`tags\s{4}:\s(.+?)$`)},
			{eventType: Address, regex: regexp.MustCompile(`udp/ip.+?public IP from Steam: (\d+\.\d+\.\d+\.\d+)`)},
			{eventType: Version, regex: regexp.MustCompile(`^version\s+:.+?\d+/\d+\s+(\d+)\s+(secure)?`)},
		},
	}
}

func (parser *parser) parse(msg string) (Event, error) {
	// the index must match the index of the EventType const values
	var outEvent Event

	for _, rxMatcher := range parser.rx {
		match := rxMatcher.regex.FindStringSubmatch(msg)
		if match == nil {
			continue
		}
		outEvent.Raw = msg
		outEvent.Type = rxMatcher.eventType
		outEvent.Timestamp = time.Now()

		switch outEvent.Type { //nolint:exhaustive
		case Connect:
			outEvent.Data = ConnectEvent{Player: match[3]}
		case Disconnect:
			outEvent.Data = DisconnectEvent{Player: match[3]}
		case Msg:
			outEvent.Data = parseMsg(match)
		case StatusID:
			userID, errUserID := strconv.ParseInt(match[1], 10, 32)
			if errUserID != nil {
				slog.Error("Failed to parse status userid", slog.String("error", errUserID.Error()))

				continue
			}

			ping, errPing := strconv.ParseInt(match[5], 10, 32)
			if errPing != nil {
				slog.Error("Failed to parse status ping", slog.String("error", errPing.Error()))

				continue
			}

			loss, errLoss := strconv.ParseInt(match[6], 10, 32)
			if errLoss != nil {
				slog.Error("Failed to parse status loss", slog.String("error", errLoss.Error()))

				continue
			}

			dur, durErr := parseConnected(match[4])
			if durErr != nil {
				slog.Error("Failed to parse status duration", slog.String("error", durErr.Error()))

				continue
			}
			sie := StatusIDEvent{
				UserID:    int(userID),
				Player:    match[2],
				PlayerSID: steamid.New(match[3]),
				Connected: int(dur.Seconds()),
				Ping:      int(ping),
				Loss:      int(loss),
				State:     match[7],
			}
			if len(match) == 9 {
				sie.Address = strings.TrimSpace(match[8])
			}
			// TODO different data for server/client modes is avail
			outEvent.Data = sie
		case Version:
			version, errVersion := strconv.ParseInt(match[1], 10, 64)
			if errVersion != nil {
				slog.Warn("Failed to parse version", slog.String("error", errVersion.Error()))

				continue
			}

			outEvent.Data = VersionEvent{Version: int(version), Secure: len(match) > 2 && match[2] == "secure"}
		case Kill:
			outEvent.Data = KillEvent{Player: match[1], Victim: match[2], Weapon: match[3], Crit: strings.Contains(match[4], "crit")}
		case Hostname:
			outEvent.Data = HostnameEvent{Hostname: match[1]}
		case Map:
			outEvent.Data = MapEvent{MapName: match[1]}
		case Tags:
			outEvent.Data = TagsEvent{Tags: strings.Split(match[1], ",")}
		case Address:
			addr, errAddr := netip.ParseAddr(match[1])
			if errAddr != nil {
				slog.Warn("Failed to parse address", slog.String("error", errAddr.Error()), slog.String("address", match[1]))

				continue
			}
			outEvent.Data = AddressEvent{Address: addr}
		case Any:
			outEvent.Data = AnyEvent{Raw: msg}
		}

		return outEvent, nil
	}

	return outEvent, ErrNoMatch
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
