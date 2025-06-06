package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config uygulamanın konfigürasyon yapısı
type Config struct {
	Database DatabaseConfig `mapstructure:"database"`
	Gemini   GeminiConfig   `mapstructure:"gemini"`
	MCP      MCPConfig      `mapstructure:"mcp"`
	Debug    bool           `mapstructure:"debug"`
}

// DatabaseConfig veritabanı konfigürasyonu
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

// GeminiConfig Google Gemini API konfigürasyonu
type GeminiConfig struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
}

// MCPConfig Model Context Protocol konfigürasyonu
type MCPConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

var appConfig *Config

// Load konfigürasyonu yükler
func Load() (*Config, error) {
	if appConfig != nil {
		return appConfig, nil
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("$HOME/.jbraincli")

	// Environment variable desteği
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("JBRAINCLI")

	// Varsayılan değerler
	setDefaults()

	// Config dosyasını oku
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config dosyası bulunamadı, environment variable'lar kullanılacak
			fmt.Println("Config file not found, using environment variables and defaults")
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	appConfig = config
	return config, nil
}

// setDefaults varsayılan değerleri ayarlar
func setDefaults() {
	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.dbname", "jbraincli")
	viper.SetDefault("database.sslmode", "disable")

	// Gemini defaults
	viper.SetDefault("gemini.model", "gemini-1.5-pro")

	// MCP defaults
	viper.SetDefault("mcp.host", "localhost")
	viper.SetDefault("mcp.port", 8080)

	// Debug default
	viper.SetDefault("debug", false)
}

// GetConfig mevcut konfigürasyonu döndürür
func GetConfig() *Config {
	if appConfig == nil {
		// Panic yerine varsayılan konfigürasyon döndür
		Load()
	}
	return appConfig
}

// GetDatabaseDSN PostgreSQL connection string oluşturur
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
} 