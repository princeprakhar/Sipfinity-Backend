package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/princeprakhar/ecommerce-backend/internal/models"
	"gorm.io/gorm"
)

const (
	DefaultPageSize = 10
	MaxPageSize     = 100
	QueryTimeout    = 30 * time.Second
)

var (
	ErrProductNotFound = errors.New("product not found")
	ErrInvalidFilter   = errors.New("invalid filter parameters")
	ErrDatabaseQuery   = errors.New("database query failed")
)

type ProductService struct {
	db *gorm.DB
}

func NewProductService(db *gorm.DB) *ProductService {
	if db == nil {
		panic("database connection cannot be nil")
	}
	return &ProductService{
		db: db,
	}
}

type ProductFilter struct {
	Category string  `form:"category" validate:"max=100"`
	Material string  `form:"material" validate:"max=100"`
	Status   string  `form:"status" validate:"oneof=active inactive"`
	MinPrice float64 `form:"min_price" validate:"min=0"`
	MaxPrice float64 `form:"max_price" validate:"min=0"`
	Search   string  `form:"search" validate:"max=255"`
	Page     int     `form:"page" validate:"min=1"`
	Limit    int     `form:"limit" validate:"min=1,max=100"`
}

type ProductResponse struct {
	Products []models.Product `json:"products"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	Limit    int              `json:"limit"`
	Pages    int              `json:"pages"`
}

type ProductRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=255"`
	Description string  `json:"description" binding:"required,min=1,max=2000"`
	Price       float64 `json:"price" binding:"required,gt=0"`
	Category    string  `json:"category" binding:"required,min=1,max=100"`
	Brand       string  `json:"brand" binding:"required,min=1,max=100"`
	Image       string  `json:"image" binding:"omitempty,url"`
}

// ValidateAndNormalize validates and normalizes filter parameters
func (f *ProductFilter) ValidateAndNormalize() error {
	// Set default pagination
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.Limit <= 0 {
		f.Limit = DefaultPageSize
	}

	// Enforce maximum page size
	if f.Limit > MaxPageSize {
		f.Limit = MaxPageSize
	}

	// Validate price range
	if f.MinPrice < 0 || f.MaxPrice < 0 {
		return fmt.Errorf("%w: prices cannot be negative", ErrInvalidFilter)
	}

	if f.MinPrice > 0 && f.MaxPrice > 0 && f.MinPrice > f.MaxPrice {
		return fmt.Errorf("%w: min_price cannot be greater than max_price", ErrInvalidFilter)
	}

	// Normalize and validate search terms
	f.Search = strings.TrimSpace(f.Search)
	f.Category = strings.TrimSpace(f.Category)
	f.Material = strings.TrimSpace(f.Material)

	// Validate search term length
	if len(f.Search) > 255 {
		return fmt.Errorf("%w: search term too long", ErrInvalidFilter)
	}

	return nil
}

// GetProducts retrieves products with filtering and pagination (public access - active products only)
func (s *ProductService) GetProducts(ctx context.Context, filter ProductFilter) (*ProductResponse, error) {
	// Validate and normalize filter
	if err := filter.ValidateAndNormalize(); err != nil {
		return nil, err
	}

	// Set query timeout
	ctx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	var products []models.Product
	var total int64

	// Build base query - only active products for public access
	query := s.db.WithContext(ctx).Model(&models.Product{}).Where("status = ?", "active")

	// Apply filters
	query = s.applyFilters(query, filter)

	// Count total records first (more efficient)
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("%w: failed to count products: %v", ErrDatabaseQuery, err)
	}

	// Early return if no products found
	if total == 0 {
		return &ProductResponse{
			Products: []models.Product{},
			Total:    0,
			Page:     filter.Page,
			Limit:    filter.Limit,
			Pages:    0,
		}, nil
	}

	// Apply pagination and ordering
	offset := (filter.Page - 1) * filter.Limit
	if err := query.
		Offset(offset).
		Limit(filter.Limit).
		Order("created_at DESC").
		Find(&products).Error; err != nil {
		return nil, fmt.Errorf("%w: failed to fetch products: %v", ErrDatabaseQuery, err)
	}

	// Load related data efficiently
	if err := s.loadProductRelations(ctx, products); err != nil {
		return nil, fmt.Errorf("failed to load product relations: %v", err)
	}

	// Calculate total pages
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

