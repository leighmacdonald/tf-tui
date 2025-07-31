package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"time"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/styles"
	"gopkg.in/yaml.v3"
)

var (
	errConfigWrite = errors.New("failed to write config file")
	errInvalidPath = errors.New("invalid path")
	errConfigValue = errors.New("failed to validate config")
)

const (
	configDirName      = "tf-tui"
	defaultConfigName  = "tf-tui.yaml"
	defaultDBName      = "tf-tui.db"
	defaultLogName     = "tf-tui.log"
	defaultHTTPTimeout = 15 * time.Second
)

type Config struct {
	Address        string          `yaml:"address"`
	Password       string          `yaml:"password"`
	ConsoleLogPath string          `yaml:"console_log_path"`
	APIBaseURL     string          `yaml:"api_base_url,omitempty"`
	BDLists        []UserList      `yaml:"bd_lists"`
	Links          []UserLink      `yaml:"links"`
	SteamID        steamid.SteamID `yaml:"steam_id"`
}

type SIDFormats string

const (
	Steam64 SIDFormats = "steam64"
	Steam2  SIDFormats = "steam"
	Steam3  SIDFormats = "steam3"
)

type UserLink struct {
	URL    string     `yaml:"url"`
	Name   string     `yaml:"name"`
	Format SIDFormats `yaml:"format"`
}

func (u UserLink) Generate(steamID steamid.SteamID) string {
	switch u.Format {
	case Steam2:
		return fmt.Sprintf(u.URL, steamID.Steam(false))
	case Steam3:
		return fmt.Sprintf(u.URL, steamID.Steam3())
	case Steam64:
		fallthrough
	default:
		return fmt.Sprintf(u.URL, steamID.String())
	}
}

type UserList struct {
	URL  string `yaml:"url"`
	Name string `yaml:"name"`
}

var defaultConfig = Config{
	Address:        "127.0.0.1:27015",
	Password:       "tf-tui",
	ConsoleLogPath: "",
}

type keymap struct {
	start    key.Binding
	stop     key.Binding
	reset    key.Binding
	quit     key.Binding
	config   key.Binding
	chat     key.Binding
	up       key.Binding
	down     key.Binding
	left     key.Binding
	right    key.Binding
	accept   key.Binding
	back     key.Binding
	nextTab  key.Binding
	overview key.Binding
	bans     key.Binding
	comp     key.Binding
	notes    key.Binding
	console  key.Binding
}

// TODO make configurable.
var DefaultKeyMap = keymap{
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
	overview: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "Overview"),
	),
	bans: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "Bans"),
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
		key.WithKeys("~"),
		key.WithHelp("~", "Console"),
	),
	chat: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "Chat"),
	),
}

// ConfigPath generates a path pointing to the filename under this apps defined $XDG_CONFIG_HOME.
func ConfigPath(name string) string {
	fullPath, errFullPath := xdg.ConfigFile(path.Join(configDirName, name))
	if errFullPath != nil {
		panic(errFullPath)
	}

	return fullPath
}

func ConfigRead(name string) (Config, bool) {
	var config Config
	if err := os.MkdirAll(path.Join(xdg.ConfigHome, configDirName), 0o600); err != nil {
		tea.Println("Failed to make config root: " + err.Error())

		return defaultConfig, false
	}

	inFile, errOpen := os.Open(ConfigPath(name))
	if errOpen != nil {
		return defaultConfig, false
	}
	defer inFile.Close()

	if err := yaml.NewDecoder(inFile).Decode(&config); err != nil {
		return Config{}, false
	}

	if config.APIBaseURL == "" {
		config.APIBaseURL = "https://tf-api.roto.lol/"
	}

	if config.ConsoleLogPath == "" {
		config.ConsoleLogPath = LinuxDefaultPath()
	}

	return config, true
}

