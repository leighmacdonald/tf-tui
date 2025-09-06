package events

import (
	"errors"
	"log/slog"
	"sync"
)

func NewRouter() *Router {
	return &Router{
		parser:     newParser(),
		readers:    make(map[string]map[EventType][]chan<- Event),
		readersAny: make(map[string][]chan<- Event),
		readersMu:  &sync.RWMutex{},
	}
}

// Router handles receiving raw log line events from a console.Source, parsing them into
// a Event and sending the parsed event to any registered handlers for the parsed event.
type Router struct {
	readersAny map[string][]chan<- Event
	readers    map[string]map[EventType][]chan<- Event
	readersMu  *sync.RWMutex
	parser     *parser
}

// ListenFor registers a channel to start receiving events for the specified event.
func (l *Router) ListenFor(serverAddress string, logType EventType, handler chan<- Event) {
	l.readersMu.Lock()
	defer l.readersMu.Unlock()

	// Any case is handled more generally
	if logType == Any {
		if _, found := l.readersAny[serverAddress]; !found {
			l.readersAny[serverAddress] = make([]chan<- Event, 0)
		}
		l.readersAny[serverAddress] = append(l.readersAny[serverAddress], handler)

		return
	}

	if _, found := l.readers[serverAddress]; !found {
		l.readers[serverAddress] = make(map[EventType][]chan<- Event)
	}

	if _, found := l.readers[serverAddress][logType]; !found {
		l.readers[serverAddress][logType] = make([]chan<- Event, 0)
	}

	l.readers[serverAddress][logType] = append(l.readers[serverAddress][logType], handler)
}

// Send is responding for parsing and sending the result to any matching registered channels.
func (l *Router) Send(logSecret int, line string) {
	var logEvent Event
	if err := l.parser.parse(line, &logEvent); err != nil || errors.Is(err, ErrNoMatch) {
		logEvent.Type = Any
		logEvent.Raw = line
	}

	l.readersMu.RLock()
	defer l.readersMu.RUnlock()

	if handlers, found := l.readers[logEvent.Type]; found {
		for _, handler := range handlers {
			handler <- logEvent
		}
	}

	for _, handler := range l.readersAny {
		select {
		case handler <- logEvent:
		default:
			slog.Warn("Failed to send event", slog.String("event", line))
		}
	}
}
