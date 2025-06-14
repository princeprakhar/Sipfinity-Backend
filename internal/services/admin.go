// services/admin.go
package services

import (
	"encoding/csv"
	"errors"
	"fmt"
	"mime/multipart"
	"strconv"
	"strings"

	"github.com/princeprakhar/ecommerce-backend/internal/config"
	"github.com/princeprakhar/ecommerce-backend/internal/models"
	"gorm.io/gorm"
)

type AdminService struct {
	db             *gorm.DB
	fastAPIService *FastAPIService
	cfg            *config.Config
	emailService   *EmailService
	s3Service      *S3Service
}

func NewAdminService(db *gorm.DB, cfg *config.Config, fastAPIService *FastAPIService, emailService *EmailService) *AdminService {
	return &AdminService{
		db:             db,
		cfg:            cfg,
		fastAPIService: fastAPIService,
		emailService:   emailService,
		s3Service:      NewS3Service(cfg.S3Region, cfg.S3BucketName, cfg.S3AccessKey, cfg.S3SecretKey),
	}
}

func (s *AdminService) CreateProduct(productReq *models.CreateProductRequest, imageFiles []*multipart.FileHeader) (*models.Product, error) {
	if productReq == nil {
		return nil, errors.New("product request cannot be nil")
	}

	// Validate product data
	if err := s.validateProductRequest(productReq); err != nil {
		return nil, err
	}

	// Start database transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create product first
	product := &models.Product{
		Name:        productReq.Name,
		Description: productReq.Description,
		Price:       productReq.Price,
		Category:    productReq.Category,
		Brand:       productReq.Brand,
		SKU:         productReq.SKU,
		Stock:       productReq.Stock,
		IsActive:    true,
	}

	if err := tx.Create(product).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create product: %v", err)
	}

	// Upload images if provided
	if len(imageFiles) > 0 {
		uploadResults, err := s.s3Service.UploadMultipleImages(imageFiles)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to upload images: %v", err)
		}

		// Create image records
		var images []models.Image
		for _, result := range uploadResults {
			image := models.Image{
				ProductID:   product.ID,
				FileName:    result.FileName,
				S3Key:       result.Key,
				S3URL:       result.URL,
				ContentType: result.ContentType,
				Size:        result.Size,
				IsActive:    true,
			}
			images = append(images, image)
		}

		if err := tx.Create(&images).Error; err != nil {
			tx.Rollback()
			// Clean up uploaded files
			var keys []string
			for _, result := range uploadResults {
				keys = append(keys, result.Key)
			}
			s.s3Service.DeleteMultipleImages(keys)
			return nil, fmt.Errorf("failed to create image records: %v", err)
		}

		product.Images = images
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	// Load the complete product with images
	if err := s.db.Preload("Images").First(product, product.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to load created product: %v", err)
	}

	return product, nil
}

