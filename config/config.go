package config

import (
	"fmt"
	"os"

	"gorm.io/gorm"
)

var (
	db     *gorm.DB
	logger *Logger
)

func Init() error {
	var err error

	driver := os.Getenv("DATABASE_DRIVER")
	switch driver {
	case "sqlite":
		db, err = InitializeSQLite()
	case "postgres", "", "postgresql":
		db, err = InitializePostgreSQL()
	default:
		return fmt.Errorf("driver de banco desconhecido: %s", driver)
	}

	if err != nil {
		return fmt.Errorf("erro inicializando banco de dados: %w", err)
	}

	return nil
}

func GetDatabase() *gorm.DB {
	return db
}

func GetPostgreSQL() *gorm.DB {
	return db
}

func GetLogger(p string) *Logger {
	//INITIALIZER LOGGER

	logger = NewLogger(p)
	return logger
}
