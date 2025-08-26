package internal

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/store"
	"github.com/leighmacdonald/tf-tui/internal/tf"
)

type PlayerKill struct {
	Source    steamid.SteamID
	Victim    steamid.SteamID
	Weapon    string
	Crit      bool
	CreatedOn time.Time
}

type PlayerHistroy struct {
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
	Players  []PlayerHistroy
	Messages []ChatMessage
}

// BlackBox handles recording various game events for long term storage.
type BlackBox struct {
	db        *store.Queries
	logEvents chan tf.LogEvent
	validIds  []steamid.SteamID
	match     Match
}

func NewBlackBox(conn *store.Queries) *BlackBox {
	return &BlackBox{db: conn, logEvents: make(chan tf.LogEvent)}
}

func (b *BlackBox) start(ctx context.Context) {
	for {
		select {
		case event := <-b.logEvents:
			var err error
			switch event.Type {
			case tf.EvtMsg:
				err = b.saveChatMsg(ctx, event)
			case tf.EvtKill:
				err = b.onKill(ctx, event)
			case tf.EvtConnect:
				err = b.onConnect(ctx, event)
			case tf.EvtDisconnect:
			}

			if err != nil {
				slog.Error("Failed to handle log event", slog.String("error", err.Error()))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (b *BlackBox) onConnect(ctx context.Context, event tf.LogEvent) error {
	if len(b.match.Players) > 0 {
		// Save it
	}

	b.match = Match{
		Players: []PlayerHistroy{},
	}

	return nil
}

func (b *BlackBox) onKill(ctx context.Context, event tf.LogEvent) error {
	for _, playerSID := range []steamid.SteamID{event.PlayerSID, event.VictimSID} {
		if err := b.ensureSID(ctx, playerSID); err != nil {
			return err
		}
	}
}

// ensureSID handles making sure the players steam_id FK is satisfied.
func (b *BlackBox) ensureSID(ctx context.Context, steamID steamid.SteamID) error {
	if slices.Contains(b.validIds, steamID) {
		return nil
	}

	args := store.InsertPlayerParams{
		SteamID:   steamID.Int64(),
		Name:      "",
		CreatedOn: time.Now().Unix(),
		UpdatedOn: time.Now().Unix(),
	}
	if err := b.db.InsertPlayer(ctx, args); err != nil {
		return err
	}

	b.validIds = append(b.validIds, steamID)

	return nil
}

func (b *BlackBox) saveChatMsg(ctx context.Context, event tf.LogEvent) error {
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
		return err
	}

	return nil
}
