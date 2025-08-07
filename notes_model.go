package main

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type NotesModel struct {
	textarea textarea.Model
	player   Player
	width    int
}

func NewNotesModel() tea.Model {
	textArea := textarea.New()
	textArea.Focus()
	textArea.SetHeight(10)

	return NotesModel{textarea: textArea}
}

func (m NotesModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m NotesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) { //nolint:gocritic
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case SelectedPlayerMsg:
		m.player = msg.player
		m.textarea.SetValue(msg.notes)
	}

	return m, nil
}

func (m NotesModel) View() string {
	title := renderTitleBar(m.width, "Player Notes (doesnt work)")
	return lipgloss.JoinVertical(lipgloss.Top, title, m.textarea.View())
}
