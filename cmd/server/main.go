package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/princeprakhar/ecommerce-backend/internal/api/routes"
	"github.com/princeprakhar/ecommerce-backend/internal/config"
	"github.com/princeprakhar/ecommerce-backend/internal/database"
	"github.com/princeprakhar/ecommerce-backend/pkg/logger"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize logger
	logger.Init()

	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Init(cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("Failed to initialize database", err)
	}


	

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize router
	router := gin.New()

	// Setup routes
	routes.SetupRoutes(router, db, cfg)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info("Server starting on port " + port)
	if err := router.Run(":" + port); err != nil {
		logger.Fatal("Failed to start server", err)
	}
}
