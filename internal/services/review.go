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
	Rating    int    `json:"rating"`
	Comment   string `json:"comment"`
}

type ReviewResponse struct {
	ID           uint   `json:"id"`
	UserID       uint   `json:"user_id"`
	ProductID    uint   `json:"product_id"`
	Rating       int    `json:"rating"`
	Comment      string `json:"comment"`
	UserName     string `json:"user_name"`
	CreatedAt    string `json:"created_at"`
	LikeCount    int    `json:"like_count"`
	DislikeCount int    `json:"dislike_count"`
}

func (s *ReviewService) CreateReview(userID uint, req CreateReviewRequest) (*models.Review, error) {
	// Validate rating
	if !utils.IsValidRating(req.Rating) {
		return nil, errors.New("rating must be between 1 and 5")
	}

	// Check if product exists
	var product models.Product
	if err := s.db.Where("id = ? AND status = ?", req.ProductID, "active").First(&product).Error; err != nil {
		return nil, errors.New("product not found")
	}

	// Check if user already reviewed this product
	var review models.Review
	if err := s.db.Where("user_id = ? AND product_id = ?", userID, req.ProductID).First(&review).Error; err == nil {
		// Review exists — update it
		review.Rating = req.Rating
		review.Comment = utils.SanitizeString(req.Comment)
		review.IsActive = true

		if err := s.db.Save(&review).Error; err != nil {
			return nil, errors.New("failed to update existing review")
		}

		// Preload user and product info
		s.db.Preload("User").Preload("Product").First(&review, review.ID)
		return &review, nil
	}

	// If not found, create a new review
	review = models.Review{
		UserID:    userID,
		ProductID: req.ProductID,
		Rating:    req.Rating,
		Comment:   utils.SanitizeString(req.Comment),
		IsActive:  true,
	}

	if err := s.db.Create(&review).Error; err != nil {
		return nil, errors.New("failed to create review")
	}

	s.db.Preload("User").Preload("Product").First(&review, review.ID)
	return &review, nil
}


func (s *ReviewService) GetProductReviews(productID uint, page, limit int) ([]ReviewResponse, error) {
	// First check if product exists
	var product models.Product
	if err := s.db.Where("id = ? AND status = ?", productID, "active").First(&product).Error; err != nil {
		return nil, errors.New("product not found")
	}

	var reviews []models.Review
	offset := (page - 1) * limit

	query := s.db.Preload("User").
		Where("product_id = ? AND is_active = ?", productID, true).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit)

	if err := query.Find(&reviews).Error; err != nil {
		return nil, errors.New("failed to fetch reviews")
	}

	var response []ReviewResponse
	for _, review := range reviews {
		// Count likes and dislikes
		var likeCount, dislikeCount int64
		s.db.Model(&models.ReviewLike{}).Where("review_id = ? AND is_like = ?", review.ID, true).Count(&likeCount)
		s.db.Model(&models.ReviewLike{}).Where("review_id = ? AND is_like = ?", review.ID, false).Count(&dislikeCount)

		// Handle case where User might be nil
		userName := "Anonymous"
		if review.User.ID != 0 {
			userName = review.User.FirstName + " " + review.User.LastName
		}

		reviewResp := ReviewResponse{
			ID:           review.ID,
			UserID:       review.UserID,
			ProductID:    review.ProductID,
			Rating:       review.Rating,
			Comment:      review.Comment,
			UserName:     userName,
			CreatedAt:    review.CreatedAt.Format("2006-01-02 15:04:05"),
			LikeCount:    int(likeCount),
			DislikeCount: int(dislikeCount),
		}
		response = append(response, reviewResp)
	}

	return response, nil
}

func (s *ReviewService) LikeReview(userID, reviewID uint, isLike bool) error {
	// Check if review exists and is active
	var review models.Review
	if err := s.db.Where("id = ? AND is_active = ?", reviewID, true).First(&review).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("review not found")
		}
		return errors.New("failed to find review")
	}

	// Check existing like/dislike
	var existingLike models.ReviewLike
	err := s.db.Where("user_id = ? AND review_id = ?", userID, reviewID).First(&existingLike).Error

	if err == nil {
		// Update existing like/dislike
		existingLike.IsLike = isLike
		if err := s.db.Save(&existingLike).Error; err != nil {
			return errors.New("failed to update like/dislike")
		}
		return nil
	} else if err == gorm.ErrRecordNotFound {
		// Create new like/dislike
		newLike := models.ReviewLike{
			UserID:   userID,
			ReviewID: reviewID,
			IsLike:   isLike,
		}
		if err := s.db.Create(&newLike).Error; err != nil {
			return errors.New("failed to create like/dislike")
		}
		return nil
	}

	return errors.New("failed to process like/dislike")
}

func (s *ReviewService) FlagReview(reviewID uint) error {
	// Check if review exists and is active
	var review models.Review
	if err := s.db.Where("id = ? AND is_active = ?", reviewID, true).First(&review).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("review not found")
		}
		return errors.New("failed to find review")
	}

	// Update the review to flagged
	if err := s.db.Model(&models.Review{}).Where("id = ?", reviewID).Update("is_flagged", true).Error; err != nil {
		return errors.New("failed to flag review")
	}

	return nil
}

func (s *ReviewService) GetFlaggedReviews() ([]models.Review, error) {
	var reviews []models.Review
	err := s.db.Preload("User").Preload("Product").
		Where("is_flagged = ? AND is_active = ?", true, true).
		Find(&reviews).Error

	if err != nil {
		return nil, errors.New("failed to fetch flagged reviews")
	}

	return reviews, nil
}

func (s *ReviewService) ModerateReview(reviewID uint, action string) error {
	// Check if review exists
	var review models.Review
	if err := s.db.Where("id = ?", reviewID).First(&review).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("review not found")
		}
		return errors.New("failed to find review")
	}

	switch action {
	case "approve":
		if err := s.db.Model(&models.Review{}).Where("id = ?", reviewID).Update("is_flagged", false).Error; err != nil {
			return errors.New("failed to approve review")
		}
		return nil
	case "remove":
		if err := s.db.Model(&models.Review{}).Where("id = ?", reviewID).Update("is_active", false).Error; err != nil {
			return errors.New("failed to remove review")
		}
		return nil
	default:
		return errors.New("invalid action, use 'approve' or 'remove'")
	}
}
