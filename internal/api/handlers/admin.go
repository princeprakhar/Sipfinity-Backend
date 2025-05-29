package handlers

import (
	"net/http"
	"strconv"
	"github.com/gin-gonic/gin"
	"github.com/princeprakhar/ecommerce-backend/internal/services"
	"github.com/princeprakhar/ecommerce-backend/internal/utils"
)

type AdminHandler struct {
	adminService *services.AdminService
}

func NewAdminHandler(adminService *services.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

func (h *AdminHandler) UploadImages(c *gin.Context) {
	userEmail := c.GetString("user_email")
	
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

	response, err := h.adminService.ProcessImageUpload(images, userEmail)
	if err != nil {
		utils.SendInternalError(c, "Failed to process images", err)
		return
	}

	utils.SendSuccess(c, "Images processed successfully", response)
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

	utils.SendSuccess(c, "Products retrieved successfully", products)
}

func (h *AdminHandler) UpdateProduct(c *gin.Context) {
	productIDStr := c.Param("product_id")
	productID, err := strconv.ParseUint(productIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid product ID")
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		utils.SendValidationError(c, "Invalid request data")
		return
	}

	err = h.adminService.UpdateProduct(uint(productID), updates)
	if err != nil {
		utils.SendInternalError(c, "Failed to update product", err)
		return
	}

	utils.SendSuccess(c, "Product updated successfully", nil)
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
		utils.SendInternalError(c, "Failed to delete product", err)
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