package repository

import (
	"fmt"
	"log"

	"github.com/terzigolu/josepshbrain-go/config"
	"github.com/terzigolu/josepshbrain-go/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database veritabanı bağlantısını yönetir
type Database struct {
	DB *gorm.DB
}

var db *Database

// NewDatabase yeni bir veritabanı bağlantısı oluşturur
func NewDatabase(cfg *config.Config) (*Database, error) {
	if db != nil {
		return db, nil
	}

	// GORM logger konfigürasyonu
	var logLevel logger.LogLevel
	if cfg.Debug {
		logLevel = logger.Info
	} else {
		logLevel = logger.Warn
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	}

	// PostgreSQL bağlantısı
	database, err := gorm.Open(postgres.Open(cfg.GetDatabaseDSN()), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Connection pool ayarları
	sqlDB, err := database.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get SQL DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	db = &Database{DB: database}

	// Auto migration
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database connection established successfully")
	return db, nil
}

// GetDB mevcut veritabanı bağlantısını döndürür
func GetDB() *Database {
	if db == nil {
		panic("Database not initialized. Call NewDatabase first.")
	}
	return db
}

// migrate veritabanı şemasını oluşturur/günceller
func (d *Database) migrate() error {
	// GORM auto migration
	return d.DB.AutoMigrate(
		&models.Project{},
		&models.Context{},
		&models.Tag{},
		&models.Task{},
		&models.Annotation{},
		&models.Memory{},
		&models.MemoryTaskLink{},
	)
}

// Close veritabanı bağlantısını kapatır
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Health veritabanı sağlığını kontrol eder
func (d *Database) Health() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
} 