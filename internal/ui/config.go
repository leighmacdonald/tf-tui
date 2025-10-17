package ui

import (
	"os"
	"path"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
)

type keymap struct {
	start         key.Binding
	stop          key.Binding
	reset         key.Binding
	quit          key.Binding
	config        key.Binding
	chat          key.Binding
	up            key.Binding
	down          key.Binding
	left          key.Binding
	right         key.Binding
	accept        key.Binding
	back          key.Binding
	prevTab       key.Binding
	nextTab       key.Binding
	overview      key.Binding
	bans          key.Binding
	bd            key.Binding
	comp          key.Binding
	notes         key.Binding
	console       key.Binding
	help          key.Binding
	consoleInput  key.Binding
	consoleCancel key.Binding
}

// TODO make configurable.
var defaultKeyMap = keymap{
	consoleInput: key.NewBinding(
		key.WithKeys("return"),
		key.WithHelp("<return>", "Send command")),
	consoleCancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("<esc>", "Cancel input")),
	help: key.NewBinding(
		key.WithKeys("h", "H"),
		key.WithHelp("h", "Help"),
	),
	accept: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "Select"),
	),
	back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "Back"),
	),
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
	nextTab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "Next Tab"),
	),
	prevTab: key.NewBinding(
		key.WithKeys("shift tab"),
		key.WithHelp("shift tab", "Prev Tab"),
	),
	overview: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "Overview"),
	),
	bans: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "Bans"),
	),
	bd: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "Bot Detector"),
	),
	comp: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "Comp"),
	),
	notes: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "Notes"),
	),
	console: key.NewBinding(
		key.WithKeys("`"),
		key.WithHelp("`", "Console"),
	),
	chat: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "Chat"),
	),
}

type configIdx int

const (
	fieldSteamID configIdx = iota
	fieldConsoleLogPath
	fieldTFAPIBaseURL
	fieldSave
)

type ConfigWriter interface {
	Write(config.Config) error
	Path() string
}

type configModel struct {
	fields     []*validatingTextInputModel
	focusIndex configIdx
	config     config.Config
	activeView contentView
	width      int
	height     int
	loader     ConfigWriter
}

func newConfigModal(config config.Config, loader ConfigWriter) tea.Model {
	homedir, err := os.UserHomeDir()
	if err != nil {
		homedir = "/"
	}

	logPath := path.Join(homedir, ".steam/steam/steamapps/common/Team Fortress 2/tf")
	if config.ConsoleLogPath == "" {
		config.ConsoleLogPath = logPath
	}

	return &configModel{
		config: config,
		fields: []*validatingTextInputModel{
			newValidatingTextInputModel("Steam ID", config.SteamID.String(), "", steamIDValidator{}),
			newValidatingTextInputModel("Path to console.log", config.ConsoleLogPath, logPath, pathValidator{}),
			newValidatingTextInputModel("TF-API Base URL", config.APIBaseURL, "", urlValidator{}),
		},
		activeView: viewConfig,
		focusIndex: fieldSteamID,
		loader:     loader,
	}
}

func (m *configModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, func() tea.Msg {
		return m.config
	})
}

func (m *configModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 3)

	m.fields[fieldConsoleLogPath], cmds[0] = m.fields[fieldConsoleLogPath].Update(msg)
	m.fields[fieldSteamID], cmds[1] = m.fields[fieldSteamID].Update(msg)
	m.fields[fieldTFAPIBaseURL], cmds[2] = m.fields[fieldTFAPIBaseURL].Update(msg)

	switch msg := msg.(type) {
	case contentView:
		m.activeView = msg
		if m.activeView == viewConfig {
			cmds = append(cmds, m.fields[fieldSteamID].focus()) //nolint:makezero
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if m.activeView != viewConfig {
			break
		}
		switch {
		case key.Matches(msg, defaultKeyMap.back):
			// go back to main view
			if m.activeView == viewConfig {
				m.activeView = viewMain
				cmds = append(cmds, setContentView(viewMain)) //nolint:makezero
			}
		case key.Matches(msg, defaultKeyMap.up):
			if m.focusIndex > 0 && m.focusIndex <= fieldSave {
				cmds = append(cmds, m.changeInput(up)) //nolint:makezero
			}
		case key.Matches(msg, defaultKeyMap.down):
			if m.focusIndex >= 0 && m.focusIndex < fieldSave {
				cmds = append(cmds, m.changeInput(down)) //nolint:makezero
			}
		case key.Matches(msg, defaultKeyMap.accept):
			switch m.focusIndex {
			case fieldSteamID:
				fallthrough
			case fieldConsoleLogPath:
				fallthrough
			case fieldTFAPIBaseURL:
				cmds = append(cmds, m.changeInput(down)) //nolint:makezero
			case fieldSave:
				for _, field := range m.fields {
					if field.input.Err != nil {
						return m, setStatusMessage("Config is not valid, cannot save", true)
					}
				}

				cfg := m.config
				cfg.SteamID = steamid.New(m.fields[fieldSteamID].input.Value())
				cfg.ConsoleLogPath = m.fields[fieldConsoleLogPath].input.Value()
				cfg.APIBaseURL = m.fields[fieldTFAPIBaseURL].input.Value()

				if err := m.loader.Write(cfg); err != nil {
					return m, setStatusMessage(err.Error(), true)
				}

				m.config = cfg

				return m, tea.Batch(
					setConfig(cfg),
					setStatusMessage("Saved config", false),
					setContentView(viewMain))
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *configModel) changeInput(direction direction) tea.Cmd {
	switch direction { //nolint:exhaustive
	case up:
		m.focusIndex--
	case down:
		m.focusIndex++
	default:
		return nil
	}

	var cmd tea.Cmd
	for i := range m.fields {
		if configIdx(i) == m.focusIndex {
			cmd = m.fields[i].focus()
		} else {
			m.fields[i].blur()
		}
	}

	return cmd
}

func (m *configModel) View() string {
	fields := []string{
		m.fields[fieldSteamID].View(),
		m.fields[fieldConsoleLogPath].View(),
		m.fields[fieldTFAPIBaseURL].View(),
	}

	if m.focusIndex == fieldSave {
		fields = append(fields, styles.FocusedSubmitButton)
	} else {
		fields = append(fields, styles.BlurredSubmitButton)
	}

	return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Left).
		Render(lipgloss.JoinVertical(lipgloss.Top, fields...))
}
