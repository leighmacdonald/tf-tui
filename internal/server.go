package internal

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/tfapi"
)

func newServerState(server config.Server) *serverState {
	return &serverState{
		mu:     &sync.RWMutex{},
		server: server,
	}
}

type serverState struct {
	mu      *sync.RWMutex
	players Players
	server  config.Server
}

func (s *serverState) SetPlayer(updates ...Player) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var existing bool
	for _, player := range updates {
		for playerIdx := range s.players {
			if s.players[playerIdx].SteamID.Equal(player.SteamID) {
				s.players[playerIdx] = player
				existing = true

				continue
			}
		}
		if !existing {
			s.players = append(s.players, player)
		}
	}
}

func (s *serverState) onDumpTick(ctx context.Context) {
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(3)

	go func() {
		defer waitGroup.Done()
		s.updateMetaProfile(ctx)
	}()

	go func() {
		defer waitGroup.Done()
		s.updateBD()
	}()

	go func() {
		defer waitGroup.Done()
		s.updateDump(ctx)
	}()

	waitGroup.Wait()
}

func (s *serverState) Player(steamID steamid.SteamID) (Player, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, player := range s.players {
		if steamID.Equal(player.SteamID) {
			return player, nil
		}
	}

	return Player{}, errPlayerNotFound
}

func (s *serverState) Players() Players {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.players
}

func (s *serverState) updateDump(ctx context.Context) {
	dump, stats, errDump := s.dumpFetcher.Fetch(ctx)
	if errDump != nil {
		// s.uiUpdates <- ui.StatusMsg{
		// 	Err:     true,
		// 	Message: errDump.Error(),
		// }
		//
		// An error result will return a copy of the last successful dump still.
		slog.Error("Failed to fetch player dump", slog.String("error", errDump.Error()))
	}

	s.UpdateStats(stats)
	s.UpdateDumpPlayer(dump)
}

func (s *serverState) UpdateMetaProfile(metaProfiles ...tfapi.MetaProfile) {
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

func (s *serverState) removeExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	var valid Players
	for _, player := range s.players {
		if time.Since(player.G15UpdatedOn) > playerTimeout {
			continue
		}

		valid = append(valid, player)
	}

	s.players = valid
}

func (s *serverState) UpdateDumpPlayer(stats tf.DumpPlayer) {
	var players Players
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
		player.Loss = stats.Loss[idx]
		player.Address = stats.Address[idx]
		player.Time = stats.Time[idx]
		player.Team = stats.Team[idx]
		player.UserID = stats.UserID[idx]
		player.G15UpdatedOn = time.Now()
		players = append(players, player)
	}

	s.SetPlayer(players...)
}
