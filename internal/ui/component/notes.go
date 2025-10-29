package component

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/internal/ui/command"
	"github.com/leighmacdonald/tf-tui/internal/ui/model"
)

type NotesModel struct {
	viewPort  viewport.Model
	textarea  textarea.Model
	player    model.Player
	viewState model.ViewState
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
	case model.ViewState:
		m.viewState = msg
		m.viewPort.Width = msg.Width
		m.viewPort.Height = msg.Lower
	case command.SelectedPlayerMsg:
		m.player = msg.Player
		m.textarea.SetValue(msg.Notes)
	}

	return m, tea.Batch()
}

func (m NotesModel) View(height int) string {
	m.viewPort.SetContent(m.textarea.Value())

	m.viewPort.Height = height

	return lipgloss.JoinVertical(lipgloss.Top, m.textarea.View())
}
