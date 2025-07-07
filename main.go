package main

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config config.yaml ./openapi.yaml

import (
	"cmp"
	"context"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/styles"

	"os"
)

type Team int

const (
	UNASSIGNED = iota
	SPEC
	BLU
	RED
)

type keymap struct {
	start  key.Binding
	stop   key.Binding
	reset  key.Binding
	quit   key.Binding
	config key.Binding
}

type errMsg error

type TickMsg struct {
	err  error
	t    time.Time
	dump *DumpPlayer
}

type appState struct {
	config         Config
	api            *ClientWithResponses
	table          *tableModel
	loadingSpinner spinner.Model
	keymap         keymap
	titleState     string
	quitting       bool
	err            errMsg
	dump           *DumpPlayer
	messages       []string
	windowSize     tea.WindowSizeMsg
	help           widgetConfig
	selectedTeam   Team
	selectedRow    int
	inConfig       bool
	statusMsg      string
}

func (m appState) Init() tea.Cmd {
	return tea.Batch(tea.SetWindowTitle("tf-tui"), m.tickEvery(), textinput.Blink, m.help.inputAddr.Focus())
}

func (m appState) View() string {
	var b strings.Builder
	b.WriteString(m.renderHeading())
	if m.inConfig {
		b.WriteString(m.renderConfig())
	} else {
		b.WriteString(m.renderPlayerTables())
	}

	// The footer
	b.WriteString(strings.Join(m.messages, "\n"))

	// Send the UI for rendering
	return b.String()
}

func (m appState) handleHelpInputs(msg string) (appState, tea.Cmd) {
	switch msg {
	case "ctrl+c", "esc":
		m.inConfig = false
		return m, nil
	case "up":
		if m.help.focusIndex > 0 && m.help.focusIndex <= 2 {
			m.help.focusIndex--
		}
	case "down":
		if m.help.focusIndex >= 0 && m.help.focusIndex < 2 {
			m.help.focusIndex++
		}
	case "enter":
		switch m.help.focusIndex {
		case 0:
			m.help.focusIndex++
		case 1:
			m.help.focusIndex++
		case 2:
			return m, tea.Batch(func() tea.Msg {
				return m.config
			})
		}
	}

	cmds := make([]tea.Cmd, 2)

	switch m.help.focusIndex {
	case 0:
		cmds = append(cmds, m.help.inputAddr.Focus())
		m.help.inputAddr.PromptStyle = styles.FocusedStyle
		m.help.inputAddr.TextStyle = styles.FocusedStyle

		m.help.passwordAddr.Blur()
		m.help.passwordAddr.PromptStyle = styles.NoStyle
		m.help.passwordAddr.TextStyle = styles.NoStyle
	case 1:
		cmds = append(cmds, m.help.passwordAddr.Focus())
		m.help.passwordAddr.PromptStyle = styles.FocusedStyle
		m.help.passwordAddr.TextStyle = styles.FocusedStyle

		m.help.inputAddr.Blur()
		m.help.inputAddr.PromptStyle = styles.NoStyle
		m.help.inputAddr.TextStyle = styles.NoStyle
	case 2:
		m.help.passwordAddr.Blur()
		m.help.passwordAddr.PromptStyle = styles.NoStyle
		m.help.passwordAddr.TextStyle = styles.NoStyle
		m.help.inputAddr.Blur()
		m.help.inputAddr.PromptStyle = styles.NoStyle
		m.help.inputAddr.TextStyle = styles.NoStyle
	}

	return m, tea.Batch(append(cmds, m.updateInputs(msg)...)...)
}
func (m appState) handleDefaultInpuits(msg string) (appState, tea.Cmd) {
	switch msg {
	// These keys should exit the program.
	case "ctrl+c", "q", "esc":
		return m, tea.Quit
	case "E":
		m.inConfig = true
		return m, nil
	// The "up" and "k" keys move the cursor up
	case "up", "k":
		//if m.cursor > 0 {
		//	m.cursor--
		//}

	// The "down" and "j" keys move the cursor down
	case "down", "j":
		//if m.cursor < len(m.choices)-1 {
		//	m.cursor++
		//}

	// The "enter" key and the spacebar (a literal space) toggle
	// the selected state for the item that the cursor is pointing at.
	case "enter", " ":
		//_, ok := m.selected[m.cursor]
		//if ok {
		//	delete(m.selected, m.cursor)
		//} else {
		//	m.selected[m.cursor] = struct{}{}
		//}
	}

	return m, nil
}

