package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	configDir  = ".jbrain"
	configFile = "config.json"
)

// CliConfig stores the CLI configuration, like the active project.
type CliConfig struct {
	ActiveProjectID string `json:"active_project_id"`
	// In the future, we can add more fields like APIKey, APIBaseURL, etc.
}

// getConfigPath returns the full path to the configuration file.
func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDir, configFile), nil
}

// LoadCliConfig reads the configuration from the file.
// If the file or directory doesn't exist, it creates them and returns an empty config.
func LoadCliConfig() (CliConfig, error) {
	path, err := getConfigPath()
	if err != nil {
		return CliConfig{}, err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return CliConfig{}, err
	}

	// Read file
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return empty config, it will be created on save
			return CliConfig{}, nil
		}
		return CliConfig{}, err
	}
	defer file.Close()

	var cfg CliConfig
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		// If file is empty or corrupted, return empty config
		return CliConfig{}, nil
	}
	return cfg, nil
}

// SaveCliConfig writes the configuration to the file.
func SaveCliConfig(cfg CliConfig) error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // for pretty printing
	return encoder.Encode(cfg)
} 