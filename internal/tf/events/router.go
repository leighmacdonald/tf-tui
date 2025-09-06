package events

import (
	"errors"
	"sync"
)

func NewRouter() *Router {
	return &Router{
		parser:    newParser(),
		readers:   make(map[EventType][]chan<- Event),
		readersMu: &sync.RWMutex{},
	}
}

// Router handles receiving raw log line events from a console.Source, parsing them into
// a Event and sending the parsed event to any registered handlers for the parsed event.
type Router struct {
	readersAny []chan<- Event
	readers    map[EventType][]chan<- Event
	readersMu  *sync.RWMutex
	parser     *parser
}

// ListenFor registers a channel to start receiving events for the specified event.
func (l *Router) ListenFor(logType EventType, handler chan<- Event) {
	l.readersMu.Lock()
	defer l.readersMu.Unlock()

	// Any case is handled more generally
	if logType == Any {
		l.readersAny = append(l.readersAny, handler)

		return
	}

	if _, found := l.readers[logType]; !found {
		l.readers[logType] = make([]chan<- Event, 0)
	}

	l.readers[logType] = append(l.readers[logType], handler)
}

// Send is responding for parsing and sending the result to any matching registered channels.
func (l *Router) Send(line string) {
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
		handler <- logEvent
	}
}
