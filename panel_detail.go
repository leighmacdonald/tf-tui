package main

import (
	"fmt"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/leighmacdonald/tf-tui/styles"
	"golang.org/x/exp/slices"
)

func NewDetailPanel(links []UserLink) DetailPanel {
	return DetailPanel{links: links}
}

type DetailPanel struct {
	links  []UserLink
	player Player
	width  int
	height int
}

func (m DetailPanel) Init() tea.Cmd {
	return nil
}

func (m DetailPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case Config:
		m.links = msg.Links
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case SelectedPlayerMsg:
		m.player = msg.player
	}

	return m, nil
}

func (m DetailPanel) View() string {
	if !m.player.SteamID.Valid() {
		return ""
	}

	var rows []string
	rows = append(rows,
		styles.DetailRow("SteamID", m.player.SteamID.String()),
		styles.DetailRow("Steam Profile",
			fmt.Sprintf("https://steamcommunity.com/profiles/%s", m.player.SteamID.String())))

	for _, link := range m.links {
		rows = append(rows, styles.DetailRow(link.Name, link.Generate(m.player.SteamID)))
	}

	if len(m.player.meta.CompetitiveTeams) > 0 {
		var leagues []string
		for _, team := range m.player.meta.CompetitiveTeams {
			if !slices.Contains(leagues, team.League) {
				leagues = append(leagues, team.League)
			}
		}
		for _, league := range leagues {
			switch league { //nolint:gocritic
			case "rgl":
				rows = append(rows, styles.DetailRow("RGL Profile",
					fmt.Sprintf("https://rgl.gg/Public/PlayerProfile?p=%s", m.player.SteamID.String())))
			case "ugc":
				rows = append(rows, styles.DetailRow("UGC Profile",
					fmt.Sprintf("https://www.ugcleague.com/players_page.cfm?player_id=%s", m.player.SteamID.String())))
			}
		}
	}

	if m.player.meta.TimeCreated > 0 {
		rows = append(rows, styles.DetailRow("Acct. Age",
			humanize.RelTime(time.Unix(m.player.meta.TimeCreated, 0), time.Now(), "", "")))
	}

	if m.player.meta.EconomyBan != "none" && m.player.meta.EconomyBan != "" {
		rows = append(rows, styles.DetailRow("Econ Ban", m.player.meta.EconomyBan))
	}

	if m.player.meta.CommunityBanned {
		rows = append(rows, styles.DetailRow("Comm Ban", "true"))
	}

	if m.player.meta.NumberOfVacBans > 0 {
		rows = append(rows, styles.DetailRow("Vac Bans",
			fmt.Sprintf("%d (%d days)", m.player.meta.NumberOfVacBans, m.player.meta.DaysSinceLastBan)))
	}

	if m.player.meta.NumberOfGameBans > 0 {
		rows = append(rows, styles.DetailRow("Game Bans", strconv.Itoa(int(m.player.meta.NumberOfGameBans))))
	}

	if m.player.meta.LogsCount > 0 {
		rows = append(rows, styles.DetailRow("Logs.tf", strconv.Itoa(int(m.player.meta.LogsCount))))
	}

	return styles.PanelBorder.Width(m.width - 2).Render(lipgloss.JoinVertical(lipgloss.Top, rows...))
}
