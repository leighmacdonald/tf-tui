package ui

import (
	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/table"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
)

func newTextInputModel(value string, placeholder string) textinput.Model {
	input := textinput.New()
	input.Styles.Cursor.Shape = tea.CursorBar
	input.SetValue(value)
	input.CharLimit = 127
	input.Placeholder = placeholder
	input.Styles.Focused.Prompt = styles.NoStyle
	// input.PromptStyle = styles.NoStyle
	// input.TextStyle = styles.NoStyle

	return input
}

func renderTitleBar(width int, value string) string {
	return lipgloss.
		NewStyle().
		Width(width - 2).
		Bold(false).
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
