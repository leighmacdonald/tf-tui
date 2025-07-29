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
		" ",
		styles.ConsoleMsg.Render(r.Content))
}

type ConsoleView struct {
	viewPort     viewport.Model
	ready        bool
	rows         []LogRow
	rowsRendered string
}

func NewConsoleView() *ConsoleView {
	return &ConsoleView{}
}

func (m ConsoleView) Init() tea.Cmd {
	return nil
}

func (m ConsoleView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case LogEvent:
		parts := strings.SplitN(msg.Raw, ": ", 2)
		if len(parts) != 2 {
			break
		}

		newRow := LogRow{Content: parts[1], CreatedOn: time.Now()}
		m.rows = append(m.rows, newRow)
		m.rowsRendered = lipgloss.JoinVertical(lipgloss.Left, m.rowsRendered, newRow.View())
		wasBottom := m.viewPort.AtBottom()
		m.viewPort.SetContent(m.rowsRendered)
		if wasBottom {
			m.viewPort.GotoBottom()
		}

	case tea.KeyMsg:
		if k := msg.String(); k == "`" || k == "esc" {
			return m, func() tea.Msg {
				return SetViewMsg{view: viewPlayerTables}
			}
		}

	case tea.WindowSizeMsg:
		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewPort = viewport.New(msg.Width, msg.Height-3)
			//m.viewPort.YPosition = headerHeight
			m.viewPort.SetContent("Start of console")
			m.ready = true
		} else {
			m.viewPort.Width = msg.Width
			m.viewPort.Height = msg.Height - 3
		}
	}

	// Handle keyboard and mouse events in the viewport
	m.viewPort, cmd = m.viewPort.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m ConsoleView) View() string {
	return m.viewPort.View()
}
