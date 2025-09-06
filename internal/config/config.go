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
	SteamID        steamid.SteamID `mapstructure:"-"`
	SteamIDString  string          `mapstructure:"steam_id"`
	Address        string          `mapstructure:"address"`
	Password       string          `mapstructure:"password"`
	ConsoleLogPath string          `mapstructure:"console_log_path"`
	UpdateFreqMs   int             `mapstructure:"update_freq_ms,omitempty"`
	APIBaseURL     string          `mapstructure:"api_base_url,omitempty"`
	// ServerModeEnabled controls if the app is running in server mode where instead
	// of connecting to your local TF2 client, you can removely monitor a server over RCON.
	// Similar to other tools like HLSW, you can receive server logs over UDP when setup
	// correctly.
	ServerModeEnabled bool `mapstructure:"server_mode_enabled"`
	// ServerLogAddress should point to an address where the server can reach you to send logs.
	ServerLogAddress string `mapstructure:"server_log_address"`
	// ServerLogSecret is the sv_logsecret values used for log message auth.
	ServerLogSecret     int64      `mapstructure:"server_log_secret"`
	ServerListenAddress string     `mapstructure:"server_listen_address"`
	BDLists             []UserList `mapstructure:"bd_lists"`
	Links               []UserLink `mapstructure:"links"`
	Servers             []Server   `mapstructure:"servers"`
}

type Server struct {
	Address   string `mapstructure:"address"`
	Password  string `mapstructure:"password"`
	LogSecret int    `mapstructure:"log_secret"`
}

type SIDFormats string

const (
	Steam64 SIDFormats = "steam64"
	Steam2  SIDFormats = "steam"
	Steam3  SIDFormats = "steam3"
)

type UserLink struct {
	URL    string     `mapstructure:"url"`
	Name   string     `mapstructure:"name"`
	Format SIDFormats `mapstructure:"format"`
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
