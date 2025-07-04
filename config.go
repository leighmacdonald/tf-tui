package main

import (
	"os"
	"path"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Address  string `yaml:"address"`
	Password string `yaml:"password"`
}

var defaultConfig = Config{
	Address:  "127.0.0.1:27015",
	Password: "test",
}

const configName = "config.yaml"

func configPath(name string) string {
	if err := os.MkdirAll(path.Join(xdg.ConfigHome, "tf-tui"), 0755); err != nil {
		panic(err)
	}
	fullPath, errFullPath := xdg.ConfigFile(path.Join("tf-tui", name))
	if errFullPath != nil {
		panic(errFullPath)
	}
	return fullPath
}

func configRead(name string) (Config, bool) {
	var config Config
	inFile, errOpen := os.Open(configPath(configName))
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
	outFile, errOpen := os.Open(configPath(configName))
	if errOpen != nil {
		return errOpen
	}

	defer outFile.Close()

	return yaml.NewEncoder(outFile).Encode(&config)
}
