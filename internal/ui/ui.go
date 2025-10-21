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
	HostPort string
	Server   Server
	Status   tf.Status
}

func (s Snapshot) AvgPing() float64 {
	if len(s.Status.Players) == 0 {
		return 0
	}

	var pings float64
	for _, player := range s.Status.Players {
		pings += float64(player.Ping)
	}

	return pings / float64(len(s.Status.Players))

}

type UI struct {
	program *tea.Program
}

func New(ctx context.Context, config config.Config, doSetup bool, buildVersion string, buildDate string, buildCommit string,
	loader ConfigWriter, cachePath string, parentCtx chan any) *UI {
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
				cachePath,
				parentCtx),
			tea.WithMouseCellMotion(),
			tea.WithAltScreen(),
			// tea.WithMouseAllMotion(),
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
