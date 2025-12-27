package config

import (
	"fmt"
	"os"
	"strconv"

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
	// Try to load .env file from multiple locations (silently)
	possibleEnvPaths := []string{
		".env", // Current directory
		"/Users/terzigolu/GitHub/josepshbrain-go/.env", // Project directory
		"/Users/terzigolu/.env",                        // Home directory
	}

	envLoaded := false
	for _, path := range possibleEnvPaths {
		if err := godotenv.Load(path); err == nil {
			envLoaded = true
			break // Successfully loaded, stop trying
		}
	}

	// Initialize viper
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

	// Try to read config file (silently handle if not found)
	configFileFound := true
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			configFileFound = false
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Only log status if in verbose mode (check DEBUG env var)
	if os.Getenv("DEBUG") == "true" {
		fmt.Printf("Working directory: %s\n", getWorkingDir())
		fmt.Printf("Env file loaded: %v\n", envLoaded)
		fmt.Printf("Config file found: %v\n", configFileFound)

		// Debug actual config values
		fmt.Printf("DB Host: %s\n", viper.GetString("database.host"))
		fmt.Printf("DB Port: %d\n", viper.GetInt("database.port"))
		fmt.Printf("DB Name: %s\n", viper.GetString("database.name"))
		fmt.Printf("DB User: %s\n", viper.GetString("database.user"))

		if !envLoaded && !configFileFound {
			fmt.Println("Config file not found, using environment variables and defaults")
		} else if envLoaded {
			fmt.Println("Configuration loaded from .env file")
		} else if configFileFound {
			fmt.Printf("Configuration loaded from %s\n", viper.ConfigFileUsed())
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
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getWorkingDir() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "unknown"
}
