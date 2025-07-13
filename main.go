package main

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config config.yaml ./openapi.yaml

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
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

type PlayerStateMsg struct {
	err  error
	t    time.Time
	dump shared.PlayerState
}

type MetaProfileMsg struct {
	profiles []MetaProfile
	t        time.Time
	err      error
}

type AppModel struct {
	config       Config
	cache        *MetaCache
	api          ClientInterface
	altScreen    bool
	table        *tableModel
	keymap       keymap
	titleState   string
	quitting     bool
	err          errMsg
	dump         shared.PlayerState
	messages     []string
	windowSize   tea.WindowSizeMsg
	selectedTeam Team
	selectedRow  int
	inConfig     bool
	statusMsg    string
	scripting    *Scripting
	helpView     help.Model
	banTable     *BanTableModel
	filepicker   filepicker.Model
	selectedFile string
	inputAddr    textinput.Model
	passwordAddr textinput.Model
	pickerActive bool
	focusIndex   int
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(tea.SetWindowTitle("tf-tui"), m.tickEvery(), m.filepicker.Init(), textinput.Blink, m.inputAddr.Focus())
}

func (m AppModel) View() string {
	var b strings.Builder
	b.WriteString(m.renderHeading())
	if m.inConfig {
		b.WriteString(m.renderConfigView())
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

func (m AppModel) handleHelpInputs(msg string) (AppModel, tea.Cmd) {
	switch msg {
	case "ctrl+c", "esc":
		m.inConfig = false
		return m, nil
	case "up":
		if m.focusIndex > 0 && m.focusIndex <= 2 {
			m.focusIndex--
		}
	case "down":
		if m.focusIndex >= 0 && m.focusIndex < 2 {
			m.focusIndex++
		}
	case "enter":
		switch m.focusIndex {
		case 0:
			m.focusIndex++
		case 1:
			m.focusIndex++
		case 2:
			return m, tea.Batch(func() tea.Msg {
				return m.config
			})
		}
	}

	cmds := make([]tea.Cmd, 2)

	switch m.focusIndex {
	case 0:
		cmds = append(cmds, m.inputAddr.Focus())
		m.inputAddr.PromptStyle = styles.FocusedStyle
		m.inputAddr.TextStyle = styles.FocusedStyle

		m.passwordAddr.Blur()
		m.passwordAddr.PromptStyle = styles.NoStyle
		m.passwordAddr.TextStyle = styles.NoStyle
	case 1:
		cmds = append(cmds, m.passwordAddr.Focus())
		m.passwordAddr.PromptStyle = styles.FocusedStyle
		m.passwordAddr.TextStyle = styles.FocusedStyle

		m.inputAddr.Blur()
		m.inputAddr.PromptStyle = styles.NoStyle
		m.inputAddr.TextStyle = styles.NoStyle
	case 2:
		m.passwordAddr.Blur()
		m.passwordAddr.PromptStyle = styles.NoStyle
		m.passwordAddr.TextStyle = styles.NoStyle
		m.inputAddr.Blur()
		m.inputAddr.PromptStyle = styles.NoStyle
		m.inputAddr.TextStyle = styles.NoStyle
	}

	return m, tea.Batch(append(cmds, m.updateInputs(msg)...)...)
}

func (m AppModel) handleDefaultInpuits(msg string) (AppModel, tea.Cmd) {
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

func (m AppModel) onPlayerStateMsg(msg PlayerStateMsg) (tea.Model, tea.Cmd) {
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
}

func (m AppModel) onConfig(msg Config) (tea.Model, tea.Cmd) {
	if err := configWrite(defaultConfigName, msg); err != nil {
		m.err = err
		return m, nil
	}

	m.statusMsg = "Saved config"
	m.config = msg
	m.inConfig = false

	return m, nil
}

func (m AppModel) onWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.windowSize = msg
	return m, nil
}

func (m AppModel) onKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
}

func (m AppModel) onMetaProfileMsg(msg MetaProfileMsg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m AppModel) Update(inMsg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := inMsg.(type) {
	case Config:
		return m.onConfig(msg)
	case MetaProfileMsg:
		return m.onMetaProfileMsg(msg)
	case PlayerStateMsg:
		return m.onPlayerStateMsg(msg)
	case tea.WindowSizeMsg:
		return m.onWindowSizeMsg(msg)
	// Is it a key press?
	case tea.KeyMsg:
		return m.onKeyMsg(msg)
	case clearErrorMsg:
		m.err = nil
		return m, nil
	case errMsg:
		m.err = msg
		return m, nil
		//default:
		//	var cmd tea.Cmd
		//	m.loadingSpinner, cmd = m.loadingSpinner.Update(msg)
		//	return m, cmd
	}

	outCmd := m.updateInputs(inMsg)

	//var cfgCmd tea.Cmd
	//m.configView.consoleLogPath, cfgCmd = m.configView.consoleLogPath.Update(inMsg)

	var cmd tea.Cmd
	m.filepicker, cmd = m.filepicker.Update(inMsg)

	// Did the user select a file?
	if didSelect, selectedPath := m.filepicker.DidSelectFile(inMsg); didSelect {
		// Get the selectedPath of the selected file.
		m.selectedFile = selectedPath
	}

	// Did the user select a disabled file?
	// This is only necessary to display an error to the user.
	if didSelect, selectedPath := m.filepicker.DidSelectDisabledFile(inMsg); didSelect {
		// Let's clear the selectedFile and display an error.
		m.err = errors.New(selectedPath + " is not valid.")
		m.selectedFile = ""
		outCmd = append(outCmd, cmd, clearErrorAfter(2*time.Second))
	}

	return m, tea.Batch(outCmd...)
}

func (m *AppModel) updateInputs(msg tea.Msg) []tea.Cmd {
	cmds := make([]tea.Cmd, 2)

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	m.inputAddr, cmds[0] = m.inputAddr.Update(msg)
	m.passwordAddr, cmds[1] = m.passwordAddr.Update(msg)

	return cmds
}
func (m AppModel) title() string {
	return styles.Title.Width(m.windowSize.Width / 2).Render(fmt.Sprintf("c: %d r: %d", m.table.selectedTeam, m.table.selectedRow))
}

func (m AppModel) status() string {
	if m.err != nil {
		return styles.Title.Render(m.err.Error())
	}
	return styles.Status.Width(m.windowSize.Width / 2).Render(m.statusMsg)
}

func (m AppModel) renderHeading() string {
	out := lipgloss.JoinHorizontal(lipgloss.Top, m.title(), m.status())

	if m.quitting {
		return out + "\n"
	}

	return out
}
func (m AppModel) renderConfigView() string {
	var b strings.Builder
	if m.pickerActive {
		b.WriteString(m.renderFilePicker())
	} else {
		b.WriteString(styles.HelpStyle.Render("\n🟥 RCON Address:  "))
		b.WriteString(m.inputAddr.View() + "\n")
		b.WriteString(styles.HelpStyle.Render("🟩 RCON Password: "))
		b.WriteString(m.passwordAddr.View())
	}
	if m.focusIndex == 2 {
		b.WriteString("\n\n" + styles.FocusedSubmitButton)
	} else {
		b.WriteString("\n\n" + styles.BlurredSubmitButton)
	}

	helpView := help.New()

	b.WriteString("\n\n" + helpView.ShortHelpView([]key.Binding{
		m.keymap.up,
		m.keymap.down,
		m.keymap.quit,
	}))

	return b.String()
}

func (m AppModel) renderPlayerTables() string {
	m.table.dump = m.dump
	return "\n" + m.table.View()
}

func (m AppModel) renderFilePicker() string {
	if m.quitting {
		return ""
	}
	var s strings.Builder
	s.WriteString(fmt.Sprintf("\n  Dir: %s \n ", m.filepicker.CurrentDirectory))
	if m.err != nil {
		s.WriteString(m.filepicker.Styles.DisabledFile.Render(m.err.Error()))
	} else if m.selectedFile == "" {
		s.WriteString("Pick a file:")
	} else {
		s.WriteString("Selected file: " + m.filepicker.Styles.Selected.Render(m.selectedFile))
	}
	s.WriteString("\n\n" + m.filepicker.View() + "\n")
	return s.String()
}

func srt(rows [][]string) {
	slices.SortFunc(rows, func(a, b []string) int {
		av, _ := strconv.Atoi(a[1])
		bv, _ := strconv.Atoi(b[1])
		return cmp.Compare(bv, av)
	})
}

func (m AppModel) tickEvery() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		m.messages = append(m.messages, "updating dump")

		dump, errDump := fetchPlayerState(context.Background(), m.config.Address, m.config.Password)
		if errDump != nil {
			m.messages = append(m.messages, "fatal:", errDump.Error())

			return PlayerStateMsg{err: errDump, t: t}
		}

		return PlayerStateMsg{t: t, dump: dump}
	})
}

