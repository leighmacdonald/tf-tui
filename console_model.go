package main

import (
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/styles"
)

type LogRow struct {
	Content   string
	CreatedOn time.Time
}

func (r LogRow) View() string {
	return lipgloss.JoinHorizontal(lipgloss.Top,
		styles.ConsoleTime.Render(" "+r.CreatedOn.Format(time.TimeOnly)+" "),
		styles.ConsoleMsg.Render(r.Content))
}

type ConsoleModel struct {
	ready          bool
	rowsMu         sync.RWMutex
	rows           []LogRow
	rowsRendered   string
	console        *ConsoleLog
	consoleLogPath string
	viewPort       viewport.Model
	width          int
}

func NewConsoleModel(consoleLogPath string) *ConsoleModel {
	cm := &ConsoleModel{console: NewConsoleLog(), consoleLogPath: consoleLogPath, viewPort: viewport.New(10, 10)}
	if consoleLogPath != "" {
		cm.console.Read(consoleLogPath)
	}

	return cm
}

type ContentViewPortHeightMsg struct {
	contentViewPortHeight int
	height                int
	width                 int
}

func (m *ConsoleModel) Init() tea.Cmd {
	return m.logTick()
}

func (m *ConsoleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ContentViewPortHeightMsg:
		m.width = msg.width
		m.viewPort.Width = msg.width
		m.viewPort.Height = msg.contentViewPortHeight
		m.updateView()
	case ConsoleLogMsg:
		m.onLogs(msg.logs)

		return m, m.logTick()
	}

	var cmd tea.Cmd
	m.viewPort, cmd = m.viewPort.Update(msg)

	return m, cmd
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
		newRow := LogRow{Content: parts[1], CreatedOn: time.Now()}
		m.rowsMu.Lock()
		m.rows = append(m.rows, newRow)
		m.rowsRendered = lipgloss.JoinVertical(lipgloss.Left, m.rowsRendered, newRow.View())
		m.rowsMu.Unlock()
	}

	m.updateView()
}

func (m *ConsoleModel) View() string {
	return m.viewPort.View()
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
