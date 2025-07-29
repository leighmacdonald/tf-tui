package main

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/leighmacdonald/tf-tui/styles"
)

func NewTextInputModel(value string, placeholder string) textinput.Model {
	input := textinput.New()
	input.Cursor.Style = styles.CursorStyle
	input.SetValue(value)
	input.CharLimit = 127
	input.Placeholder = placeholder
	input.PromptStyle = styles.FocusedStyle
	input.TextStyle = styles.FocusedStyle

	return input
}

func NewTextInputPasswordModel(value string, placeholder string) textinput.Model {
	input := NewTextInputModel(value, placeholder)
	input.Cursor.Style = styles.CursorStyle
	input.CharLimit = 128
	input.EchoMode = textinput.EchoPassword

	return input
}
