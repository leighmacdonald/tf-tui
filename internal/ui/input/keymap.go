package input

import "github.com/charmbracelet/bubbles/key"

type Map struct {
	Start         key.Binding
	Stop          key.Binding
	Reset         key.Binding
	Quit          key.Binding
	Config        key.Binding
	Chat          key.Binding
	Up            key.Binding
	Down          key.Binding
	Left          key.Binding
	Right         key.Binding
	Accept        key.Binding
	Back          key.Binding
	PrevTab       key.Binding
	NextTab       key.Binding
	Overview      key.Binding
	Bans          key.Binding
	BD            key.Binding
	Comp          key.Binding
	Notes         key.Binding
	Console       key.Binding
	Help          key.Binding
	ConsoleInput  key.Binding
	ConsoleCancel key.Binding
}

// TODO make configurable.
var Default = Map{
	ConsoleInput: key.NewBinding(
		key.WithKeys("return"),
		key.WithHelp("<return>", "Send command")),
	ConsoleCancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("<esc>", "Cancel input")),
	Help: key.NewBinding(
		key.WithKeys("h", "H"),
		key.WithHelp("h", "Help"),
	),
	Accept: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "Select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "Back"),
	),
	Reset: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "reset"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "Quit"),
	),
	Config: key.NewBinding(
		key.WithKeys("E"),
		key.WithHelp("E", "Conf"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑", "Up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓", "Down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←", "RED"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→", "BLU"),
	),
	NextTab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "Next Tab"),
	),
	PrevTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift tab", "Prev Tab"),
	),
	Overview: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "Overview"),
	),
	Bans: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "Bans"),
	),
	BD: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "Bot Detector"),
	),
	Comp: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "Comp"),
	),
	Notes: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "Notes"),
	),
	Console: key.NewBinding(
		key.WithKeys("`"),
		key.WithHelp("`", "Console"),
	),
	Chat: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "Chat"),
	),
}
