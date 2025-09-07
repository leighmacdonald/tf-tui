package config

import (
	"errors"
	"log/slog"

	"github.com/fsnotify/fsnotify"
	"github.com/joho/godotenv"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/spf13/viper"
)

// Loader handles setting up viper, loading configuration from files, and broadcasting configuration changes.
type Loader struct {
	viper   *viper.Viper
	changes chan<- Config
}

func NewLoader(changes chan<- Config) *Loader {
	var (
		loader = Loader{changes: changes}
		con    = viper.New()
	)

	errDotEnv := godotenv.Load()
	if errDotEnv != nil {
		slog.Debug("Could not load .env file", slog.String("error", errDotEnv.Error()))
	}

	con.SetDefault("steam_id", "")
	con.SetDefault("console_log_path", defaultConsoleLogPath())
	con.SetDefault("update_freq_ms", 2000)
	con.SetDefault("server_mode_enabled", false)
	con.SetDefault("server_log_address", "1.2.3.4:27115")
	con.SetDefault("server_bind_address", "1.2.3.4:27115")
	con.SetDefault("api_base_url", "https://tf-api.roto.lol/")
	con.SetDefault("bd_lists", []map[string]string{})
	con.SetDefault("links", []map[string]string{
		{
			"url":    "https://demos.tf/profiles/%s",
			"name":   "demos.tf",
			"format": "",
		},
	})
	con.SetDefault("servers", []map[string]any{
		{
			"address":    "127.0.0.1:27015",
			"password":   "tf-tui",
			"log_secret": 0,
		},
	})
	con.SetConfigName(DefaultConfigName)
	con.SetConfigType("yaml")
	con.SetEnvPrefix(EnvPrefix)
	con.AddConfigPath(Path(ConfigDirName))
	con.AddConfigPath(".")
	con.AutomaticEnv()
	con.WatchConfig()
	con.OnConfigChange(loader.onConfigChange)
	loader.viper = con

	return &loader
}

func (cl *Loader) Path() string {
	return cl.viper.ConfigFileUsed()
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
		cl.viper.Set("steam_id", config.SteamID.String())
	} else {
		cl.viper.Set("steam_id", "")
	}
	cl.viper.Set("console_log_path", config.ConsoleLogPath)
	cl.viper.Set("update_freq_ms", config.UpdateFreqMs)
	cl.viper.Set("server_mode_enabled", config.ServerModeEnabled)
	cl.viper.Set("server_log_address", config.ServerLogAddress)
	cl.viper.Set("server_bind_address", config.ServerBindAddress)
	cl.viper.Set("api_base_url", config.APIBaseURL)
	cl.viper.Set("bd_lists", config.BDLists)
	cl.viper.Set("links", config.Links)
	cl.viper.Set("servers", config.Servers)

	if err := cl.viper.WriteConfig(); err != nil {
		return errors.Join(err, errConfigWrite)
	}

	return nil
}

func (cl *Loader) Read() (Config, error) {
	if err := cl.viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			return Config{}, errors.Join(err, errConfigRead)
		}
	}

	var config Config
	if err := cl.viper.Unmarshal(&config); err != nil {
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
