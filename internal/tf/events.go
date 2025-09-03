package tf

import (
	"errors"
	"sync"
)

func NewLogBroadcaster() *LogBroadcaster {
	return &LogBroadcaster{
		parser:    newLogParser(),
		readers:   make(map[EventType][]chan<- LogEvent),
		readersMu: &sync.RWMutex{},
	}
}

type LogBroadcaster struct {
	readersAny []chan<- LogEvent
	readers    map[EventType][]chan<- LogEvent
	readersMu  *sync.RWMutex
	parser     *logParser
}

func (l *LogBroadcaster) ListenFor(logType EventType, handler chan<- LogEvent) {
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

func (l *LogBroadcaster) Send(line string) {
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
