package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/leighmacdonald/tf-tui/ui/styles"
)

func newTextInputModel(value string, placeholder string) textinput.Model {
	input := textinput.New()
	input.Cursor.Style = styles.CursorStyle
	input.SetValue(value)
	input.CharLimit = 127
	input.Placeholder = placeholder
	input.PromptStyle = styles.NoStyle
	input.TextStyle = styles.NoStyle

	return input
}

func renderTitleBar(width int, value string) string {
	return lipgloss.
		NewStyle().
		Width(width - 2).
		Bold(true).
		Align(lipgloss.Center).
		Background(styles.Black).
		Foreground(styles.ColourStrange).
		PaddingLeft(0).
		PaddingRight(0).
		Render(value)
}

func newUnstyledTable(headers ...string) *table.Table {
	return table.New().
		Border(lipgloss.NormalBorder()).
		BorderColumn(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderTop(false).
		BorderHeader(false).
		Headers(headers...)
}
