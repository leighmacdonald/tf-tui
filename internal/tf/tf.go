// Package tf handles interfacing with the game client.
package tf

const (
	MaxPlayerCount   = 102
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
