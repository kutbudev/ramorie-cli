package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Database DatabaseConfig `mapstructure:"database"`
	Server   ServerConfig   `mapstructure:"server"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

// Load loads configuration from environment variables and config files
func Load() (*Config, error) {
	// Try to load .env file from multiple locations
	possibleEnvPaths := []string{
		".env",                                                    // Current directory
		"/Users/terzigolu/GitHub/josepshbrain-go/.env",           // Project directory
		"/Users/terzigolu/.env",                                   // Home directory
	}
	
	for _, path := range possibleEnvPaths {
		if err := godotenv.Load(path); err == nil {
			break // Successfully loaded, stop trying
		}
	}
	
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set default values
	viper.SetDefault("database.host", getEnv("PG_HOST", "localhost"))
	viper.SetDefault("database.port", getEnvInt("PG_PORT", 5432))
	viper.SetDefault("database.user", getEnv("PG_USER", "postgres"))
	viper.SetDefault("database.password", getEnv("PG_PASSWORD", ""))
	viper.SetDefault("database.name", getEnv("PG_DATABASE", "jbrain_dev"))
	viper.SetDefault("database.ssl_mode", getEnv("PG_SSL_MODE", "disable"))
	viper.SetDefault("server.port", getEnvInt("SERVER_PORT", 8080))
	viper.SetDefault("server.host", getEnv("SERVER_HOST", "localhost"))

	// Enable environment variable support
	viper.AutomaticEnv()

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist, we'll use defaults and env vars
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	return &config, nil
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		// Simple conversion for now
		switch value {
		case "5432":
			return 5432
		case "8080":
			return 8080
		}
	}
	return defaultValue
} 