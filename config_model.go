package main

import (
	"os"
	"path"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/styles"
)

type configIdx int

const (
	fieldSteamID configIdx = iota
	fieldAddress
	fieldPassword
	fieldConsoleLogPath
	fieldTFAPIBaseURL
	fieldSave
)

type ConfigModel struct {
	fields     []*ValidatingTextInputModel
	focusIndex configIdx
	config     Config
	activeView contentView
	width      int
	height     int
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

	passInput := NewValidatingTextInputModel("RCON Password", config.Password, "")
	passInput.input.EchoMode = textinput.EchoPassword

	return &ConfigModel{
		config: config,
		fields: []*ValidatingTextInputModel{
			NewValidatingTextInputModel("Steam ID", config.SteamID.String(), "", SteamIDValidator{}),
			NewValidatingTextInputModel("RCON Address", config.Address, "127.0.0.1:27015", AddressValidator{}),
			passInput,
			NewValidatingTextInputModel("Path to console.log", config.ConsoleLogPath, logPath, PathValidator{}),
			NewValidatingTextInputModel("TF-API Base URL", config.APIBaseURL, "", URLValidator{}),
		},
		activeView: viewConfig,
		focusIndex: fieldSteamID,
	}
}

func (m *ConfigModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, func() tea.Msg {
		return m.config
	})
}

func (m *ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 5)

	m.fields[fieldAddress], cmds[0] = m.fields[fieldAddress].Update(msg)
	m.fields[fieldPassword], cmds[1] = m.fields[fieldPassword].Update(msg)
	m.fields[fieldConsoleLogPath], cmds[2] = m.fields[fieldConsoleLogPath].Update(msg)
	m.fields[fieldSteamID], cmds[3] = m.fields[fieldSteamID].Update(msg)
	m.fields[fieldTFAPIBaseURL], cmds[4] = m.fields[fieldTFAPIBaseURL].Update(msg)

	switch msg := msg.(type) {
	case SetViewMsg:
		m.activeView = msg.view
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
		case key.Matches(msg, DefaultKeyMap.up):
			if m.focusIndex > 0 && m.focusIndex <= fieldSave {
				cmds = append(cmds, m.changeInput(Up)) //nolint:makezero
			}
		case key.Matches(msg, DefaultKeyMap.down):
			if m.focusIndex >= 0 && m.focusIndex < fieldSave {
				cmds = append(cmds, m.changeInput(Down)) //nolint:makezero
			}
		case key.Matches(msg, DefaultKeyMap.accept):
			switch m.focusIndex {
			case fieldSteamID:
				fallthrough
			case fieldAddress:
				fallthrough
			case fieldPassword:
				fallthrough
			case fieldConsoleLogPath:
				fallthrough
			case fieldTFAPIBaseURL:
				cmds = append(cmds, m.changeInput(Down)) //nolint:makezero
			case fieldSave:
				for _, field := range m.fields {
					if field.input.Err != nil {
						return m, func() tea.Msg { return StatusMsg{message: "Config is not valid, cannot save", error: true} }
					}
				}

				cfg := m.config
				cfg.SteamID = steamid.New(m.fields[fieldSteamID].input.Value())
				cfg.Address = m.fields[fieldAddress].input.Value()
				cfg.Password = m.fields[fieldPassword].input.Value()
				cfg.ConsoleLogPath = m.fields[fieldConsoleLogPath].input.Value()
				cfg.APIBaseURL = m.fields[fieldTFAPIBaseURL].input.Value()

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
	}

	return m, tea.Batch(cmds...)
}

func (m *ConfigModel) changeInput(direction Direction) tea.Cmd {
	switch direction { //nolint:exhaustive
	case Up:
		m.focusIndex--
	case Down:
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

func (m *ConfigModel) View() string {
	fields := []string{
		m.fields[fieldSteamID].View(),
		m.fields[fieldAddress].View(),
		m.fields[fieldPassword].View(),
		m.fields[fieldConsoleLogPath].View(),
		m.fields[fieldTFAPIBaseURL].View(),
	}

	if m.focusIndex == fieldSave {
		fields = append(fields, styles.FocusedSubmitButton)
	} else {
		fields = append(fields, styles.BlurredSubmitButton)
	}

	return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).
		Render(lipgloss.JoinVertical(lipgloss.Top, fields...))
}
