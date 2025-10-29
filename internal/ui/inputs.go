package ui

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"
)

// direction defines the cardinal directions the users can use in the UI.
type direction int

const (
	up direction = iota //nolint:varnamelen
	down
	left
	right
)

var (
	serverZones = keyZoneGroup{serverTable, serverOverview, listMetamod, listSourcemod, listCvars}
	playerZones = keyZoneGroup{playerTableRED, playerTableBLU, playerOverview}
	banZones    = keyZoneGroup{playerTableRED, playerTableBLU, banTable}
	bdZones     = keyZoneGroup{playerTableRED, playerTableBLU, bdTable}
	compZones   = keyZoneGroup{playerTableRED, playerTableBLU, compTable}
)

// keyZone defines the distinct areas of the ui in which the keyboard can be interacted with.
// Only one zone, with the addition of the default global zone, will be active at any one time.
type keyZone int

const (
	playerTableRED keyZone = iota
	playerTableBLU
	playerOverview
	bdTable
	banTable
	compTable
	listCvars
	listSourcemod
	listMetamod
	serverTable
	serverOverview
	chatInput
	consoleInput
	configInput
)

func setKeyZone(zone keyZone) tea.Cmd {
	return func() tea.Msg { return zone }
}

type keyZoneGroup []keyZone

func (z keyZoneGroup) next(current keyZone, dir direction) keyZone {
	index := slices.Index(z, current)
	if index == -1 {
		return z[0]
	}

	switch dir {
	case left:
		// Wrap into the last entry
		if index-1 < 0 {
			return z[len(z)-1]
		}
		return z[index-1]
	case right:
		// Wrap into the first entry
		if index+1 >= len(z) {
			return z[0]
		}
		return z[index+1]
	default:
		return current
	}
}
