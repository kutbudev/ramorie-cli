package database

import (
	"log"

	"github.com/terzigolu/josepshbrain-go/api/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Connect initializes a new database connection.
func Connect() {
	cfg := config.LoadConfig()
	
	db, err := gorm.Open(postgres.Open(cfg.DB_DSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Database connection successfully established.")

	DB = db
} 