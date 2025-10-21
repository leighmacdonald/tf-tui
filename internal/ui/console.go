package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/internal/tf/events"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
	zone "github.com/lrstanley/bubblezone"
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
	case events.Kill:
		body = styles.ConsoleKill.Render(body)
	case events.Stats:
		body = styles.ConsoleKill.Render(body)
	case events.Version:
		// TODO
	case events.Any:
		fallthrough
	default:
		body = styles.ConsoleOther.Render(body)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, timeStamp, body)
}

type consoleModel struct {
	ready  bool
	rowsMu *sync.RWMutex
	// indexed by log secret
	rowsRendered   map[string]string
	rowsCount      map[string]int
	width          int
	viewPort       viewport.Model
	focused        bool
	filterNoisy    bool
	selectedServer Snapshot
	input          textinput.Model
	inputActive    bool
	inputZoneID    string
}

func newConsoleModel() *consoleModel {
	input := textinput.New()
	input.Prompt = lipgloss.NewStyle().Foreground(styles.ColourVintage).Background(styles.Black).Inline(true).Render("RCON  ")
	model := consoleModel{
		rowsMu:       &sync.RWMutex{},
		rowsRendered: map[string]string{},
		rowsCount:    map[string]int{},
		viewPort:     viewport.New(10, 20),
		input:        input,
		inputZoneID:  zone.NewPrefix(),
	}

	return &model
}

func (m *consoleModel) Init() tea.Cmd {
	return nil
}

func (m *consoleModel) Update(msg tea.Msg) (*consoleModel, tea.Cmd) {
	cmds := make([]tea.Cmd, 2)

	m.viewPort, cmds[0] = m.viewPort.Update(msg)

	if m.inputActive {
		m.input, cmds[1] = m.input.Update(msg)
	}

	switch msg := msg.(type) {
	case inputZoneChangeMsg:
		m.inputActive = msg.zone == zoneConsoleInput
	case selectServerSnapshotMsg:
		m.selectedServer = msg.server
	case contentViewPortHeightMsg:
		m.width = msg.width
		m.viewPort.Width = msg.width
	case events.Event:
		return m.onLogs(msg), tea.Batch(cmds...)
	}

	return m, tea.Batch(cmds...)
}

func (m *consoleModel) onLogs(event events.Event) *consoleModel {
	// if slices.Contains([]tf.EventType{tf.EvtStatusID, tf.EvtHostname, tf.EvtMsg, tf.EvtTags, tf.EvtAddress, tf.EvtLobby}, log.Type) {
	// 	return m
	// }
	parts := strings.SplitN(event.Raw, ": ", 2)
	if len(parts) != 2 {
		return m
	}
	if parts[1] == "" {
		return m
	}

	if m.filterNoisy {
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
	}

	newRow := LogRow{Content: safeString(parts[1]), CreatedOn: time.Now(), EventType: event.Type}
	m.rowsMu.Lock()
	// This does not use JoinVertical currently as it takes more and more CPU as time goes on
	// and the console log fills becoming unusable.
	prev := m.rowsRendered[event.HostPort]
	m.rowsRendered[event.HostPort] = prev + "\n" + newRow.Render(m.width-10)
	if _, ok := m.rowsCount[event.HostPort]; !ok {
		m.rowsCount[event.HostPort] = 0
	}
	m.rowsCount[event.HostPort]++
	m.rowsMu.Unlock()

	return m
}

func safeString(s string) string {
	s = strings.TrimFunc(s, func(r rune) bool {
		return !unicode.IsGraphic(r) || unicode.IsControl(r)
	})

	return s
}

func (m *consoleModel) Render(height int) string {
	title := "Console Log"
	content, found := m.rowsRendered[m.selectedServer.HostPort]
	if !found || content == "" {
		content = "<<< Start of logs >>>\n"
	} else {
		title = renderTitleBar(m.width, fmt.Sprintf("Console Log: %d Messages", m.rowsCount[m.selectedServer.HostPort]))
	}

	input := zone.Mark(m.inputZoneID, m.input.View())

	m.viewPort.Height = height - lipgloss.Height(title) - lipgloss.Height(input)
	wasBottom := m.viewPort.AtBottom()

	m.viewPort.SetContent(content)
	if wasBottom {
		m.viewPort.GotoBottom()
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, m.viewPort.View(), input)
}
