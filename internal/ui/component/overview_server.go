package component

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
	"github.com/leighmacdonald/tf-tui/internal/ui/command"
	"github.com/leighmacdonald/tf-tui/internal/ui/model"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
)

func NewServerDetailPanel() ServerDetailPanelModel {
	cvars := NewCVarList()
	cvars.SetStatusBarItemName("cvar", "cvars")

	return ServerDetailPanelModel{
		listSM:   NewPluginList("Sourcemod Plugins"),
		listMeta: NewPluginList("Metamod Plugins"),
		listCvar: cvars,
	}
}

type ServerDetailPanelModel struct {
	snapshot       model.Snapshot
	viewportDetail viewport.Model
	listSM         list.Model
	listMeta       list.Model
	listCvar       list.Model
	ready          bool
	viewState      model.ViewState
}

func (m ServerDetailPanelModel) Init() tea.Cmd {
	return nil
}

func (m ServerDetailPanelModel) Update(msg tea.Msg) (ServerDetailPanelModel, tea.Cmd) {
	switch msg := msg.(type) {
	case command.SelectServerSnapshotMsg:
		var smPlugins []list.Item
		for _, plugin := range m.snapshot.PluginsSM {
			smPlugins = append(smPlugins, GamePluginItem[tf.GamePlugin]{Item: plugin})
		}

		var mmPlugins []list.Item
		for _, plugin := range m.snapshot.PluginsMeta {
			mmPlugins = append(mmPlugins, GamePluginItem[tf.GamePlugin]{Item: plugin})
		}

		var cvars []list.Item
		for _, cvar := range m.snapshot.CVars {
			cvars = append(cvars, CVarItem[tf.CVar]{Item: cvar})
		}

		m.listSM.SetItems(smPlugins)
		m.listMeta.SetItems(mmPlugins)
		m.listCvar.SetItems(cvars)
		m.snapshot = msg.Server
	case model.ViewState:
		m.viewState = msg
		if !m.ready {
			m.viewportDetail = viewport.New(msg.Width/2, msg.Lower)
			m = m.resize(msg.Width-4, msg.Lower)
			m.ready = true
		} else {
			m = m.resize(msg.Width-4, msg.Lower)
		}
		m.listCvar.SetHeight(msg.Lower - 4)
		m.listSM.SetHeight(msg.Lower - 4)
		m.listMeta.SetHeight(msg.Lower - 4)
		m.viewportDetail.Height = msg.Lower - 4
	}

	return m, nil
}

func calcPct(size int, percent float64) int {
	return int(math.Floor(float64(size) * percent / 100))
}

func (m ServerDetailPanelModel) resize(width int, height int) ServerDetailPanelModel {
	m.viewportDetail.Height = height / 2
	m.viewportDetail.Width = calcPct(width, 50)
	m.listSM.SetWidth(calcPct(width, 15))
	m.listMeta.SetWidth(calcPct(width, 15))
	m.listCvar.SetWidth(calcPct(width, 20))

	return m
}

func (m ServerDetailPanelModel) Render(height int) string {
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
		Container("Overview", calcPct(m.viewState.Width-borderSize, 30), height, m.viewportDetail.View(), m.viewState.KeyZone == model.KZserverOverview),
		Container(fmt.Sprintf("Meta (%d)", len(m.listMeta.Items())), calcPct(m.viewState.Width-borderSize, 20), height, m.listMeta.View(), m.viewState.KeyZone == model.KZlistMetamod),
		Container(fmt.Sprintf("Sourcemod (%d)", len(m.listSM.Items())), calcPct(m.viewState.Width-borderSize, 20), height, m.listSM.View(), m.viewState.KeyZone == model.KZlistSourcemod),
		Container(fmt.Sprintf("CVars (%d)", len(m.listCvar.Items())), calcPct(m.viewState.Width-borderSize, 30), height, m.listCvar.View(), m.viewState.KeyZone == model.KZlistCvars),
	)

	return lipgloss.NewStyle().Width(m.viewState.Width).Render(bottomViews)
}
