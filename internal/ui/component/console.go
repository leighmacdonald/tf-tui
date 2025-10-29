package component

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	"github.com/leighmacdonald/tf-tui/internal/tf/events"
	"github.com/leighmacdonald/tf-tui/internal/ui/command"
	"github.com/leighmacdonald/tf-tui/internal/ui/input"
	"github.com/leighmacdonald/tf-tui/internal/ui/model"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
	zone "github.com/lrstanley/bubblezone"
	"github.com/muesli/reflow/wordwrap"
)

type LogRow struct {
	Content   string
	CreatedOn time.Time
	EventType events.EventType
}

func (r LogRow) Render(width int) string {
	timeStamp := styles.ConsoleTime.Render(" " + r.CreatedOn.Format(time.TimeOnly) + " ")
	body := " " + strings.TrimSpace(wordwrap.String(r.Content, width-lipgloss.Width(timeStamp)-2)) + " "

	switch r.EventType {
	case events.Msg:
		body = styles.ConsoleMsg.Render(body)
	case events.Connect:
		body = styles.ConsoleConnect.Render(body)
	case events.Disconnect:
		body = styles.ConsoleDisconnect.Render(body)
	case events.Address:
		body = styles.ConsoleAddress.Render(body)
	case events.Hostname:
		body = styles.ConsoleHostname.Render(body)
	case events.StatusID:
		body = styles.ConsoleStatusID.Render(body)
	case events.Map:
		body = styles.ConsoleMap.Render(body)
	case events.Tags:
		body = styles.ConsoleTags.Render(body)
	case events.Kill:
		body = styles.ConsoleKill.Render(body)
	case events.Stats:
		body = styles.ConsoleKill.Render(body)
	case events.Version:
		// TODO
	case events.Any:
		fallthrough
	default:
		body = styles.ConsoleOther.Render(body)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, timeStamp, body)
}

type ConsoleModel struct {
	ready  bool
	rowsMu *sync.RWMutex
	// indexed by log secret
	rowsRendered   map[string]string
	rowsCount      map[string]int
	cvarList       map[string]tf.CVarList
	viewPort       viewport.Model
	focused        bool
	filterNoisy    bool
	selectedServer model.Snapshot
	input          textinput.Model
	inputZoneID    string
	viewState      model.ViewState
}

func NewConsoleModel() *ConsoleModel {
	input := textinput.New()
	input.CharLimit = 120
	input.Placeholder = "cmd..."
	input.Prompt = styles.ConsolePrompt
	model := ConsoleModel{
		rowsMu:       &sync.RWMutex{},
		rowsRendered: map[string]string{},
		rowsCount:    map[string]int{},
		cvarList:     map[string]tf.CVarList{},
		viewPort:     viewport.New(10, 20),
		input:        input,
		inputZoneID:  zone.NewPrefix(),
	}

	return &model
}

func (m *ConsoleModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *ConsoleModel) Update(msg tea.Msg) (*ConsoleModel, tea.Cmd) {
	cmds := make([]tea.Cmd, 2)

	m.viewPort, cmds[0] = m.viewPort.Update(msg)

	if m.viewState.KeyZone == model.KZconsoleInput {
		m.input, cmds[1] = m.input.Update(msg)
		m.input.SetValue("")
	}

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch { //nolint:gocritic
		case key.Matches(msg, input.Default.Accept):
			cmd := m.input.Value()
			if cmd == "" {
				break
			}
			cmds = append(cmds, command.SendRCONCommand(m.selectedServer.HostPort, cmd))
		}
	case model.ViewState:
		m.viewState = msg
		m.viewPort.Width = msg.Width
		m.input.Width = msg.Width - 8
		if msg.Section == model.SectionConsole {
			if m.viewState.KeyZone == model.KZconsoleInput && !m.input.Focused() {
				cmds = append(cmds, m.input.Focus())
			}
		}
	case command.SelectServerSnapshotMsg:
		m.selectedServer = msg.Server
		if cvars, ok := m.cvarList[msg.Server.HostPort]; ok {
			m.input.SetSuggestions(cvars.Filter("").Names())
		}
	case events.Event:
		return m.onLogs(msg), tea.Batch(cmds...)
	case command.ServerCVarList:
		m.cvarList[msg.HostPort] = msg.List
	}

	return m, tea.Batch(cmds...)
}

func (m *ConsoleModel) onLogs(event events.Event) *ConsoleModel {
	// if slices.Contains([]tf.EventType{tf.EvtStatusID, tf.EvtHostname, tf.EvtMsg, tf.EvtTags, tf.EvtAddress, tf.EvtLobby}, log.Type) {
	// 	return m
	// }
	parts := strings.SplitN(event.Raw, ": ", 2)
	if len(parts) != 2 {
		return m
	}
	if parts[1] == "" {
		return m
	}

	if m.filterNoisy {
		valid := true
		for _, prefix := range []string{"# ", "version ", "steamid ", "players ", "map ", "account ", "edicts "} {
			if strings.HasPrefix(parts[1], prefix) {
				valid = false

				break
			}
		}
		if !valid {
			return m
		}
	}

	newRow := LogRow{Content: safeString(parts[1]), CreatedOn: time.Now(), EventType: event.Type}
	m.rowsMu.Lock()
	// This does not use JoinVertical currently as it takes more and more CPU as time goes on
	// and the console log fills becoming unusable.
	prev := m.rowsRendered[event.HostPort]
	m.rowsRendered[event.HostPort] = prev + "\n" + newRow.Render(m.viewState.Width-10)
	if _, ok := m.rowsCount[event.HostPort]; !ok {
		m.rowsCount[event.HostPort] = 0
	}
	m.rowsCount[event.HostPort]++
	m.rowsMu.Unlock()

	return m
}

func safeString(s string) string {
	s = strings.TrimFunc(s, func(r rune) bool {
		return !unicode.IsGraphic(r) || unicode.IsControl(r)
	})

	return s
}

func (m *ConsoleModel) Render(height int) string {
	content, found := m.rowsRendered[m.selectedServer.HostPort]
	if !found || content == "" {
		content = "<<< Start of logs >>>\n"
	}

	input := zone.Mark(m.inputZoneID, m.input.View())

	m.viewPort.Height = height - lipgloss.Height(input)
	wasBottom := m.viewPort.AtBottom()

	m.viewPort.SetContent(content)
	if wasBottom {
		m.viewPort.GotoBottom()
	}

	title := fmt.Sprintf("Console Log: %d Messages", m.rowsCount[m.selectedServer.HostPort])

	return Container(
		title,
		m.viewState.Width,
		height,
		lipgloss.JoinVertical(lipgloss.Left, m.viewPort.View(), input),
		m.viewState.KeyZone == model.KZconfigInput)
}
