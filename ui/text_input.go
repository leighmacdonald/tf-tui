package ui

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
	"github.com/leighmacdonald/tf-tui/ui/styles"
)

var (
	errAddressInvalid = errors.New("invalid address")
	errFilePath       = errors.New("invalid file path")
	errSteamIDInvalid = errors.New("invalid steamid")
	errInvalidURL     = errors.New("invalid URL")
	errConfigValue    = errors.New("failed to validate config")
)

type inputValidator interface {
	validate(string) error
}

func newValidatingTextInputModel(label string, value string, placeholder string, validators ...inputValidator) *validatingTextInputModel {
	input := newTextInputModel(value, placeholder)

	if len(validators) > 0 {
		input.Validate = func(s string) error {
			for _, validator := range validators {
				if err := validator.validate(s); err != nil {
					return err
				}
			}

			return nil
		}
	}

	return &validatingTextInputModel{input: input, active: false, label: label}
}

type validatingTextInputModel struct {
	label  string
	input  textinput.Model
	active bool
}

func (m *validatingTextInputModel) Init() tea.Cmd {
	return nil
}

func (m *validatingTextInputModel) Update(msg tea.Msg) (*validatingTextInputModel, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	return m, cmd
}

func (m *validatingTextInputModel) View() string {
	var errRow string
	if m.input.Err != nil {
		errRow = lipgloss.NewStyle().Foreground(styles.Red).Render("Validation Error: " + m.input.Err.Error())
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		styles.HelpStyle.Render(m.label+": "),
		lipgloss.JoinVertical(lipgloss.Top, m.input.View(), errRow))
}

func (m *validatingTextInputModel) focus() tea.Cmd {
	m.input.PromptStyle = styles.FocusedStyle
	m.input.TextStyle = styles.FocusedStyle

	return m.input.Focus()
}

func (m *validatingTextInputModel) blur() {
	m.input.PromptStyle = styles.NoStyle
	m.input.TextStyle = styles.NoStyle
	m.input.Blur()
}

type urlValidator struct {
	emptyOk bool
}

func (v urlValidator) validate(value string) error {
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

type pathValidator struct{}

func (v pathValidator) validate(value string) error {
	if value == "" {
		return fmt.Errorf("%w: Cannot be empty", errFilePath)
	}

	if _, err := os.Stat(value); err != nil {
		return fmt.Errorf("%w: Invalid log path", errors.Join(err, errConfigValue))
	}

	return nil
}

type steamIDValidator struct{}

func (v steamIDValidator) validate(value string) error {
	steamID := steamid.New(value)
	if !steamID.Valid() {
		return errSteamIDInvalid
	}

	return nil
}

type addressValidator struct{}

func (v addressValidator) validate(value string) error {
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
