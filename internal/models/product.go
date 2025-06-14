// models/product.go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
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
	Images      []Image   `json:"images" gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relations
	Reviews []Review `json:"reviews,omitempty"`
}

type Image struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ProductID   uint      `gorm:"not null;index" json:"product_id"`
	FileName    string    `gorm:"not null" json:"file_name"`
	S3Key       string    `gorm:"not null;unique" json:"s3_key"`
	S3URL       string    `gorm:"not null" json:"s3_url"`
	ContentType string    `gorm:"not null" json:"content_type"`
	Size        int64     `json:"size"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (i *Image) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

type ProductUploadResponse struct {
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	ProcessedCount int    `json:"processed_count"`
	FilePath       string `json:"file_path,omitempty"`
}

// Request structs for API
type CreateProductRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required,gt=0"`
	Category    string  `json:"category"`
	Brand       string  `json:"brand"`
	SKU         string  `json:"sku"`
	Stock       int     `json:"stock"`
}

type UpdateProductRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Price       *float64 `json:"price,omitempty"`
	Category    *string  `json:"category,omitempty"`
	Brand       *string  `json:"brand,omitempty"`
	SKU         *string  `json:"sku,omitempty"`
	Stock       *int     `json:"stock,omitempty"`
	IsActive    *bool    `json:"is_active,omitempty"`
}
