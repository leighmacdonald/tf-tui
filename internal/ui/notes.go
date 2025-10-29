package ui

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type notesModel struct {
	viewPort  viewport.Model
	textarea  textarea.Model
	player    Player
	viewState viewState
}

func newNotesModel() notesModel {
	textArea := textarea.New()
	// textArea.SetHeight(10)
	textArea.SetValue("A note...")
	viewPort := viewport.New(10, 10)

	return notesModel{textarea: textArea, viewPort: viewPort}
}

func (m notesModel) Init() tea.Cmd {
	return tea.Batch(m.textarea.Cursor.BlinkCmd(), m.textarea.Focus())
}

func (m notesModel) Update(msg tea.Msg) (notesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case viewState:
		m.viewState = msg
		m.viewPort.Width = msg.width
		m.viewPort.Height = msg.lowerSize
	case selectedPlayerMsg:
		m.player = msg.player
		m.textarea.SetValue(msg.notes)
	}

	return m, tea.Batch()
}

func (m notesModel) View(height int) string {
	m.viewPort.SetContent(m.textarea.Value())

	m.viewPort.Height = height

	return lipgloss.JoinVertical(lipgloss.Top, m.textarea.View())
}
