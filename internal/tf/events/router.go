package events

import (
	"errors"
	"log/slog"
	"sync"

	"github.com/leighmacdonald/tf-tui/internal/config"
)

func NewRouter() *Router {
	return &Router{
		parser:     NewParser(),
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
	parser     *Parser
}

// ListenFor registers a channel to start receiving events for the specified event.
func (r *Router) ListenFor(logSecret int, handler chan<- Event, logTypes ...EventType) {
	r.readersMu.Lock()
	defer r.readersMu.Unlock()

	for _, logType := range logTypes {
		// Any case is handled more generally
		if logType == Any {
			if _, found := r.readersAny[logSecret]; !found {
				r.readersAny[logSecret] = make([]chan<- Event, 0)
			}
			r.readersAny[logSecret] = append(r.readersAny[logSecret], handler)

			break
		}

		if _, found := r.readers[logSecret]; !found {
			r.readers[logSecret] = make(map[EventType][]chan<- Event)
		}

		if _, found := r.readers[logSecret][logType]; !found {
			r.readers[logSecret][logType] = make([]chan<- Event, 0)
		}

		r.readers[logSecret][logType] = append(r.readers[logSecret][logType], handler)
	}
}

// Send is responding for parsing and sending the result to any matching registered channels.
func (r *Router) Send(logSecret int, line string) {
	// TODO move the parser outside of the router, instead sending already parsed events to the router instead.
	logEvent, err := r.parser.Parse(line)
	if err != nil || errors.Is(err, ErrNoMatch) {
		logEvent.Type = Any
		logEvent.Raw = line
	}

	r.readersMu.RLock()
	defer r.readersMu.RUnlock()

	if handlers, found := r.readers[logSecret][logEvent.Type]; found {
		for _, handler := range handlers {
			select {
			case handler <- logEvent:
			default:
				slog.Warn("Failed to send event", slog.String("event", line), slog.Int("log_secret", logSecret))
			}
		}
	}

	anyHandlers, found := r.readersAny[logSecret]
	if found {
		for _, handler := range anyHandlers {
			// handler <- logEvent
			select {
			case handler <- logEvent:
			default:
				slog.Warn("Failed to send event", slog.String("event", line), slog.Int("log_secret", logSecret))
			}
		}
	}
}
