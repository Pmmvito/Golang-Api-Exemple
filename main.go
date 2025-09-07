package main

import (
	

	"github.com/Pmmvito/Golang-Api-Exemple/config"
	"github.com/Pmmvito/Golang-Api-Exemple/router"
)

var(
	logger *config.Logger
)

func main() {

	logger = config.GetLogger("main")

	//initialize configs of projects\
	err := config.Init()
	if err != nil{
		logger.ErrorF("config initialization erro: %v",err)
		return
	}

	router.Initialize()
}