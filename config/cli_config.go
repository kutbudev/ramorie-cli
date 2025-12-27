package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	// New config directory name
	configDirName = ".ramorie"
	// Legacy config directory name (for backward compatibility)
	configDirLegacy    = ".jbrain"
	configFileName     = "config.json"
	DefaultApiURL      = "https://jbraincli-go-backend-production.up.railway.app"
	DefaultAuthURL     = "https://jbraincli-go-backend-production.up.railway.app/auth"
	V1ApiURL           = "https://jbraincli-go-backend-production.up.railway.app/v1"
	activeProjectIDKey = "active_project_id"
	apiURLKey          = "api_url"
)

// GetV1ApiURL returns the V1 API URL.
func GetV1ApiURL() string {
	// In a real app, you might read this from viper or another config manager
	return V1ApiURL
}

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

// getLegacyConfigPath returns the full path to the legacy configuration file.
func getLegacyConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDirLegacy, configFileName), nil
}

// LoadCliConfig loads the CLI configuration from new location, falling back to legacy.
// If config is found in legacy location, it will be migrated to the new location.
func LoadCliConfig() (CliConfig, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return CliConfig{}, err
	}

	// Try new location first
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return CliConfig{}, err
		}
		var cfg CliConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return CliConfig{}, err
		}
		if cfg.ApiURL == "" {
			cfg.ApiURL = DefaultApiURL
		}
		return cfg, nil
	}

	// Try legacy location
	legacyPath, err := getLegacyConfigPath()
	if err != nil {
		return CliConfig{}, err
	}

	if _, err := os.Stat(legacyPath); err == nil {
		data, err := os.ReadFile(legacyPath)
		if err != nil {
			return CliConfig{}, err
		}
		var cfg CliConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return CliConfig{}, err
		}
		if cfg.ApiURL == "" {
			cfg.ApiURL = DefaultApiURL
		}
		// Migrate to new location
		_ = SaveCliConfig(cfg) // Best effort migration
		return cfg, nil
	}

	// Config file does not exist, create a default one.
	defaultConfig := CliConfig{
		ApiURL: DefaultApiURL,
	}
	if err := SaveCliConfig(defaultConfig); err != nil {
		return CliConfig{}, err
	}
	return defaultConfig, nil
}
