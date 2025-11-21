package services

import (
	"fmt"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB() *db.Database {
	// Use unique DB name to avoid sharing between tests
	dbName := fmt.Sprintf("file:memdb_%s?mode=memory&cache=shared", uuid.New().String())
	gormDB, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	database := &db.Database{DB: gormDB}

	// Migrate the schema
	err = database.DB.AutoMigrate(
		&models.Cluster{},
		&models.Alert{},
		&models.RCARun{},
		&models.SystemSetting{},
	)
	if err != nil {
		panic("failed to migrate database: " + err.Error())
	}

	return database
}
