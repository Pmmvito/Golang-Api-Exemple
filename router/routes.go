package router

import (

	docs "github.com/Pmmvito/Golang-Api-Exemple/docs"
	"github.com/Pmmvito/Golang-Api-Exemple/handler"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func InitializeRoutes(router *gin.Engine) {

	//initialize Handler
	handler.InitializerHandler()
	basePatch := "/api/v1"
	docs.SwaggerInfo.BasePath = basePatch

	v1 := router.Group(basePatch)
	{
		v1.GET("/opening", handler.ShowOpeningHandler)

		v1.POST("/opening", handler.CreateOpeningHandler)

		v1.DELETE("/opening", handler.DeleteOpeningHandler)

		v1.PUT("/opening", handler.UpdateOpeningHandler)

		v1.GET("/openings", handler.ListOpeningHandler)

	}

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
}
