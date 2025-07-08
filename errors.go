package main

import "errors"

var (
	errRCONExec  = errors.New("RCON exec error")
	errRCONParse = errors.New("RCON parse error")
	errRCON      = errors.New("errors making rcon request")
)
