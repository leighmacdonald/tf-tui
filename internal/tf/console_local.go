package tf

import (
	"bufio"
	"errors"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/nxadm/tail"
)

var (
	errConsoleLog     = errors.New("failed to read console.log")
	errDuration       = errors.New("failed to parse connected duration")
	errParseTimestamp = errors.New("failed to parse timestamp")
)

func NewConsoleLog(broadcater *LogBroadcaster) *ConsoleLog {
	return &ConsoleLog{
		tail:           nil,
		stopChan:       make(chan bool),
		logBroadcaster: broadcater,
	}
}

// ConsoleLog handles "tail"-ing the console.log file that TF2 produces. Some useful
// events are parsed out into typed events. Remaining events are also returned in a raw form.
type ConsoleLog struct {
	tail           *tail.Tail
	stopChan       chan bool
	logBroadcaster *LogBroadcaster
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
					//	l.handleLine(scanner.Text())
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

			l.logBroadcaster.Send(msg.Text)
		case <-l.stopChan:
			if errStop := l.tail.Stop(); errStop != nil {
				slog.Error("Failed to stop tailing console.log cleanly", slog.String("error", errStop.Error()))
			}

			return
		}
	}
}
