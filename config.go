package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/user"
	"path"
	"runtime"
	"time"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
	"github.com/joho/godotenv"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"gopkg.in/yaml.v3"
)

var (
	errConfigWrite = errors.New("failed to write config file")
	errInvalidPath = errors.New("invalid path")
	errConfigValue = errors.New("failed to validate config")
	errLoggerInit  = errors.New("failed to initialize logger")
)

const (
	configDirName      = "tf-tui"
	defaultConfigName  = "tf-tui.yaml"
	defaultDBName      = "tf-tui.db"
	defaultLogName     = "tf-tui.log"
	defaultHTTPTimeout = 15 * time.Second
)

type configIdx int

const (
	fieldAddress configIdx = iota
	fieldPassword
	fieldConsoleLogPath
	fieldSave
)

type Config struct {
	SteamID        steamid.SteamID `yaml:"steam_id"`
	Address        string          `yaml:"address"`
	Password       string          `yaml:"password"`
	ConsoleLogPath string          `yaml:"console_log_path"`
	APIBaseURL     string          `yaml:"api_base_url,omitempty"`
	BDLists        []UserList      `yaml:"bd_lists"`
	Links          []UserLink      `yaml:"links"`
}

type SIDFormats string

const (
	Steam64 SIDFormats = "steam64"
	Steam2  SIDFormats = "steam"
	Steam3  SIDFormats = "steam3"
)

type UserLink struct {
	URL    string     `yaml:"url"`
	Name   string     `yaml:"name"`
	Format SIDFormats `yaml:"format"`
}

func (u UserLink) Generate(steamID steamid.SteamID) string {
	switch u.Format {
	case Steam2:
		return fmt.Sprintf(u.URL, steamID.Steam(false))
	case Steam3:
		return fmt.Sprintf(u.URL, steamID.Steam3())
	case Steam64:
		fallthrough
	default:
		return fmt.Sprintf(u.URL, steamID.String())
	}
}

type UserList struct {
	URL  string `yaml:"url"`
	Name string `yaml:"name"`
}

var defaultConfig = Config{
	Address:        "127.0.0.1:27015",
	Password:       "tf-tui",
	ConsoleLogPath: "",
}

type keymap struct {
	start         key.Binding
	stop          key.Binding
	reset         key.Binding
	quit          key.Binding
	config        key.Binding
	chat          key.Binding
	up            key.Binding
	down          key.Binding
	left          key.Binding
	right         key.Binding
	accept        key.Binding
	back          key.Binding
	prevTab       key.Binding
	nextTab       key.Binding
	overview      key.Binding
	bans          key.Binding
	comp          key.Binding
	notes         key.Binding
	console       key.Binding
	help          key.Binding
	consoleInput  key.Binding
	consoleCancel key.Binding
}

// TODO make configurable.
var DefaultKeyMap = keymap{
	consoleInput: key.NewBinding(
		key.WithKeys("return"),
		key.WithHelp("<return>", "Send command")),
	consoleCancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("<esc>", "Cancel input")),
	help: key.NewBinding(
		key.WithKeys("h", "H"),
		key.WithHelp("h", "Help"),
	),
	accept: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "Select"),
	),
	back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "Back"),
	),
	reset: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "reset"),
	),
	quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "Quit"),
	),
	config: key.NewBinding(
		key.WithKeys("E"),
		key.WithHelp("E", "Conf"),
	),
	up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑", "Up"),
	),
	down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓", "Down"),
	),
	left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←", "RED"),
	),
	right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→", "BLU"),
	),
	nextTab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "Next Tab"),
	),
	prevTab: key.NewBinding(
		key.WithKeys("shift tab"),
		key.WithHelp("shift tab", "Prev Tab"),
	),
	overview: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "Overview"),
	),
	bans: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "Bans"),
	),
	comp: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "Comp"),
	),
	notes: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "Notes"),
	),
	console: key.NewBinding(
		key.WithKeys("~"),
		key.WithHelp("~", "Console"),
	),
	chat: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "Chat"),
	),
}

