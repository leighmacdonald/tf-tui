package main

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type NotesModel struct {
	textarea textarea.Model
	player   Player
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
	case SelectedPlayerMsg:
		m.player = msg.player
		m.textarea.SetValue(msg.notes)
	}

	return m, nil
}

func (m NotesModel) View() string {
	return m.textarea.View()
}