func (m appState) Update(inMsg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := inMsg.(type) {
	case Config:
		if err := configWrite(defaultConfigName, msg); err != nil {
			m.err = err
			return m, nil
		}

		m.statusMsg = "Saved config"
		m.config = msg
		m.inConfig = false

		return m, nil

	case TickMsg:
		if msg.dump != nil {
			m.dump = msg.dump
		}
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
		}

		return m, tea.Batch(m.tickEvery())

	case tea.WindowSizeMsg:
		m.windowSize = msg

	// Is it a key press?
	case tea.KeyMsg:
		keyMsg := msg.String()
		if m.inConfig {
			return m.handleHelpInputs(keyMsg)
		}
		return m.handleDefaultInpuits(keyMsg)

	case errMsg:
		m.err = msg
		return m, nil

	default:
		var cmd tea.Cmd
		m.loadingSpinner, cmd = m.loadingSpinner.Update(msg)
		return m, cmd
	}

	cmd := m.updateInputs(inMsg)

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, tea.Batch(cmd...)
}
func (m *appState) updateInputs(msg tea.Msg) []tea.Cmd {
	cmds := make([]tea.Cmd, 2)

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	m.help.inputAddr, cmds[0] = m.help.inputAddr.Update(msg)
	m.help.passwordAddr, cmds[1] = m.help.passwordAddr.Update(msg)

	return cmds
}
func (m appState) title() string {
	return styles.Title.Width(m.windowSize.Width / 2).Render("Welcome to tf-tui")
}

func (m appState) status() string {
	if m.err != nil {
		return styles.Title.Render(m.err.Error())
	}
	return styles.Status.Width(m.windowSize.Width / 2).Render(m.statusMsg)
}

func (m appState) renderConfig() string {
	return m.help.Render()
}

func (m appState) renderHeading() string {
	out := lipgloss.JoinHorizontal(lipgloss.Top, m.title(), m.status())

	if m.quitting {
		return out + "\n"
	}

	return out
}

func (m appState) renderPlayerTables() string {
	m.table.dump = m.dump
	return "\n" + m.table.render()
}

func srt(rows [][]string) {
	slices.SortFunc(rows, func(a, b []string) int {
		av, _ := strconv.Atoi(a[1])
		bv, _ := strconv.Atoi(b[1])
		return cmp.Compare(bv, av)
	})
}

func (m appState) tickEvery() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		m.messages = append(m.messages, "updating dump")
		dump, errDump := fetchDumpPlayer(context.Background(), m.config.Address, m.config.Password)
		if errDump != nil {
			m.messages = append(m.messages, "fatal:", errDump.Error())

			return TickMsg{err: errDump, t: t}
		}

		return TickMsg{t: t, dump: dump}
	})
}

func newAppState(client *ClientWithResponses, config Config, doSetup bool) *appState {
	return &appState{
		api:            client,
		config:         config,
		table:          newTableModel(),
		inConfig:       doSetup,
		loadingSpinner: newSpinner(),
		help:           newWidgetConfig(defaultConfig),
		keymap: keymap{
			start: key.NewBinding(
				key.WithKeys("s"),
				key.WithHelp("s", "start"),
			),
			stop: key.NewBinding(
				key.WithKeys("s"),
				key.WithHelp("s", "stop"),
			),
			reset: key.NewBinding(
				key.WithKeys("r"),
				key.WithHelp("r", "reset"),
			),
			quit: key.NewBinding(
				key.WithKeys("q", "ctrl+c"),
				key.WithHelp("q", "quit"),
			),
			config: key.NewBinding(
				key.WithKeys("c"),
				key.WithHelp("c", "confi"),
			),
		}}
}

func main() {
	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	}

	client, errClient := NewClientWithResponses("http://localhost:8888/")
	if errClient != nil {
		fmt.Println("fatal:", errClient)
		os.Exit(1)
	}

	if err := os.MkdirAll(path.Join(xdg.ConfigHome, "tf-tui"), 0755); err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}

	config, exists := configRead(defaultConfigName)
	program := tea.NewProgram(newAppState(client, config, !exists) /**, tea.WithAltScreen()**/)
	if _, err := program.Run(); err != nil {
		fmt.Printf("There's been an error :( %v", err)
		os.Exit(1)
	}
}
