package services

import (
	"errors"
	"github.com/princeprakhar/ecommerce-backend/internal/models"
	"github.com/princeprakhar/ecommerce-backend/internal/utils"
	"gorm.io/gorm"
)

type ReviewService struct {
	db *gorm.DB
}

func NewReviewService(db *gorm.DB) *ReviewService {
	return &ReviewService{db: db}
}

type CreateReviewRequest struct {
	ProductID uint   `json:"product_id" binding:"required"`
	Rating    int    `json:"rating" binding:"required"`
	Comment   string `json:"comment"`
}

type ReviewResponse struct {
	ID        uint   `json:"id"`
	UserID    uint   `json:"user_id"`
	ProductID uint   `json:"product_id"`
	Rating    int    `json:"rating"`
	Comment   string `json:"comment"`
	UserName  string `json:"user_name"`
	CreatedAt string `json:"created_at"`
	LikeCount int    `json:"like_count"`
	DislikeCount int `json:"dislike_count"`
}

func (s *ReviewService) CreateReview(userID uint, req CreateReviewRequest) (*models.Review, error) {
	// Validate rating
	if !utils.IsValidRating(req.Rating) {
		return nil, errors.New("rating must be between 1 and 5")
	}

	// Check if product exists
	var product models.Product
	if err := s.db.Where("id = ? AND is_active = ?", req.ProductID, true).First(&product).Error; err != nil {
		return nil, errors.New("product not found")
	}

	// Check if user already reviewed this product
	var existingReview models.Review
	if err := s.db.Where("user_id = ? AND product_id = ?", userID, req.ProductID).First(&existingReview).Error; err == nil {
		return nil, errors.New("you have already reviewed this product")
	}

	// Create review
	review := models.Review{
		UserID:    userID,
		ProductID: req.ProductID,
		Rating:    req.Rating,
		Comment:   utils.SanitizeString(req.Comment),
		IsActive:  true,
	}

	if err := s.db.Create(&review).Error; err != nil {
		return nil, errors.New("failed to create review")
	}

	// Preload user information
	s.db.Preload("User").First(&review, review.ID)

	return &review, nil
}

func (s *ReviewService) GetProductReviews(productID uint, page, limit int) ([]ReviewResponse, error) {
	var reviews []models.Review
	offset := (page - 1) * limit

	query := s.db.Preload("User").
		Where("product_id = ? AND is_active = ?", productID, true).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit)

	if err := query.Find(&reviews).Error; err != nil {
		return nil, err
	}

	var response []ReviewResponse
	for _, review := range reviews {
		// Count likes and dislikes
		var likeCount, dislikeCount int64
		s.db.Model(&models.ReviewLike{}).Where("review_id = ? AND is_like = ?", review.ID, true).Count(&likeCount)
		s.db.Model(&models.ReviewLike{}).Where("review_id = ? AND is_like = ?", review.ID, false).Count(&dislikeCount)

		reviewResp := ReviewResponse{
			ID:           review.ID,
			UserID:       review.UserID,
			ProductID:    review.ProductID,
			Rating:       review.Rating,
			Comment:      review.Comment,
			UserName:     review.User.FirstName + " " + review.User.LastName,
			CreatedAt:    review.CreatedAt.Format("2006-01-02 15:04:05"),
			LikeCount:    int(likeCount),
			DislikeCount: int(dislikeCount),
		}
		response = append(response, reviewResp)
	}

	return response, nil
}

func (s *ReviewService) LikeReview(userID, reviewID uint, isLike bool) error {
	// Check if review exists
	var review models.Review
	if err := s.db.Where("id = ? AND is_active = ?", reviewID, true).First(&review).Error; err != nil {
		return errors.New("review not found")
	}

	// Check existing like/dislike
	var existingLike models.ReviewLike
	err := s.db.Where("user_id = ? AND review_id = ?", userID, reviewID).First(&existingLike).Error

	if err == nil {
		// Update existing like/dislike
		existingLike.IsLike = isLike
		return s.db.Save(&existingLike).Error
	} else if err == gorm.ErrRecordNotFound {
		// Create new like/dislike
		newLike := models.ReviewLike{
			UserID:   userID,
			ReviewID: reviewID,
			IsLike:   isLike,
		}
		return s.db.Create(&newLike).Error
	}

	return err
}

func (s *ReviewService) FlagReview(reviewID uint) error {
	return s.db.Model(&models.Review{}).Where("id = ?", reviewID).Update("is_flagged", true).Error
}

func (s *ReviewService) GetFlaggedReviews() ([]models.Review, error) {
	var reviews []models.Review
	err := s.db.Preload("User").Preload("Product").
		Where("is_flagged = ? AND is_active = ?", true, true).
		Find(&reviews).Error
	return reviews, err
}

func (s *ReviewService) ModerateReview(reviewID uint, action string) error {
	switch action {
	case "approve":
		return s.db.Model(&models.Review{}).Where("id = ?", reviewID).Update("is_flagged", false).Error
	case "remove":
		return s.db.Model(&models.Review{}).Where("id = ?", reviewID).Update("is_active", false).Error
	default:
		return errors.New("invalid action")
	}
}