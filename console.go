package main

import (
	"context"
	"io"
	"strings"

	"github.com/nxadm/tail"
)

type ConsoleLog struct {
	tail *tail.Tail
}

func (l *ConsoleLog) ReadFile(filePath string) error {
	tailConfig := tail.Config{
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
		return errTail
	}

	l.tail = tailFile

	return nil

}

func (l *ConsoleLog) lineEmitter(ctx context.Context, incoming chan string) {
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

			incoming <- line
		case <-ctx.Done():
			return
		}
	}
}

// start begins reading incoming log events, parsing events from the lines and emitting any found events as a LogEvent.
func (l *ConsoleLog) start(ctx context.Context) {
	defer l.tail.Cleanup()
	incomingLogLines := make(chan string)

	go l.lineEmitter(ctx, incomingLogLines)

	//for {
	//	select {
	//	case line := <-incomingLogLines:
	//		var logEvent LogEvent
	//		if err := li.parser.parse(line, &logEvent); err != nil || errors.Is(err, ErrNoMatch) {
	//			// slog.Debug("could not match line", slog.String("line", line))
	//			continue
	//		}
	//
	//		slog.Debug("matched line", slog.String("line", line))
	//
	//		l.broadcaster.broadcast(logEvent)
	//	case <-ctx.Done():
	//		if errStop := li.tail.Stop(); errStop != nil {
	//			li.logger.Error("Failed to stop tailing console.log cleanly", errAttr(errStop))
	//		}
	//
	//		return
	//	}
	//}
}

func newConsoleLog() *ConsoleLog {

	return &ConsoleLog{tail: nil}
}
