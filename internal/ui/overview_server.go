package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
)

func newServerDetailPanel() serverDetailPanelModel {
	return serverDetailPanelModel{}
}

type serverDetailPanelModel struct {
	snapshot Snapshot
	width    int
	viewport viewport.Model
	ready    bool
}

func (m serverDetailPanelModel) Init() tea.Cmd {
	return nil
}

func (m serverDetailPanelModel) Update(msg tea.Msg) (serverDetailPanelModel, tea.Cmd) {
	switch msg := msg.(type) {
	case selectServerSnapshotMsg:
		m.snapshot = msg.server
	case contentViewPortHeightMsg:
		m.width = msg.width
		if !m.ready {
			m.viewport = viewport.New(msg.width, msg.contentViewPortHeight)
			m.ready = true
		} else {
			m.viewport.Height = msg.contentViewPortHeight
			m.viewport.Width = msg.width
		}
	}

	return m, nil
}

func (m serverDetailPanelModel) Render(height int) string {
	var rows []string
	rows = append(rows, styles.DetailRow("Game Tags", strings.Join(m.snapshot.Status.Tags, ", ")))
	var edicts []string
	for _, edict := range m.snapshot.Status.Edicts {
		edicts = append(edicts, strconv.Itoa(edict))
	}
	rows = append(rows, styles.DetailRow("EDicts", strings.Join(edicts, ", ")))
	rows = append(rows, styles.DetailRow("Game", m.snapshot.Server.Game))

	m.viewport.SetContent(lipgloss.JoinVertical(lipgloss.Top, rows...))

	titleBar := renderTitleBar(m.width, "Server Overview: "+m.snapshot.Status.ServerName)
	m.viewport.Height = height - lipgloss.Height(titleBar)

	return lipgloss.JoinVertical(lipgloss.Top, titleBar, "", m.viewport.View())
}
