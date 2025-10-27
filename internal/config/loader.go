package config

import (
	"errors"
	"log/slog"

	"github.com/fsnotify/fsnotify"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/spf13/viper"
)

// Loader handles setting up viper, loading configuration from files, and broadcasting configuration changes.
type Loader struct {
	*viper.Viper
	changes chan<- Config
}

func NewLoader(changes chan<- Config) *Loader {
	loader := Loader{changes: changes, Viper: viper.New()}
	loader.SetDefault("steam_id", "")
	loader.SetDefault("console_log_path", defaultConsoleLogPath())
	loader.SetDefault("update_freq_ms", 2000)
	loader.SetDefault("server_mode_enabled", false)
	loader.SetDefault("server_log_address", "1.2.3.4:27115")
	loader.SetDefault("server_upnp_enabled", false)
	loader.SetDefault("server_bind_address", "1.2.3.4:27115")
	loader.SetDefault("api_base_url", "https://tf-api.roto.lol/")
	loader.SetDefault("bd_lists", []map[string]string{})
	loader.SetDefault("links", []map[string]string{
		{
			"url":    "https://demos.tf/profiles/%s",
			"name":   "demos.tf",
			"format": "",
		},
	})
	loader.SetDefault("servers", []map[string]any{
		{
			"address":    "127.0.0.1:27015",
			"password":   "tf-tui",
			"log_secret": 0,
		},
	})
	loader.SetDefault("debug", false)
	loader.SetDefault("fps", 60)
	loader.SetConfigName(DefaultConfigName)
	loader.SetConfigType("yaml")
	loader.SetEnvPrefix(EnvPrefix)
	loader.AddConfigPath(Path(""))
	loader.AddConfigPath(".")
	loader.AutomaticEnv()
	loader.WatchConfig()
	loader.OnConfigChange(loader.onConfigChange)

	return &loader
}

func (cl *Loader) Path() string {
	return cl.ConfigFileUsed()
}

func (cl *Loader) onConfigChange(in fsnotify.Event) {
	if in.Op != fsnotify.Write && in.Op != fsnotify.Rename {
		return
	}

	slog.Debug("External config reload triggered")
	config, err := cl.Read()
	if err != nil {
		slog.Error("Error reading config", slog.String("error", err.Error()))

		return
	}

	cl.changes <- config
}

func (cl *Loader) Write(config Config) error {
	if config.SteamID.Valid() {
		cl.Set("steam_id", config.SteamID.String())
	} else {
		cl.Set("steam_id", "")
	}
	cl.Set("console_log_path", config.ConsoleLogPath)
	cl.Set("update_freq_ms", config.UpdateFreqMs)
	cl.Set("server_mode_enabled", config.ServerModeEnabled)
	cl.Set("server_log_address", config.ServerLogAddress)
	cl.Set("server_bind_address", config.ServerBindAddress)
	cl.Set("api_base_url", config.APIBaseURL)
	cl.Set("bd_lists", config.BDLists)
	cl.Set("links", config.Links)
	cl.Set("servers", config.Servers)

	if err := cl.WriteConfig(); err != nil {
		return errors.Join(err, errConfigWrite)
	}

	return nil
}

func (cl *Loader) Read() (Config, error) {
	if err := cl.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			return Config{}, errors.Join(err, errConfigRead)
		}
	}

	var config Config
	if err := cl.Unmarshal(&config); err != nil {
		return Config{}, errors.Join(err, errConfigRead)
	}

	if config.SteamIDString != "" {
		sid := steamid.New(config.SteamIDString)
		if !sid.Valid() {
			return Config{}, errConfigRead
		}
		config.SteamID = sid
	}

	return config, nil
}
