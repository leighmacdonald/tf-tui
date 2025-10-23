package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/ui/model"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
)

func newServerDetailPanel() serverDetailPanelModel {
	return serverDetailPanelModel{
		listSM:   model.NewPluginList("Sourcemod Plugins"),
		listMeta: model.NewPluginList("Metamod Plugins"),
	}
}

type serverDetailPanelModel struct {
	snapshot       Snapshot
	width          int
	viewportDetail viewport.Model
	listSM         list.Model
	listMeta       list.Model
	ready          bool
}

func (m serverDetailPanelModel) Init() tea.Cmd {

	return nil
}

func (m serverDetailPanelModel) Update(msg tea.Msg) (serverDetailPanelModel, tea.Cmd) {
	switch msg := msg.(type) {
	case selectServerSnapshotMsg:
		m.snapshot = msg.server
		var smPlugins []list.Item
		for _, plugin := range m.snapshot.PluginsSM {
			smPlugins = append(smPlugins, model.PluginItem[tf.GamePlugin]{Item: plugin})
		}
		var mmPlugins []list.Item
		for _, plugin := range m.snapshot.PluginsMeta {
			mmPlugins = append(mmPlugins, model.PluginItem[tf.GamePlugin]{Item: plugin})
		}

		m.listSM.SetItems(smPlugins)
		m.listMeta.SetItems(mmPlugins)
	case contentViewPortHeightMsg:
		m.width = msg.width
		if !m.ready {
			m.viewportDetail = viewport.New(msg.width/3, msg.contentViewPortHeight)
			m = m.resize(msg.width, msg.contentViewPortHeight)
			m.ready = true
		} else {
			m = m.resize(msg.width, msg.contentViewPortHeight)
		}
	}

	return m, nil
}

func (m serverDetailPanelModel) resize(width int, height int) serverDetailPanelModel {
	m.viewportDetail.Height = height / 2
	m.viewportDetail.Width = width / 3
	m.listSM.SetHeight(height / 2)
	m.listSM.SetWidth(width / 3)
	m.listMeta.SetHeight(height / 2)
	m.listMeta.SetWidth(width / 3)

	return m
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

	m.viewportDetail.SetContent(lipgloss.JoinVertical(lipgloss.Top, rows...))

	titleBar := renderTitleBar(m.width, "Server Overview: "+m.snapshot.Status.ServerName)
	m.viewportDetail.Height = height - lipgloss.Height(titleBar)

	var metaPlugins []string
	for _, plugin := range m.snapshot.PluginsMeta {
		metaPlugins = append(metaPlugins, lipgloss.JoinHorizontal(lipgloss.Top), strconv.Itoa(plugin.Index), plugin.Name, plugin.Version, plugin.Author)
	}

	var smPlugins []string
	for _, plugin := range m.snapshot.PluginsSM {
		smPlugins = append(smPlugins, lipgloss.JoinHorizontal(lipgloss.Top), strconv.Itoa(plugin.Index), plugin.Name, plugin.Version, plugin.Author)
	}

	bottomViews := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.viewportDetail.View(),
		m.listMeta.View(),
		m.listSM.View(),
	)

	return lipgloss.JoinVertical(lipgloss.Top, titleBar, bottomViews)
}
