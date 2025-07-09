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
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/shared"
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
	up     key.Binding
	down   key.Binding
	left   key.Binding
	right  key.Binding
	fs     key.Binding
}

type errMsg error

type TickMsg struct {
	err  error
	t    time.Time
	dump shared.PlayerState
}

type appState struct {
	config         Config
	api            *ClientWithResponses
	altScreen      bool
	table          *tableModel
	loadingSpinner spinner.Model
	keymap         keymap
	titleState     string
	quitting       bool
	err            errMsg
	dump           shared.PlayerState
	messages       []string
	windowSize     tea.WindowSizeMsg
	configView     widgetConfig
	selectedTeam   Team
	selectedRow    int
	inConfig       bool
	statusMsg      string
	scripting      *Scripting
	helpView       help.Model
	banTable       *BanTableModel
}

func (m appState) Init() tea.Cmd {
	return tea.Batch(tea.SetWindowTitle("tf-tui"), m.tickEvery(), m.configView.Init())
}

func (m appState) View() string {
	var b strings.Builder
	b.WriteString(m.renderHeading())
	if m.inConfig {
		b.WriteString(m.configView.View())
	} else {
		b.WriteString(m.renderPlayerTables())
		b.WriteString("\n")
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, m.banTable.table.Render()))
		b.WriteString("\n")
		b.WriteString(m.helpView.ShortHelpView([]key.Binding{
			m.keymap.quit,
			m.keymap.config,
			m.keymap.fs,
			m.keymap.up,
			m.keymap.down,
			m.keymap.left,
			m.keymap.right,
		}))
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
		if m.configView.focusIndex > 0 && m.configView.focusIndex <= 2 {
			m.configView.focusIndex--
		}
	case "down":
		if m.configView.focusIndex >= 0 && m.configView.focusIndex < 2 {
			m.configView.focusIndex++
		}
	case "enter":
		switch m.configView.focusIndex {
		case 0:
			m.configView.focusIndex++
		case 1:
			m.configView.focusIndex++
		case 2:
			return m, tea.Batch(func() tea.Msg {
				return m.config
			})
		}
	}

	cmds := make([]tea.Cmd, 2)

	switch m.configView.focusIndex {
	case 0:
		cmds = append(cmds, m.configView.inputAddr.Focus())
		m.configView.inputAddr.PromptStyle = styles.FocusedStyle
		m.configView.inputAddr.TextStyle = styles.FocusedStyle

		m.configView.passwordAddr.Blur()
		m.configView.passwordAddr.PromptStyle = styles.NoStyle
		m.configView.passwordAddr.TextStyle = styles.NoStyle
	case 1:
		cmds = append(cmds, m.configView.passwordAddr.Focus())
		m.configView.passwordAddr.PromptStyle = styles.FocusedStyle
		m.configView.passwordAddr.TextStyle = styles.FocusedStyle

		m.configView.inputAddr.Blur()
		m.configView.inputAddr.PromptStyle = styles.NoStyle
		m.configView.inputAddr.TextStyle = styles.NoStyle
	case 2:
		m.configView.passwordAddr.Blur()
		m.configView.passwordAddr.PromptStyle = styles.NoStyle
		m.configView.passwordAddr.TextStyle = styles.NoStyle
		m.configView.inputAddr.Blur()
		m.configView.inputAddr.PromptStyle = styles.NoStyle
		m.configView.inputAddr.TextStyle = styles.NoStyle
	}

	return m, tea.Batch(append(cmds, m.updateInputs(msg)...)...)
}

