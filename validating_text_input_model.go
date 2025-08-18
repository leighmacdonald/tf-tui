package main

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/styles"
)

var (
	errAddressInvalid = errors.New("invalid address")
	errFilePath       = errors.New("invalid file path")
	errSteamIDInvalid = errors.New("invalid steamid")
	errInvalidURL     = errors.New("invalid URL")
)

type InputValidator interface {
	Validate(string) error
}

func NewValidatingTextInputModel(label string, value string, placeholder string, validators ...InputValidator) *ValidatingTextInputModel {
	input := NewTextInputModel(value, placeholder)

	if len(validators) > 0 {
		input.Validate = func(s string) error {
			for _, validator := range validators {
				if err := validator.Validate(s); err != nil {
					return err //nolint:wrapcheck
				}
			}

			return nil
		}
	}

	return &ValidatingTextInputModel{input: input, active: false, label: label}
}

type ValidatingTextInputModel struct {
	label  string
	input  textinput.Model
	active bool
}

func (m *ValidatingTextInputModel) Init() tea.Cmd {
	return nil
}

func (m *ValidatingTextInputModel) Update(msg tea.Msg) (*ValidatingTextInputModel, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	return m, cmd
}

func (m *ValidatingTextInputModel) View() string {
	var errRow string
	if m.input.Err != nil {
		errRow = lipgloss.NewStyle().Foreground(styles.Red).Render("Validation Error: " + m.input.Err.Error())
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		styles.HelpStyle.Render(m.label+": "),
		lipgloss.JoinVertical(lipgloss.Top, m.input.View(), errRow))
}

func (m *ValidatingTextInputModel) focus() tea.Cmd {
	m.input.PromptStyle = styles.FocusedStyle
	m.input.TextStyle = styles.FocusedStyle

	return m.input.Focus()
}

func (m *ValidatingTextInputModel) blur() {
	m.input.PromptStyle = styles.NoStyle
	m.input.TextStyle = styles.NoStyle
	m.input.Blur()
}

type URLValidator struct {
	emptyOk bool
}

func (v URLValidator) Validate(value string) error {
	if value == "" {
		if v.emptyOk {
			return nil
		}

		return errInvalidURL
	}

	_, errParse := url.Parse(value)
	if errParse != nil {
		return errors.Join(errParse, errInvalidURL)
	}

	return nil
}

type PathValidator struct{}

func (v PathValidator) Validate(value string) error {
	if value == "" {
		return fmt.Errorf("%w: Cannot be empty", errFilePath)
	}

	if _, err := os.Stat(value); err != nil {
		return fmt.Errorf("%w: Invalid log path", errors.Join(err, errConfigValue))
	}

	return nil
}

type SteamIDValidator struct{}

func (v SteamIDValidator) Validate(value string) error {
	steamID := steamid.New(value)
	if !steamID.Valid() {
		return errSteamIDInvalid
	}

	return nil
}

type AddressValidator struct{}

func (v AddressValidator) Validate(value string) error {
	if value == "" {
		return fmt.Errorf("%w: Cannot be empty", errAddressInvalid)
	}

	_, port, err := net.SplitHostPort(value)
	if err != nil {
		return fmt.Errorf("%w: Invalid address", errors.Join(err, errConfigValue))
	}

	portValue, errParse := strconv.ParseUint(port, 10, 16)
	if errParse != nil {
		return errors.Join(errParse, errAddressInvalid)
	}
	if portValue == 0 {
		return fmt.Errorf("%w: port cannot be 0", errAddressInvalid)
	}

	return nil
}
