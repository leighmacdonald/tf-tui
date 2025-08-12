package main

import (
	"log/slog"
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
}

func (r LogRow) View(width int) string {
	timeStamp := styles.ConsoleTime.Render(" " + r.CreatedOn.Format(time.TimeOnly) + " ")

	return lipgloss.JoinHorizontal(lipgloss.Top, timeStamp,
		styles.ConsoleMsg.Render(" "+strings.TrimSpace(wordwrap.String(r.Content, width-lipgloss.Width(timeStamp)-2))+" "))
}

type ConsoleModel struct {
	ready          bool
	rowsMu         sync.RWMutex
	rowsRendered   string
	console        *ConsoleLog
	consoleLogPath string
	width          int
	viewPort       viewport.Model
	focused        bool
}

func NewConsoleModel(consoleLogPath string) *ConsoleModel {
	model := &ConsoleModel{
		console:        NewConsoleLog(),
		consoleLogPath: consoleLogPath,
		viewPort:       viewport.New(10, 10),
	}

	if consoleLogPath != "" {
		if err := model.console.Read(consoleLogPath); err != nil {
			slog.Error("Failed to read console file", slog.String("error", err.Error()),
				slog.String("path", consoleLogPath))
		}
	}

	return model
}

func (m *ConsoleModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.logTick())
}

func (m *ConsoleModel) Update(msg tea.Msg) (*ConsoleModel, tea.Cmd) {
	cmds := make([]tea.Cmd, 2)

	m.viewPort, cmds[0] = m.viewPort.Update(msg)

	switch msg := msg.(type) {
	case ContentViewPortHeightMsg:
		m.width = msg.width
		m.viewPort.Width = msg.width
		m.updateView()
	case ConsoleLogMsg:
		m.onLogs(msg.logs)

		return m, m.logTick()
	}

	return m, tea.Batch(cmds...)
}

func (m *ConsoleModel) onLogs(logs []LogEvent) {
	if len(logs) == 0 {
		return
	}

	for _, msg := range logs {
		parts := strings.SplitN(msg.Raw, ": ", 2)
		if len(parts) != 2 {
			break
		}
		if parts[1] == "" {
			continue
		}
		newRow := LogRow{Content: safeString(parts[1]), CreatedOn: time.Now()}
		m.rowsMu.Lock()
		m.rowsRendered = lipgloss.JoinVertical(lipgloss.Left, m.rowsRendered, newRow.View(m.width-10))
		m.rowsMu.Unlock()
	}

	m.updateView()
}

func safeString(s string) string {
	s = strings.TrimFunc(s, func(r rune) bool {
		return !unicode.IsGraphic(r) || unicode.IsControl(r)
	})

	return s
}

func (m *ConsoleModel) View(height int) string {
	title := renderTitleBar(m.width, "Console Log")

	m.viewPort.Height = height - lipgloss.Height(title)

	return lipgloss.JoinVertical(lipgloss.Left, title, m.viewPort.View())
}

func (m *ConsoleModel) updateView() {
	wasBottom := m.viewPort.AtBottom()
	m.viewPort.SetContent(m.rowsRendered)
	if wasBottom {
		m.viewPort.GotoBottom()
	}
}

func (m *ConsoleModel) logTick() tea.Cmd {
	return tea.Tick(time.Second, func(lastTime time.Time) tea.Msg {
		return ConsoleLogMsg{t: lastTime, logs: m.console.Dequeue()}
	})
}
