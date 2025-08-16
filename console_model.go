package main

import (
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/styles"
	"github.com/muesli/reflow/wordwrap"
)

type LogRow struct {
	Content   string
	CreatedOn time.Time
	EventType EventType
}

func (r LogRow) View(width int) string {
	timeStamp := styles.ConsoleTime.Render(" " + r.CreatedOn.Format(time.TimeOnly) + " ")
	body := " " + strings.TrimSpace(wordwrap.String(r.Content, width-lipgloss.Width(timeStamp)-2)) + " "

	switch r.EventType {
	case EvtMsg:
		body = styles.ConsoleMsg.Render(body)
	case EvtConnect:
		body = styles.ConsoleConnect.Render(body)
	case EvtDisconnect:
		body = styles.ConsoleDisconnect.Render(body)
	case EvtAddress:
		body = styles.ConsoleAddress.Render(body)
	case EvtHostname:
		body = styles.ConsoleHostname.Render(body)
	case EvtStatusID:
		body = styles.ConsoleStatusID.Render(body)
	case EvtMap:
		body = styles.ConsoleMap.Render(body)
	case EvtTags:
		body = styles.ConsoleTags.Render(body)
	case EvtLobby:
		body = styles.ConsoleLobby.Render(body)
	case EvtKill:
		body = styles.ConsoleKill.Render(body)
	default:
		body = styles.ConsoleOther.Render(body)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, timeStamp, body)
}

type ConsoleModel struct {
	ready          bool
	rowsMu         *sync.RWMutex
	rowsRendered   string
	console        *ConsoleLog
	consoleLogPath string
	width          int
	viewPort       viewport.Model
	focused        bool
}

func NewConsoleModel(consoleLogPath string) ConsoleModel {
	model := ConsoleModel{
		rowsMu:         &sync.RWMutex{},
		console:        NewConsoleLog(),
		consoleLogPath: consoleLogPath,
		viewPort:       viewport.New(10, 20),
	}

	if consoleLogPath != "" {
		if err := model.console.Read(consoleLogPath); err != nil {
			slog.Error("Failed to read console file", slog.String("error", err.Error()),
				slog.String("path", consoleLogPath))
		}
	}

	return model
}

func (m ConsoleModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.logTick())
}

func (m ConsoleModel) Update(msg tea.Msg) (ConsoleModel, tea.Cmd) {
	cmds := make([]tea.Cmd, 2)

	m.viewPort, cmds[0] = m.viewPort.Update(msg)

	switch msg := msg.(type) {
	case ContentViewPortHeightMsg:
		m.width = msg.width
		m.viewPort.Width = msg.width
	case ConsoleLogMsg:
		cmds = append(cmds, m.logTick()) //nolint:makezero

		return m.onLogs(msg.logs), tea.Batch(cmds...)
	}

	return m, tea.Batch(cmds...)
}

func (m ConsoleModel) onLogs(logs []LogEvent) ConsoleModel {
	if len(logs) == 0 {
		return m
	}

	for _, msg := range logs {
		if slices.Contains([]EventType{EvtStatusID, EvtHostname, EvtMsg, EvtTags, EvtAddress, EvtLobby}, msg.Type) {
			continue
		}

		parts := strings.SplitN(msg.Raw, ": ", 2)
		if len(parts) != 2 {
			break
		}
		if parts[1] == "" {
			continue
		}

		valid := true
		for _, prefix := range []string{"# ", "version ", "steamid ", "players ", "map ", "account ", "edicts "} {
			if strings.HasPrefix(parts[1], prefix) {
				valid = false

				break
			}
		}
		if !valid {
			continue
		}

		newRow := LogRow{Content: safeString(parts[1]), CreatedOn: time.Now(), EventType: msg.Type}
		m.rowsMu.Lock()
		// This does not use JoinVertical currently as it takes *way* more and more CPU as time goes on
		// and the console log fills.
		m.rowsRendered = m.rowsRendered + "\n" + newRow.View(m.width-10)
		m.rowsMu.Unlock()
	}

	return m
}

func safeString(s string) string {
	s = strings.TrimFunc(s, func(r rune) bool {
		return !unicode.IsGraphic(r) || unicode.IsControl(r)
	})

	return s
}

func (m ConsoleModel) Render(height int) string {
	title := renderTitleBar(m.width, "Console Log")
	m.viewPort.Height = height - lipgloss.Height(title)
	wasBottom := m.viewPort.AtBottom()
	m.viewPort.SetContent(m.rowsRendered)
	if wasBottom {
		m.viewPort.GotoBottom()
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, m.viewPort.View())
}

func (m ConsoleModel) logTick() tea.Cmd {
	return tea.Tick(time.Second, func(lastTime time.Time) tea.Msg {
		return ConsoleLogMsg{t: lastTime, logs: m.console.Dequeue()}
	})
}
