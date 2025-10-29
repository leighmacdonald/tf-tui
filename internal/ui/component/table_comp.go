package component

import (
	"slices"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/leighmacdonald/tf-tui/internal/tfapi"
	"github.com/leighmacdonald/tf-tui/internal/ui/command"
	"github.com/leighmacdonald/tf-tui/internal/ui/model"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
)

type compTableCol int

const (
	colLeague compTableCol = iota
	colCompetition
	colJoined
	colLeft
	colFormat
	colDivision
	colTeamName
)

type compTableSize int

const (
	colLeagueSize      compTableSize = 16
	colCompetitionSize compTableSize = 30
	colJoinedSize      compTableSize = 12
	colLeftSize        compTableSize = 12
	colFormatSize      compTableSize = 12
	colDivisionSize    compTableSize = 15
	colTeamNameSize    compTableSize = -1
)

type TableCompModel struct {
	player    model.Player
	table     *table.Table
	viewState model.ViewState
	ready     bool
	viewport  viewport.Model
}

func NewTableCompModel() TableCompModel {
	return TableCompModel{
		table: NewUnstyledTable("League", "Competition", "Joined", "Left", "Format", "Division", "Team Name"),
	}
}

func (m TableCompModel) Init() tea.Cmd {
	return nil
}

func (m TableCompModel) Update(msg tea.Msg) (TableCompModel, tea.Cmd) {
	switch msg := msg.(type) {
	case model.ViewState:
		m.viewState = msg
		m.table.Height(msg.Lower - 2)
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Lower)
			m.ready = true
		} else {
			m.viewport.Height = msg.Lower - 1
		}
	case command.SelectedPlayerMsg:
		m.player = msg.Player
		m.table.ClearRows()

		var rows [][]string
		if m.player.CompetitiveTeams != nil {
			slices.SortStableFunc(m.player.CompetitiveTeams, func(a, b tfapi.LeaguePlayerTeamHistory) int {
				return a.JoinedTeam.Compare(b.LeftTeam)
			})
			slices.Reverse(m.player.CompetitiveTeams)
			for _, team := range m.player.CompetitiveTeams {
				var (
					joined string
					left   string
				)

				if !team.JoinedTeam.IsZero() {
					joined = team.JoinedTeam.Format(time.DateOnly)
				}

				if !team.LeftTeam.IsZero() {
					left = team.LeftTeam.Format(time.DateOnly)
				}

				rows = append(rows, []string{
					team.League,
					team.SeasonName,
					joined,
					left,
					team.Format,
					team.DivisionName,
					team.TeamName,
				})
			}
		}
		// m.table.Height(len(rows))
		m.table.Rows(rows...)
		var content string
		if len(m.player.CompetitiveTeams) == 0 {
			content = styles.InfoMessage.Width(m.viewState.Width).Render("No league history found " + styles.IconNoComp)
		} else {
			content = m.table.
				StyleFunc(func(row int, col int) lipgloss.Style {
					var width compTableSize
					switch compTableCol(col) {
					case colLeague:
						width = colLeagueSize
					case colCompetition:
						width = colCompetitionSize
					case colJoined:
						width = colJoinedSize
					case colLeft:
						width = colLeftSize
					case colFormat:
						width = colFormatSize
					case colDivision:
						width = colDivisionSize
					case colTeamName:
						// consts are just an illusion of course :)
						width = compTableSize(m.viewState.Width - int(colLeagueSize) - int(colCompetitionSize) -
							int(colJoinedSize) - int(colLeftSize) - int(colFormatSize) - int(colDivisionSize) - 4)
					}
					switch {
					case row == table.HeaderRow:
						return styles.HeaderStyleRed.Padding(0).Width(int(width))
					case row%2 == 0:
						return styles.TableRowValuesEven.Width(int(width))
					default:
						return styles.TableRowValuesOdd.Width(int(width))
					}
				}).
				Render()
		}
		m.viewport.SetContent(content)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

func (m TableCompModel) Render(height int) string {
	m.viewport.Height = height - 2

	return Container("Competitive History", m.viewState.Width, height, m.viewport.View(), false)
}
