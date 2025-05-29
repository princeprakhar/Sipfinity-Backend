package handlers

import (
	"net/http"
	"strconv"
	"github.com/gin-gonic/gin"
	"github.com/princeprakhar/ecommerce-backend/internal/services"
	"github.com/princeprakhar/ecommerce-backend/internal/utils"
)

type ReviewHandler struct {
	reviewService *services.ReviewService
}

func NewReviewHandler(reviewService *services.ReviewService) *ReviewHandler {
	return &ReviewHandler{reviewService: reviewService}
}

func (h *ReviewHandler) CreateReview(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	var req services.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request data")
		return
	}

	review, err := h.reviewService.CreateReview(userID, req)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Failed to create review", err)
		return
	}

	utils.SendSuccess(c, "Review created successfully", review)
}

func (h *ReviewHandler) GetProductReviews(c *gin.Context) {
	productIDStr := c.Param("product_id")
	productID, err := strconv.ParseUint(productIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid product ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	reviews, err := h.reviewService.GetProductReviews(uint(productID), page, limit)
	if err != nil {
		utils.SendInternalError(c, "Failed to fetch reviews", err)
		return
	}

	utils.SendSuccess(c, "Reviews retrieved successfully", reviews)
}

func (h *ReviewHandler) LikeReview(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	reviewIDStr := c.Param("review_id")
	reviewID, err := strconv.ParseUint(reviewIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid review ID")
		return
	}

	var req struct {
		IsLike bool `json:"is_like"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request data")
		return
	}

	err = h.reviewService.LikeReview(userID, uint(reviewID), req.IsLike)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Failed to like/dislike review", err)
		return
	}

	message := "Review liked successfully"
	if !req.IsLike {
		message = "Review disliked successfully"
	}

	utils.SendSuccess(c, message, nil)
}

func (h *ReviewHandler) FlagReview(c *gin.Context) {
	reviewIDStr := c.Param("review_id")
	reviewID, err := strconv.ParseUint(reviewIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid review ID")
		return
	}

	err = h.reviewService.FlagReview(uint(reviewID))
	if err != nil {
		utils.SendInternalError(c, "Failed to flag review", err)
		return
	}

	utils.SendSuccess(c, "Review flagged successfully", nil)
}

func (h *ReviewHandler) GetFlaggedReviews(c *gin.Context) {
	reviews, err := h.reviewService.GetFlaggedReviews()
	if err != nil {
		utils.SendInternalError(c, "Failed to fetch flagged reviews", err)
		return
	}

	utils.SendSuccess(c, "Flagged reviews retrieved successfully", reviews)
}

func (h *ReviewHandler) ModerateReview(c *gin.Context) {
	reviewIDStr := c.Param("review_id")
	reviewID, err := strconv.ParseUint(reviewIDStr, 10, 32)
	if err != nil {
		utils.SendValidationError(c, "Invalid review ID")
		return
	}

	var req struct {
		Action string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendValidationError(c, "Invalid request data")
		return
	}

	err = h.reviewService.ModerateReview(uint(reviewID), req.Action)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Failed to moderate review", err)
		return
	}

	utils.SendSuccess(c, "Review moderated successfully", nil)
}