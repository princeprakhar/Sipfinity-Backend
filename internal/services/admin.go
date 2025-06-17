// services/admin.go
package services

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"mime/multipart"
	"strconv"
	"strings"

	"github.com/princeprakhar/ecommerce-backend/internal/config"
	"github.com/princeprakhar/ecommerce-backend/internal/models"
	"gorm.io/gorm"
	"time"
)

const MaxImageSize = 10 * 1024 * 1024

var (
	ErrInvalidInput          = errors.New("invalid input parameters")
	ErrS3Upload              = errors.New("S3 upload failed")
	ErrProductAlreadyDeleted = errors.New("product already deleted")
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
		Title:       productReq.Title,
		Description: productReq.Description,
		Price:       productReq.Price,
		Category:    productReq.Category,
		Size:        productReq.Size,
		Material:    productReq.Material,
		Status:      productReq.Status,
		Stock:       productReq.Stock,
		Images:      []models.Image{},
		Services:    []models.Service{},
	}

	if productReq.Services != nil {
		// Handle services if provided
		for _, svc := range productReq.Services {
			service := models.Service{
				Name: svc.Name,
				Link: svc.Link,
			}
			product.Services = append(product.Services, service)
		}
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

func (s *AdminService) UpdateProduct(ctx context.Context, productID uint, updateReq *models.UpdateProductRequest, imageFiles []*multipart.FileHeader, deleteImageIDs []string) (*models.Product, error) {
	// Input validation
	if productID == 0 {
		return nil, fmt.Errorf("%w: invalid product ID", ErrInvalidInput)
	}
	if updateReq == nil {
		return nil, fmt.Errorf("%w: update request cannot be nil", ErrInvalidInput)
	}

	// Set context timeout
	ctx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	// Start transaction
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Find existing product
	var product models.Product
	if err := tx.Preload("Images").First(&product, productID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: product with ID %d not found", ErrProductNotFound, productID)
		}
		return nil, fmt.Errorf("%w: failed to find product: %v", ErrDatabaseQuery, err)
	}

	// Build update data
	updateData := make(map[string]interface{})
	hasUpdates := false

	if updateReq.Title != nil {
		updateData["title"] = strings.TrimSpace(*updateReq.Title)
		hasUpdates = true
	}
	if updateReq.Description != nil {
		updateData["description"] = strings.TrimSpace(*updateReq.Description)
		hasUpdates = true
	}
	if updateReq.Price != nil {
		if *updateReq.Price <= 0 {
			tx.Rollback()
			return nil, fmt.Errorf("%w: price must be greater than 0", ErrInvalidInput)
		}
		updateData["price"] = *updateReq.Price
		hasUpdates = true
	}
	if updateReq.Category != nil {
		updateData["category"] = strings.TrimSpace(*updateReq.Category)
		hasUpdates = true
	}
	if updateReq.Status != nil {
		updateData["status"] = strings.TrimSpace(*updateReq.Status)
		hasUpdates = true
	}
	if updateReq.Material != nil {
		updateData["material"] = strings.TrimSpace(*updateReq.Material)
		hasUpdates = true
	}
	if updateReq.Stock != nil {
		if *updateReq.Stock < 0 {
			tx.Rollback()
			return nil, fmt.Errorf("%w: stock cannot be negative", ErrInvalidInput)
		}
		updateData["stock"] = *updateReq.Stock
		hasUpdates = true
	}
	if updateReq.Size != nil {
		updateData["size"] = strings.TrimSpace(*updateReq.Size)
		hasUpdates = true
	}

	// Add updated_at timestamp
	if hasUpdates {
		updateData["updated_at"] = time.Now()
	}

	// **THIS WAS MISSING** - Actually update the product with the updateData
	if hasUpdates {
		if err := tx.Model(&product).Updates(updateData).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("%w: failed to update product: %v", ErrDatabaseQuery, err)
		}
	}

	// Handle services update
	if updateReq.Services != nil {
		// First, delete old services
		if err := tx.Where("product_id = ?", product.ID).Delete(&models.Service{}).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("%w: failed to delete old services: %v", ErrDatabaseQuery, err)
		}

		// Then, insert new services
		var services []models.Service
		for _, svc := range updateReq.Services {
			if svc.Name == "" {
				tx.Rollback()
				return nil, fmt.Errorf("%w: service name cannot be empty", ErrInvalidInput)
			}
			services = append(services, models.Service{
				ProductID: product.ID,
				Name:      strings.TrimSpace(svc.Name),
				Link:      strings.TrimSpace(svc.Link),
			})
		}

		if len(services) > 0 {
			if err := tx.Create(&services).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("%w: failed to insert new services: %v", ErrDatabaseQuery, err)
			}
		}
	}

	// Handle image deletions
	var keysToDelete []string
	if len(deleteImageIDs) > 0 {
		var imagesToDelete []models.Image
		if err := tx.Where("product_id = ? AND id IN ?", productID, deleteImageIDs).Find(&imagesToDelete).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("%w: failed to find images to delete: %v", ErrDatabaseQuery, err)
		}

		for _, img := range imagesToDelete {
			keysToDelete = append(keysToDelete, img.S3Key)
		}

		// Soft delete images from database
		if err := tx.Model(&models.Image{}).Where("product_id = ? AND id IN ?", productID, deleteImageIDs).Update("is_active", false).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("%w: failed to delete images: %v", ErrDatabaseQuery, err)
		}
	}

	// Handle new image uploads
	if len(imageFiles) > 0 {
		// Validate image files
		for _, file := range imageFiles {
			if file.Size > MaxImageSize {
				tx.Rollback()
				return nil, fmt.Errorf("%w: image size exceeds maximum allowed size", ErrInvalidInput)
			}
		}

		uploadResults, err := s.s3Service.UploadMultipleImages(imageFiles)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("%w: failed to upload new images: %v", ErrS3Upload, err)
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
			go func() {
				if cleanupErr := s.s3Service.DeleteMultipleImages(keys); cleanupErr != nil {
					// Log cleanup error
					fmt.Printf("Warning: Failed to cleanup uploaded images: %v\n", cleanupErr)
				}
			}()
			return nil, fmt.Errorf("%w: failed to create new image records: %v", ErrDatabaseQuery, err)
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("%w: failed to commit transaction: %v", ErrDatabaseQuery, err)
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

	// Load updated product with all relations
	var updatedProduct models.Product
	if err := s.db.WithContext(ctx).
		Preload("Images", "is_active = ?", true).
		Preload("Services").
		First(&updatedProduct, productID).Error; err != nil {
		return nil, fmt.Errorf("%w: failed to load updated product: %v", ErrDatabaseQuery, err)
	}

	return &updatedProduct, nil
}

