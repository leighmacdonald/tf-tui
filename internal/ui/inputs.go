package ui

import tea "github.com/charmbracelet/bubbletea"

// keyZone defines the distinct areas of the ui in which the keyboard can be interacted with.
// Only one zone, with the addition of the default global zone, will be active at any one time.
type keyZone int

const (
	playerTableRED keyZone = iota
	playerTableBLU
	listCvars
	listSourcemod
	listMetamod
	serverTable
	chatInput
	consoleInput
	configInput
)

func setKeyZone(zone keyZone) tea.Cmd {
	return func() tea.Msg { return zone }
}
