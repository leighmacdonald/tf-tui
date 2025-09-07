package state

import (
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/bd"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/tfapi"
)

type Player struct {
	SteamID       steamid.SteamID
	Name          string
	Ping          int
	Loss          int
	Address       string
	Time          int
	Score         int
	Deaths        int
	Connected     bool
	Team          tf.Team
	Alive         bool
	Health        int
	Valid         bool
	UserID        int
	BDMatches     []bd.Match
	Meta          tfapi.MetaProfile
	MetaUpdatedOn time.Time
	G15UpdatedOn  time.Time
}

type Players []Player
