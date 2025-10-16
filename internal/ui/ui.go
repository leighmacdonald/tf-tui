package ui

import (
	"context"
	"errors"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leighmacdonald/tf-tui/internal/config"
	"github.com/leighmacdonald/tf-tui/internal/tf"
	zone "github.com/lrstanley/bubblezone"
)

const (
	clearMessageTimeout = time.Second * 10
)

var ErrUIExit = errors.New("ui error returned")

type contentView int

const (
	viewMain contentView = iota
	viewConfig
	viewHelp
)

type Snapshot struct {
	LogSecret int
	Server    Server
	Status    tf.Status
}

type UI struct {
	program *tea.Program
}

func New(ctx context.Context, config config.Config, doSetup bool, buildVersion string, buildDate string, buildCommit string, loader ConfigWriter, cachePath string) *UI {
	zone.NewGlobal()

	return &UI{
		program: tea.NewProgram(
			newRootModel(
				config,
				doSetup,
				buildVersion,
				buildDate,
				buildCommit,
				loader,
				cachePath),
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
