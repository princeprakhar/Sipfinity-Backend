package services

import (
	"errors"
	"strings"

	"github.com/princeprakhar/ecommerce-backend/internal/models"

	"gorm.io/gorm"
)

type ProductService struct {
	db *gorm.DB
}


func NewProductService(db *gorm.DB) *ProductService {
	return &ProductService{
		db: db,
	}
}

type ProductFilter struct {
	Category   string  `form:"category"`
	Brand      string  `form:"brand"`
	MinPrice   float64 `form:"min_price"`
	MaxPrice   float64 `form:"max_price"`
	Search     string  `form:"search"`
	IsActive   *bool   `form:"is_active"`
	Page       int     `form:"page"`
	Limit      int     `form:"limit"`
}

type ProductResponse struct {
	Products []models.Product `json:"products"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	Limit    int              `json:"limit"`
	Pages    int              `json:"pages"`
}


type ProductRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Price       float64 `json:"price" binding:"required"`
	Category    string  `json:"category" binding:"required"`
	Brand       string  `json:"brand" binding:"required"`
	Image       string  `json:"image"`
}


func (s *ProductService) GetProducts(filter ProductFilter) (*ProductResponse, error) {
	var products []models.Product
	var total int64

	// Set default pagination
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 10
	}

	query := s.db.Model(&models.Product{}).Where("is_active = ?", true)

	// Apply filters
	if filter.Category != "" {
		query = query.Where("LOWER(category) LIKE ?", "%"+strings.ToLower(filter.Category)+"%")
	}

	if filter.Brand != "" {
		query = query.Where("LOWER(brand) LIKE ?", "%"+strings.ToLower(filter.Brand)+"%")
	}

	if filter.MinPrice > 0 {
		query = query.Where("price >= ?", filter.MinPrice)
	}

	if filter.MaxPrice > 0 {
		query = query.Where("price <= ?", filter.MaxPrice)
	}

	if filter.Search != "" {
		searchTerm := "%" + strings.ToLower(filter.Search) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ?", searchTerm, searchTerm)
	}

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply pagination and get results
	offset := (filter.Page - 1) * filter.Limit
	if err := query.Offset(offset).Limit(filter.Limit).Order("created_at DESC").Find(&products).Error; err != nil {
		return nil, err
	}

	pages := int(total) / filter.Limit
	if int(total)%filter.Limit > 0 {
		pages++
	}

	return &ProductResponse{
		Products: products,
		Total:    total,
		Page:     filter.Page,
		Limit:    filter.Limit,
		Pages:    pages,
	}, nil
}

// GetProductByID - for public access (only active products)
func (s *ProductService) GetProductByID(id uint) (*models.Product, error) {
	var product models.Product
	
	if err := s.db.Where("id = ? AND is_active = ?", id, true).First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("product not found")
		}
		return nil, err
	}

	return &product, nil
}

