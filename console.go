package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/nxadm/tail"
)

var (
	errConsoleLog     = errors.New("failed to read console.log")
	errDuration       = errors.New("failed to parse connected duration")
	errParseTimestamp = errors.New("failed to parse timestamp")
)

type MsgSender interface {
	Send(tea.Msg)
}

type ConsoleLog struct {
	tail     *tail.Tail
	parser   Parser
	sender   MsgSender
	stopChan chan bool
}

func (l *ConsoleLog) ReadFile(filePath string) error {
	if l.tail != nil && l.tail.Filename == filePath {
		return nil
	}

	tailConfig := tail.Config{
		// Start at the end of the file, only watch for new lines.
		Location: &tail.SeekInfo{
			Offset: 0,
			Whence: io.SeekEnd,
		},
		Follow:    true,
		ReOpen:    true,
		MustExist: false,
		// Poll:      runtime.GOOS == "windows",
	}

	tailFile, errTail := tail.TailFile(filePath, tailConfig)
	if errTail != nil {
		return errors.Join(errTail, errConsoleLog)
	}

	if l.tail != nil {
		l.stopChan <- true
	}

	l.tail = tailFile
	go l.start()

	return nil
}

// start begins reading incoming log events, parsing events from the lines and emitting any found events as a LogEvent.
func (l *ConsoleLog) start() {
	defer l.tail.Cleanup()

	for {
		select {
		case msg := <-l.tail.Lines:
			if msg == nil {
				// Happens on linux only?
				continue
			}

			line := strings.TrimSuffix(msg.Text, "\r")
			if line == "" {
				continue
			}

			var logEvent LogEvent
			if err := l.parser.parse(line, &logEvent); err != nil || errors.Is(err, ErrNoMatch) {
				// slog.Debug("could not match line", slog.String("line", line))
				continue
			}

			tea.Println("matched line: " + line)

			l.sender.Send(logEvent)
		case <-l.stopChan:
			if errStop := l.tail.Stop(); errStop != nil {
				tea.Println("Failed to stop tailing console.log cleanly: " + errStop.Error())
			}

			return
		}
	}
}

func NewConsoleLog() *ConsoleLog {
	return &ConsoleLog{tail: nil, stopChan: make(chan bool), parser: newLogParser()}
}

var ErrNoMatch = errors.New("no match found")

type hostnameEvent struct {
	hostname string
}

type mapEvent struct {
	mapName string
}

type tagsEvent struct {
	tags []string
}

const logTimestampFormat = "01/02/2006 - 15:04:05"

// parseTimestamp will convert the source formatted log timestamps into a time.Time value.
func parseTimestamp(timestamp string) (time.Time, error) {
	parsedTime, errParse := time.Parse(logTimestampFormat, timestamp)
	if errParse != nil {
		return time.Time{}, errors.Join(errParse, errParseTimestamp)
	}

	return parsedTime, nil
}

type LogEvent struct {
	Type            EventType
	Player          string
	PlayerPing      int
	PlayerConnected time.Duration
	Team            Team
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

func (e *LogEvent) ApplyTimestamp(tsString string) error {
	ts, errTS := parseTimestamp(tsString)
	if errTS != nil {
		return errTS
	}

	e.Timestamp = ts

	return nil
}

type Event struct {
	Name  EventType
	Value any
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
	default:
		return "unknown"
	}
}

type killEvent struct {
	sourceName string
	victimName string
}

type statusEvent struct {
	ping      int
	userID    int
	name      string
	connected time.Duration
}

type updateStateEvent struct {
	kind   updateType
	source steamid.SteamID
	data   any
}

type messageEvent struct {
	steamID   steamid.SteamID
	name      string
	createdAt time.Time
	message   string
	teamOnly  bool
	dead      bool
}

type Parser interface {
	parse(msg string, outEvent *LogEvent) error
}

type logParser struct {
	evtChan     chan LogEvent
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

func newLogParser() *logParser {
	return &logParser{
		rx: []*regexp.Regexp{
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
		},
	}
}

func (parser *logParser) parse(msg string, outEvent *LogEvent) error {
	// the index must match the index of the EventType const values
	for i, rxMatcher := range parser.rx {
		if match := rxMatcher.FindStringSubmatch(msg); match != nil { //nolint:nestif
			outEvent.Raw = msg
			outEvent.Type = EventType(i)
			if outEvent.Type != EvtLobby {
				if errTS := outEvent.ApplyTimestamp(match[1]); errTS != nil {
					tea.Println("Failed to parse timestamp: " + errTS.Error())
				}
			}

			switch outEvent.Type {
			case EvtConnect:
				outEvent.Player = match[2]
			case EvtDisconnect:
				outEvent.MetaData = match[2]
			case EvtMsg:
				name := match[2]
				dead := false
				team := false

				if strings.HasPrefix(name, teamPrefix) {
					name = strings.TrimPrefix(name, teamPrefix)
					team = true
				}

				if strings.HasPrefix(name, deadTeamPrefix) {
					name = strings.TrimPrefix(name, deadTeamPrefix)
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
			case EvtStatusID:
				userID, errUserID := strconv.ParseInt(match[2], 10, 32)
				if errUserID != nil {
					tea.Println("Failed to parse status userid: " + errUserID.Error())

					continue
				}

				ping, errPing := strconv.ParseInt(match[7], 10, 32)
				if errPing != nil {
					tea.Println("Failed to parse status ping: " + errPing.Error())

					continue
				}

				dur, durErr := parseConnected(match[5])
				if durErr != nil {
					tea.Println("Failed to parse status duration: " + durErr.Error())

					continue
				}

				outEvent.UserID = int(userID)
				outEvent.Player = match[3]
				outEvent.PlayerSID = steamid.New(match[4])
				outEvent.PlayerConnected = dur
				outEvent.PlayerPing = int(ping)
			case EvtKill:
				outEvent.Player = match[2]
				outEvent.Victim = match[3]
			case EvtHostname:
				outEvent.MetaData = match[2]
			case EvtMap:
				outEvent.MetaData = match[2]
			case EvtTags:
				outEvent.MetaData = match[2]
			case EvtAddress:
				outEvent.MetaData = match[2]
			case EvtLobby:
				outEvent.PlayerSID = steamid.New(match[2])
				if match[3] == "INVADERS" {
					outEvent.Team = BLU
				} else {
					outEvent.Team = RED
				}
			}

			return nil
		}
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
		return 0, errors.Join(parseErr, errDuration)
	}

	return dur, nil
}

type EventType int

const (
	EvtAny = iota - 1
	EvtKill
	EvtMsg
	EvtConnect
	EvtDisconnect
	EvtStatusID
	EvtHostname
	EvtMap
	EvtTags
	EvtAddress
	EvtLobby
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
