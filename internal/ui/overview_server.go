package ui

import (
	"math"
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
	cvars := model.NewCVarList()
	cvars.SetStatusBarItemName("cvar", "cvars")

	return serverDetailPanelModel{
		listSM:   model.NewPluginList("Sourcemod Plugins"),
		listMeta: model.NewPluginList("Metamod Plugins"),
		listCvar: cvars,
	}
}

type serverDetailPanelModel struct {
	snapshot       Snapshot
	width          int
	viewportDetail viewport.Model
	listSM         list.Model
	listMeta       list.Model
	listCvar       list.Model
	ready          bool
}

func (m serverDetailPanelModel) Init() tea.Cmd {
	return nil
}

func (m serverDetailPanelModel) Update(msg tea.Msg) (serverDetailPanelModel, tea.Cmd) {
	switch msg := msg.(type) {
	case selectServerSnapshotMsg:
		var smPlugins []list.Item
		for _, plugin := range m.snapshot.PluginsSM {
			smPlugins = append(smPlugins, model.PluginItem[tf.GamePlugin]{Item: plugin})
		}

		var mmPlugins []list.Item
		for _, plugin := range m.snapshot.PluginsMeta {
			mmPlugins = append(mmPlugins, model.PluginItem[tf.GamePlugin]{Item: plugin})
		}

		var cvars []list.Item
		for _, cvar := range m.snapshot.CVars {
			cvars = append(cvars, model.CVarItem[tf.CVar]{Item: cvar})
		}

		m.listSM.SetItems(smPlugins)
		m.listMeta.SetItems(mmPlugins)
		m.listCvar.SetItems(cvars)
		m.snapshot = msg.server
	case contentViewPortHeightMsg:
		m.width = msg.width
		if !m.ready {
			m.viewportDetail = viewport.New(msg.width/2, msg.contentViewPortHeight)
			m = m.resize(msg.width, msg.contentViewPortHeight)
			m.ready = true
		} else {
			m = m.resize(msg.width, msg.contentViewPortHeight)
		}
	}

	return m, nil
}

func calcPct(size int, percent float64) int {
	return int(math.Floor(float64(size) * percent / 100))
}

func (m serverDetailPanelModel) resize(width int, height int) serverDetailPanelModel {
	m.viewportDetail.Height = height / 2
	m.viewportDetail.Width = calcPct(width, 50)
	m.listSM.SetHeight(height / 3)
	m.listSM.SetWidth(calcPct(width, 15))
	m.listMeta.SetHeight(height / 2)
	m.listMeta.SetWidth(calcPct(width, 15))
	m.listCvar.SetHeight(height / 2)
	m.listCvar.SetWidth(calcPct(width, 20))

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

	bottomViews := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.viewportDetail.View(),
		m.listMeta.View(),
		m.listSM.View(),
		m.listCvar.View(),
	)

	return lipgloss.NewStyle().Width(m.width).Render(lipgloss.JoinVertical(lipgloss.Top, titleBar, bottomViews))
}
