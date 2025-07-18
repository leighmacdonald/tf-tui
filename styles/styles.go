package styles

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	Accent = lipgloss.Color("205")
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
	Black     = lipgloss.Color("#111111")
	Gray      = lipgloss.Color("245")
	LightGray = lipgloss.Color("241")

	Red = lipgloss.Color("#B8383B")
	Blu = lipgloss.Color("#5885A2")

	HeaderStyleRed  = lipgloss.NewStyle().Foreground(Red).Bold(true).Align(lipgloss.Center)
	HeaderStyleBlu  = lipgloss.NewStyle().Foreground(Blu).Bold(true).Align(lipgloss.Center)
	CellStyleName   = lipgloss.NewStyle().Padding(0, 1).Width(32)
	CellStyle       = lipgloss.NewStyle().Padding(0, 1).Width(6)
	OddRowStyleName = CellStyleName.Foreground(Gray)
	OddRowStyle     = CellStyle.Foreground(Gray)
	EvenRowStyle    = CellStyle.Foreground(LightGray)

	SelectedCellStyleRed     = lipgloss.NewStyle().Padding(0, 1).Bold(true).Background(Red).Foreground(Black)
	SelectedCellStyleNameRed = lipgloss.NewStyle().Padding(0, 1).Bold(true).Width(32).Background(Red).Foreground(Black)

	SelectedCellStyleBlu     = lipgloss.NewStyle().Padding(0, 1).Bold(true).Background(Blu).Foreground(Black)
	SelectedCellStyleNameBlu = SelectedCellStyleBlu.Width(32).Background(Blu).Foreground(Black)

	PanelBorder = lipgloss.NewStyle().
			Bold(true).
			Foreground(Red).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(Gray).
		//BorderBackground(Gray).
		//BorderTop(true).
		//BorderLeft(true).
		Padding(0)

	PanelLabel   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	PanelValue   = lipgloss.NewStyle().Width(60)
	TabWidth     = 12
	TabsInactive = lipgloss.NewStyle().Inline(true).Bold(true).
			Border(lipgloss.NormalBorder()).BorderStyle(lipgloss.InnerHalfBlockBorder()).Padding(1).Width(TabWidth)
	TabsActive = lipgloss.NewStyle().Inline(true).Bold(true).
			Border(lipgloss.NormalBorder()).Padding(1).Width(TabWidth).Foreground(Blu)
)