func (s *AdminService) DeleteProduct(ctx context.Context, productID uint) error {
	if productID == 0 {
		return fmt.Errorf("%w: invalid product ID", ErrInvalidInput)
	}

	ctx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get product with images
	var product models.Product
	if err := tx.Preload("Images").First(&product, productID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: product with ID %d not found", ErrProductNotFound, productID)
		}
		return fmt.Errorf("%w: failed to find product: %v", ErrDatabaseQuery, err)
	}

	// Collect image S3 keys for deletion
	var keysToDelete []string
	for _, img := range product.Images {
		if img.S3Key != "" {
			keysToDelete = append(keysToDelete, img.S3Key)
		}
	}

	// 1. Delete review likes
	// Delete review likes where the related review belongs to the product
if err := tx.Where("review_id IN (?)",
	tx.Model(&models.Review{}).Select("id").Where("product_id = ?", productID),
).Delete(&models.ReviewLike{}).Error; err != nil {
	tx.Rollback()
	return fmt.Errorf("failed to delete review likes: %v", err)
}


	// 2. Delete reviews
	if err := tx.Where("product_id = ?", productID).Delete(&models.Review{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete reviews: %v", err)
	}

	// 3. Delete product reactions
	if err := tx.Where("product_id = ?", productID).Delete(&models.ProductReaction{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete product reactions: %v", err)
	}

	// Delete images from DB
	if err := tx.Where("product_id = ?", productID).Delete(&models.Image{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("%w: failed to delete product images: %v", ErrDatabaseQuery, err)
	}

	// Delete services from DB
	if err := tx.Where("product_id = ?", productID).Delete(&models.Service{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("%w: failed to delete product services: %v", ErrDatabaseQuery, err)
	}

	// Finally, delete the product itself
	if err := tx.Delete(&product).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("%w: failed to delete product: %v", ErrDatabaseQuery, err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", ErrDatabaseQuery, err)
	}

	// Delete images from S3 (async)
	if len(keysToDelete) > 0 {
		go func() {
			if err := s.s3Service.DeleteMultipleImages(keysToDelete); err != nil {
				fmt.Printf("Warning: Failed to delete images from S3 for product %d: %v\n", productID, err)
			} else {
				fmt.Printf("Successfully deleted %d images from S3 for product %d\n", len(keysToDelete), productID)
			}
		}()
	}

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
			Title:       strings.TrimSpace(record[0]),
			Description: strings.TrimSpace(record[1]),
			Price:       price,
			Category:    strings.TrimSpace(record[3]),
			Material:    strings.TrimSpace(record[4]),
			Size:        strings.TrimSpace(record[5]),
			Stock:       stock,
			Status:      "active",         // Default status
			Images:      []models.Image{}, // No images in CSV upload
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
		Preload("Reviews").Preload("Services").
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
	if req.Title == "" {
		return errors.New("product title cannot be empty")
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

func (s *AdminService) GetProductByID(ctx context.Context, productID uint) (*models.Product, error) {
	// Input validation
	if productID == 0 {
		return nil, fmt.Errorf("invalid product ID")
	}

	// Set query timeout
	ctx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	var product models.Product

	// Admin can access products regardless of status (active/inactive)
	err := s.db.WithContext(ctx).
		Preload("Images"). // Load all images (active and inactive for admin)
		Preload("Reviews").
		Preload("Services"). // If you have services relation
		Where("id = ?", productID).
		First(&product).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: product with ID %d not found", ErrProductNotFound, productID)
		}
		return nil, fmt.Errorf("%w: failed to fetch product: %v", ErrDatabaseQuery, err)
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
