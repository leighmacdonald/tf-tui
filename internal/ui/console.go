package ui

import (
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/internal/tf/events"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
	"github.com/muesli/reflow/wordwrap"
)

type LogRow struct {
	Content   string
	CreatedOn time.Time
	EventType events.EventType
}

func (r LogRow) Render(width int) string {
	timeStamp := styles.ConsoleTime.Render(" " + r.CreatedOn.Format(time.TimeOnly) + " ")
	body := " " + strings.TrimSpace(wordwrap.String(r.Content, width-lipgloss.Width(timeStamp)-2)) + " "

	switch r.EventType {
	case events.Msg:
		body = styles.ConsoleMsg.Render(body)
	case events.Connect:
		body = styles.ConsoleConnect.Render(body)
	case events.Disconnect:
		body = styles.ConsoleDisconnect.Render(body)
	case events.Address:
		body = styles.ConsoleAddress.Render(body)
	case events.Hostname:
		body = styles.ConsoleHostname.Render(body)
	case events.StatusID:
		body = styles.ConsoleStatusID.Render(body)
	case events.Map:
		body = styles.ConsoleMap.Render(body)
	case events.Tags:
		body = styles.ConsoleTags.Render(body)
	case events.Lobby:
		body = styles.ConsoleLobby.Render(body)
	case events.Kill:
		body = styles.ConsoleKill.Render(body)
	default:
		body = styles.ConsoleOther.Render(body)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, timeStamp, body)
}

type consoleModel struct {
	ready        bool
	rowsMu       *sync.RWMutex
	rowsRendered string
	width        int
	viewPort     viewport.Model
	focused      bool
}

func newConsoleModel() consoleModel {
	model := consoleModel{
		rowsMu:   &sync.RWMutex{},
		viewPort: viewport.New(10, 20),
	}

	return model
}

func (m consoleModel) Init() tea.Cmd {
	return nil
}

func (m consoleModel) Update(msg tea.Msg) (consoleModel, tea.Cmd) {
	cmds := make([]tea.Cmd, 2)

	m.viewPort, cmds[0] = m.viewPort.Update(msg)

	switch msg := msg.(type) {
	case ContentViewPortHeightMsg:
		m.width = msg.width
		m.viewPort.Width = msg.width
	case events.Event:
		return m.onLogs(msg), tea.Batch(cmds...)
	}

	return m, tea.Batch(cmds...)
}

func (m consoleModel) onLogs(log events.Event) consoleModel {
	// if slices.Contains([]tf.EventType{tf.EvtStatusID, tf.EvtHostname, tf.EvtMsg, tf.EvtTags, tf.EvtAddress, tf.EvtLobby}, log.Type) {
	// 	return m
	// }
	parts := strings.SplitN(log.Raw, ": ", 2)
	if len(parts) != 2 {
		return m
	}
	if parts[1] == "" {
		return m
	}

	valid := true
	for _, prefix := range []string{"# ", "version ", "steamid ", "players ", "map ", "account ", "edicts "} {
		if strings.HasPrefix(parts[1], prefix) {
			valid = false

			break
		}
	}
	if !valid {
		return m
	}

	newRow := LogRow{Content: safeString(parts[1]), CreatedOn: time.Now(), EventType: log.Type}
	m.rowsMu.Lock()
	// This does not use JoinVertical currently as it takes more and more CPU as time goes on
	// and the console log fills becoming unusuable.
	m.rowsRendered = m.rowsRendered + "\n" + newRow.Render(m.width-10)
	m.rowsMu.Unlock()

	return m
}

func safeString(s string) string {
	s = strings.TrimFunc(s, func(r rune) bool {
		return !unicode.IsGraphic(r) || unicode.IsControl(r)
	})

	return s
}

func (m consoleModel) Render(height int) string {
	title := renderTitleBar(m.width, "Console Log")
	m.viewPort.Height = height - lipgloss.Height(title)
	wasBottom := m.viewPort.AtBottom()
	m.viewPort.SetContent(m.rowsRendered)
	if wasBottom {
		m.viewPort.GotoBottom()
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, m.viewPort.View())
}
