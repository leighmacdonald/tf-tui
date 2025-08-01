package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/tf-tui/styles"
)

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
