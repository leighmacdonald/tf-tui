package player_transform

import (
	"github.com/leighmacdonald/tf-tui/shared"
)

func OnPlayerState(state shared.PlayerState) shared.PlayerState {
	state.Score[0] = 100

	return state
}
