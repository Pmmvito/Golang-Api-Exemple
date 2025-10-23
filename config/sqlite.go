package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitializeSQLite() (*gorm.DB, error) {
	logger := GetLogger("sqlite")

	dbPath := os.Getenv("SQLITE_PATH")
	if dbPath == "" {
		dbPath = "./db/main.db"
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), os.ModePerm); err != nil {
		return nil, fmt.Errorf("erro criando diretório do banco: %w", err)
	}

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		logger.Info("Arquivo de banco SQLite não encontrado, criando...")
		file, err := os.Create(dbPath)
		if err != nil {
			return nil, fmt.Errorf("erro criando arquivo SQLite: %w", err)
		}
		file.Close()
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		logger.ErrorF("Erro abrindo SQLite: %v", err)
		return nil, err
	}

	if err := runMigrations(db); err != nil {
		return nil, err
	}

	return db, nil
}
