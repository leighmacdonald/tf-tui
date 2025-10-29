package model

import (
	"slices"

	"github.com/leighmacdonald/tf-tui/internal/ui/input"
)

var (
	ServerZones  = KeyZoneGroup{KZserverTable, KZserverOverview, KZlistMetamod, KZlistSourcemod, KZlistCvars}
	PlayerZones  = KeyZoneGroup{KZplayerTableRED, KZplayerTableBLU, KZplayerOverview}
	BanZones     = KeyZoneGroup{KZplayerTableRED, KZplayerTableBLU, KZbanTable}
	BDZones      = KeyZoneGroup{KZplayerTableRED, KZplayerTableBLU, KZbdTable}
	CompZones    = KeyZoneGroup{KZplayerTableRED, KZplayerTableBLU, KZcompTable}
	ChatZones    = KeyZoneGroup{KZplayerTableRED, KZplayerTableBLU, KZchatInput}
	ConsoleZones = KeyZoneGroup{KZplayerTableRED, KZplayerTableBLU, KZconsoleInput}
)

// KeyZone defines the distinct areas of the ui in which the keyboard can be interacted with.
// Only one zone, with the addition of the default global zone, will be active at any one time.
type KeyZone int

const (
	KZplayerTableRED KeyZone = iota
	KZplayerTableBLU
	KZplayerOverview
	KZbdTable
	KZbanTable
	KZcompTable
	KZlistCvars
	KZlistSourcemod
	KZlistMetamod
	KZserverTable
	KZserverOverview
	KZchatInput
	KZconsoleInput
	KZconfigInput
)

type KeyZoneGroup []KeyZone

func (z KeyZoneGroup) Next(current KeyZone, dir input.Direction) KeyZone {
	index := slices.Index(z, current)
	if index == -1 {
		return z[0]
	}

	switch dir {
	case input.Left:
		// Wrap into the last entry
		if index-1 < 0 {
			return z[len(z)-1]
		}
		return z[index-1]
	case input.Right:
		// Wrap into the first entry
		if index+1 >= len(z) {
			return z[0]
		}
		return z[index+1]
	default:
		return current
	}
}