func (s *AdminService) UpdateProduct(productID uint, updateReq *models.UpdateProductRequest, imageFiles []*multipart.FileHeader, deleteImageIDs []string) (*models.Product, error) {
	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Find existing product
	var product models.Product
	if err := tx.Preload("Images").First(&product, productID).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("product not found: %v", err)
	}

	// Update product fields
	updateData := make(map[string]interface{})
	if updateReq.Name != nil {
		updateData["name"] = *updateReq.Name
	}
	if updateReq.Description != nil {
		updateData["description"] = *updateReq.Description
	}
	if updateReq.Price != nil {
		if *updateReq.Price <= 0 {
			tx.Rollback()
			return nil, errors.New("price must be greater than 0")
		}
		updateData["price"] = *updateReq.Price
	}
	if updateReq.Category != nil {
		updateData["category"] = *updateReq.Category
	}
	if updateReq.Brand != nil {
		updateData["brand"] = *updateReq.Brand
	}
	if updateReq.SKU != nil {
		updateData["sku"] = *updateReq.SKU
	}
	if updateReq.Stock != nil {
		updateData["stock"] = *updateReq.Stock
	}
	if updateReq.IsActive != nil {
		updateData["is_active"] = *updateReq.IsActive
	}

	if len(updateData) > 0 {
		if err := tx.Model(&product).Updates(updateData).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to update product: %v", err)
		}
	}

	// Handle image deletions
	var keysToDelete []string
	if len(deleteImageIDs) > 0 {
		var imagesToDelete []models.Image
		if err := tx.Where("product_id = ? AND id IN ?", productID, deleteImageIDs).Find(&imagesToDelete).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to find images to delete: %v", err)
		}

		for _, img := range imagesToDelete {
			keysToDelete = append(keysToDelete, img.S3Key)
		}

		// Soft delete images from database
		if err := tx.Model(&models.Image{}).Where("product_id = ? AND id IN ?", productID, deleteImageIDs).Update("is_active", false).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to delete images: %v", err)
		}
	}

	// Handle new image uploads
	if len(imageFiles) > 0 {
		uploadResults, err := s.s3Service.UploadMultipleImages(imageFiles)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to upload new images: %v", err)
		}

		// Create new image records
		var newImages []models.Image
		for _, result := range uploadResults {
			image := models.Image{
				ProductID:   productID,
				FileName:    result.FileName,
				S3Key:       result.Key,
				S3URL:       result.URL,
				ContentType: result.ContentType,
				Size:        result.Size,
				IsActive:    true,
			}
			newImages = append(newImages, image)
		}

		if err := tx.Create(&newImages).Error; err != nil {
			tx.Rollback()
			// Clean up uploaded files
			var keys []string
			for _, result := range uploadResults {
				keys = append(keys, result.Key)
			}
			s.s3Service.DeleteMultipleImages(keys)
			return nil, fmt.Errorf("failed to create new image records: %v", err)
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	// Delete old images from S3 after successful database commit
	if len(keysToDelete) > 0 {
		go func() {
			if err := s.s3Service.DeleteMultipleImages(keysToDelete); err != nil {
				// Log error but don't fail the operation
				fmt.Printf("Warning: Failed to delete images from S3: %v\n", err)
			}
		}()
	}

	// Load updated product with active images
	if err := s.db.Preload("Images", "is_active = ?", true).First(&product, productID).Error; err != nil {
		return nil, fmt.Errorf("failed to load updated product: %v", err)
	}

	return &product, nil
}

func (s *AdminService) DeleteProduct(productID uint) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get product with images
	var product models.Product
	if err := tx.Preload("Images").First(&product, productID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("product not found: %v", err)
	}

	// Soft delete product and images
	if err := tx.Model(&product).Update("is_active", false).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete product: %v", err)
	}

	if err := tx.Model(&models.Image{}).Where("product_id = ?", productID).Update("is_active", false).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete product images: %v", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	// Delete images from S3 asynchronously
	go func() {
		var keys []string
		for _, img := range product.Images {
			keys = append(keys, img.S3Key)
		}
		if err := s.s3Service.DeleteMultipleImages(keys); err != nil {
			fmt.Printf("Warning: Failed to delete images from S3: %v\n", err)
		}
	}()

	return nil
}

