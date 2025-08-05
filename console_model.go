package main

import (
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
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
		styles.ConsoleMsg.Render(" "+strings.TrimSpace(r.Content)))
}

type ConsoleModel struct {
	ready          bool
	rowsMu         sync.RWMutex
	rows           []LogRow
	rowsRendered   string
	console        *ConsoleLog
	consoleLogPath string
	width          int
	viewPort       viewport.Model
	input          textinput.Model
	focused        bool
}

func NewConsoleModel(consoleLogPath string) *ConsoleModel {
	model := &ConsoleModel{
		console:        NewConsoleLog(),
		input:          NewTextInputModel("", ""),
		consoleLogPath: consoleLogPath,
		viewPort:       viewport.New(10, 10),
	}

	if consoleLogPath != "" {
		if err := model.console.Read(consoleLogPath); err != nil {
			tea.Println(err.Error())
		}
	}

	return model
}

type ContentViewPortHeightMsg struct {
	contentViewPortHeight int
	height                int
	width                 int
}

func (m *ConsoleModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.logTick())
}

func (m *ConsoleModel) Update(msg tea.Msg) (*ConsoleModel, tea.Cmd) {
	cmds := make([]tea.Cmd, 2)

	m.viewPort, cmds[0] = m.viewPort.Update(msg)
	m.input, cmds[1] = m.input.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.consoleInput):
			if m.focused {
				m.input.PromptStyle = styles.NoStyle
				m.input.TextStyle = styles.NoStyle
				m.input.Blur()
				m.focused = false
			} else {
				m.input.PromptStyle = styles.FocusedStyle
				m.input.TextStyle = styles.FocusedStyle
				cmds = append(cmds, m.input.Focus()) // nolint:makezero
				m.focused = true
			}

		case key.Matches(msg, DefaultKeyMap.consoleCancel):
			m.input.Blur()
			m.input.PromptStyle = styles.NoStyle
			m.input.TextStyle = styles.NoStyle

			return m, nil
		}
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
		newRow := LogRow{Content: parts[1], CreatedOn: time.Now()}
		m.rowsMu.Lock()
		m.rows = append(m.rows, newRow)
		m.rowsRendered = lipgloss.JoinVertical(lipgloss.Left, m.rowsRendered, newRow.View())
		m.rowsMu.Unlock()
	}

	m.viewPort.PageUp()
	m.updateView()
}

func (m *ConsoleModel) View(height int) string {
	title := renderTitleBar(m.width, "Console Log")
	inputRow := lipgloss.JoinHorizontal(lipgloss.Top, "CONSOLE>", m.input.View())
	input := lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).Width(m.width - 4).Render(inputRow)

	m.viewPort.Height = height - lipgloss.Height(title) - lipgloss.Height(input)

	return lipgloss.JoinVertical(lipgloss.Left, title, m.viewPort.View(), input)
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
