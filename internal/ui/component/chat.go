package component

import (
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/tf/events"
	"github.com/leighmacdonald/tf-tui/internal/ui/command"
	"github.com/leighmacdonald/tf-tui/internal/ui/model"
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

type ChatMsg struct {
	Message  string
	ChatType tf.ChatType
}

func NewChatModel() ChatModel {
	return ChatModel{}
}

type ChatModel struct {
	players         model.Players
	viewport        viewport.Model
	viewState       model.ViewState
	ready           bool
	rows            map[string][]ChatRow
	rowsRendered    map[string]string
	selectedsServer string
	inputOpen       bool
	chatType        tf.ChatType
	incoming        chan events.Event
}

func (m ChatModel) Placeholder() string {
	var label string
	switch m.chatType {
	case tf.AllChat:
		label = "All"
	case tf.TeamChat:
		label = "Team"
	case tf.PartyChat:
		label = "Party"
	}

	return label + " >"
}

func (m ChatModel) Init() tea.Cmd {
	return nil
}

func (m ChatModel) Update(msg tea.Msg) (ChatModel, tea.Cmd) {
	switch msg := msg.(type) {
	case command.SelectServerSnapshotMsg:
		m.selectedsServer = msg.Server.HostPort
	case model.ViewState:
		m.viewState = msg
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Lower)
			m.ready = true
		} else {
			m.viewport.Height = msg.Lower
		}
	case model.Snapshot:
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

func (m ChatModel) View(height int) string {
	m.viewport.Height = height - 2
	m.viewport.SetContent(m.rowsRendered[m.selectedsServer])

	return Container("Chat Logs", m.viewState.Width, height, m.viewport.View(), m.viewState.KeyZone == model.KZconsoleInput)
}
