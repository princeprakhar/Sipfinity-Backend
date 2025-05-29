package models

import (
	"time"
)

type Product struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"not null"`
	Description string    `json:"description"`
	Price       float64   `json:"price" gorm:"not null"`
	Category    string    `json:"category"`
	Brand       string    `json:"brand"`
	SKU         string    `json:"sku" gorm:"uniqueIndex"`
	Stock       int       `json:"stock" gorm:"default:0"`
	Images      []string  `json:"images" gorm:"type:text[]"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	
	// Relations
	Reviews []Review `json:"reviews,omitempty"`
}

type ProductUploadResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	ProcessedCount int  `json:"processed_count"`
	FilePath     string `json:"file_path,omitempty"`
}