func newAppState(client *ClientWithResponses, config Config, doSetup bool, scripting *Scripting, cache *MetaCache) *AppModel {
	address := config.Address
	if address == "" {
		address = "127.0.0.1:27015"
	}
	return &AppModel{
		pickerActive: true,
		inputAddr:    newTextInputModel(address, "127.0.0.1:27015"),
		passwordAddr: newTextInputPasswordModel(config.Password, ""),
		api:          client,
		cache:        cache,
		altScreen:    config.FullScreen,
		config:       config,
		helpView:     help.New(),
		scripting:    scripting,
		table:        newTableModel(),
		inConfig:     doSetup,
		banTable:     NewBanTableModel(),
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
				key.WithHelp("↑", "Up"),
			),
			down: key.NewBinding(
				key.WithKeys("down", "j"),
				key.WithHelp("↓", "Down"),
			),
			left: key.NewBinding(
				key.WithKeys("left", "h"),
				key.WithHelp("←", "RED"),
			),
			right: key.NewBinding(
				key.WithKeys("right", "l"),
				key.WithHelp("→", "BLU"),
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

	program := tea.NewProgram(newAppState(client, config, !exists, scripting, NewMetaCache()), opts...)
	if _, err := program.Run(); err != nil {
		fmt.Printf("There's been an error :( %v", err)
		os.Exit(1)
	}
}
