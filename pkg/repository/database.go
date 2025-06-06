package repository

import (
	"fmt"

	"github.com/terzigolu/josepshbrain-go/pkg/config"
	"github.com/terzigolu/josepshbrain-go/pkg/models"
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

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Enable UUID extension
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		return nil, fmt.Errorf("failed to enable uuid extension: %w", err)
	}

	// Auto migrate the schema (only run when needed)
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	}

	// Only log when migration actually happens or on first connection
	// log.Println("✅ Database connected successfully")
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