package main

import (
	"log/slog"
	"time"

	"github.com/hashicorp/go-plugin"
	"github.com/leighmacdonald/tf-tui/pkg/plugins"
)

type Logger struct{}

func (p *Logger) Info(message string) {
	slog.Info(message)
}
func (p *Logger) Warn(message string) {
	slog.Info(message)
}

func (p *Logger) Error(message string) {
	slog.Info(message)
}
func (p *Logger) Debug(message string) {
	slog.Info(message)
}

func (p *Logger) start() {
	ticker := time.NewTicker(time.Second * 1)
	for {
		select {
		case <-ticker.C:
			p.Debug("Hello from a plugin")
		}
	}
}

func main() {
	logger := &plugins.LoggerPlugin{Impl: &Logger{}}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugins.Handshake,
		Plugins: map[string]plugin.Plugin{
			string(plugins.PluginRCON): logger,
		},

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
