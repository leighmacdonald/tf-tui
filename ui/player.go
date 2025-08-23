package ui

import (
	"slices"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/tf"
	"github.com/leighmacdonald/tf-tui/tfapi"
)

type Player struct {
	SteamID       steamid.SteamID
	Name          string
	Ping          int
	Score         int
	Deaths        int
	Connected     bool
	Team          tf.Team
	Alive         bool
	Health        int
	Valid         bool
	UserID        int
	Meta          tfapi.MetaProfile
	MetaUpdatedOn time.Time
	G15UpdatedOn  time.Time
}

type Players []Player

// FindFriends searches through all players in the server for friend relationships. This means
// as long as at least one of the friends has their friends list public it should link them.
func (p Players) FindFriends(steamID steamid.SteamID) steamid.Collection {
	var friends steamid.Collection

	for _, player := range p {
		for _, friend := range player.Meta.Friends {
			if friend.SteamId == steamID.String() && !slices.Contains(friends, steamID) {
				friends = append(friends, steamID)
			}
		}
	}

	return friends
}