func (s *AdminService) ProcessCSVUpload(file *multipart.FileHeader, adminEmail string) (*models.ProductUploadResponse, error) {
	// Open CSV file
	src, err := file.Open()
	if err != nil {
		return nil, errors.New("failed to open CSV file")
	}
	defer src.Close()

	// Parse CSV
	reader := csv.NewReader(src)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, errors.New("failed to parse CSV file")
	}

	if len(records) < 2 {
		return nil, errors.New("CSV file must have header and at least one data row")
	}

	// Expected CSV format: name,description,price,category,brand,sku,stock
	processedCount := 0
	var failedRows []string

	for i, record := range records[1:] { // Skip header
		if len(record) < 7 {
			failedRows = append(failedRows, fmt.Sprintf("Row %d: insufficient columns", i+2))
			continue
		}

		price, err := strconv.ParseFloat(strings.TrimSpace(record[2]), 64)
		if err != nil {
			failedRows = append(failedRows, fmt.Sprintf("Row %d: invalid price", i+2))
			continue
		}

		stock, err := strconv.Atoi(strings.TrimSpace(record[6]))
		if err != nil {
			stock = 0
		}

		product := models.Product{
			Name:        strings.TrimSpace(record[0]),
			Description: strings.TrimSpace(record[1]),
			Price:       price,
			Category:    strings.TrimSpace(record[3]),
			Brand:       strings.TrimSpace(record[4]),
			SKU:         strings.TrimSpace(record[5]),
			Stock:       stock,
			IsActive:    true,
		}

		if err := s.db.Create(&product).Error; err == nil {
			processedCount++
		} else {
			failedRows = append(failedRows, fmt.Sprintf("Row %d: %s", i+2, err.Error()))
		}
	}

	message := fmt.Sprintf("CSV processed successfully. %d products added", processedCount)
	if len(failedRows) > 0 {
		message += fmt.Sprintf(". %d rows failed", len(failedRows))
	}

	return &models.ProductUploadResponse{
		Success:        true,
		Message:        message,
		ProcessedCount: processedCount,
	}, nil
}

func (s *AdminService) GetProducts(page, limit int) ([]models.Product, error) {
	var products []models.Product
	offset := (page - 1) * limit

	err := s.db.Preload("Images", "is_active = ?", true).
		Preload("Reviews").
		Where("is_active = ?", true).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&products).Error

	return products, err
}

func (s *AdminService) GetDashboardStats() (map[string]interface{}, error) {
	var stats map[string]interface{} = make(map[string]interface{})

	// Total products
	var totalProducts int64
	s.db.Model(&models.Product{}).Where("is_active = ?", true).Count(&totalProducts)
	stats["total_products"] = totalProducts

	// Total users
	var totalUsers int64
	s.db.Model(&models.User{}).Where("is_active = ?", true).Count(&totalUsers)
	stats["total_users"] = totalUsers

	// Total reviews
	var totalReviews int64
	s.db.Model(&models.Review{}).Where("is_active = ?", true).Count(&totalReviews)
	stats["total_reviews"] = totalReviews

	// Flagged reviews
	var flaggedReviews int64
	s.db.Model(&models.Review{}).Where("is_flagged = ? AND is_active = ?", true, true).Count(&flaggedReviews)
	stats["flagged_reviews"] = flaggedReviews

	return stats, nil
}

func (s *AdminService) validateProductRequest(req *models.CreateProductRequest) error {
	if req.Name == "" {
		return errors.New("product name cannot be empty")
	}
	if req.Price <= 0 {
		return errors.New("product price must be greater than 0")
	}
	if req.Stock < 0 {
		return errors.New("product stock cannot be negative")
	}
	return nil
}





// Add these methods to your AdminService in services/admin.go

func (s *AdminService) GetProductByID(productID uint) (*models.Product, error) {
	var product models.Product
	err := s.db.Preload("Images", "is_active = ?", true).
		Preload("Reviews").
		Where("id = ? AND is_active = ?", productID, true).
		First(&product).Error
	
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (s *AdminService) SearchProducts(params map[string]interface{}) ([]models.Product, int, error) {
	var products []models.Product
	var total int64

	query := s.db.Model(&models.Product{}).Where("is_active = ?", true)

	// Apply search filters
	if searchQuery, ok := params["query"].(string); ok && searchQuery != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+searchQuery+"%", "%"+searchQuery+"%")
	}
	
	if category, ok := params["category"].(string); ok && category != "" {
		query = query.Where("category = ?", category)
	}
	
	if brand, ok := params["brand"].(string); ok && brand != "" {
		query = query.Where("brand = ?", brand)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	page := params["page"].(int)
	limit := params["limit"].(int)
	offset := (page - 1) * limit

	err := query.Preload("Images", "is_active = ?", true).
		Preload("Reviews").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&products).Error

	return products, int(total), err
}