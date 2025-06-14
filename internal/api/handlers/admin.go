package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"mime/multipart"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/princeprakhar/ecommerce-backend/internal/models"
	"github.com/princeprakhar/ecommerce-backend/internal/services"
	"github.com/princeprakhar/ecommerce-backend/internal/utils"
)

type AdminHandler struct {
	adminService *services.AdminService
}

func NewAdminHandler(adminService *services.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

// CreateProduct handles the creation of a new product with images
func (h *AdminHandler) CreateProduct(c *gin.Context) {
	// Parse form data for product information
	var productReq models.CreateProductRequest
	
	// Try to get JSON data first
	if c.ContentType() == "application/json" {
		if err := c.ShouldBindJSON(&productReq); err != nil {
			utils.SendValidationError(c, "Invalid JSON data: "+err.Error())
			return
		}
	} else {
		// Handle multipart form data
		productReq.Name = c.PostForm("name")
		productReq.Description = c.PostForm("description")
		productReq.Category = c.PostForm("category")
		productReq.Brand = c.PostForm("brand")
		productReq.SKU = c.PostForm("sku")
		
		// Parse price
		if priceStr := c.PostForm("price"); priceStr != "" {
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				utils.SendValidationError(c, "Invalid price format")
				return
			}
			productReq.Price = price
		}
		
		// Parse stock
		if stockStr := c.PostForm("stock"); stockStr != "" {
			stock, err := strconv.Atoi(stockStr)
			if err != nil {
				utils.SendValidationError(c, "Invalid stock format")
				return
			}
			productReq.Stock = stock
		}
	}

	// Validate required fields
	if productReq.Name == "" {
		utils.SendValidationError(c, "Product name is required")
		return
	}
	if productReq.Price <= 0 {
		utils.SendValidationError(c, "Product price must be greater than 0")
		return
	}

	// Handle image uploads
	var imageFiles []*multipart.FileHeader
	if c.ContentType() != "application/json" {
		form, err := c.MultipartForm()
		if err == nil && form.File["images"] != nil {
			imageFiles = form.File["images"]
		}
	}

	// Create product with images
	product, err := h.adminService.CreateProduct(&productReq, imageFiles)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Failed to create product", err)
		return
	}

	utils.SendSuccess(c, "Product created successfully", product)
}

// UpdateProduct handles updating an existing product and its images
func (h *AdminHandler) UpdateProduct(c *gin.Context) {
	productIDStr := c.Param("product_id")
	productID, err := strconv.ParseUint(productIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid product ID")
		return
	}

	var updateReq models.UpdateProductRequest
	var imageFiles []*multipart.FileHeader
	var deleteImageIDs []string

	// Handle different content types
	if c.ContentType() == "application/json" {
		if err := c.ShouldBindJSON(&updateReq); err != nil {
			utils.SendValidationError(c, "Invalid JSON data: "+err.Error())
			return
		}
	} else {
		// Handle multipart form data
		if name := c.PostForm("name"); name != "" {
			updateReq.Name = &name
		}
		if description := c.PostForm("description"); description != "" {
			updateReq.Description = &description
		}
		if category := c.PostForm("category"); category != "" {
			updateReq.Category = &category
		}
		if brand := c.PostForm("brand"); brand != "" {
			updateReq.Brand = &brand
		}
		if sku := c.PostForm("sku"); sku != "" {
			updateReq.SKU = &sku
		}
		
		// Parse price
		if priceStr := c.PostForm("price"); priceStr != "" {
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				utils.SendValidationError(c, "Invalid price format")
				return
			}
			updateReq.Price = &price
		}
		
		// Parse stock
		if stockStr := c.PostForm("stock"); stockStr != "" {
			stock, err := strconv.Atoi(stockStr)
			if err != nil {
				utils.SendValidationError(c, "Invalid stock format")
				return
			}
			updateReq.Stock = &stock
		}
		
		// Parse is_active
		if isActiveStr := c.PostForm("is_active"); isActiveStr != "" {
			isActive, err := strconv.ParseBool(isActiveStr)
			if err != nil {
				utils.SendValidationError(c, "Invalid is_active format")
				return
			}
			updateReq.IsActive = &isActive
		}

		// Handle image uploads
		form, err := c.MultipartForm()
		if err == nil && form.File["images"] != nil {
			imageFiles = form.File["images"]
		}
		
		// Handle image deletions
		if deleteIDsStr := c.PostForm("delete_image_ids"); deleteIDsStr != "" {
			deleteImageIDs = strings.Split(deleteIDsStr, ",")
			// Trim whitespace from each ID
			for i, id := range deleteImageIDs {
				deleteImageIDs[i] = strings.TrimSpace(id)
			}
		}
	}

	// Validate price if provided
	if updateReq.Price != nil && *updateReq.Price <= 0 {
		utils.SendValidationError(c, "Product price must be greater than 0")
		return
	}

	// Update product
	product, err := h.adminService.UpdateProduct(uint(productID), &updateReq, imageFiles, deleteImageIDs)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Failed to update product", err)
		return
	}

	utils.SendSuccess(c, "Product updated successfully", product)
}