// ConfigPath generates a path pointing to the filename under this apps defined $XDG_CONFIG_HOME.
func ConfigPath(name string) string {
	fullPath, errFullPath := xdg.ConfigFile(path.Join(configDirName, name))
	if errFullPath != nil {
		panic(errFullPath)
	}

	return fullPath
}

// ConfigWatcher is responsible for monitoring the config file for external changes and
// subsequently sending the new Config to the *tea.Program to broadcast the changed Config.
func ConfigWatcher(ctx context.Context, program *tea.Program, name string) {
	watcher, errWatcher := fsnotify.NewWatcher()
	if errWatcher != nil {
		return
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			// watch for events
			case event := <-watcher.Events:
				if event.Op != fsnotify.Rename && event.Op != fsnotify.Write {
					continue
				}

				conf, readOk := ConfigRead(name)
				if !readOk {
					continue
				}

				program.Send(conf)
			}
		}
	}()

	configPath := ConfigPath(name)
	if err := watcher.Add(configPath); err != nil {
		slog.Error("Error adding watch for config", slog.String("error", err.Error()))
	}

	<-ctx.Done()
}

func ConfigRead(name string) (Config, bool) {
	errDotEnv := godotenv.Load()
	if errDotEnv != nil {
		slog.Debug("Could not load .env file", slog.String("error", errDotEnv.Error()))
	}

	var config Config
	if err := os.MkdirAll(path.Join(xdg.ConfigHome, configDirName), 0o600); err != nil {
		slog.Error("Failed to make config root", slog.String("error", err.Error()))

		return defaultConfig, false
	}

	inFile, errOpen := os.Open(ConfigPath(name))
	if errOpen != nil {
		return defaultConfig, false
	}
	defer inFile.Close()

	if err := yaml.NewDecoder(inFile).Decode(&config); err != nil {
		return Config{}, false
	}

	if config.APIBaseURL == "" {
		config.APIBaseURL = "https://tf-api.roto.lol/"
	}

	if config.ConsoleLogPath == "" {
		config.ConsoleLogPath = DefaultConsoleLogPath()
	}

	return config, true
}

func DefaultConsoleLogPath() string {
	switch runtime.GOOS {
	case "darwin":
		// Untested
		usr, err := user.Current()
		if err != nil {
			panic(err)
		}

		return fmt.Sprintf("/Users/%s/Library/Application Support/Steam/steamapps/common/Team Fortress 2/tf/console.log", usr.Name)
	case "linux":
		homedir, err := os.UserHomeDir()
		if err != nil {
			homedir = "/"
		}

		return path.Join(homedir, ".steam/steam/steamapps/common/Team Fortress 2/tf/console.log")
	case "windows":
		// Untested
		return "C:\\Program Files (x86)\\Steam\\steamapps\\common\\Team Fortress 2\\tf\\console.log"
	default:
		return ""
	}
}

func ConfigWrite(name string, config Config) error {
	outFile, errOpen := os.Create(ConfigPath(name))
	if errOpen != nil {
		return errors.Join(errOpen, errConfigWrite)
	}

	defer outFile.Close()

	if err := yaml.NewEncoder(outFile).Encode(&config); err != nil {
		return errors.Join(err, errConfigWrite)
	}

	return nil
}

// LoggerInit sets up the slog global handler to use a log file as we cant print to the console.
func LoggerInit(logPath string, level slog.Level) (io.Closer, error) {
	logFile, errLogFile := os.Create(path.Join(xdg.ConfigHome, configDirName, logPath))
	if errLogFile != nil {
		return nil, errors.Join(errLogFile, errLoggerInit)
	}

	logger := slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{
		AddSource: false,
		Level:     level,
	}))

	slog.SetDefault(logger)

	return logFile, nil
}
