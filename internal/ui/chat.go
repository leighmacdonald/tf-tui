package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/tf/events"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
)

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
	case tf.UNASSIGNED:
		fallthrough
	case tf.SPEC:
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

func newChatModel() chatModel {
	return chatModel{}
}

type chatModel struct {
	players         Players
	viewport        viewport.Model
	ready           bool
	rows            map[string][]ChatRow
	rowsRendered    map[string]string
	selectedsServer string
	width           int
	inputOpen       bool
	chatType        ChatType
	incoming        chan events.Event
}

func (m chatModel) Placeholder() string {
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

func (m chatModel) Init() tea.Cmd {
	return nil
}

func (m chatModel) Update(msg tea.Msg) (chatModel, tea.Cmd) {
	switch msg := msg.(type) {
	case selectServerSnapshotMsg:
		m.selectedsServer = msg.server.HostPort
	case contentViewPortHeightMsg:
		m.width = msg.width
		if !m.ready {
			m.viewport = viewport.New(msg.width, msg.contentViewPortHeight)
			m.ready = true
		} else {
			m.viewport.Height = msg.contentViewPortHeight
		}
	case Snapshot:
		m.players = msg.Server.Players
	case events.Event:
		if msg.Type != events.Msg {
			break
		}

		evt, ok := msg.Data.(events.MsgEvent)
		if !ok {
			break
		}

		team := tf.UNASSIGNED
		for _, player := range m.players {
			if player.SteamID.Equal(evt.PlayerSID) {
				team = player.Team

				break
			}
		}

		row := ChatRow{
			steamID:   evt.PlayerSID,
			name:      evt.Player,
			createdOn: msg.Timestamp,
			message:   evt.Message,
			team:      team,
			dead:      evt.Dead,
		}

		if _, ok := m.rows[msg.HostPort]; !ok {
			m.rows[msg.HostPort] = []ChatRow{}
		}

		m.rows[msg.HostPort] = append(m.rows[msg.HostPort], row)
		previous := m.rowsRendered[msg.HostPort]
		m.rowsRendered[msg.HostPort] = lipgloss.JoinVertical(lipgloss.Left, previous, row.View())
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

func (m chatModel) View(height int) string {
	titleBar := renderTitleBar(m.width, "Game Chat")
	m.viewport.Height = height - lipgloss.Height(titleBar)
	rows := m.rowsRendered[m.selectedsServer]
	m.viewport.SetContent(rows)

	return lipgloss.JoinVertical(lipgloss.Left, titleBar, m.viewport.View())
}
