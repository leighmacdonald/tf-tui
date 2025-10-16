// Package tf handles interfacing with the game client.
package tf

import (
	"github.com/leighmacdonald/steamid/v4/extra"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	// Max number of players supported by the game.
	MaxPlayerCount = 102
	// In game max message length.
	MaxMessageLength = 127
)

type Team int

const (
	UNASSIGNED Team = iota
	SPEC
	BLU
	RED
)

type KickReason string

const (
	KickReasonIdle     KickReason = "idle"
	KickReasonScamming KickReason = "scamming"
	KickReasonCheating KickReason = "cheating"
	KickReasonOther    KickReason = "other"
)

type ChatDest string

const (
	ChatDestAll   ChatDest = "all"
	ChatDestTeam  ChatDest = "team"
	ChatDestParty ChatDest = "party"
)

// DumpPlayer holds the data returned from the `g15_dumpplayer` rcon command.
type DumpPlayer struct {
	Names     [MaxPlayerCount]string
	Ping      [MaxPlayerCount]int
	Score     [MaxPlayerCount]int
	Deaths    [MaxPlayerCount]int
	Connected [MaxPlayerCount]bool
	Team      [MaxPlayerCount]Team
	Alive     [MaxPlayerCount]bool
	Health    [MaxPlayerCount]int
	SteamID   [MaxPlayerCount]steamid.SteamID
	Valid     [MaxPlayerCount]bool
	UserID    [MaxPlayerCount]int
	Loss      [MaxPlayerCount]int
	State     [MaxPlayerCount]string
	Address   [MaxPlayerCount]string
	Time      [MaxPlayerCount]int
}

// Stats holds the data returned from the `stats` rcon command.
type Stats struct {
	CPU        float32
	InKBs      float32
	OutKBs     float32
	FPS        float32
	Uptime     int
	MapChanges int
	Players    int
	Connects   int
}

type Status struct {
	extra.Status
	Stats  Stats
	Region string
}
