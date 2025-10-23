package handler

import (
	"os"
	"strconv"
	"time"

	"github.com/Pmmvito/Golang-Api-Exemple/config"
	"gorm.io/gorm"
)

var (
	logger     *config.Logger
	db         *gorm.DB
	sessionTTL = 24 * time.Hour
)

func InitializerHandler() {
	logger = config.GetLogger("handler")
	db = config.GetDatabase()

	if db == nil {
		logger.Error("banco de dados não inicializado")
	}

	if ttlStr := os.Getenv("SESSION_TTL_HOURS"); ttlStr != "" {
		if hours, err := strconv.Atoi(ttlStr); err == nil && hours > 0 {
			sessionTTL = time.Duration(hours) * time.Hour
		}
	}
}

func getDB() *gorm.DB {
	return db
}

func getLogger() *config.Logger {
	return logger
}

func getSessionTTL() time.Duration {
	return sessionTTL
}
