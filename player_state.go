package main

import (
	"context"
	"errors"
	"sync"
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

func NewPlayerStates() *PlayerStates {
	return &PlayerStates{
		mu:            &sync.RWMutex{},
		players:       Players{},
		expiration:    time.Second * 30,
		checkInterval: time.Second,
	}
}

type PlayerStates struct {
	mu            *sync.RWMutex
	players       Players
	expiration    time.Duration
	checkInterval time.Duration
}

func (s *PlayerStates) UpdateDumpPlayer(stats tf.DumpPlayer) {
	var players Players //nolint:prealloc
	for idx := range tf.MaxPlayerCount {
		sid := stats.SteamID[idx]
		if !sid.Valid() {
			// TODO verify this is ok, however i think g15 is filled sequentially.
			continue
		}

		player, playerErr := s.Player(sid)
		if playerErr != nil {
			if !errors.Is(playerErr, errPlayerNotFound) {
				return
			}
			player = Player{SteamID: sid, Meta: tfapi.MetaProfile{Bans: []tfapi.Ban{}}}
		}

		player.Valid = stats.Valid[idx]
		player.Health = stats.Health[idx]
		player.Alive = stats.Alive[idx]
		player.Deaths = stats.Deaths[idx]
		player.Ping = stats.Ping[idx]
		player.Health = stats.Health[idx]
		player.Score = stats.Score[idx]
		player.Connected = stats.Connected[idx]
		player.Name = stats.Names[idx]
		player.Team = tf.Team(stats.Team[idx])
		player.UserID = stats.UserID[idx]
		player.G15UpdatedOn = time.Now()
		players = append(players, player)
	}

	s.SetPlayer(players...)
}

func (s *PlayerStates) SetPlayer(updates ...Player) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, player := range updates {
		for playerIdx := range s.players {
			if s.players[playerIdx].SteamID.Equal(player.SteamID) {
				s.players[playerIdx] = player

				continue
			}
		}

		s.players = append(s.players, player)
	}
}

func (s *PlayerStates) UpdateMetaProfile(metaProfiles ...tfapi.MetaProfile) {
	players := make(Players, len(metaProfiles))
	for index, meta := range metaProfiles {
		player, err := s.Player(steamid.New(meta.SteamId))
		if err != nil {
			return
		}

		player.Meta = meta
		players[index] = player
	}

	s.SetPlayer(players...)
}

func (s *PlayerStates) Player(steamID steamid.SteamID) (Player, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, player := range s.players {
		if steamID.Equal(player.SteamID) {
			return player, nil
		}
	}

	return Player{}, errPlayerNotFound
}

func (s *PlayerStates) Players() Players {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.players
}

func (s *PlayerStates) cleaner(ctx context.Context) {
	ticker := time.NewTicker(s.checkInterval)

	for {
		select {
		case <-ticker.C:
			s.removeExpired()
		case <-ctx.Done():
			return
		}
	}
}

func (s *PlayerStates) removeExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	var valid Players //nolint:prealloc
	for _, player := range s.players {
		if time.Since(player.G15UpdatedOn) > s.expiration {
			continue
		}

		valid = append(valid, player)
	}

	s.players = valid
}
