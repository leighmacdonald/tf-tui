package plugins

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"github.com/leighmacdonald/tf-tui/pkg/plugins/proto"
	"google.golang.org/grpc"
)

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "tf-tui",
}

type Players interface {
	Get(steamid string) string
}

type PlayersPlugin struct {
	plugin.NetRPCUnsupportedPlugin

	Impl Players
}

func (p *PlayersPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterPlayersServer(s, &GRPCServer{
		Impl:   p.Impl,
		broker: broker,
	})

	return nil
}

func (p *PlayersPlugin) GRPCClient(_ context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (any, error) {
	return &GRPCClient{
		client: proto.NewPlayersClient(c),
		broker: broker,
	}, nil
}

var PluginMap = map[string]plugin.Plugin{
	"players": &PlayersPlugin{},
}

var _ plugin.GRPCPlugin = &PlayersPlugin{}
