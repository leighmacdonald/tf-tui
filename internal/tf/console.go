package tf

import (
	"bufio"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nxadm/tail"
)

var (
	errConsoleLog     = errors.New("failed to read console.log")
	errDuration       = errors.New("failed to parse connected duration")
	errParseTimestamp = errors.New("failed to parse timestamp")
)

func NewConsoleLog() *ConsoleLog {
	return &ConsoleLog{
		tail:      nil,
		stopChan:  make(chan bool),
		parser:    newLogParser(),
		readers:   make(map[EventType][]chan<- LogEvent),
		readersMu: &sync.RWMutex{},
	}
}

// ConsoleLog handles "tail"-ing the console.log file that TF2 produces. Some useful
// events are parsed out into typed events. Remaining events are also returned in a raw form.
type ConsoleLog struct {
	tail       *tail.Tail
	parser     *logParser
	stopChan   chan bool
	readers    map[EventType][]chan<- LogEvent
	readersAny []chan<- LogEvent
	readersMu  *sync.RWMutex
}

func (l *ConsoleLog) Open(filePath string) error {
	if l.tail != nil && l.tail.Filename == filePath {
		return nil
	}

	tailConfig := tail.Config{
		// Start at the end of the file, only watch for new lines.
		Location: &tail.SeekInfo{
			Offset: 0,
			Whence: io.SeekEnd,
		},
		// Ensure we don't see the log messages in stdout and mangle the ui
		Logger:    tail.DiscardingLogger,
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

func (l *ConsoleLog) RegisterHandler(logType EventType, handler chan<- LogEvent) {
	l.readersMu.Lock()
	defer l.readersMu.Unlock()

	// Any case is handled more generally
	if logType == EvtAny {
		l.readersAny = append(l.readersAny, handler)

		return
	}

	if _, found := l.readers[logType]; !found {
		l.readers[logType] = make([]chan<- LogEvent, 0)
	}

	l.readers[logType] = append(l.readers[logType], handler)
}

func (l *ConsoleLog) handleLine(rawLine string) {
	line := strings.TrimSuffix(rawLine, "\r")
	if line == "" {
		return
	}

	var logEvent LogEvent
	if err := l.parser.parse(line, &logEvent); err != nil || errors.Is(err, ErrNoMatch) {
		// This is sent as a "raw" line so that the console view can show it even if it doesn't
		// match any supported events.
		logEvent.Raw = line
		logEvent.Type = EvtAny
	}

	l.readersMu.RLock()
	defer l.readersMu.RUnlock()

	if handlers, found := l.readers[logEvent.Type]; found {
		for _, handler := range handlers {
			handler <- logEvent
		}
	}

	for _, handler := range l.readersAny {
		handler <- logEvent
	}
}

// start begins reading incoming log events, parsing events from the lines and emitting any found events as a LogEvent.
func (l *ConsoleLog) start() {
	if len(os.Getenv("DEBUG")) > 0 {
		go func() {
			for {
				reader, errReader := os.Open("testdata/console.log")
				if errReader != nil {
					panic(errReader)
				}

				scanner := bufio.NewScanner(reader)
				for scanner.Scan() {
					l.handleLine(scanner.Text())
					time.Sleep(time.Millisecond * 50)
				}
			}
		}()
	}
	for {
		select {
		case msg := <-l.tail.Lines:
			if msg == nil {
				// Happens on linux only?
				continue
			}
			l.handleLine(msg.Text)
		case <-l.stopChan:
			if errStop := l.tail.Stop(); errStop != nil {
				slog.Error("Failed to stop tailing console.log cleanly", slog.String("error", errStop.Error()))
			}

			return
		}
	}
}
