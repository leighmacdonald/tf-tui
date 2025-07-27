package main

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
)

type SelectedFileMsg struct {
	filePath string
}

func NewFileSelect() tea.Model {
	return &FileSelect{
		filepicker:   NewPicker(),
		selectedFile: "",
	}
}

type FileSelect struct {
	selectedFile string
	filepicker   filepicker.Model
	err          string
}

func (m FileSelect) Init() tea.Cmd {
	return m.filepicker.Init()
}

func (m FileSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.filepicker, cmd = m.filepicker.Update(msg)

	// Did the user select a file?
	if didSelect, selectedPath := m.filepicker.DidSelectFile(msg); didSelect {
		m.selectedFile = selectedPath
		m.err = ""

		return m, tea.Batch(cmd, func() tea.Msg {
			return SelectedFileMsg{m.selectedFile}
		})
	}

	// Did the user select a disabled file?
	// This is only necessary to display an error to the user.
	if didSelect, selectedPath := m.filepicker.DidSelectDisabledFile(msg); didSelect {
		// Let's clear the selectedFile and display an error.
		m.selectedFile = ""
		m.err = errInvalidPath.Error() + ": " + selectedPath

		return m, tea.Batch(cmd, clearErrorAfter(10*time.Second))
	}

	return m, cmd
}

func (m FileSelect) View() string {
	var builder strings.Builder
	builder.WriteString("\n  ")
	switch {
	case m.err != "":
		builder.WriteString(m.filepicker.Styles.DisabledFile.Render(m.err))
	case m.selectedFile == "":
		builder.WriteString("Pick a file:")
	default:
		builder.WriteString("Selected file: " + m.filepicker.Styles.Selected.Render(m.selectedFile))
	}
	builder.WriteString("\n\n" + m.filepicker.View() + "\n")

	return builder.String()
}