func LinuxDefaultPath() string {
	homedir, err := os.UserHomeDir()
	if err != nil {
		homedir = "/"
	}
	return path.Join(homedir, ".steam/steam/steamapps/common/Team Fortress 2/tf/console.log")
}

func ConfigWrite(name string, config Config) error {
	outFile, errOpen := os.Create(ConfigPath(name))
	if errOpen != nil {
		return errors.Join(errOpen, errConfigWrite)
	}

	defer outFile.Close()

	if err := yaml.NewEncoder(outFile).Encode(&config); err != nil {
		return errors.Join(err, errConfigWrite)
	}

	return nil
}

func NewPicker() filepicker.Model {
	homedir, err := os.UserHomeDir()
	if err != nil {
		homedir = "/"
	}

	picker := filepicker.New()
	// picker.AllowedTypes = []string{"console.log"}
	picker.CurrentDirectory, _ = os.UserHomeDir()
	picker.CurrentDirectory = path.Join(homedir, ".steam/steam/steamapps/common/Team Fortress 2/tf")
	picker.ShowPermissions = true
	picker.ShowHidden = true
	picker.ShowSize = true

	return picker
}

type configModel struct {
	inputAddr      textinput.Model
	passwordAddr   textinput.Model
	consoleLogPath textinput.Model
	focusIndex     configIdx
	config         Config
	activeView     contentView
	width          int
	height         int
}

func NewConfigModal(config Config) tea.Model {
	homedir, err := os.UserHomeDir()
	if err != nil {
		homedir = "/"
	}
	logPath := path.Join(homedir, ".steam/steam/steamapps/common/Team Fortress 2/tf")

	if config.ConsoleLogPath == "" {
		config.ConsoleLogPath = logPath
	}
	return &configModel{
		config:         config,
		inputAddr:      NewTextInputModel(config.Address, "127.0.0.1:27015"),
		passwordAddr:   NewTextInputPasswordModel(config.Password, ""),
		consoleLogPath: NewTextInputModel(config.ConsoleLogPath, logPath),
		activeView:     viewConfig,
		focusIndex:     fieldAddress,
	}
}

func (m configModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.inputAddr.Focus())
}

type configIdx int

const (
	fieldAddress configIdx = iota
	fieldPassword
	fieldConsoleLogPath
	fieldSave
)

