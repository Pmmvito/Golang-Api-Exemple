package config

import (
	"os"
	"github.com/Pmmvito/Golang-Api-Exemple/schemas"
	"gorm.io/driver/sqlite" 
	"gorm.io/gorm"
)

func InitializeSQLite() (*gorm.DB, error) {

	logger := GetLogger("sqlite")
	dbPath := "./db/main.db"
	//check if exist

	_,err := os.Stat(dbPath)
	if os.IsNotExist(err) {
		logger.Info("DataBase file not found, creating......")
		
		if err = os.MkdirAll("./db", os.ModePerm)
		err != nil {
			return nil, err
		}
		file , err := os.Create(dbPath)
		if err != nil {
			return nil, err
		}
		file.Close()
		
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		logger.ErrorF("sqlite opning error: %v", err)
	}
	err = db.AutoMigrate(&schemas.Opening{})
	if err != nil {
		logger.ErrorF("sqlite automigration error: %v", err)
		return nil, err
	}
	return db, nil
}
