package models

import (
	"time"
)

type Review struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null"`
	ProductID uint      `json:"product_id" gorm:"not null"`
	Rating    int       `json:"rating" gorm:"check:rating >= 1 AND rating <= 5"`
	Comment   string    `json:"comment"`
	IsFlagged bool      `json:"is_flagged" gorm:"default:false"`
	IsActive  bool      `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	User    User         `json:"user,omitempty"`
	Product Product      `json:"product,omitempty"`
	Likes   []ReviewLike `json:"likes,omitempty"`
}

type ReviewLike struct {
	ID       uint `json:"id" gorm:"primaryKey"`
	UserID   uint `json:"user_id" gorm:"not null"`
	ReviewID uint `json:"review_id" gorm:"not null"`
	IsLike   bool `json:"is_like"` // true for like, false for dislike

	// Relations
	User   User   `json:"user,omitempty"`
	Review Review `json:"review,omitempty"`
}

// Ensure one like/dislike per user per review
func (ReviewLike) TableName() string {
	return "review_likes"
}