// GetProductByID retrieves a single product by ID (public access - active products only)
func (s *ProductService) GetProductByID(ctx context.Context, id uint) (*models.Product, error) {
	if id == 0 {
		return nil, fmt.Errorf("%w: invalid product ID", ErrInvalidFilter)
	}

	// Set query timeout
	ctx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	var product models.Product
	
	if err := s.db.WithContext(ctx).
		Where("id = ? AND status = ?", id, "active").
		First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("%w: failed to fetch product: %v", ErrDatabaseQuery, err)
	}

	// Load related data
	if err := s.loadProductRelations(ctx, []models.Product{product}); err != nil {
		return nil, fmt.Errorf("failed to load product relations: %v", err)
	}

	return &product, nil
}

// applyFilters applies search filters to the query
func (s *ProductService) applyFilters(query *gorm.DB, filter ProductFilter) *gorm.DB {
	if filter.Category != "" {
		query = query.Where("LOWER(category) LIKE ?", "%"+strings.ToLower(filter.Category)+"%")
	}

	if filter.Material != "" {
		query = query.Where("LOWER(material) LIKE ?", "%"+strings.ToLower(filter.Material)+"%")
	}

	if filter.MinPrice > 0 {
		query = query.Where("price >= ?", filter.MinPrice)
	}

	if filter.MaxPrice > 0 {
		query = query.Where("price <= ?", filter.MaxPrice)
	}

	if filter.Search != "" {
		searchTerm := "%" + strings.ToLower(filter.Search) + "%"
		query = query.Where(
			"LOWER(name) LIKE ? OR LOWER(description) LIKE ? OR LOWER(brand) LIKE ?",
			searchTerm, searchTerm, searchTerm,
		)
	}

	return query
}

func (s *ProductService) loadProductRelations(ctx context.Context, products []models.Product) error {
	if len(products) == 0 {
		return nil
	}

	// Extract product IDs
	productIDs := make([]uint, len(products))
	productMap := make(map[uint]int) // product ID to index mapping
	
	for i, product := range products {
		productIDs[i] = product.ID
		productMap[product.ID] = i
	}

	// Load all images in batch
	var images []models.Image
	if err := s.db.WithContext(ctx).
		Where("product_id IN ?", productIDs).
		Find(&images).Error; err != nil {
		return fmt.Errorf("failed to load product images: %v", err)
	}

	// Load all services in batch
	var services []models.Service
	if err := s.db.WithContext(ctx).
		Where("product_id IN ?", productIDs).
		Find(&services).Error; err != nil {
		return fmt.Errorf("failed to load product services: %v", err)
	}

	// Group images and services by product ID
	for _, image := range images {
		if idx, exists := productMap[image.ProductID]; exists {
			products[idx].Images = append(products[idx].Images, image)
		}
	}

	for _, service := range services {
		if idx, exists := productMap[service.ProductID]; exists {
			products[idx].Services = append(products[idx].Services, service)
		}
	}

	return nil
}





func (s *ProductService) GetCategories(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT category
		FROM products
		WHERE category IS NOT NULL AND category != ''
		ORDER BY category
	`
	
	categories := make([]string, 0)
	if err := s.db.WithContext(ctx).Raw(query).Scan(&categories).Error; err != nil {
		return nil, fmt.Errorf("%w: failed to fetch categories: %v", ErrDatabaseQuery, err)
	}
	
	return categories, nil
}