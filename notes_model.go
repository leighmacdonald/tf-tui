package main

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type NotesModel struct {
	viewPort viewport.Model
	textarea textarea.Model
	player   Player
	width    int
}

func NewNotesModel() NotesModel {
	textArea := textarea.New()
	// textArea.SetHeight(10)
	textArea.SetValue("A note...")
	viewPort := viewport.New(10, 10)

	return NotesModel{textarea: textArea, viewPort: viewPort}
}

func (m NotesModel) Init() tea.Cmd {
	return tea.Batch(m.textarea.Cursor.BlinkCmd(), m.textarea.Focus())
}

func (m NotesModel) Update(msg tea.Msg) (NotesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ContentViewPortHeightMsg:
		m.width = msg.width
		m.viewPort.Width = msg.width
		m.viewPort.Height = msg.contentViewPortHeight
	case SelectedPlayerMsg:
		m.player = msg.player
		m.textarea.SetValue(msg.notes)
	}

	return m, tea.Batch()
}

func (m NotesModel) View(height int) string {
	m.viewPort.SetContent(m.textarea.Value())
	title := renderTitleBar(m.width, "Player Notes (doesnt work)")

	m.viewPort.Height = height - lipgloss.Height(title)

	return lipgloss.JoinVertical(lipgloss.Top, title, m.textarea.View())
}
