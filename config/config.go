package config

import (
	"fmt"

	"gorm.io/gorm"
)

var (
	db     *gorm.DB
	logger *Logger
)

func Init() error {
	var err error

	//initialize PostgreSQL
	db, err = InitializePostgreSQL()
	if err != nil {
		return fmt.Errorf("erro initializing postgresql %v: ", err)
	}

	return nil
}

func GetPostgreSQL() *gorm.DB {
	return db
}

func GetLogger(p string) *Logger {
	//INITIALIZER LOGGER

	logger = NewLogger(p)
	return logger
}
