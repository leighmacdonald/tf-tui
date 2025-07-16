package player_transform

import (
	"tftui/tftui"
)

func OnPlayerState(state tftui.PlayerState) tftui.PlayerState {
	state.Score[0] = 100

	return state
}
