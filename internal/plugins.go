package internal

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/hashicorp/go-plugin"
	"github.com/leighmacdonald/tf-tui/pkg/plugins"
)

var errPluginOpen = errors.New("failed to open plugin host")

func NewPluginHost(pluginRoot string) *PluginHost {
	return &PluginHost{
		pluginRoot: pluginRoot,
	}
}

type PluginHost struct {
	pluginRoot string
	plugins    []*plugin.Client
	players    []plugins.Players
}

func (p *PluginHost) Open() error {
	pluginList, errList := listPlugins(p.pluginRoot)
	if errList != nil {
		return errList
	}

	for _, pluginPath := range pluginList {
		client := plugin.NewClient(&plugin.ClientConfig{
			HandshakeConfig: plugins.Handshake,
			Plugins:         plugins.PluginMap,
			Managed:         true,
			// Logger:          hclog.NewNullLogger(),
			Cmd: exec.Command("sh", "-c", pluginPath),
			AllowedProtocols: []plugin.Protocol{
				plugin.ProtocolGRPC,
			},
		})

		// Connect via RPC
		rpcClient, errClient := client.Client()
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

		p.players = append(p.players, playersImpl)
	}

	return nil
}

func (p *PluginHost) Close() {
	for _, plugin := range p.plugins {
		plugin.Kill()
	}

	p.plugins = nil
	p.players = nil
}

func listPlugins(root string) ([]string, error) {
	var valid []string
	files, errRead := os.ReadDir(root)
	if errRead != nil {
		return nil, errors.Join(errRead, errPluginOpen)
	}

	for _, entry := range files {
		if entry.IsDir() {
			continue
		}

		if err := isFileExececutable(entry.Name()); err != nil {
			continue
		}

		valid = append(valid, entry.Name())
	}

	return valid, nil
}

var errFileNotExec = errors.New("file not executable")

func isFileExececutable(file string) error {
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(file, "exe") {
			return errFileNotExec
		}

		return nil
	}

	fileInfo, err := os.Stat(file)
	if err != nil {
		return err
	}
	m := fileInfo.Mode()

	if !((m.IsRegular()) || (uint32(m&fs.ModeSymlink) == 0)) {
		return errFileNotExec
	}
	if uint32(m&0o111) == 0 {
		return errFileNotExec
	}

	// unix dep, have to build in separate file with build options if we want to check this.
	// if unix.Access(file, unix.X_OK) != nil {
	// 	return errFileNotExec
	// }

	return nil
}
