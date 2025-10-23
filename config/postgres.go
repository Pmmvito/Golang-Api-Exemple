package config

import (
	"fmt"
	"os"

	"github.com/Pmmvito/Golang-Api-Exemple/schemas"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitializePostgreSQL() (*gorm.DB, error) {
	logger := GetLogger("postgres")

	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		return nil, fmt.Errorf("variável de ambiente DATABASE_DSN não definida")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.ErrorF("Erro ao conectar com o PostgreSQL: %v", err)
		return nil, err
	}

	if err := runMigrations(db); err != nil {
		return nil, err
	}

	logger.Info("Conexão com o PostgreSQL estabelecida e migração bem-sucedida.")
	return db, nil
}

func runMigrations(db *gorm.DB) error {
	logger := GetLogger("migrations")

	if err := db.AutoMigrate(
		&schemas.User{},
		&schemas.UserConfig{},
		&schemas.Category{},
		&schemas.Expense{},
		&schemas.ExpenseItem{},
		&schemas.Receipt{},
		&schemas.GeneratedTip{},
		&schemas.MealPlan{},
		&schemas.MealItem{},
		&schemas.Session{},
		&schemas.SyncJob{},
		&schemas.TokenUsage{},
	); err != nil {
		logger.ErrorF("Erro na automigração: %v", err)
		return err
	}

	return nil
}
