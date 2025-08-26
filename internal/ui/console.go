package ui

import (
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
	"github.com/muesli/reflow/wordwrap"
)

type LogRow struct {
	Content   string
	CreatedOn time.Time
	EventType tf.EventType
}

func (r LogRow) Render(width int) string {
	timeStamp := styles.ConsoleTime.Render(" " + r.CreatedOn.Format(time.TimeOnly) + " ")
	body := " " + strings.TrimSpace(wordwrap.String(r.Content, width-lipgloss.Width(timeStamp)-2)) + " "

	switch r.EventType {
	case tf.EvtMsg:
		body = styles.ConsoleMsg.Render(body)
	case tf.EvtConnect:
		body = styles.ConsoleConnect.Render(body)
	case tf.EvtDisconnect:
		body = styles.ConsoleDisconnect.Render(body)
	case tf.EvtAddress:
		body = styles.ConsoleAddress.Render(body)
	case tf.EvtHostname:
		body = styles.ConsoleHostname.Render(body)
	case tf.EvtStatusID:
		body = styles.ConsoleStatusID.Render(body)
	case tf.EvtMap:
		body = styles.ConsoleMap.Render(body)
	case tf.EvtTags:
		body = styles.ConsoleTags.Render(body)
	case tf.EvtLobby:
		body = styles.ConsoleLobby.Render(body)
	case tf.EvtKill:
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
	case tf.LogEvent:
		return m.onLogs(msg), tea.Batch(cmds...)
	}

	return m, tea.Batch(cmds...)
}

func (m consoleModel) onLogs(log tf.LogEvent) consoleModel {
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
	// This does not use JoinVertical currently as it takes *way* more and more CPU as time goes on
	// and the console log fills.
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

// func (m ConsoleModel) logTick() tea.Cmd {
// 	return tea.Tick(time.Second, func(lastTime time.Time) tea.Msg {
// 		return ConsoleLogMsg{t: lastTime, logs: m.console.Dequeue()}
// 	})
// }
