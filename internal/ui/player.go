package ui

import (
	"slices"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/tfapi"
)

type Player struct {
	SteamID                  steamid.SteamID
	Name                     string
	Ping                     int
	Score                    int
	Deaths                   int
	Connected                bool
	Team                     tf.Team
	Alive                    bool
	Health                   int
	Valid                    bool
	UserID                   int
	Bans                     []tfapi.Ban
	CommunityBanned          bool
	CommunityVisibilityState tfapi.MetaProfileCommunityVisibilityState
	CompetitiveTeams         []tfapi.LeaguePlayerTeamHistory
	DaysSinceLastBan         int64
	EconomyBan               string
	Friends                  []tfapi.SteamFriend
	LogsCount                int64
	NumberOfGameBans         int64
	NumberOfVacBans          int64
	PersonaName              string
	ProfileState             tfapi.MetaProfileProfileState
	RealName                 string
	TimeCreated              int64
}

type Players []Player

// FindFriends searches through all players in the server for friend relationships. This means
// as long as at least one of the friends has their friends list public it should link them.
func (p Players) FindFriends(steamID steamid.SteamID) steamid.Collection {
	var friends steamid.Collection

	for _, player := range p {
		for _, friend := range player.Friends {
			if friend.SteamId == steamID.String() && !slices.Contains(friends, steamID) {
				friends = append(friends, steamID)
			}
		}
	}

	return friends
}
