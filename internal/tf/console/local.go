package console

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"github.com/nxadm/tail"
)

func NewLocal(filePath string) *Local {
	return &Local{
		tail:     nil,
		stopChan: make(chan any),
		filePath: filePath,
	}
}

// Local handles "tail"-ing the console.log file that TF2 produces. Some useful
// events are parsed out into typed events. Remaining events are also returned in a raw form.
type Local struct {
	tail     *tail.Tail
	stopChan chan any
	filePath string
}

func (l *Local) Close(_ context.Context) error {
	if l.tail == nil || l.stopChan == nil {
		return nil
	}

	l.stopChan <- "ahh!"

	return nil
}

func (l *Local) Open() error {
	if l.tail != nil && l.tail.Filename == l.filePath {
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

	tailFile, errTail := tail.TailFile(l.filePath, tailConfig)
	if errTail != nil {
		return errors.Join(errTail, ErrOpen)
	}

	if l.tail != nil {
		l.stopChan <- true
	}

	l.tail = tailFile

	return nil
}

// start begins reading incoming log events, parsing events from the lines and emitting any found events as a LogEvent.
func (l *Local) Start(ctx context.Context, receiver Receiver) {
	stop := func() {
		if l.tail == nil {
			return
		}
		if errStop := l.tail.Stop(); errStop != nil {
			slog.Error("Failed to stop tailing console.log cleanly", slog.String("error", errStop.Error()))
		}
	}

	for {
		select {
		case <-ctx.Done():
			stop()

			return
		case msg := <-l.tail.Lines:
			if msg == nil {
				continue // Happens on linux only?
			}

			receiver.Send(0, msg.Text)
		case <-l.stopChan:
			stop()

			return
		}
	}
}
