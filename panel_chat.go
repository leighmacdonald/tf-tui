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

func NewPanelChatModel() *PanelChatModel {
	input := NewTextInputModel("", ">")
	input.CharLimit = 127
	return &PanelChatModel{input: input}
}

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

type ChatMsg struct {
	Message  string
	ChatType ChatType
}

type ChatType int

const (
	AllChat ChatType = iota
	TeamChat
	PartyChat
)

type PanelChatModel struct {
	input        textinput.Model
	viewPort     viewport.Model
	team         Team
	ready        bool
	rows         []ChatRow
	rowsRendered string
	width        int
	height       int
	inputOpen    bool
	chatType     ChatType
}

func (m PanelChatModel) Init() tea.Cmd {
	return nil
}

func (m PanelChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
		wasBottom := m.viewPort.AtBottom()
		m.viewPort.SetContent(m.rowsRendered)
		if wasBottom {
			m.viewPort.GotoBottom()
		}
	case tea.KeyMsg:
		k := msg.String()
		if m.input.Focused() {
			switch k {
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
		switch k {
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

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.viewPort = viewport.New(msg.Width, msg.Height-3)
			//m.viewPort.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewPort.Width = msg.Width
			m.viewPort.Height = msg.Height - 3
		}
	}
	cmds := make([]tea.Cmd, 2)
	m.viewPort, cmds[0] = m.viewPort.Update(msg)
	m.input, cmds[1] = m.input.Update(msg)

	return m, tea.Batch(cmds...)
}

func (m PanelChatModel) View() string {
	if m.inputOpen {
		m.viewPort.Width = m.width
		m.viewPort.Height = m.height - 4
		return lipgloss.JoinVertical(lipgloss.Top, m.viewPort.View(), m.input.View())
	} else {
		m.viewPort.Width = m.width
		m.viewPort.Height = m.height - 3
		return lipgloss.JoinVertical(lipgloss.Top, m.viewPort.View())
	}

}
