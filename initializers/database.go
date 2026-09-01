package initializers

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"time"

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

var AdvoProDB *sql.DB

func ConnectAdvoPro() error {
	user := os.Getenv("MSSQL_USER")
	pass := os.Getenv("MSSQL_PASS")

	// Direct connection to 192.168.2.11:3000 avoids UDP browser lookup delays
	conn := fmt.Sprintf(
		"server=192.168.2.11;port=3000;database=AdvoPro;user id=%s;password=%s;encrypt=disable;TrustServerCertificate=true;connection timeout=30",
		user, pass,
	)

	var err error
	AdvoProDB, err = sql.Open("sqlserver", conn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	AdvoProDB.SetMaxOpenConns(25)
	AdvoProDB.SetMaxIdleConns(5)
	AdvoProDB.SetConnMaxLifetime(5 * time.Minute)

	return AdvoProDB.Ping()
}