func (m appState) handleDefaultInpuits(msg string) (appState, tea.Cmd) {
	switch msg {
	case "ctrl+c", "q", "esc":
		return m, tea.Quit
	case "E":
		m.inConfig = true
		return m, nil
	case "up", "k":
		m.table.moveSelection(Up)
	case "down", "j":
		m.table.moveSelection(Down)
	case "left", "h":
		m.table.moveSelection(Left)
	case "right", "l":
		m.table.moveSelection(Right)
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
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
		}

		m.dump = msg.dump
		if m.table.selectedRow > m.table.selectedColumnPlayerCount()-1 {
			m.table.selectedRow = m.table.selectedColumnPlayerCount() - 1
		}
		return m, tea.Batch(m.tickEvery())

	case tea.WindowSizeMsg:
		m.windowSize = msg

	// Is it a key press?
	case tea.KeyMsg:
		keyMsg := msg.String()
		switch keyMsg {
		case "f":
			var cmd tea.Cmd
			if m.altScreen {
				cmd = tea.ExitAltScreen
			} else {
				cmd = tea.EnterAltScreen
			}
			m.altScreen = !m.altScreen
			return m, cmd
		default:
			if m.inConfig {
				return m.handleHelpInputs(keyMsg)
			}
			return m.handleDefaultInpuits(keyMsg)
		}

	case errMsg:
		m.err = msg
		return m, nil

	default:
		var cmd tea.Cmd
		m.loadingSpinner, cmd = m.loadingSpinner.Update(msg)
		return m, cmd
	}

	cmd := m.updateInputs(inMsg)
	return m, tea.Batch(cmd...)
}
func (m *appState) updateInputs(msg tea.Msg) []tea.Cmd {
	cmds := make([]tea.Cmd, 2)

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	m.configView.inputAddr, cmds[0] = m.configView.inputAddr.Update(msg)
	m.configView.passwordAddr, cmds[1] = m.configView.passwordAddr.Update(msg)

	return cmds
}
func (m appState) title() string {
	return styles.Title.Width(m.windowSize.Width / 2).Render(fmt.Sprintf("c: %d r: %d", m.table.selectedTeam, m.table.selectedRow))
}

func (m appState) status() string {
	if m.err != nil {
		return styles.Title.Render(m.err.Error())
	}
	return styles.Status.Width(m.windowSize.Width / 2).Render(m.statusMsg)
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
	return "\n" + m.table.View()
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

		dump, errDump := fetchPlayerState(context.Background(), m.config.Address, m.config.Password)
		if errDump != nil {
			m.messages = append(m.messages, "fatal:", errDump.Error())

			return TickMsg{err: errDump, t: t}
		}

		return TickMsg{t: t, dump: dump}
	})
}

func newAppState(client *ClientWithResponses, config Config, doSetup bool, scripting *Scripting) *appState {
	return &appState{
		api:            client,
		altScreen:      config.FullScreen,
		config:         config,
		helpView:       help.New(),
		scripting:      scripting,
		table:          newTableModel(),
		inConfig:       doSetup,
		banTable:       NewBanTableModel(),
		loadingSpinner: newSpinner(),
		configView:     newWidgetConfig(defaultConfig),
		keymap: keymap{
			reset: key.NewBinding(
				key.WithKeys("r"),
				key.WithHelp("r", "reset"),
			),
			quit: key.NewBinding(
				key.WithKeys("q", "ctrl+c"),
				key.WithHelp("q", "Quit"),
			),
			config: key.NewBinding(
				key.WithKeys("E"),
				key.WithHelp("E", "Conf"),
			),
			up: key.NewBinding(
				key.WithKeys("up", "k"),
				key.WithHelp("↑", "up"),
			),
			down: key.NewBinding(
				key.WithKeys("down", "j"),
				key.WithHelp("↓", "down"),
			),
			left: key.NewBinding(
				key.WithKeys("left", "h"),
				key.WithHelp("←", "Red Team"),
			),
			right: key.NewBinding(
				key.WithKeys("right", "l"),
				key.WithHelp("→", "Blu Team"),
			),
			fs: key.NewBinding(
				key.WithKeys("f"),
				key.WithHelp("f", "Toggle View"),
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
		defer func(f *os.File) {
			if errClose := f.Close(); errClose != nil {
				fmt.Println("error:", errClose.Error())
			}
		}(f)
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

	scripting, errScripting := NewScripting()
	if errScripting != nil {
		fmt.Println("fatal:", errScripting.Error())
		os.Exit(1)
	}

	//if errScripts := scripting.LoadDir("scripts"); errScripts != nil {
	//	fmt.Println("fatal:", errScripts.Error())
	//	os.Exit(1)
	//}

	var opts []tea.ProgramOption
	if config.FullScreen {
		opts = append(opts, tea.WithAltScreen())
	}

	program := tea.NewProgram(newAppState(client, config, !exists, scripting), opts...)
	if _, err := program.Run(); err != nil {
		fmt.Printf("There's been an error :( %v", err)
		os.Exit(1)
	}
}
