package styles

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	Accent = lipgloss.Color("#f4722b")

	ContainerTitle       = lipgloss.NewStyle().Bold(true)
	ContainerBorder      = lipgloss.DoubleBorder()
	ContainerStyle       = lipgloss.NewStyle().Border(ContainerBorder).BorderForeground(Gray)
	ContainerStyleActive = lipgloss.NewStyle().Border(ContainerBorder).BorderForeground(Blu)

	HeaderContainerStyle  = lipgloss.NewStyle().Align(lipgloss.Center)
	ContentContainerStyle = lipgloss.NewStyle().Align(lipgloss.Center)
	FooterContainerStyle  = lipgloss.NewStyle().Align(lipgloss.Center)

	FocusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	BlurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Background(Black)
	CursorStyle  = FocusedStyle
	NoStyle      = lipgloss.NewStyle()
	HelpStyle    = BlurredStyle

	FocusedSubmitButton = lipgloss.NewStyle().Foreground(Accent).Render("[ Submit ]")
	BlurredSubmitButton = fmt.Sprintf("[ %s ]", BlurredStyle.Render("Submit"))

	// Tables.
	Black       = lipgloss.Color("#111111")
	Gray        = lipgloss.Color("#3e3e3e")
	GrayDark    = lipgloss.Color("#2f3030")
	GrayDarkAlt = lipgloss.Color("#0f0f0f")
	// LightGray   = lipgloss.Color("#111111").
	White  = lipgloss.Color("#cccccc")
	Whiter = lipgloss.Color("#aaaaaa")

	Red = lipgloss.Color("#B8383B")
	Blu = lipgloss.Color("#5885A2")

	ColourStrange = lipgloss.Color("#cf6a32")
	ColourLimited = lipgloss.Color("#ffd700")
	ColourGenuine = lipgloss.Color("#4d7455")
	ColourUnusual = lipgloss.Color("#8650ac")
	ColourVintage = lipgloss.Color("#476291")

	PluginTitle = lipgloss.NewStyle().Foreground(Gray).Padding(0)
	PluginItem  = lipgloss.NewStyle().Foreground(Gray).Padding(0)

	HeaderStyleRed = lipgloss.NewStyle().Foreground(Red).Bold(true).Align(lipgloss.Left).PaddingLeft(0)
	HeaderStyleBlu = lipgloss.NewStyle().Foreground(Blu).Bold(true).Align(lipgloss.Left).PaddingLeft(0)

	SelectedCellStyleRed     = lipgloss.NewStyle().Padding(0).Bold(true).Background(Red).Foreground(Black)
	SelectedCellStyleNameRed = lipgloss.NewStyle().Padding(0).Bold(true).Width(32).Background(Red).Foreground(Black)

	ListSelectedRow  = lipgloss.NewStyle().Padding(0).Bold(true).Foreground(Blu).Inline(true)
	ListUnelectedRow = lipgloss.NewStyle().Padding(0).Bold(false).Foreground(White).Inline(true)

	SelectedCellStyleBlu     = lipgloss.NewStyle().Padding(0).Bold(true).Background(Blu).Foreground(Black)
	SelectedCellStyleNameBlu = SelectedCellStyleBlu.Width(32).Background(Blu).Foreground(Black)

	PlayerTableRow     = lipgloss.NewStyle().Foreground(White)
	PlayerTableRowOdd  = lipgloss.NewStyle().Foreground(Whiter)
	PlayerTableRowSelf = lipgloss.NewStyle().Foreground(ColourGenuine)

	ConsoleTime       = lipgloss.NewStyle().Foreground(Gray).Background(Black)
	ConsoleOther      = lipgloss.NewStyle().Foreground(ColourVintage)
	ConsoleMsg        = lipgloss.NewStyle().Foreground(ColourLimited)
	ConsoleKill       = lipgloss.NewStyle().Foreground(Red)
	ConsoleConnect    = lipgloss.NewStyle().Foreground(ColourStrange)
	ConsoleDisconnect = lipgloss.NewStyle().Foreground(ColourStrange)
	ConsoleStatusID   = lipgloss.NewStyle().Foreground(ColourVintage)
	ConsoleHostname   = lipgloss.NewStyle().Foreground(ColourGenuine)
	ConsoleMap        = lipgloss.NewStyle().Foreground(ColourUnusual)
	ConsoleTags       = lipgloss.NewStyle().Foreground(Red)
	ConsoleAddress    = lipgloss.NewStyle().Foreground(Blu)
	ConsoleLobby      = lipgloss.NewStyle().Foreground(ColourVintage)
	ConsolePrompt     = lipgloss.NewStyle().Padding(0).Foreground(ColourVintage).Background(Black).Inline(true).Render("RCON  ")

	PanelLabel   = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Align(lipgloss.Right).Width(16)
	PanelValue   = lipgloss.NewStyle().Width(60)
	TabContainer = lipgloss.NewStyle().Align(lipgloss.Center)
	TabsInactive = lipgloss.NewStyle().Bold(true).
			Foreground(ColourVintage).PaddingLeft(2).PaddingRight(2)
	TabsActive = lipgloss.NewStyle().
			Foreground(ColourUnusual).PaddingLeft(2).PaddingRight(2)

	StatusHostname = lipgloss.NewStyle().Foreground(ColourStrange).PaddingRight(2).PaddingLeft(1).Bold(true)
	StatusMap      = lipgloss.NewStyle().Foreground(ColourGenuine).PaddingRight(2).PaddingLeft(1).Bold(true)
	StatusError    = lipgloss.NewStyle().Foreground(Red).Align(lipgloss.Right).Bold(true).PaddingRight(2)
	StatusMessage  = lipgloss.NewStyle().Foreground(ColourGenuine).Align(lipgloss.Right).Bold(true).PaddingRight(2)
	StatusRedTeam  = lipgloss.NewStyle().Foreground(Red).Bold(true).PaddingLeft(0).Align(lipgloss.Center).PaddingRight(0)
	StatusBluTeam  = lipgloss.NewStyle().Foreground(Blu).Bold(true).PaddingLeft(0).Align(lipgloss.Center).PaddingRight(0)

	StatusHelp    = lipgloss.NewStyle().Foreground(Gray).Bold(true).Align(lipgloss.Center)
	StatusVersion = lipgloss.NewStyle().Foreground(ColourGenuine).Bold(true).Align(lipgloss.Center)

	ChatNameOther = lipgloss.NewStyle().Foreground(ColourLimited).Bold(true).Align(lipgloss.Left)
	ChatNameBlu   = lipgloss.NewStyle().Width(38).Foreground(Blu).Bold(true).Align(lipgloss.Left)
	ChatNameRed   = lipgloss.NewStyle().Width(38).Foreground(Red).Bold(true).Align(lipgloss.Left)
	ChatMessage   = lipgloss.NewStyle()
	ChatTime      = lipgloss.NewStyle().Width(14).Foreground(Gray)

	BanTableHeading = lipgloss.NewStyle().Background(Black).Foreground(Red).Bold(true)

	TableRowValuesEven = lipgloss.NewStyle().Background(GrayDark)
	TableRowValuesOdd  = lipgloss.NewStyle().Background(GrayDarkAlt)

	InfoMessage = lipgloss.NewStyle().Align(lipgloss.Center).Padding(1)

	HelpBox = lipgloss.NewStyle().Padding(3)

	// 🚨 👮 💂 🕵️ 👷 🐈 🏟️ 🪵 ♻️.
	IconServers = "🌍"
	IconPlayers = "👥"
	IconDead    = "💀"
	IconComp    = "🏁"
	IconCheck   = "✅"
	IconBans    = "🛑"
	IconVac     = "👮"
	// IconNotes   = "📓".
	IconInfo    = "💡"
	IconChat    = "🌮"
	IconConsole = "🐤"
	IconNoBans  = "🍕"
	IconNoComp  = "🍣"
	IconBD      = "🕵️"
)

func DetailRow(label string, value string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top,
		PanelLabel.Render(label+" "),
		PanelValue.Render(value))
}

// WrapX will wrap a centered string with the supplied character up to the lenth specified.
func WrapX(width int, value string, character string) string {
	all := width - lipgloss.Width(value)
	return strings.Repeat(character, all/2) + value + strings.Repeat(character, all/2)
}

func TitleBorder(border lipgloss.Border, width int, title string) lipgloss.Border {
	border.Top = WrapX(width, "║"+title+"║", border.Top)

	return border
}
