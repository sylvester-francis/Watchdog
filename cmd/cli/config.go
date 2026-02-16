package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CLIConfig holds saved CLI authentication state.
type CLIConfig struct {
	HubURL string `json:"hub_url"`
	Token  string `json:"token"`
}

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".watchdog")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

func loadConfig() (*CLIConfig, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not logged in — run 'watchdog login' first")
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg CLIConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.HubURL == "" || cfg.Token == "" {
		return nil, fmt.Errorf("incomplete config — run 'watchdog login' again")
	}

	return &cfg, nil
}

func saveConfig(cfg *CLIConfig) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath(), data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}
