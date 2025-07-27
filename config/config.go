// Package config provides functionality to load and save configuration settings for the n8nctl CLI tool.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	APIToken string `json:"api_token"`
	BaseURL  string `json:"base_url"`
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(home, ".n8nctl")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.Mkdir(configDir, 0700); err != nil {
			return "", err
		}
	}
	return filepath.Join(configDir, "config.json"), nil
}

func LoadConfig() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()
	var cfg Config
	err = json.NewDecoder(f).Decode(&cfg)
	return cfg, err
}

func SaveConfig(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(cfg)
}
