package styles

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	Accent = lipgloss.Color("#f4722b")

	HeaderContainerStyle  = lipgloss.NewStyle().Align(lipgloss.Center)
	ContentContainerStyle = lipgloss.NewStyle().Align(lipgloss.Center)
	FooterContainerStyle  = lipgloss.NewStyle().Align(lipgloss.Center)

	FocusedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	BlurredStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Background(Black)
	CursorStyle         = FocusedStyle
	NoStyle             = lipgloss.NewStyle()
	HelpStyle           = BlurredStyle
	CursorModeHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	FocusedSubmitButton = lipgloss.NewStyle().Foreground(Accent).Render("[ Submit ]")
	BlurredSubmitButton = fmt.Sprintf("[ %s ]", BlurredStyle.Render("Submit"))

	// Tables.
	Black     = lipgloss.Color("#111111")
	Gray      = lipgloss.Color("#3e3e3e")
	LightGray = lipgloss.Color("#9a9280")

	Red = lipgloss.Color("#B8383B")
	Blu = lipgloss.Color("#5885A2")

	ColourStrange = lipgloss.Color("#cf6a32")
	ColourLimited = lipgloss.Color("#ffd700")
	ColourGenuine = lipgloss.Color("#4d7455")
	ColourUnusual = lipgloss.Color("#8650ac")
	ColourVintage = lipgloss.Color("#476291")

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
		// BorderBackground(Gray).
		// BorderTop(true).
		// BorderLeft(true).
		Padding(0)

	ConsoleTime = lipgloss.NewStyle().Foreground(Gray).Background(Black)
	ConsoleMsg  = lipgloss.NewStyle().Foreground(ColourLimited)

	PanelLabel   = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Align(lipgloss.Right).Width(20)
	PanelValue   = lipgloss.NewStyle().Width(60)
	TabContainer = lipgloss.NewStyle().Align(lipgloss.Center).Background(Black)
	TabsInactive = lipgloss.NewStyle().Background(Black).Bold(true).
			Foreground(ColourVintage).PaddingLeft(2).PaddingRight(2)
	TabsActive = lipgloss.NewStyle().
			Background(Black).Bold(true).
			Foreground(ColourUnusual).PaddingLeft(2).PaddingRight(2)

	StatusHostname = lipgloss.NewStyle().Foreground(ColourStrange).Background(Black).PaddingRight(2).PaddingLeft(2).Bold(true)
	StatusMap      = lipgloss.NewStyle().Foreground(ColourGenuine).Background(Black).PaddingRight(2).PaddingLeft(2).Bold(true)
	StatusError    = lipgloss.NewStyle().Foreground(Red).Background(Black).Align(lipgloss.Right).Bold(true).PaddingRight(2)
	StatusMessage  = lipgloss.NewStyle().Foreground(ColourGenuine).Background(Black).Align(lipgloss.Right).Bold(true).PaddingRight(2)
	StatusRedTeam  = lipgloss.NewStyle().Foreground(Red).Background(Black).Bold(true).PaddingLeft(0).Align(lipgloss.Center).PaddingRight(0)
	StatusBluTeam  = lipgloss.NewStyle().Foreground(Blu).Background(Black).Bold(true).PaddingLeft(0).Align(lipgloss.Center).PaddingRight(0)
	// 🚨 👮 💂 🕵️ 👷 🐈 🏟️ 🪵 ♻️.
	IconComp   = "🏁"
	IconCheck  = "✅"
	IconBans   = "🛑"
	IconVac    = "👮"
	IconDrCool = "😎"
	IconNotes  = "📓"
	IconInfo   = "💡"
)

func DetailRow(label string, value string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top,
		PanelLabel.Render(label+" "),
		PanelValue.Render(value))
}
