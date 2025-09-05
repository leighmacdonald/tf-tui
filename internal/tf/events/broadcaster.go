package events

import (
	"errors"
	"sync"
)

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		parser:    newParser(),
		readers:   make(map[EventType][]chan<- Event),
		readersMu: &sync.RWMutex{},
	}
}

type Broadcaster struct {
	readersAny []chan<- Event
	readers    map[EventType][]chan<- Event
	readersMu  *sync.RWMutex
	parser     *parser
}

func (l *Broadcaster) ListenFor(logType EventType, handler chan<- Event) {
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

func (l *Broadcaster) Send(line string) {
	var logEvent Event
	if err := l.parser.parse(line, &logEvent); err != nil || errors.Is(err, ErrNoMatch) {
		// This is sent as a "raw" line so that the console view can show it even if it doesn't
		// match any supported events.
		logEvent.Raw = line
		logEvent.Type = Any
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
