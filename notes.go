package main

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type TextAreaNotes struct {
	textarea textarea.Model
	player   Player
}

func NewTextAreaNotes() tea.Model {
	textArea := textarea.New()
	textArea.Focus()
	textArea.SetHeight(10)

	return TextAreaNotes{textarea: textArea}
}

func (m TextAreaNotes) Init() tea.Cmd {
	return textarea.Blink
}

func (m TextAreaNotes) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SelectedPlayerMsg:
		m.player = msg.player
		m.textarea.SetValue(msg.notes)
	}

	return m, nil
}

func (m TextAreaNotes) View() string {
	return m.textarea.View()
}
