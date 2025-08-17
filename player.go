package main

import (
	"slices"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	// How long we wait until a player should be ejected from our tracking.
	// This should be long enough to last through map changes without dropping the
	// known players.
	playerExpiration = time.Second * 30
)

type Team int

const (
	UNASSIGNED = iota
	SPEC
	BLU
	RED
)

type Player struct {
	SteamID       steamid.SteamID
	Name          string
	Ping          int
	Score         int
	Deaths        int
	Connected     bool
	Team          Team
	Alive         bool
	Health        int
	Valid         bool
	UserID        int
	meta          MetaProfile
	metaUpdatedOn time.Time
	g15UpdatedOn  time.Time
	BDMatches     []MatchedBDPlayer
}

func (p Player) Expired() bool {
	return time.Since(p.g15UpdatedOn) > playerExpiration
}

type Players []Player

// FindFriends searches through all players in the server for friend relationships. This means
// as long as at least one of the friends has their friends list public it should link them.
func (p Players) FindFriends(steamID steamid.SteamID) steamid.Collection {
	var friends steamid.Collection

	for _, player := range p {
		for _, friend := range player.meta.Friends {
			if friend.SteamId == steamID.String() && !slices.Contains(friends, steamID) {
				friends = append(friends, steamID)
			}
		}
	}

	return friends
}
