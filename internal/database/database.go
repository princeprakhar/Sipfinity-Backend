package database

import (
	"github.com/princeprakhar/ecommerce-backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Init(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	// Auto migrate schemas
	err = db.AutoMigrate(
		&models.User{},
		&models.Product{},
		&models.Review{},
		&models.RefreshToken{},
		&models.PasswordResetToken{},
		&models.ReviewLike{},
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}