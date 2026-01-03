package repository

import (
	"fmt"
	"os"

	"github.com/kutbudev/ramorie-cli/pkg/config"
	"github.com/kutbudev/ramorie-cli/pkg/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewDatabase creates a new database connection
func NewDatabase(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.Port,
		cfg.Database.SSLMode,
	)

	// Temporary debug - always show database info
	fmt.Printf("🔧 Connecting to database: %s@%s:%d/%s\n",
		cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)

	// Set GORM logger level based on DEBUG env var
	logLevel := logger.Silent
	if os.Getenv("DEBUG") == "true" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database %s@%s:%d: %w",
			cfg.Database.User, cfg.Database.Host, cfg.Database.Port, err)
	}

	// Enable UUID extension
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		return nil, fmt.Errorf("failed to enable uuid extension: %w", err)
	}

	// Auto migrate the schema (only run when needed)
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	}

	// Log successful connection only in DEBUG mode
	if os.Getenv("DEBUG") == "true" {
		fmt.Printf("✅ Database connected successfully (%s@%s:%d)\n",
			cfg.Database.User, cfg.Database.Host, cfg.Database.Port)
	}

	return db, nil
}

// autoMigrate runs auto migration for all models
func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Project{},
		&models.Context{},
		&models.Tag{},
		&models.Task{},
		&models.Annotation{},
		&models.Dependency{},
		&models.Memory{},
		&models.MemoryItem{},
		&models.TaskMemory{},
		&models.MemoryTaskLink{},
	)
}
