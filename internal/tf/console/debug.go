package console

import (
	"bufio"
	"context"
	"errors"
	"os"
	"time"
)

func NewDebug(logPath string) Debug {
	return Debug{logPath: logPath}
}

type Debug struct {
	logBroadcaster Receiver
	file           *os.File
	logPath        string
}

func (c *Debug) Open() error {
	reader, errReader := os.Open(c.logPath)
	if errReader != nil {
		return errors.Join(errReader, ErrOpen)
	}
	c.file = reader

	return nil
}

func (c *Debug) Close(_ context.Context) error {
	if c.file == nil {
		return nil
	}

	if err := c.file.Close(); err != nil {
		return errors.Join(err, ErrClose)
	}

	return nil
}

func (c *Debug) Start(ctx context.Context, receiver Receiver) {
	var (
		logFreq = time.NewTicker(time.Millisecond * 50)
		scanner = bufio.NewScanner(c.file)
	)

	for {
		select {
		case <-ctx.Done():
			return
		case <-logFreq.C:
			if scanner.Scan() {
				receiver.Send(0, scanner.Text())
			}
		}
	}
}
