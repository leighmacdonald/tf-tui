package main

import (
	"os"
	"path"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/bubbles/filepicker"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Address        string `yaml:"address"`
	Password       string `yaml:"password"`
	ConsoleLogPath string `yaml:"console_log_path"`
	FullScreen     bool   `yaml:"full_screen"`
}

var defaultConfig = Config{
	Address:        "127.0.0.1:27015",
	Password:       "test",
	ConsoleLogPath: "",
	FullScreen:     true,
}

const defaultConfigName = "tf-tui.yaml"

func configPath(name string) string {
	fullPath, errFullPath := xdg.ConfigFile(path.Join("tf-tui", name))
	if errFullPath != nil {
		panic(errFullPath)
	}
	return fullPath
}

func configRead(name string) (Config, bool) {
	var config Config
	inFile, errOpen := os.Open(configPath(name))
	if errOpen != nil {
		return defaultConfig, false
	}
	defer inFile.Close()

	if err := yaml.NewDecoder(inFile).Decode(&config); err != nil {
		return Config{}, false
	}

	return config, true
}

func configWrite(name string, config Config) error {
	outFile, errOpen := os.Create(configPath(defaultConfigName))
	if errOpen != nil {
		return errOpen
	}

	defer outFile.Close()

	return yaml.NewEncoder(outFile).Encode(&config)
}

func newPicker() filepicker.Model {
	fp := filepicker.New()
	fp.AllowedTypes = []string{}
	fp.CurrentDirectory, _ = os.UserHomeDir()
	fp.ShowPermissions = true
	fp.ShowHidden = true
	fp.ShowSize = true

	return fp
}
