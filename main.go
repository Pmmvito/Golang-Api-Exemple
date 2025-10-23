// @title           Golang Finance API
// @version         1.0
// @description     API para gestão financeira pessoal com suporte a OCR de recibos.
// @contact.name    Equipe Golang Finance
// @contact.email   contato@example.com
// @host            localhost:8080
// @BasePath        /api/v1
// @schemes         http
// @securityDefinitions.apikey Bearer
// @in              header
// @name            Authorization
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
