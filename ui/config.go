package ui

import "github.com/charmbracelet/bubbles/key"

type keymap struct {
	start         key.Binding
	stop          key.Binding
	reset         key.Binding
	quit          key.Binding
	config        key.Binding
	chat          key.Binding
	up            key.Binding
	down          key.Binding
	left          key.Binding
	right         key.Binding
	accept        key.Binding
	back          key.Binding
	prevTab       key.Binding
	nextTab       key.Binding
	overview      key.Binding
	bans          key.Binding
	bd            key.Binding
	comp          key.Binding
	notes         key.Binding
	console       key.Binding
	help          key.Binding
	consoleInput  key.Binding
	consoleCancel key.Binding
}

// TODO make configurable.
var DefaultKeyMap = keymap{
	consoleInput: key.NewBinding(
		key.WithKeys("return"),
		key.WithHelp("<return>", "Send command")),
	consoleCancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("<esc>", "Cancel input")),
	help: key.NewBinding(
		key.WithKeys("h", "H"),
		key.WithHelp("h", "Help"),
	),
	accept: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "Select"),
	),
	back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "Back"),
	),
	reset: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "reset"),
	),
	quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "Quit"),
	),
	config: key.NewBinding(
		key.WithKeys("E"),
		key.WithHelp("E", "Conf"),
	),
	up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑", "Up"),
	),
	down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓", "Down"),
	),
	left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←", "RED"),
	),
	right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→", "BLU"),
	),
	nextTab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "Next Tab"),
	),
	prevTab: key.NewBinding(
		key.WithKeys("shift tab"),
		key.WithHelp("shift tab", "Prev Tab"),
	),
	overview: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "Overview"),
	),
	bans: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "Bans"),
	),
	bd: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "Bot Detector"),
	),
	comp: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "Comp"),
	),
	notes: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "Notes"),
	),
	console: key.NewBinding(
		key.WithKeys("`"),
		key.WithHelp("`", "Console"),
	),
	chat: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "Chat"),
	),
}
