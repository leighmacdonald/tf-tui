package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leighmacdonald/tf-tui/styles"
	"gopkg.in/yaml.v3"
)

var (
	errConfigWrite = errors.New("failed to write config file")
	errInvalidPath = errors.New("invalid path")
)

type Config struct {
	Address        string `yaml:"address"`
	Password       string `yaml:"password"`
	ConsoleLogPath string `yaml:"console_log_path"`
	APIBaseURL     string `yaml:"api_base_url,omitempty"`
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

func newPicker() filepicker.Model {
	picker := filepicker.New()
	picker.AllowedTypes = []string{}
	picker.CurrentDirectory, _ = os.UserHomeDir()
	picker.CurrentDirectory = "."
	picker.ShowPermissions = true
	picker.ShowHidden = true
	picker.ShowSize = true

	return picker
}

type configModel struct {
	filepicker   filepicker.Model
	selectedFile string
	inputAddr    textinput.Model
	passwordAddr textinput.Model
	focusIndex   configIdx
	config       Config
	activeView   contentView
}

func NewConfigModal(config Config) tea.Model {
	return &configModel{
		config:       config,
		inputAddr:    newTextInputModel(config.Address, "127.0.0.1:27015"),
		passwordAddr: newTextInputPasswordModel(config.Password, ""),
		filepicker:   newPicker(),
		selectedFile: config.ConsoleLogPath,
	}
}

func (m configModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.inputAddr.Focus(), m.filepicker.Init())
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
	m.filepicker, cmds[2] = m.filepicker.Update(msg)

	// Did the user select a file?
	if didSelect, selectedPath := m.filepicker.DidSelectFile(msg); didSelect {
		// Get the selectedPath of the selected file.
		m.selectedFile = selectedPath
		m.config.ConsoleLogPath = selectedPath
	}

	// Did the user select a disabled file?
	// This is only necessary to display an error to the user.
	if didSelect, selectedPath := m.filepicker.DidSelectDisabledFile(msg); didSelect {
		// Let's clear the selectedFile and display an error.
		// m.selectedFile = ""
		cmds = append(cmds, clearErrorAfter(10*time.Second), func() tea.Msg {
			return StatusMsg{
				message: fmt.Errorf("%w: Invalid selected file: %s", errInvalidPath, selectedPath).Error(),
				error:   true,
			}
		})
	}

	switch msg := msg.(type) {
	case SetViewMsg:
		m.activeView = msg.view
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.up):
			if m.focusIndex > 0 && m.focusIndex <= 2 {
				m.focusIndex--
			}
		case key.Matches(msg, DefaultKeyMap.down):
			if m.focusIndex >= 0 && m.focusIndex < 2 {
				m.focusIndex++
			}
		case key.Matches(msg, DefaultKeyMap.accept):
			switch m.focusIndex {
			case fieldAddress:
				m.focusIndex++
			case fieldPassword:
				m.focusIndex++
			case fieldConsoleLogPath:
				return m, func() tea.Msg {
					return SetViewMsg{view: viewConfigFiles}
				}
			case fieldSave:
				cfg := m.config
				cfg.Address = m.inputAddr.Value()
				cfg.Password = m.passwordAddr.Value()

				if err := configWrite(defaultConfigName, cfg); err != nil {
					return m, func() tea.Msg { return StatusMsg{message: err.Error(), error: true} }
				}

				m.config = cfg

				return m, tea.Batch(
					func() tea.Msg { return cfg },
					func() tea.Msg { return StatusMsg{message: "Saved config"} })
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
	var builder strings.Builder
	if m.activeView == viewConfigFiles {
		builder.WriteString(fmt.Sprintf("\n  Dir: %s \n ", m.filepicker.CurrentDirectory))
		if m.selectedFile == "" {
			builder.WriteString("Pick a file:")
		} else {
			builder.WriteString("Selected file: " + m.filepicker.Styles.Selected.Render(m.selectedFile))
		}
		builder.WriteString("\n\n" + m.filepicker.View() + "\n")
	} else {
		builder.WriteString(styles.HelpStyle.Render("\n🟥 RCON Address:  "))
		builder.WriteString(m.inputAddr.View())
		builder.WriteString(styles.HelpStyle.Render("\n🟩 RCON Password: "))
		builder.WriteString(m.passwordAddr.View())
		builder.WriteString(styles.HelpStyle.Render("\n🟩 console.log: "))
		if m.focusIndex == fieldConsoleLogPath {
			builder.WriteString(styles.FocusedStyle.Render(m.selectedFile))
		} else {
			builder.WriteString(m.selectedFile)
		}
	}
	if m.focusIndex == fieldSave {
		builder.WriteString("\n\n" + styles.FocusedSubmitButton)
	} else {
		builder.WriteString("\n\n" + styles.BlurredSubmitButton)
	}

	return builder.String()
}
