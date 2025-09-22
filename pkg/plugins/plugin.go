package plugins

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"github.com/leighmacdonald/tf-tui/pkg/plugins/proto"
	"google.golang.org/grpc"
)

const ProtocolVersion = 1

type PluginName string

const (
	PluginLogger PluginName = "logger"
	PluginRCON   PluginName = "rcon"
)

var (
	Handshake = plugin.HandshakeConfig{
		ProtocolVersion:  ProtocolVersion,
		MagicCookieKey:   "BASIC_PLUGIN",
		MagicCookieValue: "tf-tui",
	}

	PluginMap = map[string]plugin.Plugin{
		string(PluginLogger): &LoggerPlugin{},
		string(PluginRCON):   &RCONPlugin{},
	}
)

type RCON interface {
	Command(command string) string
}

type RCONPlugin struct {
	plugin.NetRPCUnsupportedPlugin

	Impl RCON
}

func (p *RCONPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterRCONServer(s, &GRPCServerRCON{
		Impl:   p.Impl,
		broker: broker,
	})

	return nil
}

func (p *RCONPlugin) GRPCClient(_ context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (any, error) {
	return &GRPCClientRCON{
		client: proto.NewRCONClient(c),
		broker: broker,
	}, nil
}

type Logger interface {
	Info(message string)
	Warn(message string)
	Error(message string)
	Debug(message string)
}

type LoggerPlugin struct {
	plugin.NetRPCUnsupportedPlugin

	Impl Logger
}

func (p *LoggerPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterLoggerServer(s, &GRPCServerLogger{
		Impl:   p.Impl,
		broker: broker,
	})

	return nil
}

func (p *LoggerPlugin) GRPCClient(_ context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (any, error) {
	return &GRPCClientLogger{
		client: proto.NewLoggerClient(c),
		broker: broker,
	}, nil
}

var (
	_ plugin.GRPCPlugin = &LoggerPlugin{}
	_ plugin.GRPCPlugin = &RCONPlugin{}
)
