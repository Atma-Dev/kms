package config

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() *gorm.DB {
	db, err := gorm.Open(postgres.Open(DSN()), &gorm.Config{TranslateError: true})
	if err != nil {
		panic(err)
	}

	DB = db
	return db
}

// DSN is the connection string for the configured database, shared with
// internal/testdb so the tests connect exactly the way the app does.
func DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		AppConfig.DbHost,
		AppConfig.DbPort,
		AppConfig.DbUser,
		AppConfig.DbPassword,
		AppConfig.DbName,
	)
}
