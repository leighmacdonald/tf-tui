package main

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/leighmacdonald/tf-tui/plugins"
)

type FnNames string

const (
	onAdd         FnNames = "Add"
	onPlayerState FnNames = "PlayerState"
)

var CallbackNames = []FnNames{
	onAdd,
	onPlayerState,
}

type OnAdd func(a int, b int) int

type OnPlayerState func(state DumpPlayer) DumpPlayer

var errPluginInterpreter = errors.New("interpreter error")

func NewScripting() *Scripting {
	return &Scripting{}
}

type Scripting struct {
	logEvents []plugins.LogEvents
}

func (s *Scripting) LoadedPluginsCount() int {
	return len(s.logEvents)
}

func (s *Scripting) LoadPlugins(ctx context.Context, root string) error {
	eventsPlugin, err := plugins.NewLogEventsPlugin(ctx)
	if err != nil {
		return errors.Join(err, errPluginInterpreter)
	}

	pluginPaths, errPaths := findPlugins(root)
	if errPaths != nil {
		return errPaths
	}

	for _, pluginPath := range pluginPaths {
		plugin, errPlugin := eventsPlugin.Load(ctx, pluginPath, NewPluginLogger(pluginPath))
		if errPlugin != nil {
			return errors.Join(errPlugin, errPluginInterpreter)
		}

		s.logEvents = append(s.logEvents, plugin)
	}

	return nil
}

func findPlugins(rootPath string) ([]string, error) {
	var pluginPaths []string
	if err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return errors.Join(err, errPluginInterpreter)
		}

		if !d.IsDir() {
			return nil
		}

		// dirPath := filepath.Join(path, d.Name())
		dirFiles, errDir := os.ReadDir(path)
		if errDir != nil {
			return errors.Join(errDir, errPluginInterpreter)
		}

		for _, f := range dirFiles {
			if !strings.HasSuffix(f.Name(), ".wasm") {
				continue
			}

			pluginPaths = append(pluginPaths, filepath.Join(path, f.Name()))
		}

		return nil
	}); err != nil {
		return nil, errors.Join(err, errPluginInterpreter)
	}

	return pluginPaths, nil
}

func NewPluginLogger(path string) PluginLogger {
	return PluginLogger{PluginPath: path}
}

type PluginLogger struct {
	PluginPath string
}

func (l PluginLogger) Debug(_ context.Context, log *plugins.LogMessage) (*emptypb.Empty, error) {
	tea.Println("DEBUG: " + log.Message)

	return &emptypb.Empty{}, nil
}

func (l PluginLogger) Info(_ context.Context, log *plugins.LogMessage) (*emptypb.Empty, error) {
	tea.Println("INFO: " + log.Message)

	return &emptypb.Empty{}, nil
}

func (l PluginLogger) Warn(_ context.Context, log *plugins.LogMessage) (*emptypb.Empty, error) {
	tea.Println("WARN: " + log.Message)

	return &emptypb.Empty{}, nil
}

func (l PluginLogger) Error(_ context.Context, log *plugins.LogMessage) (*emptypb.Empty, error) {
	tea.Println("ERROR: " + log.Message)

	return &emptypb.Empty{}, nil
}
