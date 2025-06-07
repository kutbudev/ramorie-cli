package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	defaultApiURL      = "http://localhost:8080"
	configDirName      = ".jbrain"
	configFileName     = "config.json"
	activeProjectIDKey = "active_project_id"
	apiURLKey          = "api_url"
)

// CliConfig holds the configuration for the CLI tool.
type CliConfig struct {
	ActiveProjectID string `json:"active_project_id"`
	ApiURL          string `json:"api_url"`
}

// getConfigPath returns the full path to the configuration file.
func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDirName, configFileName), nil
}

// SaveCliConfig saves the CLI configuration to the user's home directory.
func SaveCliConfig(cfg CliConfig) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	// Ensure the directory exists.
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// LoadCliConfig loads the CLI configuration. If it doesn't exist, it creates a default one.
func LoadCliConfig() (CliConfig, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return CliConfig{}, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Config file does not exist, create a default one.
		defaultConfig := CliConfig{
			ApiURL: defaultApiURL,
		}
		if err := SaveCliConfig(defaultConfig); err != nil {
			return CliConfig{}, err
		}
		return defaultConfig, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return CliConfig{}, err
	}

	var cfg CliConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return CliConfig{}, err
	}

	// If ApiURL is missing from an existing config, set it to default.
	if cfg.ApiURL == "" {
		cfg.ApiURL = defaultApiURL
	}

	return cfg, nil
} 