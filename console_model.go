package main

import (
	"strings"
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
	viewPort       viewport.Model
	ready          bool
	rows           []LogRow
	rowsRendered   string
	console        *ConsoleLog
	consoleLogPath string
}

func NewConsoleModel(consoleLogPath string) *ConsoleModel {
	cm := &ConsoleModel{console: NewConsoleLog(), consoleLogPath: consoleLogPath}
	if consoleLogPath != "" {
		cm.console.ReadConsole(consoleLogPath)
	}

	return cm
}

func (m *ConsoleModel) Init() tea.Cmd {
	return m.logEmitter()
}

func (m *ConsoleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "`" || k == "esc" {
			return m, func() tea.Msg {
				return SetViewMsg{view: viewPlayerTables}
			}
		}

	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewPort = viewport.New(msg.Width, msg.Height-3)
			//m.viewPort.YPosition = headerHeight
			m.viewPort.SetContent("~~~ Start of console ~~~")
			m.ready = true
		} else {
			m.viewPort.Width = msg.Width
			m.viewPort.Height = msg.Height - 20
		}
	}

	// Handle keyboard and mouse events in the viewport
	m.viewPort, cmd = m.viewPort.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *ConsoleModel) View() string {
	return m.viewPort.View()
}

func (m *ConsoleModel) logEmitter() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(lastTime time.Time) tea.Msg {
		var outLogs []LogEvent
		for _, msg := range m.console.Dequeue() {
			parts := strings.SplitN(msg.Raw, ": ", 2)
			if len(parts) != 2 {
				break
			}

			newRow := LogRow{Content: parts[1], CreatedOn: time.Now()}
			m.rows = append(m.rows, newRow)
			m.rowsRendered = lipgloss.JoinVertical(lipgloss.Left, m.rowsRendered, newRow.View())

			// Automatically scroll if we are at the bottom.
			wasBottom := m.viewPort.AtBottom()
			m.viewPort.SetContent(m.rowsRendered)
			if wasBottom {
				m.viewPort.GotoBottom()
			}
			outLogs = append(outLogs, msg)
		}

		return ConsoleLogMsg{t: lastTime, logs: outLogs}
	})
}
