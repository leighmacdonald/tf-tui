package model

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
)

func Container(title string, width int, height int, content string, active bool) string {
	if height <= 0 || width <= 0 {
		return ""
	}

	var base lipgloss.Style
	if active {
		base = styles.ContainerStyleActive
	} else {
		base = styles.ContainerStyle
	}

	return base.
		Border(styles.TitleBorder(styles.ContainerBorder, width, title)).
		Width(width).
		Height(height).
		Render(content)
}
