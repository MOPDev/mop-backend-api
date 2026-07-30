package initializers

import (
	"io"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectToDB() {
	var err error

	DB, err = gorm.Open(sqlite.Open("data.db?_busy_timeout=10000&_txlock=immediate"), &gorm.Config{
		Logger: logger.New(log.New(io.Discard, "", 0), logger.Config{
			LogLevel:                  logger.Silent,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
		}),
	})
	if err != nil {
		log.Fatal("failed to connect database")
	}

	DB.Exec("PRAGMA foreign_keys = ON;")
	DB.Exec("PRAGMA journal_mode = WAL;") // lets readers run while a writer holds the lock

	sqlDB, err := DB.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(1) // ponytail: one writer ceiling; move to a real DB if write volume grows
	}

}
