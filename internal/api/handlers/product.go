package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/princeprakhar/ecommerce-backend/internal/services"
)


type ProductHandler struct {
	productService *services.ProductService
}

func NewProductHandler(productService *services.ProductService) *ProductHandler {
	return &ProductHandler{
		productService: productService,
	}
}


	func (h *ProductHandler) GetAllProducts(c *gin.Context) {
		minPrice, _ := strconv.ParseFloat(c.Query("min_price"), 64)
		maxPrice, _ := strconv.ParseFloat(c.Query("max_price"), 64)
		status := c.Query("status")
		page, _ := strconv.Atoi(c.Query("page"))
		limit, _ := strconv.Atoi(c.Query("limit"))
		filter := services.ProductFilter{
			Category:   c.Query("category"),
			Material:      c.Query("material"),
			MinPrice:   minPrice,
			MaxPrice:   maxPrice,
			Search:     c.Query("search"),
			Status:   status,
			Page:       page,
			Limit:      limit,
		}
		products, err := h.productService.GetProducts(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve products",
			"error":   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Products retrieved successfully",
		"data":    products,
	})
}



func (h *ProductHandler) GetProduct(c *gin.Context) {
	productID, err := strconv.Atoi(c.Param("id"))
	if err != nil {	
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid product ID",
			"error":   err.Error(),
		})
		return
	}
	product, err := h.productService.GetProductByID(c.Request.Context(), uint(productID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve product",
			"error":   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Product retrieved successfully",
		"data":    product,
	})
}


func (h *ProductHandler) GetCategories(c *gin.Context) {
	categories, err := h.productService.GetCategories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to retrieve categories",
			"error":   err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Categories retrieved successfully",
		"data":    categories,
	})
}