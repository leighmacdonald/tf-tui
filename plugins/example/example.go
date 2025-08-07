//go:build wasip1

package main

import (
	"context"

	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/leighmacdonald/tf-tui/plugins"
)

// main is required for Go to compile to Wasm.
func main() {}

func init() {
	// Register any of our plugins with the host so they can be called.
	plugins.RegisterLogEvents(&CustomLogEventHandler{})
}

type CustomLogEventHandler struct{}

func (m CustomLogEventHandler) OnLogEvent(ctx context.Context, request *plugins.LogEvent) (*plugins.LogEventResponse, error) {
	fn := plugins.NewLoggingFunctions()
	fn.Info(ctx, &plugins.LogMessage{Message: "Example Plugin Log Event Message"})
	return &plugins.LogEventResponse{Success: true, LogEvent: request}, nil
}

// Unload handles any shutdown procedures that need to happen in your plugin. Note that `Close` is a reserved function
// name and cannot be used.
func (m CustomLogEventHandler) Unload(ctx context.Context, _ *emptypb.Empty) (*plugins.ErrorResponse, error) {
	return &plugins.ErrorResponse{Success: true}, nil
}
