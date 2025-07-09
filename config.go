package main

import (
	"errors"
	"os"
	"path"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/help"
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

const defaultConfigName = "tf-tui.yaml"

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

type filePickerModel struct {
	filepicker   filepicker.Model
	selectedFile string
	quitting     bool
	err          error
}

func newPickerModel() filePickerModel {
	fp := filepicker.New()
	//fp.AllowedTypes = []string{".log"}
	fp.CurrentDirectory, _ = os.UserHomeDir()

	return filePickerModel{filepicker: fp}
}

func (m filePickerModel) Init() tea.Cmd {
	return m.filepicker.Init()
}

func (m filePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}
	case clearErrorMsg:
		m.err = nil
	}

	var cmd tea.Cmd
	m.filepicker, cmd = m.filepicker.Update(msg)

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
		m.selectedFile = ""
		return m, tea.Batch(cmd, clearErrorAfter(2*time.Second))
	}

	return m, cmd
}

func (m filePickerModel) View() string {
	if m.quitting {
		return ""
	}
	var s strings.Builder
	s.WriteString("\n  ")
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

type widgetConfig struct {
	config         Config
	inputAddr      textinput.Model
	passwordAddr   textinput.Model
	consoleLogPath filePickerModel
	pickerActive   bool
	focusIndex     int
}

func newWidgetConfig(config Config) widgetConfig {
	address := config.Address
	if address == "" {
		address = "127.0.0.1:27015"
	}
	return widgetConfig{
		config:         config,
		pickerActive:   false,
		inputAddr:      newTextInputModel(address, "127.0.0.1:27015"),
		passwordAddr:   newTextInputPasswordModel(config.Password, ""),
		consoleLogPath: newPickerModel(),
	}
}

type helpKeymap struct {
	up   key.Binding
	down key.Binding
	esc  key.Binding
}

func newHelpKeymap() helpKeymap {
	return helpKeymap{
		up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("up", "Move up")),
		down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("down", "Move down")),
		esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "Go back"),
		),
	}
}

func (w widgetConfig) Init() tea.Cmd {
	return tea.Batch(w.consoleLogPath.Init(), textinput.Blink, w.inputAddr.Focus())
}

func (w widgetConfig) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if w.pickerActive {
		return w.consoleLogPath.Update(msg)
	}

	return w, nil
}

func (w widgetConfig) View() string {
	keyMaps := newHelpKeymap()
	var b strings.Builder
	if w.pickerActive {
		b.WriteString(w.consoleLogPath.View())
	} else {
		b.WriteString(styles.HelpStyle.Render("\n🟥 RCON Address:  "))
		b.WriteString(w.inputAddr.View() + "\n")
		b.WriteString(styles.HelpStyle.Render("🟩 RCON Password: "))
		b.WriteString(w.passwordAddr.View())
	}
	if w.focusIndex == 2 {
		b.WriteString("\n\n" + styles.FocusedSubmitButton)
	} else {
		b.WriteString("\n\n" + styles.BlurredSubmitButton)
	}

	helpView := help.New()

	b.WriteString("\n\n" + helpView.ShortHelpView([]key.Binding{
		keyMaps.up,
		keyMaps.down,
		keyMaps.esc,
	}))

	return b.String()
}
