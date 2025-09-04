package console

import (
	"context"
	"errors"
)

type Receiver interface {
	Send(string)
}

type Source interface {
	Start(ctx context.Context)
	Open(ctx context.Context) error
	Close(ctx context.Context) error
}

var (
	ErrOpen  = errors.New("failed to open console source")
	ErrSetup = errors.New("failed to setup log source")
	ErrClose = errors.New("failed to close log source")
)
