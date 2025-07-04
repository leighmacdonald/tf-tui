package main

import (
	"context"
	"errors"
	"strings"
)

var parser = newG15Parser()

func fetchDumpPlayer(ctx context.Context, address string, password string) (*DumpPlayer, error) {
	conn := newRconConnection(address, password)
	response, errExec := conn.exec(ctx, "g15_dumpplayer", true)
	if errExec != nil {
		return nil, errors.Join(errExec, errRCONExec)
	}

	var dump DumpPlayer
	if err := parser.Parse(strings.NewReader(response), &dump); err != nil {
		return nil, errors.Join(err, errRCONParse)
	}

	return &dump, nil
}
