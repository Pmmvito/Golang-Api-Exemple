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
	
	//initialize sqlite
	db, err = InitializeSQLite()
	if err != nil {
		return fmt.Errorf("erro initializing sqlite %v: ", err)
	}
	
	return nil
}

func GetSQLite() *gorm.DB{
	return db
}

func GetLogger(p string) *Logger {
	//INITIALIZER LOGGER

	logger = NewLogger(p)
	return logger
}
