package pages

import (
	"os"
	"path"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/ui/command"
	"github.com/leighmacdonald/tf-tui/internal/ui/component"
	"github.com/leighmacdonald/tf-tui/internal/ui/input"
	"github.com/leighmacdonald/tf-tui/internal/ui/model"
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
)

type configIdx int

const (
	fieldSteamID configIdx = iota
	fieldConsoleLogPath
	fieldTFAPIBaseURL
	fieldSave
)

type Config struct {
	fields      []*component.ValidatingTextInputModel
	focusIndex  configIdx
	config      config.Config
	viewState   model.ViewState
	loader      config.Writer
	inputActive bool
}

func NewConfig(config config.Config, loader config.Writer) *Config {
	homedir, err := os.UserHomeDir()
	if err != nil {
		homedir = "/"
	}

	logPath := path.Join(homedir, ".steam/steam/steamapps/common/Team Fortress 2/tf")
	if config.ConsoleLogPath == "" {
		config.ConsoleLogPath = logPath
	}

	return &Config{
		config: config,
		fields: []*component.ValidatingTextInputModel{
			component.NewValidatingTextInputModel("Steam ID", config.SteamID.String(), "", component.SteamIDValidator{}),
			component.NewValidatingTextInputModel("Path to console.log", config.ConsoleLogPath, logPath, component.PathValidator{}),
			component.NewValidatingTextInputModel("TF-API Base URL", config.APIBaseURL, "", component.URLValidator{}),
		},
		focusIndex: fieldSteamID,
		loader:     loader,
	}
}

func (m *Config) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, func() tea.Msg {
		return m.config
	})
}

func (m *Config) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 3)

	m.fields[fieldConsoleLogPath], cmds[0] = m.fields[fieldConsoleLogPath].Update(msg)
	m.fields[fieldSteamID], cmds[1] = m.fields[fieldSteamID].Update(msg)
	m.fields[fieldTFAPIBaseURL], cmds[2] = m.fields[fieldTFAPIBaseURL].Update(msg)

	switch msg := msg.(type) {
	case model.ViewState:
		m.viewState = msg
	case tea.KeyMsg:
		if m.viewState.Page != model.PageConfig {
			break
		}
		switch {
		case key.Matches(msg, input.Default.Back):
			// go back to main view
			if m.viewState.Page == model.PageConfig {
				m.viewState.Page = model.PageMain
				cmds = append(cmds, command.SetViewState(m.viewState)) //nolint:makezero
			}
		case key.Matches(msg, input.Default.Up):
			if m.focusIndex > 0 && m.focusIndex <= fieldSave {
				cmds = append(cmds, m.changeInput(input.Up)) //nolint:makezero
			}
		case key.Matches(msg, input.Default.Down):
			if m.focusIndex >= 0 && m.focusIndex < fieldSave {
				cmds = append(cmds, m.changeInput(input.Down)) //nolint:makezero
			}
		case key.Matches(msg, input.Default.Accept):
			switch m.focusIndex {
			case fieldSteamID:
				fallthrough
			case fieldConsoleLogPath:
				fallthrough
			case fieldTFAPIBaseURL:
				cmds = append(cmds, m.changeInput(input.Down)) //nolint:makezero
			case fieldSave:
				for _, field := range m.fields {
					if field.Input.Err != nil {
						return m, command.SetStatusMessage("Config is not valid, cannot save", true)
					}
				}

				cfg := m.config
				cfg.SteamID = steamid.New(m.fields[fieldSteamID].Input.Value())
				cfg.ConsoleLogPath = m.fields[fieldConsoleLogPath].Input.Value()
				cfg.APIBaseURL = m.fields[fieldTFAPIBaseURL].Input.Value()

				if err := m.loader.Write(cfg); err != nil {
					return m, command.SetStatusMessage(err.Error(), true)
				}

				m.config = cfg

				m.viewState.Page = model.PageMain

				return m, tea.Batch(
					command.SetConfig(cfg),
					command.SetStatusMessage("Saved config", false),
					command.SetViewState(m.viewState))
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Config) changeInput(dir input.Direction) tea.Cmd {
	switch dir { //nolint:exhaustive
	case input.Up:
		m.focusIndex--
	case input.Down:
		m.focusIndex++
	default:
		return nil
	}

	var cmd tea.Cmd
	for i := range m.fields {
		if configIdx(i) == m.focusIndex {
			cmd = m.fields[i].Focus()
		} else {
			m.fields[i].Blur()
		}
	}

	return cmd
}

func (m *Config) View() string {
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

	return lipgloss.NewStyle().Width(m.viewState.Width).Align(lipgloss.Left).
		Render(lipgloss.JoinVertical(lipgloss.Top, fields...))
}
