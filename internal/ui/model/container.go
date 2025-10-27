package model

import (
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
)

func Container(title string, width int, height int, content string) string {
	if height <= 0 || width <= 0 {
		return ""
	}

	return styles.ContainerStyle.
		Border(styles.TitleBorder(styles.ContainerBorder, width, title)).
		Width(width).
		Height(height).
		Render(content)
}
