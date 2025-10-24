// Package config handles loading and reloading config files.
package config

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/user"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	errConfigWrite = errors.New("failed to write config file")
	errConfigRead  = errors.New("failed to read config file")
	errLoggerInit  = errors.New("failed to initialize logger")
)

const (
	ConfigDirName      = "tf-tui"
	DefaultConfigName  = "tf-tui"
	DefaultDBName      = "tf-tui.db"
	DefaultLogName     = "tf-tui.log"
	CacheDirName       = "cache"
	EnvPrefix          = "tftui"
	DefaultHTTPTimeout = 15 * time.Second
)

type Config struct {
	// TODO implement encoding.TextUnmarshaler so we can decode directly with viper/mapstructure
	SteamID steamid.SteamID `mapstructure:"-"`
	// SteamIDString should contain the steamid of the player running the app.
	SteamIDString string `mapstructure:"steam_id"`
	// ConsoleLogPath  defines the path to the console log file when running in local mode.
	ConsoleLogPath string `mapstructure:"console_log_path"`
	// UpdateFreqMs defines the frequency in milliseconds at which the app should update the player UI state.
	UpdateFreqMs int `mapstructure:"update_freq_ms,omitempty"`
	// CacheDir is where we can cache data.
	CacheDir string `mapstructure:"cache_dir"`
	// Debug enables some functionality useful for debugging for development, namely providing a
	// fake log source and randomly generated player data.
	Debug bool `mapstructure:"debug,omitempty"`
	// APIBaseURL is the base URL for the API where external data is fetched from.
	// Unless you are reimplementing the API, this should be left as is.
	APIBaseURL string `mapstructure:"api_base_url,omitempty"`
	// ServerModeEnabled controls if the app is running in server mode where instead
	// of connecting to your local TF2 client, you can removely monitor a server over RCON.
	// Similar to other tools like HLSW, you can receive server logs over UDP when setup
	// correctly.
	ServerModeEnabled bool `mapstructure:"server_mode_enabled"`
	// ServerLogAddress should point to an address where the server can reach you to send logs.
	ServerLogAddress string `mapstructure:"server_log_address"`
	// ServerBindAddress is the address where the server should bind to.
	ServerBindAddress string `mapstructure:"server_bind_address"`
	ServerUPNPEnabled bool   `mapstructure:"server_upnp_enabled"`
	// BDLists contains a list of bot detector lists to use.
	BDLists []UserList `mapstructure:"bd_lists"`
	// Links can be used to provide additional links to websites in the overview panel.
	Links []UserLink `mapstructure:"links"`
	// Servers contains a list of all known servers.
	Servers []ServerConfig `mapstructure:"servers"`
	// Client is the connect info for running in local client mode.
	Client ServerConfig `mapstructure:"client"`
}

func (c Config) UPNPPortMapping() (uint16, uint16) {
	return getPort(c.ServerLogAddress), getPort(c.ServerBindAddress)
}

func getPort(addr string) uint16 {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return 27115
	}

	port, errPort := strconv.ParseUint(parts[1], 10, 16)
	if errPort != nil {
		slog.Error("Invalid port mapping", slog.String("error", errPort.Error()))
		return 27115
	}

	return uint16(port)
}

type ServerConfig struct {
	// Address is the RCON address of the server. Must be unique.
	Address string `mapstructure:"address"`
	// Password is the RCON password of the server.
	Password string `mapstructure:"password"`
	// LogSecret is how we authenticate the server logs. You MUST set these to unique values for each server
	// for this functionality to work correctly.
	LogSecret int `mapstructure:"logsecret"`
}

type SIDFormats string

const (
	Steam64 SIDFormats = "steam64"
	Steam   SIDFormats = "steam"
	Steam3  SIDFormats = "steam3"
)

type UserLink struct {
	URL    string     `mapstructure:"url"`
	Name   string     `mapstructure:"name"`
	Format SIDFormats `mapstructure:"format"`
}

func (u UserLink) Generate(steamID steamid.SteamID) string {
	switch u.Format {
	case Steam:
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
	URL       string `mapstructure:"url"`
	Name      string `mapstructure:"name"`
	LogSecret int    `mapstructure:"log_secret"`
}

// Path generates a path pointing to the filename under this apps defined $XDG_CONFIG_HOME.
func Path(name string) string {
	fullPath, errFullPath := xdg.ConfigFile(path.Join(ConfigDirName, name))
	if errFullPath != nil {
		panic(errFullPath)
	}

	return fullPath
}

func PathCache(name string) string {
	cacheDir, found := os.LookupEnv("CACHE_DIR")
	if found && cacheDir != "" {
		return cacheDir
	}

	return path.Join(xdg.CacheHome, ConfigDirName, name)
}

func defaultConsoleLogPath() string {
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

// LoggerInit sets up the slog global handler to use a log file as we cant print to the console.
func LoggerInit(logPath string, level slog.Level) (io.Closer, error) {
	logFile, errLogFile := os.Create(path.Join(xdg.ConfigHome, ConfigDirName, logPath))
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
