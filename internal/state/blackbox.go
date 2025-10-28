package state

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/store"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/tf/events"
)

var errBlackBox = errors.New("failed to save blackbox event")

// blackBox handles recording various game events for long term storage.
type blackBox struct {
	db        *store.Queries
	logEvents chan events.Event
	validIDs  []steamid.SteamID
	match     Match
}

func newBlackBox(conn *store.Queries, incomingEvents chan events.Event) *blackBox {
	return &blackBox{db: conn, logEvents: incomingEvents}
}

func (b *blackBox) Start(ctx context.Context) {
	for {
		select {
		case event := <-b.logEvents:
			slog.Info("event")
			var err error
			switch data := event.Data.(type) {
			case events.MsgEvent:
				err = b.onMsg(ctx, event.Timestamp, data)
			case events.KillEvent:
				b.onKill(ctx, data)
			case events.ConnectEvent:
				b.onConnect(ctx, event)
			case events.DisconnectEvent:
			case events.AddressEvent:
				b.match.Address = data.Address.String()
			case events.HostnameEvent:
				b.match.Hostname = data.Hostname
			case events.TagsEvent:
				b.match.Tags = data.Tags
			case events.LobbyEvent:
			case events.StatusIDEvent:
			case events.MapEvent:
			case events.AnyEvent:
			}

			if err != nil {
				slog.Error("Failed to handle log event", slog.String("error", err.Error()))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (b *blackBox) onConnect(_ context.Context, _ events.Event) {
	// if len(b.match.Players) > 0 {
	// 	// Save it
	// }
	b.match = Match{
		Players:  []*PlayerHistory{},
		Messages: []ChatMessage{},
		Tags:     []string{},
	}
}

func (b *blackBox) player(steamID steamid.SteamID) *PlayerHistory {
	for _, player := range b.match.Players {
		if player.SteamID.Equal(steamID) {
			return player
		}
	}

	player := &PlayerHistory{SteamID: steamID}
	b.match.Players = append(b.match.Players, player)

	return player
}

func (b *blackBox) onKill(_ context.Context, event events.KillEvent) {
	player := b.player(event.PlayerSID)
	player.Kills = append(player.Kills, PlayerKill{
		Source:    event.PlayerSID,
		Victim:    event.VictimSID,
		Weapon:    event.Weapon,
		Crit:      event.Crit,
		CreatedOn: event.Timestamp,
	})
}

// ensureSID handles making sure the players steam_id FK is satisfied.
func (b *blackBox) ensureSID(ctx context.Context, steamID steamid.SteamID) error {
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

func (b *blackBox) onMsg(ctx context.Context, timeStamp time.Time, event events.MsgEvent) error {
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
		CreatedOn: timeStamp.Unix(),
	}); err != nil {
		return errors.Join(err, errBlackBox)
	}

	return nil
}

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
