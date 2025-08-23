package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/tf"
	"github.com/leighmacdonald/tf-tui/ui/styles"
)

const MaxTF2MessageLength = 127

type ChatRow struct {
	steamID   steamid.SteamID
	name      string
	createdOn time.Time
	message   string
	team      tf.Team
	dead      bool
}

func (m ChatRow) View() string {
	var name string
	switch m.team {
	case tf.RED:
		name = styles.ChatNameRed.Render(m.name)
	case tf.BLU:
		name = styles.ChatNameBlu.Render(m.name)
	default:
		name = styles.ChatNameOther.Render(m.name)
	}

	msg := m.message
	if m.dead {
		msg = styles.IconDead + " " + msg
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		styles.ChatTime.Render(m.createdOn.Format(time.TimeOnly)),
		name,
		styles.ChatMessage.Render(msg),
	)
}

type ChatType int

const (
	AllChat ChatType = iota
	TeamChat
	PartyChat
)

func NewChatModel() ChatModel {
	return ChatModel{}
}

type ChatModel struct {
	viewport     viewport.Model
	ready        bool
	rows         []ChatRow
	rowsRendered string
	width        int
	inputOpen    bool
	chatType     ChatType
}

func (m ChatModel) Placeholder() string {
	var label string
	switch m.chatType {
	case AllChat:
		label = "All"
	case TeamChat:
		label = "Team"
	case PartyChat:
		label = "Party"
	}

	return label + " >"
}

func (m ChatModel) Init() tea.Cmd {
	return nil
}

func (m ChatModel) Update(msg tea.Msg) (ChatModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ContentViewPortHeightMsg:
		m.width = msg.width
		if !m.ready {
			m.viewport = viewport.New(msg.width, msg.contentViewPortHeight)
			m.ready = true
		} else {
			m.viewport.Height = msg.contentViewPortHeight
		}
	case ChatRow:
		m.rows = append(m.rows, msg)
		m.rowsRendered = lipgloss.JoinVertical(lipgloss.Left, m.rowsRendered, msg.View())
		m.viewport.SetContent(m.rowsRendered)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

func (m ChatModel) View(height int) string {
	titleBar := renderTitleBar(m.width, "Game Chat")
	m.viewport.Height = height - lipgloss.Height(titleBar)

	return lipgloss.JoinVertical(lipgloss.Left, titleBar, m.viewport.View())
}
