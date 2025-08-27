package internal

import (
	"errors"
	"os/exec"

	"github.com/hashicorp/go-plugin"
	"github.com/leighmacdonald/tf-tui/pkg/plugins"
)

var errPluginOpen = errors.New("failed to open plugin host")

func NewPluginHost() *PluginHost {
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: plugins.Handshake,
		Plugins:         plugins.PluginMap,
		Managed:         true,
		// Logger:          hclog.NewNullLogger(),
		Cmd: exec.Command("sh", "-c", "./pkg/plugins/plugin-example"),
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolGRPC,
		},
	})

	return &PluginHost{
		client: client,
	}
}

type PluginHost struct {
	client  *plugin.Client
	players plugins.Players
}

func (p *PluginHost) Open() error {
	// Connect via RPC
	rpcClient, errClient := p.client.Client()
	if errClient != nil {
		return errors.Join(errClient, errPluginOpen)
	}

	// Request the plugin
	raw, errDispense := rpcClient.Dispense("players")
	if errDispense != nil {
		return errors.Join(errDispense, errPluginOpen)
	}

	playersImpl, ok := raw.(plugins.Players)
	if !ok {
		return errPluginOpen
	}

	p.players = playersImpl

	return nil
}

func (p *PluginHost) Close() {
	p.client.Kill()
}
