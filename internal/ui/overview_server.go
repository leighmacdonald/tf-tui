package ui

import (
	"fmt"
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
	keyZone        keyZone
}

func (m serverDetailPanelModel) Init() tea.Cmd {
	return nil
}

func (m serverDetailPanelModel) Update(msg tea.Msg) (serverDetailPanelModel, tea.Cmd) {
	switch msg := msg.(type) {
	case keyZone:
		m.keyZone = msg
	case selectServerSnapshotMsg:
		var smPlugins []list.Item
		for _, plugin := range m.snapshot.PluginsSM {
			smPlugins = append(smPlugins, model.GamePluginItem[tf.GamePlugin]{Item: plugin})
		}

		var mmPlugins []list.Item
		for _, plugin := range m.snapshot.PluginsMeta {
			mmPlugins = append(mmPlugins, model.GamePluginItem[tf.GamePlugin]{Item: plugin})
		}

		var cvars []list.Item
		for _, cvar := range m.snapshot.CVars {
			cvars = append(cvars, model.CVarItem[tf.CVar]{Item: cvar})
		}

		m.listSM.SetItems(smPlugins)
		m.listMeta.SetItems(mmPlugins)
		m.listCvar.SetItems(cvars)
		m.snapshot = msg.server
	case viewState:
		m.width = msg.width
		if !m.ready {
			m.viewportDetail = viewport.New(msg.width/2, msg.lowerSize)
			m = m.resize(msg.width-4, msg.lowerSize)
			m.ready = true
		} else {
			m = m.resize(msg.width-4, msg.lowerSize)
		}
		m.listCvar.SetHeight(msg.lowerSize - 4)
		m.listSM.SetHeight(msg.lowerSize - 4)
		m.listMeta.SetHeight(msg.lowerSize - 4)
		m.viewportDetail.Height = msg.lowerSize - 4
	}

	return m, nil
}

func calcPct(size int, percent float64) int {
	return int(math.Floor(float64(size) * percent / 100))
}

func (m serverDetailPanelModel) resize(width int, height int) serverDetailPanelModel {
	m.viewportDetail.Height = height / 2
	m.viewportDetail.Width = calcPct(width, 50)
	m.listSM.SetWidth(calcPct(width, 15))
	m.listMeta.SetWidth(calcPct(width, 15))
	m.listCvar.SetWidth(calcPct(width, 20))

	return m
}

func (m serverDetailPanelModel) Render(height int) string {
	m.listCvar.SetHeight(height)
	m.listSM.SetHeight(height)
	m.listMeta.SetHeight(height)

	var rows []string
	rows = append(rows, styles.DetailRow("Game Tags", strings.Join(m.snapshot.Status.Tags, ", ")))
	var edicts []string
	for _, edict := range m.snapshot.Status.Edicts {
		edicts = append(edicts, strconv.Itoa(edict))
	}
	rows = append(rows, styles.DetailRow("EDicts", strings.Join(edicts, ", ")))
	rows = append(rows, styles.DetailRow("Game", m.snapshot.Server.Game))

	m.viewportDetail.SetContent(lipgloss.JoinVertical(lipgloss.Top, rows...))

	// titleBar := renderTitleBar(m.width, "Server Overview: "+m.snapshot.Status.ServerName)
	m.viewportDetail.Height = height - 2 // - lipgloss.Height(titleBar)

	// TODO compute this
	borderSize := 8 // 4 containers, 2 sides each
	bottomViews := lipgloss.JoinHorizontal(
		lipgloss.Top,
		model.Container("Overview", calcPct(m.width-borderSize, 30), height, m.viewportDetail.View(), m.keyZone == serverOverview),
		model.Container(fmt.Sprintf("Meta (%d)", len(m.listMeta.Items())), calcPct(m.width-borderSize, 20), height, m.listMeta.View(), m.keyZone == listMetamod),
		model.Container(fmt.Sprintf("Sourcemod (%d)", len(m.listSM.Items())), calcPct(m.width-borderSize, 20), height, m.listSM.View(), m.keyZone == listSourcemod),
		model.Container(fmt.Sprintf("CVars (%d)", len(m.listCvar.Items())), calcPct(m.width-borderSize, 30), height, m.listCvar.View(), m.keyZone == listCvars),
	)

	return lipgloss.NewStyle().Width(m.width).Render(bottomViews)
}
