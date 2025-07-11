package styles

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	Status = lipgloss.NewStyle().Bold(true).Foreground(Red).
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(Blu).GetBorderStyle()).Padding(1)
	Title = lipgloss.NewStyle().Bold(true).Foreground(Blu).Padding(1)

	FocusedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	BlurredStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	CursorStyle         = FocusedStyle
	NoStyle             = lipgloss.NewStyle()
	HelpStyle           = BlurredStyle
	CursorModeHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	FocusedSubmitButton = FocusedStyle.Render("[ Submit ]")
	BlurredSubmitButton = fmt.Sprintf("[ %s ]", BlurredStyle.Render("Submit"))

	// Tables
	Gray      = lipgloss.Color("245")
	LightGray = lipgloss.Color("241")

	Red = lipgloss.Color("#B8383B")
	Blu = lipgloss.Color("#5885A2")

	HeaderStyleRed  = lipgloss.NewStyle().Foreground(Red).Bold(true).Align(lipgloss.Center)
	HeaderStyleBlu  = lipgloss.NewStyle().Foreground(Blu).Bold(true).Align(lipgloss.Center)
	CellStyleName   = lipgloss.NewStyle().Padding(0, 1).Width(30)
	CellStyle       = lipgloss.NewStyle().Padding(0, 1).Width(6)
	OddRowStyleName = CellStyleName.Foreground(Gray)
	OddRowStyle     = CellStyle.Foreground(Gray)
	EvenRowStyle    = CellStyle.Foreground(LightGray)
)
