package main

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/styles"
)

const MaxTF2MessageLength = 127

type ChatRow struct {
	steamID   steamid.SteamID
	name      string
	createdOn time.Time
	message   string
	team      Team
	dead      bool
}

func (m ChatRow) View() string {
	var name string
	switch m.team {
	case RED:
		name = styles.ChatNameRed.Render(m.name)
	case BLU:
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

func NewChatModel() *ChatModel {
	input := NewTextInputModel("", "(ALL) >")
	input.CharLimit = MaxTF2MessageLength

	return &ChatModel{input: input}
}

type ChatModel struct {
	viewport     viewport.Model
	input        textinput.Model
	ready        bool
	rows         []ChatRow
	rowsRendered string
	width        int
	inputOpen    bool
	chatType     ChatType
}

func (m *ChatModel) Placeholder() string {
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

func (m *ChatModel) Init() tea.Cmd {
	return nil
}

func (m *ChatModel) Update(msg tea.Msg) (*ChatModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ContentViewPortHeightMsg:
		m.width = msg.width
		if !m.ready {
			m.viewport = viewport.New(msg.width, msg.contentViewPortHeight)
			m.ready = true
		} else {
			m.viewport.Height = msg.contentViewPortHeight
		}
	case LogEvent:
		if msg.Type != EvtMsg {
			break
		}

		newRow := ChatRow{
			steamID:   msg.PlayerSID,
			name:      msg.Player,
			createdOn: msg.Timestamp,
			message:   msg.Message,
			dead:      msg.Dead,
			team:      msg.Team,
		}
		m.rows = append(m.rows, newRow)
		m.rowsRendered = lipgloss.JoinVertical(lipgloss.Left, m.rowsRendered, newRow.View())
	case tea.KeyMsg:
		key := msg.String()
		if m.input.Focused() {
			switch key {
			case "esc":
				m.input.Blur()
				m.inputOpen = false

				return m, nil
			case "return":
				m.inputOpen = false
				message := m.input.Value()
				if message == "" {
					return m, nil
				}
				m.input.SetValue("")

				return m, func() tea.Msg {
					return ChatMsg{
						Message:  message,
						ChatType: m.chatType,
					}
				}
			}
		}
		switch key {
		case "y":
			m.inputOpen = true
			m.chatType = AllChat
		case "u":
			m.inputOpen = true
			m.chatType = TeamChat
		case "p":
			m.inputOpen = true
			m.chatType = PartyChat
		case "c":
			fallthrough
		case "esc":
			return m, func() tea.Msg {
				return SetViewMsg{view: viewPlayerTables}
			}
		}
	}

	cmds := make([]tea.Cmd, 1)
	m.input, cmds[0] = m.input.Update(msg)

	return m, tea.Batch(cmds...)
}

func (m *ChatModel) View(height int) string {
	titleBar := renderTitleBar(m.width, "Game Chat")
	content := lipgloss.JoinVertical(lipgloss.Top, m.rowsRendered)
	m.input.Placeholder = m.Placeholder()
	input := m.input.View()
	m.viewport.Height = height - lipgloss.Height(titleBar) - lipgloss.Height(input)
	m.viewport.SetContent(content)

	return lipgloss.JoinVertical(lipgloss.Left, titleBar, m.viewport.View(), input)
}
