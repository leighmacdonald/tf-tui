package main

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/leighmacdonald/tf-tui/styles"
)

func newTextInputModel(value string, placeholder string) textinput.Model {
	input := textinput.New()
	input.Cursor.Style = styles.CursorStyle
	input.SetValue(value)
	input.CharLimit = 128
	input.Placeholder = placeholder
	input.PromptStyle = styles.FocusedStyle
	input.TextStyle = styles.FocusedStyle

	return input
}

func newTextInputPasswordModel(value string, placeholder string) textinput.Model {
	input := newTextInputModel(value, placeholder)
	input.Cursor.Style = styles.CursorStyle
	input.CharLimit = 128
	input.EchoMode = textinput.EchoPassword

	return input
}
