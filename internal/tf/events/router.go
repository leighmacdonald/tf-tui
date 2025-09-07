package events

import (
	"errors"
	"log/slog"
	"sync"

	"github.com/leighmacdonald/tf-tui/internal/config"
)

func NewRouter() *Router {
	return &Router{
		parser:     newParser(),
		readers:    make(map[int]map[EventType][]chan<- Event),
		readersAny: make(map[int][]chan<- Event),
		readersMu:  &sync.RWMutex{},
	}
}

// Router handles receiving raw log line events from a console.Source, parsing them into
// a Event and sending the parsed event to any registered handlers for the parsed event.
type Router struct {
	config     config.Config
	readersAny map[int][]chan<- Event
	readers    map[int]map[EventType][]chan<- Event
	readersMu  *sync.RWMutex
	parser     *parser
}

// ListenFor registers a channel to start receiving events for the specified event.
func (l *Router) ListenFor(logSecret int, handler chan<- Event, logTypes ...EventType) {
	l.readersMu.Lock()
	defer l.readersMu.Unlock()

	for _, logType := range logTypes {
		// Any case is handled more generally
		if logType == Any {
			if _, found := l.readersAny[logSecret]; !found {
				l.readersAny[logSecret] = make([]chan<- Event, 0)
			}
			l.readersAny[logSecret] = append(l.readersAny[logSecret], handler)

			break
		}

		if _, found := l.readers[logSecret]; !found {
			l.readers[logSecret] = make(map[EventType][]chan<- Event)
		}

		if _, found := l.readers[logSecret][logType]; !found {
			l.readers[logSecret][logType] = make([]chan<- Event, 0)
		}

		l.readers[logSecret][logType] = append(l.readers[logSecret][logType], handler)
	}
}

// Send is responding for parsing and sending the result to any matching registered channels.
func (l *Router) Send(logSecret int, line string) {
	var logEvent Event
	// TODO move the parser outside of the router, instead sending already parsed events to the router instead.
	if err := l.parser.parse(line, &logEvent); err != nil || errors.Is(err, ErrNoMatch) {
		logEvent.Type = Any
		logEvent.Raw = line
	}

	l.readersMu.RLock()
	defer l.readersMu.RUnlock()

	if handlers, found := l.readers[logSecret][logEvent.Type]; found {
		for _, handler := range handlers {
			handler <- logEvent
		}
	}

	anyHandlers, found := l.readersAny[logSecret]
	if found {
		for _, handler := range anyHandlers {
			select {
			case handler <- logEvent:
			default:
				slog.Warn("Failed to send event", slog.String("event", line), slog.Int("log_secret", logSecret))
			}
		}
	}
}