// UploadProductImages handles uploading images for an existing product
func (h *AdminHandler) UploadProductImages(c *gin.Context) {
	productIDStr := c.Param("product_id")
	productID, err := strconv.ParseUint(productIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid product ID")
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Failed to parse multipart form", err)
		return
	}

	images := form.File["images"]
	if len(images) == 0 {
		utils.SendValidationError(c, "No images provided")
		return
	}

	// Use the update method to add images
	updateReq := models.UpdateProductRequest{} // Empty update request
	product, err := h.adminService.UpdateProduct(uint(productID), &updateReq, images, nil)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Failed to upload images", err)
		return
	}

	utils.SendSuccess(c, "Images uploaded successfully", product)
}

// DeleteProductImage handles deleting a specific image from a product
func (h *AdminHandler) DeleteProductImage(c *gin.Context) {
	productIDStr := c.Param("product_id")
	imageIDStr := c.Param("image_id")
	
	productID, err := strconv.ParseUint(productIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid product ID")
		return
	}

	// Use the update method to delete specific image
	updateReq := models.UpdateProductRequest{} // Empty update request
	product, err := h.adminService.UpdateProduct(uint(productID), &updateReq, nil, []string{imageIDStr})
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Failed to delete image", err)
		return
	}

	utils.SendSuccess(c, "Image deleted successfully", product)
}

// Legacy upload methods for backward compatibility
func (h *AdminHandler) UploadImages(c *gin.Context) {
	utils.SendError(c, http.StatusBadRequest, "This endpoint is deprecated. Use /products endpoint with images", nil)
}

func (h *AdminHandler) UploadCSV(c *gin.Context) {
	userEmail := c.GetString("user_email")
	
	file, err := c.FormFile("csv")
	if err != nil {
		utils.SendValidationError(c, "No CSV file provided")
		return
	}

	response, err := h.adminService.ProcessCSVUpload(file, userEmail)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Failed to process CSV", err)
		return
	}

	utils.SendSuccess(c, "CSV processed successfully", response)
}

func (h *AdminHandler) GetProducts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	products, err := h.adminService.GetProducts(page, limit)
	if err != nil {
		utils.SendInternalError(c, "Failed to fetch products", err)
		return
	}

	// Response with pagination info
	response := map[string]interface{}{
		"products": products,
		"pagination": map[string]interface{}{
			"page":  page,
			"limit": limit,
			"total": len(products),
		},
	}

	utils.SendSuccess(c, "Products retrieved successfully", response)
}

// GetProduct handles fetching a single product by ID
func (h *AdminHandler) GetProduct(c *gin.Context) {
	productIDStr := c.Param("product_id")
	productID, err := strconv.ParseUint(productIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid product ID")
		return
	}

	// You'll need to add this method to AdminService
	product, err := h.adminService.GetProductByID(uint(productID))
	if err != nil {
		utils.SendError(c, http.StatusNotFound, "Product not found", err)
		return
	}

	utils.SendSuccess(c, "Product retrieved successfully", product)
}

func (h *AdminHandler) DeleteProduct(c *gin.Context) {
	productIDStr := c.Param("product_id")
	productID, err := strconv.ParseUint(productIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid product ID")
		return
	}

	err = h.adminService.DeleteProduct(uint(productID))
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Failed to delete product", err)
		return
	}

	utils.SendSuccess(c, "Product deleted successfully", nil)
}

func (h *AdminHandler) GetDashboard(c *gin.Context) {
	stats, err := h.adminService.GetDashboardStats()
	if err != nil {
		utils.SendInternalError(c, "Failed to fetch dashboard stats", err)
		return
	}

	utils.SendSuccess(c, "Dashboard stats retrieved successfully", stats)
}

// Batch operations
func (h *AdminHandler) BatchDeleteProducts(c *gin.Context) {
	var request struct {
		ProductIDs []uint `json:"product_ids" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendValidationError(c, "Invalid request data: "+err.Error())
		return
	}

	var errors []string
	successCount := 0

	for _, productID := range request.ProductIDs {
		if err := h.adminService.DeleteProduct(productID); err != nil {
			errors = append(errors, fmt.Sprintf("Product %d: %v", productID, err))
		} else {
			successCount++
		}
	}

	response := map[string]interface{}{
		"success_count": successCount,
		"total_count":   len(request.ProductIDs),
	}

	if len(errors) > 0 {
		response["errors"] = errors
		utils.SendSuccess(c, fmt.Sprintf("Batch delete completed with %d successes and %d errors", successCount, len(errors)), response)
	} else {
		utils.SendSuccess(c, "All products deleted successfully", response)
	}
}

// Product search
func (h *AdminHandler) SearchProducts(c *gin.Context) {
	query := c.Query("q")
	category := c.Query("category")
	brand := c.Query("brand")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	searchParams := map[string]interface{}{
		"query":    query,
		"category": category,
		"brand":    brand,
		"page":     page,
		"limit":    limit,
	}

	// You'll need to add this method to AdminService
	products, total, err := h.adminService.SearchProducts(searchParams)
	if err != nil {
		utils.SendInternalError(c, "Failed to search products", err)
		return
	}

	response := map[string]interface{}{
		"products": products,
		"pagination": map[string]interface{}{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"total_pages": (total + limit - 1) / limit,
		},
	}

	utils.SendSuccess(c, "Products search completed", response)
}