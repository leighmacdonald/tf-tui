package rcon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/leighmacdonald/rcon/rcon"
)

var errRCON = errors.New("errors making rcon request")

type Connection struct {
	addr     string
	password string
	timeout  time.Duration
}

// ExecP is a shortcut for executing a rcon command and parsing it to a gerically typed receiver.
func ExecP[T any](ctx context.Context, addr string, password string, command string, parser func(string) T) T {
	response, errExec := New(addr, password).Exec(ctx, command, true)
	if errExec != nil {
		slog.Error("Failed to exec rcon", slog.String("error", errExec.Error()), slog.String("host", addr))
		var empty T

		return empty
	}

	return parser(response)
}

func New(addr string, password string) Connection {
	return Connection{
		addr:     addr,
		password: password,
		timeout:  time.Second,
	}
}

func (r Connection) Exec(ctx context.Context, cmd string, large bool) (string, error) {
	conn, errConn := rcon.Dial(ctx, r.addr, r.password, r.timeout)
	if errConn != nil {
		return "", errors.Join(errConn, fmt.Errorf("%w: %s", errRCON, r.addr))
	}
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Error("failed to close rcon connection", slog.String("error", err.Error()))
		}
	}()

	if large {
		return r.rconLarge(conn, cmd)
	}

	return r.rcon(conn, cmd)
}

func (r Connection) rcon(conn *rcon.RemoteConsole, cmd string) (string, error) {
	cmdID, errWrite := conn.Write(cmd)
	if errWrite != nil {
		return "", errors.Join(errWrite, errRCON)
	}

	resp, respID, errRead := conn.Read()
	if errRead != nil {
		return "", errors.Join(errRead, errRCON)
	}

	if respID != cmdID {
		slog.Warn("Mismatched command response ID", slog.Int("req", cmdID), slog.Int("resp", respID))
	}

	return resp, nil
}

// rconLarge is used for rcon responses that exceed the size of a single rcon packet (g15_dumpplayer).
func (r Connection) rconLarge(conn *rcon.RemoteConsole, cmd string) (string, error) {
	cmdID, errWrite := conn.Write(cmd)
	if errWrite != nil {
		return "", errors.Join(errWrite, errRCON)
	}

	var response string

	for {
		resp, respID, errRead := conn.Read()
		if errRead != nil {
			return "", errors.Join(errRead, errRCON)
		}

		if cmdID == respID {
			s := len(resp)
			response += resp

			if s < 4096 {
				break
			}
		}
	}

	return response, nil
}
