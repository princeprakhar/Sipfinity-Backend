package services

import (
	"encoding/csv"
	"errors"
	"fmt"
	// "io"
	"mime/multipart"
	"strconv"
	"strings"
	"github.com/princeprakhar/ecommerce-backend/internal/models"
	"gorm.io/gorm"
)

type AdminService struct {
	db           *gorm.DB
	fastAPIService *FastAPIService
	emailService   *EmailService
}

func NewAdminService(db *gorm.DB, fastAPIService *FastAPIService, emailService *EmailService) *AdminService {
	return &AdminService{
		db:           db,
		fastAPIService: fastAPIService,
		emailService:   emailService,
	}
}

func (s *AdminService) ProcessImageUpload(images []*multipart.FileHeader, adminEmail string) (*models.ProductUploadResponse, error) {
	// Save uploaded images temporarily
	var imagePaths []string
	for _, image := range images {
		// Save image to temp directory
		tempPath := fmt.Sprintf("./temp/%s", image.Filename)
		// Implementation to save multipart file to disk would go here
		imagePaths = append(imagePaths, tempPath)
	}

	// Process images through FastAPI
	fastAPIResp, err := s.fastAPIService.ProcessImages(imagePaths)
	if err != nil {
		return nil, fmt.Errorf("failed to process images: %v", err)
	}

	// Save products to database
	processedCount := 0
	for _, productData := range fastAPIResp.ProductData {
		product := models.Product{
			Name:        productData.Name,
			Description: productData.Description,
			Price:       productData.Price,
			Category:    productData.Category,
			Brand:       productData.Brand,
			SKU:         productData.SKU,
			Images:      productData.Images,
			Stock:       0,
			IsActive:    true,
		}

		if err := s.db.Create(&product).Error; err == nil {
			processedCount++
		}
	}

	// Send email with Excel file
	if fastAPIResp.ExcelPath != "" {
		err := s.emailService.SendProductUploadNotification(adminEmail, fastAPIResp.ExcelPath, processedCount)
		if err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Failed to send email: %v\n", err)
		}
	}

	return &models.ProductUploadResponse{
		Success:        true,
		Message:        "Images processed successfully",
		ProcessedCount: processedCount,
		FilePath:       fastAPIResp.ExcelPath,
	}, nil
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

	err := s.db.Preload("Reviews").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&products).Error

	return products, err
}

func (s *AdminService) UpdateProduct(productID uint, updates map[string]interface{}) error {
	return s.db.Model(&models.Product{}).Where("id = ?", productID).Updates(updates).Error
}

func (s *AdminService) DeleteProduct(productID uint) error {
	return s.db.Model(&models.Product{}).Where("id = ?", productID).Update("is_active", false).Error
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