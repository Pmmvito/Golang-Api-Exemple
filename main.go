package main

import (
	"github.com/Pmmvito/Golang-Api-Exemple/config"
	"github.com/Pmmvito/Golang-Api-Exemple/router"
	"github.com/joho/godotenv"
)

var (
	logger *config.Logger
)

func main() {
	logger = config.GetLogger("main")

	// Carrega as variáveis de ambiente do arquivo .env
	err := godotenv.Load()
	if err != nil {
		logger.ErrorF("Erro ao carregar o arquivo .env: %v", err)
		return
	}

	//initialize configs of projects
	err = config.Init()
	if err != nil {
		logger.ErrorF("config initialization erro: %v", err)
		return
	}

	router.Initialize()
}
