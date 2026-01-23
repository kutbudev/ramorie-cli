package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	// New config directory name
	configDirNew = ".ramorie"
	// Legacy config directory name (for backward compatibility)
	configDirLegacy = ".jbrain"
	configFileName  = "config.json"
)

type Config struct {
	APIKey string `json:"api_key"`
	// Encryption fields (cached from server after login)
	EncryptionEnabled     bool   `json:"encryption_enabled,omitempty"`
	EncryptedSymmetricKey string `json:"encrypted_symmetric_key,omitempty"` // base64
	KeyNonce              string `json:"key_nonce,omitempty"`               // base64
	Salt                  string `json:"salt,omitempty"`                    // base64
	KDFIterations         int    `json:"kdf_iterations,omitempty"`
	KDFAlgorithm          string `json:"kdf_algorithm,omitempty"`
}

// GetConfigPath returns the path to the new config file (~/.ramorie/config.json)
func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDirNew, configFileName), nil
}

// getLegacyConfigPath returns the path to the legacy config file (~/.jbrain/config.json)
func getLegacyConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDirLegacy, configFileName), nil
}

// LoadConfig loads config from the new location, falling back to legacy location if needed.
// If config is found in legacy location, it will be migrated to the new location.
func LoadConfig() (*Config, error) {
	newPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// Try to load from new location first
	if data, err := os.ReadFile(newPath); err == nil {
		var cfg Config
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
		return &cfg, nil
	}

	// Try legacy location
	legacyPath, err := getLegacyConfigPath()
	if err != nil {
		return nil, err
	}

	if data, err := os.ReadFile(legacyPath); err == nil {
		var cfg Config
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
		// Migrate to new location
		_ = SaveConfig(&cfg) // Best effort migration
		return &cfg, nil
	}

	// No config found, return empty
	return &Config{}, nil
}

func SaveConfig(cfg *Config) error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Ensure the directory exists
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
