package ui

import (
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/leighmacdonald/tf-tui/config"
	"github.com/leighmacdonald/tf-tui/ui/styles"
	"golang.org/x/exp/slices"
)

func newDetailPanelModel(links []config.UserLink) detailPanelModel {
	return detailPanelModel{
		links:    links,
		viewport: viewport.New(1, 1),
	}
}

type detailPanelModel struct {
	links                 []config.UserLink
	players               Players
	player                Player
	width                 int
	height                int
	contentViewPortHeight int
	ready                 bool
	viewport              viewport.Model
}

func (m detailPanelModel) Init() tea.Cmd {
	return nil
}

func (m detailPanelModel) Update(msg tea.Msg) (detailPanelModel, tea.Cmd) {
	switch msg := msg.(type) {
	case config.Config:
		m.links = msg.Links
	case FullStateUpdateMsg:
		m.players = msg.players
	case ContentViewPortHeightMsg:
		m.width = msg.width
		m.height = msg.height
		if !m.ready {
			m.viewport = viewport.New(msg.width, msg.contentViewPortHeight)
			m.ready = true
		} else {
			m.contentViewPortHeight = msg.contentViewPortHeight
			m.viewport.Height = msg.contentViewPortHeight
		}
	case SelectedPlayerMsg:
		m.player = msg.player
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

func (m detailPanelModel) Render(height int) string {
	if !m.player.SteamID.Valid() {
		return ""
	}

	var rows []string //nolint:prealloc
	rows = append(rows,
		styles.DetailRow("SteamID", m.player.SteamID.String()),
		styles.DetailRow("Steam Profile",
			fmt.Sprintf("https://steamcommunity.com/profiles/%s", m.player.SteamID.String())))

	for _, link := range m.links {
		rows = append(rows, styles.DetailRow(link.Name, link.Generate(m.player.SteamID)))
	}

	if len(m.player.CompetitiveTeams) > 0 {
		var leagues []string
		for _, team := range m.player.CompetitiveTeams {
			if !slices.Contains(leagues, team.League) {
				leagues = append(leagues, team.League)
			}
		}
		for _, league := range leagues {
			switch league {
			case "rgl":
				rows = append(rows, styles.DetailRow("RGL Profile",
					fmt.Sprintf("https://rgl.gg/Public/PlayerProfile?p=%s", m.player.SteamID.String())))
			case "ugc":
				rows = append(rows, styles.DetailRow("UGC Profile",
					fmt.Sprintf("https://www.ugcleague.com/players_page.cfm?player_id=%s", m.player.SteamID.String())))
			}
		}
	}

	if m.player.TimeCreated > 0 {
		rows = append(rows, styles.DetailRow("Acct. Age",
			humanize.RelTime(time.Unix(m.player.TimeCreated, 0), time.Now(), "", "")))
	}

	if m.player.EconomyBan != "none" && m.player.EconomyBan != "" {
		rows = append(rows, styles.DetailRow("Econ Ban", m.player.EconomyBan))
	}

	if m.player.CommunityBanned {
		rows = append(rows, styles.DetailRow("Comm Ban", "true"))
	}

	if m.player.NumberOfVacBans > 0 {
		rows = append(rows, styles.DetailRow("Vac Bans",
			fmt.Sprintf("%d (%d days)", m.player.NumberOfVacBans, m.player.DaysSinceLastBan)))
	}

	if m.player.NumberOfGameBans > 0 {
		rows = append(rows, styles.DetailRow("Game Bans", strconv.Itoa(int(m.player.NumberOfGameBans))))
	}
	// FIXME
	// if len(m.player.BDMatches) > 0 {
	// 	rows = append(rows, styles.DetailRow("Bot Detector Entries",
	// 		strconv.Itoa(len(m.player.BDMatches))))
	// }

	if m.player.LogsCount > 0 {
		rows = append(rows, styles.DetailRow("Logs.tf", strconv.Itoa(int(m.player.LogsCount))))
	}

	rows = append(rows, styles.DetailRow("Friends (Steam)", strconv.Itoa(len(m.player.Friends))))

	friends := m.players.FindFriends(m.player.SteamID)
	rows = append(rows, styles.DetailRow("Friends (In Game)", strconv.Itoa(len(friends))))

	m.viewport.SetContent(lipgloss.JoinVertical(lipgloss.Top, rows...))

	titleBar := renderTitleBar(m.width, "Player Overview")
	m.viewport.Height = height - lipgloss.Height(titleBar)

	return lipgloss.JoinVertical(lipgloss.Top, titleBar, m.viewport.View())
}
