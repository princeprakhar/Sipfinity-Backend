// models/product.go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Product struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Title       string    `json:"title" gorm:"not null"`
	Description string    `json:"description"`
	Price       float64   `json:"price" gorm:"not null"`
	Category    string    `json:"category"`
	Size        string    `json:"size"`
	Material    string    `json:"material,omitempty"`
	Status      string    `json:"status" gorm:"default:'active'"`
	Stock       int       `json:"stock" gorm:"default:0"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Images      []Image   `json:"images" gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE"`
	LikeCount    int  `gorm:"default:0"`
	DislikeCount int  `gorm:"default:0"`

	// Fixed Services relationship
	Services []Service `json:"services,omitempty" gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE"`

	// Relations
	Reviews []Review `json:"reviews,omitempty"`
}
type ProductReaction struct {
	ID         uint `gorm:"primaryKey"`
	UserID     uint
	ProductID  uint
	IsLike     bool 
	IsDislike  bool
	CreatedAt  time.Time
}


// Fixed Service struct (singular name)
type Service struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ProductID uint      `json:"product_id" gorm:"not null;index"` // Remove duplicate foreignKey here
	Name      string    `json:"name" gorm:"not null"`
	Link      string    `json:"link" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Belongs to relationship
	Product Product `json:"-" gorm:"constraint:OnDelete:CASCADE"`
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

	// Belongs to relationship
	Product Product `json:"-" gorm:"constraint:OnDelete:CASCADE"`
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



type CreateProductRequest struct {
	Title       string                 `json:"title" binding:"required"`
	Description string                 `json:"description"`
	Price       float64                `json:"price" binding:"required,gt=0"`
	Category    string                 `json:"category"`
	Material    string                 `json:"material,omitempty"`
	Size        string                 `json:"size"`
	Stock       int                    `json:"stock"`
	Status      string                 `json:"status" binding:"required,oneof=active inactive"`
	Services    []CreateServiceRequest `json:"services,omitempty"`
}

type CreateServiceRequest struct {
	Name string `json:"name" binding:"required"`
	Link string `json:"link" binding:"required"`
}

type UpdateProductRequest struct {
	Title       *string  `json:"title,omitempty"`
	Description *string  `json:"description,omitempty"`
	Price       *float64 `json:"price,omitempty"`
	Category    *string  `json:"category,omitempty"`
	Material    *string  `json:"material,omitempty"`
	Size        *string  `json:"size,omitempty"`
	Stock       *int     `json:"stock,omitempty"`
	Status      *string  `json:"status,omitempty"`
	Services    []CreateServiceRequest `json:"services,omitempty"` 
}
