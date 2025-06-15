package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/princeprakhar/ecommerce-backend/internal/api/handlers"
	"github.com/princeprakhar/ecommerce-backend/internal/api/middleware"
	"github.com/princeprakhar/ecommerce-backend/internal/config"
	"github.com/princeprakhar/ecommerce-backend/internal/services"
	"github.com/princeprakhar/ecommerce-backend/pkg/logger"
	"gorm.io/gorm"
)

func SetupRoutes(router *gin.Engine, db *gorm.DB, cfg *config.Config) {
	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.RateLimitMiddleware(cfg))


	validationService := services.NewValidationService(
        cfg.AbstractEmailAPIKey,
        cfg.AbstractPhoneNumberAPIKey,
    )



	// Initialize services
	emailService := services.NewEmailService(cfg)
	authService := services.NewAuthService(db, cfg.JWTSecret, validationService, emailService, cfg.BaseURL)
	reviewService := services.NewReviewService(db)
	productService := services.NewProductService(db)
	
	fastAPIService := services.NewFastAPIService(cfg)
	adminService := services.NewAdminService(db,cfg, fastAPIService, emailService)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	passwordHandler := handlers.NewPasswordHandler(authService)
	reviewHandler := handlers.NewReviewHandler(reviewService)
	adminHandler := handlers.NewAdminHandler(adminService)
	productHandler := handlers.NewProductHandler(productService)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "message": "Server is running"})
	})

	// API routes
	api := router.Group("/api/v1")

	// Auth routes (public)
	auth := api.Group("/auth")
	{
		auth.POST("/signup", authHandler.Signup)
		auth.POST("/login", authHandler.Login)
		auth.POST("/logout", middleware.AuthMiddleware(cfg), authHandler.Logout)
		auth.POST("/refresh-token", authHandler.RefreshToken)
		auth.GET("/profile", middleware.AuthMiddleware(cfg), authHandler.GetProfile)
		auth.PUT("/profile-update", middleware.AuthMiddleware(cfg), authHandler.UpdateProfile)
	}

	// Password reset routes
	passwordGroup := api.Group("/password")
	{
		passwordGroup.POST("/forgot", passwordHandler.ForgotPassword)
		passwordGroup.GET("/validate-reset-token",  passwordHandler.ValidateResetToken, ) // Requires authentication
		passwordGroup.POST("/reset", passwordHandler.ResetPassword)
		passwordGroup.POST("/change", middleware.AuthMiddleware(cfg), passwordHandler.ChangePassword) // Requires authentication
	}
	// Review routes
	reviews := api.Group("/reviews")
	{
		reviews.GET("/product/:product_id", reviewHandler.GetProductReviews)
		reviews.POST("/", middleware.AuthMiddleware(cfg), middleware.CustomerOrAdmin(), reviewHandler.CreateReview)
		reviews.POST("/:review_id/like", middleware.AuthMiddleware(cfg), middleware.CustomerOrAdmin(), reviewHandler.LikeReview)
		reviews.POST("/:review_id/flag", middleware.AuthMiddleware(cfg), middleware.CustomerOrAdmin(), reviewHandler.FlagReview)
	}


	// Product routes
	products := api.Group("/products")
	{
		products.GET("/", productHandler.GetAllProducts)
		products.GET("/:product_id", productHandler.GetProduct)
		products.GET("/category",productHandler.GetCategories)
	}

	// Admin routes
	admin := api.Group("/admin", middleware.AuthMiddleware(cfg), middleware.AdminOnly())
	{
		admin.GET("/dashboard", adminHandler.GetDashboard)
		
		// Product management
		// admin.POST("/upload/images", adminHandler.UploadImages)
		// admin.POST("/upload/csv", adminHandler.UploadCSV)
		admin.GET("/products", adminHandler.GetProducts)
		admin.POST("/products", adminHandler.CreateProduct)
		admin.GET("/products/:product_id", adminHandler.GetProduct)

		admin.PUT("/products/:product_id", adminHandler.UpdateProduct)
		admin.POST("/products/:product_id/images", adminHandler.UploadProductImages)
		admin.DELETE("/products/:product_id/images/:image_id", adminHandler.DeleteProductImage)
		admin.DELETE("/products/batch", adminHandler.BatchDeleteProducts)
		admin.DELETE("/products/:product_id", adminHandler.DeleteProduct)
		admin.GET("/products/search", adminHandler.SearchProducts)

		// Review moderation
		admin.GET("/reviews/flagged", reviewHandler.GetFlaggedReviews)
		admin.POST("/reviews/:review_id/moderate", reviewHandler.ModerateReview)
	}

	logger.Info("Routes initialized successfully")
}