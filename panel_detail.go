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

type DetailPanel struct {
	player Player
	width  int
	height int
}

func (m DetailPanel) Init() tea.Cmd {
	return nil
}

func (m DetailPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
		lipgloss.JoinHorizontal(lipgloss.Top,
			styles.PanelLabel.Render("SteamID: "),
			styles.PanelValue.Render(m.player.SteamID.String())))

	rows = append(rows,
		lipgloss.JoinHorizontal(lipgloss.Top,
			styles.PanelLabel.Render("Steam Profile: "),
			styles.PanelValue.Render(fmt.Sprintf("https://steamcommunity.com/profiles/%s", m.player.SteamID.String()))))

	if m.player.meta.CompetitiveTeams != nil && len(*m.player.meta.CompetitiveTeams) > 0 {
		var leagues []string
		for _, team := range *m.player.meta.CompetitiveTeams {
			if !slices.Contains(leagues, team.League) {
				leagues = append(leagues, team.League)
			}
		}
		for _, league := range leagues {
			switch league {
			case "rgl":
				rows = append(rows,
					lipgloss.JoinHorizontal(lipgloss.Top,
						styles.PanelLabel.Render("RGL Profile: "),
						styles.PanelValue.Render(fmt.Sprintf("https://rgl.gg/Public/PlayerProfile?p=%s", m.player.SteamID.String()))))
			}
		}
	}

	if m.player.meta.TimeCreated > 0 {
		diff := humanize.RelTime(time.Unix(m.player.meta.TimeCreated, 0), time.Now(), "", "")
		// age := time.Since(time.Unix(m.player.meta.TimeCreated, 0))
		rows = append(rows,
			lipgloss.JoinHorizontal(lipgloss.Top,
				styles.PanelLabel.Render("Acct. Age: "),
				styles.PanelValue.Render(diff)))
	}

	if m.player.meta.EconomyBan != "none" && m.player.meta.EconomyBan != "" {
		rows = append(rows,
			lipgloss.JoinHorizontal(lipgloss.Top,
				styles.PanelLabel.Render("Econ Ban: "),
				styles.PanelValue.Render(m.player.meta.EconomyBan)))
	}

	if m.player.meta.CommunityBanned {
		rows = append(rows,
			lipgloss.JoinHorizontal(lipgloss.Top,
				styles.PanelLabel.Render("Comm Ban: "),
				styles.PanelValue.Render("true")))
	}

	if m.player.meta.NumberOfVacBans > 0 {
		rows = append(rows,
			lipgloss.JoinHorizontal(lipgloss.Top,
				styles.PanelLabel.Render("Vac Bans: "),
				styles.PanelValue.Render(fmt.Sprintf("%d (%d days)", m.player.meta.NumberOfVacBans, m.player.meta.DaysSinceLastBan))))
	}

	if m.player.meta.NumberOfGameBans > 0 {
		rows = append(rows,
			lipgloss.JoinHorizontal(lipgloss.Top,
				styles.BlurredStyle.Render("Game Bans: "),
				styles.NoStyle.Render(strconv.Itoa(int(m.player.meta.NumberOfGameBans)))))
	}

	if m.player.meta.LogsCount > 0 {
		rows = append(rows,
			lipgloss.JoinHorizontal(lipgloss.Top,
				styles.BlurredStyle.Render("Logs.tf #: "),
				styles.NoStyle.Render(strconv.Itoa(int(m.player.meta.LogsCount)))))
	}

	return styles.PanelBorder.Width(m.width).Render(lipgloss.JoinVertical(lipgloss.Top, rows...))
}
