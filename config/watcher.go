package config

import (
	"context"
	"io"
	"log/slog"

	"github.com/fsnotify/fsnotify"
)

// Notify is responsible for monitoring the config file for external changes and
// subsequently sending the new Config to the *tea.Program to broadcast the changed Config.
func Notify(ctx context.Context, name string, changes chan<- Config) {
	watcher, errWatcher := fsnotify.NewWatcher()
	if errWatcher != nil {
		return
	}
	defer func(closer io.Closer) {
		if err := closer.Close(); err != nil {
			slog.Error("watcher close error", slog.String("err", err.Error()))
		}
	}(watcher)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-watcher.Events:
				if event.Op != fsnotify.Rename && event.Op != fsnotify.Write {
					continue
				}

				conf, errRead := Read(name)
				if errRead != nil {
					slog.Error("Failed to read config", slog.String("error", errRead.Error()))

					continue
				}
				changes <- conf
			}
		}
	}()

	configPath := PathConfig(name)
	if err := watcher.Add(configPath); err != nil {
		slog.Error("Error adding watch for config", slog.String("error", err.Error()))
	}

	<-ctx.Done()
}
