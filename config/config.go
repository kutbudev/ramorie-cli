package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config stores all configuration for the application.
type Config struct {
	DB_DSN string
}

// LoadConfig loads configuration from the .env file and constructs the DSN.
func LoadConfig() *Config {
	// Load .env file from the current directory
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	host := os.Getenv("PG_HOST")
	port := os.Getenv("PG_PORT")
	user := os.Getenv("PG_USER")
	password := os.Getenv("PG_PASSWORD")
	dbname := os.Getenv("PG_DATABASE")
	sslmode := os.Getenv("PG_SSL_MODE")

	if host == "" || port == "" || user == "" || dbname == "" {
		log.Fatal("One or more required database environment variables (PG_HOST, PG_PORT, PG_USER, PG_DATABASE) are not set.")
	}
	
	if sslmode == "" {
		sslmode = "disable"
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		host, user, password, dbname, port, sslmode)

	return &Config{
		DB_DSN: dsn,
	}
} 