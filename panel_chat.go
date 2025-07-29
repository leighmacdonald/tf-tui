package main

import tea "github.com/charmbracelet/bubbletea"

func NewPanelChatModel() *PanelChatModel {
	return &PanelChatModel{}
}

type PanelChatModel struct {
}

func (m PanelChatModel) Init() tea.Cmd {
	return nil
}

func (m PanelChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m PanelChatModel) View() string {
	return ""
}
