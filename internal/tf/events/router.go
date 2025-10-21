package events

import (
	"errors"
	"sync"

	"github.com/leighmacdonald/tf-tui/internal/config"
)

func NewRouter() *Router {
	return &Router{
		parser:     NewParser(),
		readers:    make(map[string]map[EventType][]chan<- Event),
		readersAny: make(map[string][]chan<- Event),
		readersMu:  &sync.RWMutex{},
	}
}

// Router handles receiving raw log line events from a console.Source, parsing them into
// a Event and sending the parsed event to any registered handlers for the parsed event.
type Router struct {
	config     config.Config
	readersAll []chan<- Event
	readersAny map[string][]chan<- Event
	readers    map[string]map[EventType][]chan<- Event
	readersMu  *sync.RWMutex
	parser     *Parser
}

// ListenFor registers a channel to start receiving events for the specified event.
func (r *Router) ListenFor(hostPort string, handler chan<- Event, logTypes ...EventType) {
	r.readersMu.Lock()
	defer r.readersMu.Unlock()

	for _, logType := range logTypes {
		if hostPort == "" {
			r.readersAll = append(r.readersAll, handler)

			continue
		}

		// Any case is handled more generally
		if logType == Any {
			if _, found := r.readersAny[hostPort]; !found {
				r.readersAny[hostPort] = make([]chan<- Event, 0)
			}
			r.readersAny[hostPort] = append(r.readersAny[hostPort], handler)

			break
		}

		if _, found := r.readers[hostPort]; !found {
			r.readers[hostPort] = make(map[EventType][]chan<- Event)
		}

		if _, found := r.readers[hostPort][logType]; !found {
			r.readers[hostPort][logType] = make([]chan<- Event, 0)
		}

		r.readers[hostPort][logType] = append(r.readers[hostPort][logType], handler)
	}
}

// Send is responding for parsing and sending the result to any matching registered channels.
func (r *Router) Send(hostPort string, line string) {
	// TODO move the parser outside of the router, instead sending already parsed events to the router instead.
	logEvent, err := r.parser.Parse(line)
	if err != nil || errors.Is(err, ErrNoMatch) {
		logEvent.Type = Any
		logEvent.Raw = line
	}
	logEvent.HostPort = hostPort

	r.readersMu.RLock()
	defer r.readersMu.RUnlock()

	if handlers, found := r.readers[hostPort][logEvent.Type]; found {
		for _, handler := range handlers {
			select {
			case handler <- logEvent:
			default:
				// slog.Warn("Failed to send event", slog.String("event", line), slog.String("host", hostPort))
			}
		}
	}

	anyHandlers, found := r.readersAny[hostPort]
	if found {
		for _, handler := range anyHandlers {
			// handler <- logEvent
			select {
			case handler <- logEvent:
			default:
				// slog.Warn("Failed to send event", slog.String("event", line), slog.String("host", hostPort))
			}
		}
	}

	for _, handler := range r.readersAll {
		// handler <- logEvent
		select {
		case handler <- logEvent:
		default:
			// slog.Warn("Failed to send event", slog.String("event", line), slog.String("host", hostPort))
		}
	}
}
