package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"
	"github.com/leighmacdonald/tf-tui/styles"
)

type DetailPanel struct {
	player Player
}

func (m DetailPanel) Init() tea.Cmd {
	return nil
}

func (m DetailPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SelectedPlayerMsg:
		m.player = msg.player
	}

	return m, nil
}

func (m DetailPanel) View() string {
	if !m.player.SteamID.Valid() {
		return ""
	}

	var b strings.Builder

	b.WriteString(styles.PanelLabel.Render("SteamID: "))
	b.WriteString(styles.PanelValue.Render(m.player.SteamID.String()))

	b.WriteString(styles.PanelValue.Render(fmt.Sprintf("\nhttps://steamcommunity.com/profiles/%s", m.player.SteamID.String())))

	if m.player.meta.TimeCreated > 0 {
		diff := humanize.RelTime(time.Unix(m.player.meta.TimeCreated, 0), time.Now(), "", "")
		//age := time.Since(time.Unix(m.player.meta.TimeCreated, 0))
		b.WriteString(styles.PanelLabel.Render("\nAcct. Age: "))
		b.WriteString(styles.PanelValue.Render(diff))
	}

	if m.player.meta.EconomyBan != "none" {
		b.WriteString(styles.PanelLabel.Render("\nEcon Ban: "))
		b.WriteString(styles.PanelValue.Render(m.player.meta.EconomyBan))
	}

	if m.player.meta.CommunityBanned {
		b.WriteString(styles.PanelLabel.Render("\nComm Ban: "))
		b.WriteString(styles.PanelValue.Render("true"))
	}

	if m.player.meta.NumberOfVacBans > 0 {
		b.WriteString(styles.PanelLabel.Render("\nVac Bans: "))
		b.WriteString(styles.PanelValue.Render(fmt.Sprintf("%d (%d days)", m.player.meta.NumberOfVacBans, m.player.meta.DaysSinceLastBan)))
	}

	if m.player.meta.NumberOfGameBans > 0 {
		b.WriteString(styles.BlurredStyle.Render("\nGame Bans: "))
		b.WriteString(styles.NoStyle.Render(strconv.Itoa(int(m.player.meta.NumberOfGameBans))))
	}

	if m.player.meta.LogsCount > 0 {
		b.WriteString(styles.BlurredStyle.Render("\nLogs.tf #: "))
		b.WriteString(styles.NoStyle.Render(strconv.Itoa(int(m.player.meta.LogsCount))))
	}

	return styles.PanelBorder.Width(50).Render(b.String())
}
