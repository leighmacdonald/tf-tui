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
	fp.ShowPermissions = true
	fp.ShowHidden = true
	fp.ShowSize = true

	return fp
}

type configModal struct {
	filepicker   filepicker.Model
	selectedFile string
	inputAddr    textinput.Model
	passwordAddr textinput.Model
	pickerActive bool
	focusIndex   configIdx
	config       Config
	err          error
	statusMsg    string
}

func newConfigModal(config Config) tea.Model {
	return configModal{
		pickerActive: true,
		config:       config,
		inputAddr:    newTextInputModel(config.Address, "127.0.0.1:27015"),
		passwordAddr: newTextInputPasswordModel(config.Password, ""),
		filepicker:   newPicker(),
	}
}

func (m configModal) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.inputAddr.Focus())
}

type configIdx int

const (
	fieldAddress configIdx = iota
	fieldPassword
	fieldSave
)

func (m configModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg {
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
		case fieldAddress:
			m.focusIndex++
		case fieldPassword:
			m.focusIndex++
		case fieldSave:
			cfg := m.config
			cfg.Address = m.inputAddr.Value()
			cfg.Password = m.passwordAddr.Value()

			if err := configWrite(defaultConfigName, cfg); err != nil {
				m.err = err
				return m, nil
			}

			m.statusMsg = "Saved config"
			m.config = cfg
			return m, tea.Batch(func() tea.Msg {
				return cfg
			})
		}
	}

	cmds := make([]tea.Cmd, 2)

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	m.inputAddr, cmds[0] = m.inputAddr.Update(msg)
	m.passwordAddr, cmds[1] = m.passwordAddr.Update(msg)

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

	var cmd tea.Cmd
	m.filepicker, cmd = m.filepicker.Update(msg)
	cmds = append(cmds, cmd)

	// Did the user select a file?
	if didSelect, selectedPath := m.filepicker.DidSelectFile(msg); didSelect {
		// Get the selectedPath of the selected file.
		m.selectedFile = selectedPath
	}

	// Did the user select a disabled file?
	// This is only necessary to display an error to the user.
	if didSelect, selectedPath := m.filepicker.DidSelectDisabledFile(msg); didSelect {
		// Let's clear the selectedFile and display an error.
		m.err = errors.New(selectedPath + " is not valid.")
		//m.selectedFile = ""
		cmds = append(cmds, clearErrorAfter(10*time.Second))
	}

	return m, tea.Batch(cmds...)
}

func (m configModal) View() string {
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

	//helpView := help.New()

	//b.WriteString("\n\n" + helpView.ShortHelpView([]key.Binding{
	//	m.keymap.up,
	//	m.keymap.down,
	//	m.keymap.quit,
	//}))

	return b.String()
}

func (m configModal) renderFilePicker() string {
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
