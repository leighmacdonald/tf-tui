package main

import (
	"context"
	"errors"
	"strings"

	"github.com/leighmacdonald/tf-tui/shared"
)

func fetchPlayerState(ctx context.Context, address string, password string) (shared.PlayerState, error) {
	conn := newRconConnection(address, password)
	response, errExec := conn.exec(ctx, "g15_dumpplayer", true)
	if errExec != nil {
		return shared.PlayerState{}, errors.Join(errExec, errRCONExec)
	}

	dump, err := Parse(strings.NewReader(response))
	if err != nil {
		return shared.PlayerState{}, errors.Join(err, errRCONParse)
	}

	return dump, nil
}
