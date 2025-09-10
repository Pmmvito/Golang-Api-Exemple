package config

import (
	"os"

	"github.com/Pmmvito/Golang-Api-Exemple/schemas"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitializePostgreSQL() (*gorm.DB, error) {
	logger := GetLogger("postgres")

	// Carrega a DSN da variável de ambiente
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		logger.ErrorF("Variável de ambiente DATABASE_DSN não definida")
		return nil, nil // Ou retorne um erro apropriado
	}

	// Conecta ao banco de dados
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.ErrorF("Erro ao conectar com o PostgreSQL: %v", err)
		return nil, err
	}

	// Migra o schema
	err = db.AutoMigrate(&schemas.Opening{})
	if err != nil {
		logger.ErrorF("Erro na automigração do PostgreSQL: %v", err)
		return nil, err
	}

	logger.Info("Conexão com o PostgreSQL estabelecida e migração bem-sucedida.")
	return db, nil
}