func (m configModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 3)

	m.inputAddr, cmds[0] = m.inputAddr.Update(msg)
	m.passwordAddr, cmds[1] = m.passwordAddr.Update(msg)
	m.consoleLogPath, cmds[2] = m.consoleLogPath.Update(msg)

	switch msg := msg.(type) {
	case SetViewMsg:
		m.activeView = msg.view
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if m.activeView != viewConfig {
			break
		}
		switch {
		case key.Matches(msg, DefaultKeyMap.up):
			if m.focusIndex > 0 && m.focusIndex <= 3 {
				m.focusIndex--
			}
		case key.Matches(msg, DefaultKeyMap.down):
			if m.focusIndex >= 0 && m.focusIndex < 3 {
				m.focusIndex++
			}
		case key.Matches(msg, DefaultKeyMap.accept):
			switch m.focusIndex {
			case fieldAddress:
				m.focusIndex++
			case fieldPassword:
				m.focusIndex++
			case fieldConsoleLogPath:
				m.focusIndex++
			case fieldSave:
				if err := m.validate(); err != nil {
					return m, func() tea.Msg { return StatusMsg{message: err.Error(), error: true} }
				}

				cfg := m.config
				cfg.Address = m.inputAddr.Value()
				cfg.Password = m.passwordAddr.Value()
				cfg.ConsoleLogPath = m.consoleLogPath.Value()

				if err := ConfigWrite(defaultConfigName, cfg); err != nil {
					return m, func() tea.Msg { return StatusMsg{message: err.Error(), error: true} }
				}

				m.config = cfg

				return m, tea.Batch(
					func() tea.Msg { return cfg },
					func() tea.Msg { return StatusMsg{message: "Saved config"} },
					func() tea.Msg { return SetViewMsg{view: viewPlayerTables} })
			}
		}

		switch m.focusIndex {
		case fieldAddress:
			cmds = append(cmds, m.inputAddr.Focus())
			m.inputAddr.PromptStyle = styles.FocusedStyle
			m.inputAddr.TextStyle = styles.FocusedStyle

			m.passwordAddr.Blur()
			m.passwordAddr.PromptStyle = styles.NoStyle
			m.passwordAddr.TextStyle = styles.NoStyle
			m.consoleLogPath.PromptStyle = styles.NoStyle
			m.consoleLogPath.TextStyle = styles.NoStyle
		case fieldPassword:
			cmds = append(cmds, m.passwordAddr.Focus())
			m.passwordAddr.PromptStyle = styles.FocusedStyle
			m.passwordAddr.TextStyle = styles.FocusedStyle

			m.inputAddr.Blur()
			m.inputAddr.PromptStyle = styles.NoStyle
			m.inputAddr.TextStyle = styles.NoStyle

			m.consoleLogPath.PromptStyle = styles.NoStyle
			m.consoleLogPath.TextStyle = styles.NoStyle
		case fieldConsoleLogPath:
			cmds = append(cmds, m.consoleLogPath.Focus())
			m.passwordAddr.Blur()
			m.passwordAddr.PromptStyle = styles.NoStyle
			m.passwordAddr.TextStyle = styles.NoStyle
			m.inputAddr.Blur()
			m.inputAddr.PromptStyle = styles.NoStyle
			m.inputAddr.TextStyle = styles.NoStyle
			m.consoleLogPath.PromptStyle = styles.FocusedStyle
			m.consoleLogPath.TextStyle = styles.FocusedStyle
		case fieldSave:
			m.passwordAddr.Blur()
			m.passwordAddr.PromptStyle = styles.NoStyle
			m.passwordAddr.TextStyle = styles.NoStyle
			m.inputAddr.Blur()
			m.inputAddr.PromptStyle = styles.NoStyle
			m.inputAddr.TextStyle = styles.NoStyle
			m.passwordAddr.Blur()
			m.consoleLogPath.PromptStyle = styles.NoStyle
			m.consoleLogPath.TextStyle = styles.NoStyle
		}
	}

	return m, tea.Batch(cmds...)
}

func (m configModel) validate() error {
	_, _, err := net.SplitHostPort(m.inputAddr.Value())
	if err != nil {
		return fmt.Errorf("%w: Invalid address", errors.Join(err, errConfigValue))
	}

	if _, err := os.Stat(m.consoleLogPath.Value()); err != nil {
		return fmt.Errorf("%w: Invalid log path", errors.Join(err, errConfigValue))
	}

	if len(m.passwordAddr.Value()) == 0 {
		return fmt.Errorf("%w: Invalid password", errConfigValue)
	}

	return nil
}

func (m configModel) View() string {
	return m.renderConfig()
}

func (m configModel) renderConfig() string {
	var fields []string
	fields = append(fields,
		lipgloss.JoinHorizontal(lipgloss.Top,
			styles.HelpStyle.Render("RCON Address:  "), m.inputAddr.View()))

	fields = append(fields, lipgloss.JoinHorizontal(lipgloss.Top, styles.HelpStyle.Render("RCON Password: "), m.passwordAddr.View()))
	fields = append(fields, lipgloss.JoinHorizontal(lipgloss.Top, styles.HelpStyle.Render("Path to console.log: "), m.consoleLogPath.View()))

	if m.focusIndex == fieldSave {
		fields = append(fields, styles.FocusedSubmitButton)
	} else {
		fields = append(fields, styles.BlurredSubmitButton)
	}

	return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(lipgloss.JoinVertical(lipgloss.Top, fields...))
}
