package console

import (
	"context"
	"errors"
)

// Receiver handles incoming raw log message lines.
type Receiver interface {
	Send(string)
}

// Source is responsible for setting up and sending console log messages
// to a Receiver.
type Source interface {
	Start(ctx context.Context, receiver Receiver)
	Open(ctx context.Context) error
	Close(ctx context.Context) error
}

var (
	ErrOpen  = errors.New("failed to open console source")
	ErrSetup = errors.New("failed to setup log source")
	ErrClose = errors.New("failed to close log source")
)
