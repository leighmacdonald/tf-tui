package component

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
	"github.com/leighmacdonald/tf-tui/internal/ui/styles"
)

var (
	errAddressInvalid = errors.New("invalid address")
	errFilePath       = errors.New("invalid file path")
	errSteamIDInvalid = errors.New("invalid steamid")
	errInvalidURL     = errors.New("invalid URL")
	errConfigValue    = errors.New("failed to validate config")
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
					return err
				}
			}

			return nil
		}
	}

	return &ValidatingTextInputModel{Input: input, active: false, Label: label}
}

type ValidatingTextInputModel struct {
	Label  string
	Input  textinput.Model
	active bool
}

func (m *ValidatingTextInputModel) Init() tea.Cmd {
	return nil
}

func (m *ValidatingTextInputModel) Update(msg tea.Msg) (*ValidatingTextInputModel, tea.Cmd) {
	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)

	return m, cmd
}

func (m *ValidatingTextInputModel) View() string {
	var errRow string
	if m.Input.Err != nil {
		errRow = lipgloss.NewStyle().Foreground(styles.Red).Render("Validation Error: " + m.Input.Err.Error())
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		styles.HelpStyle.Render(m.Label+": "),
		lipgloss.JoinVertical(lipgloss.Top, m.Input.View(), errRow))
}

func (m *ValidatingTextInputModel) Focus() tea.Cmd {
	m.Input.PromptStyle = styles.FocusedStyle
	m.Input.TextStyle = styles.FocusedStyle

	return m.Input.Focus()
}

func (m *ValidatingTextInputModel) Blur() {
	m.Input.PromptStyle = styles.NoStyle
	m.Input.TextStyle = styles.NoStyle
	m.Input.Blur()
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
