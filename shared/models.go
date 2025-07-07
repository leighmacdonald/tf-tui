package shared

import "github.com/leighmacdonald/steamid/v4/steamid"

const MaxDataSize = 102

type PlayerState struct {
	Names     [MaxDataSize]string
	Ping      [MaxDataSize]int
	Score     [MaxDataSize]int
	Deaths    [MaxDataSize]int
	Connected [MaxDataSize]bool
	Team      [MaxDataSize]int
	Alive     [MaxDataSize]bool
	Health    [MaxDataSize]int
	SteamID   [MaxDataSize]steamid.SteamID
	Valid     [MaxDataSize]bool
	UserID    [MaxDataSize]int
}
