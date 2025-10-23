package router

import (
	"time"

	docs "github.com/Pmmvito/Golang-Api-Exemple/docs"
	"github.com/Pmmvito/Golang-Api-Exemple/handler"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func InitializeRoutes(router *gin.Engine) {
	handler.InitializerHandler()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	basePath := "/api/v1"
	docs.SwaggerInfo.BasePath = basePath

	api := router.Group(basePath)

	authGroup := api.Group("/auth")
	{
		authGroup.POST("/register", handler.RegisterHandler)
		authGroup.POST("/login", handler.LoginHandler)
		authGroup.Use(handler.AuthMiddleware())
		authGroup.POST("/logout", handler.LogoutHandler)
		authGroup.GET("/me", handler.MeHandler)
	}

	protected := api.Group("")
	protected.Use(handler.AuthMiddleware())
	{
		protected.GET("/categories", handler.ListCategoriesHandler)
		protected.POST("/categories", handler.CreateCategoryHandler)
		protected.PUT("/categories/:id", handler.UpdateCategoryHandler)
		protected.DELETE("/categories/:id", handler.DeleteCategoryHandler)

		protected.GET("/expenses", handler.ListExpensesHandler)
		protected.POST("/expenses", handler.CreateExpenseHandler)
		protected.GET("/expenses/:id", handler.GetExpenseHandler)
		protected.PUT("/expenses/:id", handler.UpdateExpenseHandler)
		protected.DELETE("/expenses/:id", handler.DeleteExpenseHandler)

		protected.POST("/receipts/scan", handler.ScanReceiptHandler)

		protected.GET("/dashboard/summary", handler.DashboardSummaryHandler)

		protected.POST("/sync/jobs", handler.TriggerSyncHandler)

		protected.GET("/tips", handler.ListTipsHandler)
		protected.POST("/tips/generate", handler.GenerateTipsHandler)

		protected.GET("/token-usage", handler.ListTokenUsageHandler)

		protected.GET("/meal-plans", handler.GetMealPlanHandler)
		protected.POST("/meal-plans/generate", handler.GenerateMealPlanHandler)
	}

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
}
