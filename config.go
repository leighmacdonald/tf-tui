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

type Config struct {
	Address        string `yaml:"address"`
	Password       string `yaml:"password"`
	ConsoleLogPath string `yaml:"console_log_path"`
	FullScreen     bool   `yaml:"full_screen"`
}

var defaultConfig = Config{
	Address:        "127.0.0.1:27015",
	Password:       "test",
	ConsoleLogPath: "",
	FullScreen:     true,
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
	fs       key.Binding
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
	fs: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "Toggle View"),
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

	return config, true
}

func configWrite(name string, config Config) error {
	outFile, errOpen := os.Create(configPath(defaultConfigName))
	if errOpen != nil {
		return errOpen
	}

	defer outFile.Close()

	return yaml.NewEncoder(outFile).Encode(&config)
}

func newPicker() filepicker.Model {
	fp := filepicker.New()
	fp.AllowedTypes = []string{}
	fp.CurrentDirectory, _ = os.UserHomeDir()
	fp.CurrentDirectory = "."
	fp.ShowPermissions = true
	fp.ShowHidden = true
	fp.ShowSize = true

	return fp
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

func newConfigModal(config Config) tea.Model {
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
		//m.selectedFile = ""
		cmds = append(cmds, clearErrorAfter(10*time.Second), func() tea.Msg {
			return StatusMsg{
				message: errors.New(selectedPath + " is not valid.").Error(),
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
	var b strings.Builder
	if m.activeView == viewConfigFiles {
		b.WriteString(fmt.Sprintf("\n  Dir: %s \n ", m.filepicker.CurrentDirectory))
		if m.selectedFile == "" {
			b.WriteString("Pick a file:")
		} else {
			b.WriteString("Selected file: " + m.filepicker.Styles.Selected.Render(m.selectedFile))
		}
		b.WriteString("\n\n" + m.filepicker.View() + "\n")
	} else {
		b.WriteString(styles.HelpStyle.Render("\n🟥 RCON Address:  "))
		b.WriteString(m.inputAddr.View())
		b.WriteString(styles.HelpStyle.Render("\n🟩 RCON Password: "))
		b.WriteString(m.passwordAddr.View())
		b.WriteString(styles.HelpStyle.Render("\n🟩 console.log: "))
		if m.focusIndex == fieldConsoleLogPath {
			b.WriteString(styles.FocusedStyle.Render(m.selectedFile))
		} else {
			b.WriteString(m.selectedFile)
		}
	}
	if m.focusIndex == fieldSave {
		b.WriteString("\n\n" + styles.FocusedSubmitButton)
	} else {
		b.WriteString("\n\n" + styles.BlurredSubmitButton)
	}

	return b.String()
}
