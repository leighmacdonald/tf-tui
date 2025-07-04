package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/leighmacdonald/tf-tui/styles"
)

type widgetConfig struct {
	config       Config
	inputAddr    textinput.Model
	passwordAddr textinput.Model
	focusIndex   int
}

func newWidgetConfig(config Config) widgetConfig {
	address := config.Address
	if address == "" {
		address = "127.0.0.1:27015"
	}
	return widgetConfig{
		config:       config,
		inputAddr:    newTextInputModel(address, "127.0.0.1:27015"),
		passwordAddr: newTextInputPasswordModel(config.Password, ""),
	}
}

type helpKeymap struct {
	up   key.Binding
	down key.Binding
	esc  key.Binding
}

func newHelpKeymap() helpKeymap {
	return helpKeymap{
		up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("up", "Move up")),
		down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("down", "Move down")),
		esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "Go back"),
		),
	}
}
func (w widgetConfig) Render() string {
	keyMaps := newHelpKeymap()
	var b strings.Builder

	b.WriteString(styles.HelpStyle.Render("\nRCON Address:  "))
	b.WriteString(w.inputAddr.View() + "\n")
	b.WriteString(styles.HelpStyle.Render("RCON Password: "))
	b.WriteString(w.passwordAddr.View())

	if w.focusIndex == 2 {
		b.WriteString("\n\n" + styles.FocusedSubmitButton)
	} else {
		b.WriteString("\n\n" + styles.BlurredSubmitButton)
	}

	helpView := help.New()

	b.WriteString("\n\n" + helpView.ShortHelpView([]key.Binding{
		keyMaps.up,
		keyMaps.down,
		keyMaps.esc,
	}))

	return b.String()
}
