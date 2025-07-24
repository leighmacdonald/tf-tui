package main

import (
	"errors"
	"os"
	"path"
	"time"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/styles"
	"gopkg.in/yaml.v3"
)

var (
	errConfigWrite = errors.New("failed to write config file")
	errInvalidPath = errors.New("invalid path")
)

type Config struct {
	Address        string     `yaml:"address"`
	Password       string     `yaml:"password"`
	ConsoleLogPath string     `yaml:"console_log_path"`
	APIBaseURL     string     `yaml:"api_base_url,omitempty"`
	BDLists        []UserList `yaml:"bd_lists"`
}

type UserList struct {
	URL  string `yaml:"url"`
	Name string `yaml:"name"`
}

var defaultConfig = Config{
	Address:        "127.0.0.1:27015",
	Password:       "test",
	ConsoleLogPath: "",
}

const (
	defaultConfigName  = "tf-tui.yaml"
	defaultHTTPTimeout = 15 * time.Second
)

type keymap struct {
	start    key.Binding
	stop     key.Binding
	reset    key.Binding
	quit     key.Binding
	config   key.Binding
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
}

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
}

func configPath(name string) string {
	fullPath, errFullPath := xdg.ConfigFile(path.Join("tf-tui", name))
	if errFullPath != nil {
		panic(errFullPath)
	}

	return fullPath
}

func configRead(name string) (Config, bool) {
	var config Config
	inFile, errOpen := os.Open(configPath(name))
	if errOpen != nil {
		return defaultConfig, false
	}
	defer inFile.Close()

	if err := yaml.NewDecoder(inFile).Decode(&config); err != nil {
		return Config{}, false
	}

	if config.APIBaseURL == "" {
		config.APIBaseURL = "http://localhost:8888/"
	}

	return config, true
}

func configWrite(name string, config Config) error {
	outFile, errOpen := os.Create(configPath(name))
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
	//picker.AllowedTypes = []string{"console.log"}
	picker.CurrentDirectory, _ = os.UserHomeDir()
	picker.CurrentDirectory = path.Join(homedir, ".steam/steam/steamapps/common/Team Fortress 2/tf")
	picker.ShowPermissions = true
	picker.ShowHidden = true
	picker.ShowSize = true

	return picker
}

type configModel struct {
	inputAddr    textinput.Model
	passwordAddr textinput.Model
	fileSelect   tea.Model
	focusIndex   configIdx
	config       Config
	activeView   contentView
	selectedFile string
	width        int
	height       int
}

func NewConfigModal(config Config) tea.Model {
	return &configModel{
		config:       config,
		inputAddr:    NewTextInputModel(config.Address, "127.0.0.1:27015"),
		passwordAddr: NewTextInputPasswordModel(config.Password, ""),
		fileSelect:   NewFileSelect(),
		activeView:   viewConfig,
		selectedFile: "",
	}
}

func (m configModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.inputAddr.Focus(), m.fileSelect.Init())
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
	m.fileSelect, cmds[2] = m.fileSelect.Update(msg)

	switch msg := msg.(type) {
	case SetViewMsg:
		m.activeView = msg.view
	case SelectedFileMsg:
		m.selectedFile = msg.filePath
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
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
				return m, func() tea.Msg {
					return SetViewMsg{view: viewConfigFiles}
				}
			case fieldSave:
				cfg := m.config
				cfg.Address = m.inputAddr.Value()
				cfg.Password = m.passwordAddr.Value()
				cfg.ConsoleLogPath = m.selectedFile

				if err := configWrite(defaultConfigName, cfg); err != nil {
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

		case fieldPassword:
			cmds = append(cmds, m.passwordAddr.Focus())
			m.passwordAddr.PromptStyle = styles.FocusedStyle
			m.passwordAddr.TextStyle = styles.FocusedStyle

			m.inputAddr.Blur()
			m.inputAddr.PromptStyle = styles.NoStyle
			m.inputAddr.TextStyle = styles.NoStyle
		case fieldConsoleLogPath:
			m.passwordAddr.Blur()
			m.passwordAddr.PromptStyle = styles.NoStyle
			m.passwordAddr.TextStyle = styles.NoStyle
			m.inputAddr.Blur()
			m.inputAddr.PromptStyle = styles.NoStyle
			m.inputAddr.TextStyle = styles.NoStyle

		case fieldSave:
			m.passwordAddr.Blur()
			m.passwordAddr.PromptStyle = styles.NoStyle
			m.passwordAddr.TextStyle = styles.NoStyle
			m.inputAddr.Blur()
			m.inputAddr.PromptStyle = styles.NoStyle
			m.inputAddr.TextStyle = styles.NoStyle
		}
	}

	return m, tea.Batch(cmds...)
}

func (m configModel) View() string {
	if m.activeView == viewConfigFiles {
		return m.fileSelect.View()
	} else {
		return m.renderConfig()
	}
}

func (m configModel) renderConfig() string {
	var fields []string
	fields = append(fields,
		lipgloss.JoinHorizontal(lipgloss.Top,
			styles.HelpStyle.Render("RCON Address:  "), m.inputAddr.View()))

	fields = append(fields, lipgloss.JoinHorizontal(lipgloss.Top, styles.HelpStyle.Render("RCON Password: "), m.passwordAddr.View()))

	if m.focusIndex == fieldConsoleLogPath {
		fields = append(fields, lipgloss.JoinHorizontal(lipgloss.Top, styles.FocusedStyle.Render("console.log: "), styles.FocusedStyle.Render(m.selectedFile)))
	} else {
		fields = append(fields, lipgloss.JoinHorizontal(lipgloss.Top, styles.HelpStyle.Render("console.log: "), m.selectedFile))
	}

	if m.focusIndex == fieldSave {
		fields = append(fields, styles.FocusedSubmitButton)
	} else {
		fields = append(fields, styles.BlurredSubmitButton)
	}

	return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(lipgloss.JoinVertical(lipgloss.Top, fields...))
}
