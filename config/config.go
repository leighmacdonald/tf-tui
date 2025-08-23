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
	"github.com/joho/godotenv"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"gopkg.in/yaml.v3"
)

var (
	errConfigWrite = errors.New("failed to write config file")
	errConfigRead  = errors.New("failed to read config file")
	errInvalidPath = errors.New("invalid path")
	errLoggerInit  = errors.New("failed to initialize logger")
)

const (
	ConfigDirName      = "tf-tui"
	DefaultConfigName  = "tf-tui.yaml"
	DefaultDBName      = "tf-tui.db"
	DefaultLogName     = "tf-tui.log"
	CacheDirName       = "cache"
	DefaultHTTPTimeout = 15 * time.Second
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
	ConsoleLogPath: DefaultConsoleLogPath(),
	APIBaseURL:     "https://tf-api.roto.lol/",
	BDLists:        []UserList{},
	Links:          []UserLink{},
}

// PathConfig generates a path pointing to the filename under this apps defined $XDG_CONFIG_HOME.
func PathConfig(name string) string {
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

func Read(name string) (Config, error) {
	errDotEnv := godotenv.Load()
	if errDotEnv != nil {
		slog.Debug("Could not load .env file", slog.String("error", errDotEnv.Error()))
	}

	if err := os.MkdirAll(path.Join(xdg.ConfigHome, ConfigDirName), 0o700); err != nil {
		return defaultConfig, errors.Join(err, errConfigRead)
	}

	inFile, errOpen := os.Open(PathConfig(name))
	if errOpen != nil {
		return defaultConfig, errors.Join(errOpen, errConfigRead)
	}
	defer func(closer io.Closer) {
		if err := closer.Close(); err != nil {
			slog.Error("Failed to close config file", slog.String("error", err.Error()))
		}
	}(inFile)

	var config Config
	if err := yaml.NewDecoder(inFile).Decode(&config); err != nil {
		return defaultConfig, errors.Join(err, errConfigRead)
	}

	if config.APIBaseURL == "" {
		config.APIBaseURL = "https://tf-api.roto.lol/"
	}

	if config.ConsoleLogPath == "" {
		config.ConsoleLogPath = DefaultConsoleLogPath()
	}

	return config, nil
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

func Write(name string, config Config) error {
	outFile, errOpen := os.Create(PathConfig(name))
	if errOpen != nil {
		return errors.Join(errOpen, errConfigWrite)
	}

	defer func(file io.Closer) {
		if err := file.Close(); err != nil {
			slog.Error("Failed to close config file", slog.String("error", err.Error()))
		}
	}(outFile)

	if err := yaml.NewEncoder(outFile).Encode(&config); err != nil {
		return errors.Join(err, errConfigWrite)
	}

	return nil
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
