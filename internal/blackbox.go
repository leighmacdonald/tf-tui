package internal

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/store"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/tf/events"
)

var errBlackBox = errors.New("failed to save blackbox event")

type PlayerKill struct {
	Source    steamid.SteamID
	Victim    steamid.SteamID
	Weapon    string
	Crit      bool
	CreatedOn time.Time
}

type PlayerHistory struct {
	SteamID   steamid.SteamID
	Name      string
	Score     int
	Deaths    int
	Ping      int
	Team      tf.Team
	Connected int
	Kills     []PlayerKill
}

type ChatMessage struct {
	SteamID steamid.SteamID
}

type Match struct {
	Players  []*PlayerHistory
	Messages []ChatMessage
	Hostname string
	Address  string
	Tags     []string
}

// BlackBox handles recording various game events for long term storage.
type BlackBox struct {
	db        *store.Queries
	logEvents chan events.Event
	validIDs  []steamid.SteamID
	match     Match
}

func NewBlackBox(conn *store.Queries) *BlackBox {
	return &BlackBox{db: conn, logEvents: make(chan events.Event)}
}

func (b *BlackBox) start(ctx context.Context) {
	for {
		select {
		case event := <-b.logEvents:
			var err error
			switch event.Type {
			case events.Msg:
				err = b.onMsg(ctx, event)
			case events.Kill:
				b.onKill(ctx, event)
			case events.Connect:
				b.onConnect(ctx, event)
			case events.Disconnect:
			case events.Address:
				b.match.Address = event.MetaData
			case events.Hostname:
				b.match.Hostname = event.MetaData
			case events.Tags:
				b.match.Tags = strings.Split(event.MetaData, ",")
			case events.Lobby:
			case events.StatusID:
			}

			if err != nil {
				slog.Error("Failed to handle log event", slog.String("error", err.Error()))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (b *BlackBox) onConnect(_ context.Context, _ events.Event) {
	// if len(b.match.Players) > 0 {
	// 	// Save it
	// }
	b.match = Match{
		Players:  []*PlayerHistory{},
		Messages: []ChatMessage{},
		Tags:     []string{},
	}
}

func (b *BlackBox) player(steamID steamid.SteamID) *PlayerHistory {
	for _, player := range b.match.Players {
		if player.SteamID.Equal(steamID) {
			return player
		}
	}

	player := &PlayerHistory{SteamID: steamID}
	b.match.Players = append(b.match.Players, player)

	return player
}

func (b *BlackBox) onKill(_ context.Context, event events.Event) {
	player := b.player(event.PlayerSID)
	player.Kills = append(player.Kills, PlayerKill{
		Source:    event.PlayerSID,
		Victim:    event.VictimSID,
		Weapon:    event.MetaData,
		Crit:      false,
		CreatedOn: event.Timestamp,
	})
}

// ensureSID handles making sure the players steam_id FK is satisfied.
func (b *BlackBox) ensureSID(ctx context.Context, steamID steamid.SteamID) error {
	if slices.Contains(b.validIDs, steamID) {
		return nil
	}

	args := store.InsertPlayerParams{
		SteamID:   steamID.Int64(),
		Name:      "",
		CreatedOn: time.Now().Unix(),
		UpdatedOn: time.Now().Unix(),
	}
	if err := b.db.InsertPlayer(ctx, args); err != nil {
		return errors.Join(err, errBlackBox)
	}

	b.validIDs = append(b.validIDs, steamID)

	return nil
}

func (b *BlackBox) onMsg(ctx context.Context, event events.Event) error {
	if errEnsure := b.ensureSID(ctx, event.PlayerSID); errEnsure != nil {
		return errEnsure
	}

	teamOnly := int64(0)
	if event.TeamOnly {
		teamOnly = 1
	}

	if err := b.db.InsertChat(ctx, store.InsertChatParams{
		SteamID:   event.PlayerSID.Int64(),
		Name:      event.Player,
		Message:   event.Message,
		TeamOnly:  teamOnly,
		CreatedOn: event.Timestamp.Unix(),
	}); err != nil {
		return errors.Join(err, errBlackBox)
	}

	return nil
}
