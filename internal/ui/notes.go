package ui

import (
	"github.com/charmbracelet/bubbles/v2/textarea"
	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

type notesModel struct {
	viewPort viewport.Model
	textarea textarea.Model
	player   Player
	width    int
}

func newNotesModel() notesModel {
	textArea := textarea.New()
	// textArea.SetHeight(10)
	textArea.SetValue("A note...")
	viewPort := viewport.New()

	return notesModel{textarea: textArea, viewPort: viewPort}
}

func (m notesModel) Init() tea.Cmd {
	return tea.Batch(m.textarea.Focus())
}

func (m notesModel) Update(msg tea.Msg) (notesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ContentViewPortHeightMsg:
		m.width = msg.width
		m.viewPort.SetWidth(msg.width)
		m.viewPort.SetHeight(msg.contentViewPortHeight)
	case SelectedPlayerMsg:
		m.player = msg.player
		m.textarea.SetValue(msg.notes)
	}

	return m, tea.Batch()
}

func (m notesModel) View(height int) string {
	m.viewPort.SetContent(m.textarea.Value())
	title := renderTitleBar(m.width, "Player Notes (doesnt work)")

	m.viewPort.SetHeight(height - lipgloss.Height(title))

	return lipgloss.JoinVertical(lipgloss.Top, title, m.textarea.View())
}
