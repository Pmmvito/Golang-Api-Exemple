package handler

import (
	"github.com/Pmmvito/Golang-Api-Exemple/config"
	"gorm.io/gorm"
)

var (
	logger *config.Logger
	db     *gorm.DB
)

func InitializerHandler() {
	logger = config.GetLogger("handler")
	db = config.GetPostgreSQL()
}
