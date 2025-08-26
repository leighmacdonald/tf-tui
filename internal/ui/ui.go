package ui

import (
	"context"
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leighmacdonald/tf-tui/internal/config"
	zone "github.com/lrstanley/bubblezone"
)

var ErrUIExit = errors.New("ui error returned")

type contentView int

const (
	viewPlayerTables contentView = iota
	viewConfig
	viewHelp
)

type UI struct {
	program *tea.Program
}

func New(ctx context.Context, config config.Config, doSetup bool, buildVersion string, buildDate string, buildCommit string) *UI {
	zone.NewGlobal()

	return &UI{
		program: tea.NewProgram(
			newRootModel(
				config,
				doSetup,
				buildVersion,
				buildDate,
				buildCommit),
			tea.WithMouseCellMotion(),
			tea.WithAltScreen(),
			tea.WithContext(ctx)),
	}
}

func (t UI) Run() error {
	if _, err := t.program.Run(); err != nil {
		return errors.Join(err, ErrUIExit)
	}

	return nil
}

func (t UI) Send(msg tea.Msg) {
	t.program.Send(msg)
}
