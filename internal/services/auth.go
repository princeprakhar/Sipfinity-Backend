package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/princeprakhar/ecommerce-backend/internal/models"
	"github.com/princeprakhar/ecommerce-backend/internal/utils"
	"gorm.io/gorm"
	"github.com/princeprakhar/ecommerce-backend/internal/types"
)

type AuthService struct {
	db                *gorm.DB
	jwtSecret         string
	validationService *ValidationService
	emailService      *EmailService
	baseURL           string
}

type ForgotPasswordRequest struct {
    Email string `json:"email" binding:"required"`
}

type ResetPasswordRequest struct {
    Token       string `json:"token" binding:"required"`
    NewPassword string `json:"new_password" binding:"required"`
}

type ChangePasswordRequest struct {
    CurrentPassword string `json:"current_password" binding:"required"`
    NewPassword     string `json:"new_password" binding:"required"`
}

type UpdateProfileRequest struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email      string `json:"email" binding:"required,email"`
	PhoneNumber string `json:"phone_number"`
}

func NewAuthService(db *gorm.DB, jwtSecret string, validationService *ValidationService, emailService *EmailService, baseURL string) *AuthService {
	return &AuthService{
		db:                db,
		jwtSecret:         jwtSecret,
		validationService: validationService,
		emailService:      emailService,
		baseURL:           baseURL,
	}
}

type SignupRequest struct {
	Email       string `json:"email" binding:"required"`
	Password    string `json:"password" binding:"required"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	PhoneNumber string `json:"phone_number" binding:"required"`
	Role        string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	IsAdmin  bool   `json:"is_admin"` // Optional, for admin login
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type AuthResponse struct {
	Token struct {
		AccessToken           string `json:"access_token"`
		RefreshToken          string `json:"refresh_token"`
		AccessTokenExpiresAt  int64  `json:"access_token_expires_at"`
		RefreshTokenExpiresAt int64  `json:"refresh_token_expires_at"`
	} `json:"tokens"`
	User models.User `json:"user"`
}

func (s *AuthService) Signup(req SignupRequest) (*AuthResponse, error) {
	// Basic email format validation first
	if !utils.IsValidEmail(req.Email) {
		return nil, errors.New("invalid email format")
	}

	// Basic password validation
	if !utils.IsValidPassword(req.Password) {
		return nil, errors.New("password must be at least 8 characters")
	}

	// Email validation
	if s.validationService != nil {
		emailValid, err := s.validationService.IsEmailValid(req.Email)
		if err != nil {
			return nil, fmt.Errorf("email validation failed: %v", err)
		}
		if !emailValid {
			return nil, errors.New("email address is not valid or deliverable")
		}
	} else {
		return nil, errors.New("email validation service unavailable")
	}

	// Phone validation
	if req.PhoneNumber != "" {
		if s.validationService != nil {
			phoneValid, err := s.validationService.IsPhoneValid(req.PhoneNumber)
			if err != nil {
				return nil, fmt.Errorf("phone validation failed: %v", err)
			}
			if !phoneValid {
				return nil, errors.New("phone number is not valid")
			}
		} else {
			return nil, errors.New("phone validation service unavailable")
		}
	}

	// Set default role
	if req.Role == "" {
		req.Role = "customer"
	}

	if !utils.IsValidRole(req.Role) {
		return nil, errors.New("invalid role")
	}

	// Check if user already exists
	var existingUser models.User
	if err := s.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return nil, errors.New("user already exists")
	}

	// Create user
	user := models.User{
		Email:       utils.SanitizeString(req.Email),
		Password:    req.Password, // Will be hashed in BeforeCreate hook
		FirstName:   utils.SanitizeString(req.FirstName),
		LastName:    utils.SanitizeString(req.LastName),
		PhoneNumber: utils.SanitizeString(req.PhoneNumber),
		Role:        req.Role,
		IsActive:    true,
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, errors.New("failed to create user")
	}

	// Generate token pair
	tokenPair, err := utils.GenerateTokenPair(user.ID, user.Email, user.Role, s.jwtSecret)
	if err != nil {
		return nil, errors.New("failed to generate tokens")
	}

	// Store refresh token in database
	refreshToken := models.RefreshToken{
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: time.Unix(tokenPair.RefreshTokenExpiresAt, 0),
		IsRevoked: false,
	}

	if err := s.db.Create(&refreshToken).Error; err != nil {
		return nil, errors.New("failed to store refresh token")
	}

	return &AuthResponse{
		Token: struct {
			AccessToken           string `json:"access_token"`
			RefreshToken          string `json:"refresh_token"`
			AccessTokenExpiresAt  int64  `json:"access_token_expires_at"`
			RefreshTokenExpiresAt int64  `json:"refresh_token_expires_at"`
		}{
			AccessToken:           tokenPair.AccessToken,
			RefreshToken:          tokenPair.RefreshToken,
			AccessTokenExpiresAt:  tokenPair.AccessTokenExpiresAt,
			RefreshTokenExpiresAt: tokenPair.RefreshTokenExpiresAt,
		},
		User: user,
	}, nil
}

func (s *AuthService) Login(req LoginRequest) (*AuthResponse, error) {
	// Validate input
	if !utils.IsValidEmail(req.Email) {
		return nil, errors.New("invalid email format")
	}
	
	var role string
	if req.IsAdmin {
		role = "admin"
	} else {
		role = "customer"
	}

	// Find user
	var user models.User
	if err := s.db.Where("email = ? AND is_active = ?", req.Email, true).First(&user).Error; err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Check password
	if !user.CheckPassword(req.Password)  {
		return nil, errors.New("invalid credentials")
	}

	if user.Role != role {
		return nil, errors.New("invalid credentials")
	}

	// Revoke all existing refresh tokens for this user (optional security measure)
	s.db.Model(&models.RefreshToken{}).Where("user_id = ?", user.ID).Update("is_revoked", true)

	// Generate new token pair
	tokenPair, err := utils.GenerateTokenPair(user.ID, user.Email, user.Role, s.jwtSecret)
	if err != nil {
		return nil, errors.New("failed to generate tokens")
	}

	// Store new refresh token
	refreshToken := models.RefreshToken{
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: time.Unix(tokenPair.RefreshTokenExpiresAt, 0),
		IsRevoked: false,
	}

	if err := s.db.Create(&refreshToken).Error; err != nil {
		return nil, errors.New("failed to store refresh token")
	}

	return &AuthResponse{
		Token: struct {
			AccessToken           string `json:"access_token"`
			RefreshToken          string `json:"refresh_token"`
			AccessTokenExpiresAt  int64  `json:"access_token_expires_at"`
			RefreshTokenExpiresAt int64  `json:"refresh_token_expires_at"`
		}{
			AccessToken:           tokenPair.AccessToken,
			RefreshToken:          tokenPair.RefreshToken,
			AccessTokenExpiresAt:  tokenPair.AccessTokenExpiresAt,
			RefreshTokenExpiresAt: tokenPair.RefreshTokenExpiresAt,
		},
		User: user,
	}, nil
}

// services/auth_service.go
func (s *AuthService) RefreshToken(req RefreshRequest) (*types.AuthResponse, error) {
	claims, err := utils.ValidateToken(req.RefreshToken, s.jwtSecret)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	if claims.Type != string(utils.RefreshToken) {
		return nil, errors.New("invalid token type")
	}

	var refreshToken models.RefreshToken
	if err := s.db.Where("token = ? AND is_revoked = ? AND expires_at > ?", req.RefreshToken, false, time.Now()).
		First(&refreshToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("refresh token not found or expired")
		}
		return nil, err
	}

	var user models.User
	if err := s.db.Where("id = ? AND is_active = ?", refreshToken.UserID, true).
		First(&user).Error; err != nil {
		return nil, errors.New("user not found")
	}

	// Transactional revoke and new insert
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	refreshToken.IsRevoked = true
	if err := tx.Save(&refreshToken).Error; err != nil {
		tx.Rollback()
		return nil, errors.New("failed to revoke old token")
	}

	tokenPair, err := utils.GenerateTokenPair(user.ID, user.Email, user.Role, s.jwtSecret)
	if err != nil {
		tx.Rollback()
		return nil, errors.New("failed to generate new tokens")
	}

	newRefresh := models.RefreshToken{
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: time.Unix(tokenPair.RefreshTokenExpiresAt, 0),
		IsRevoked: false,
	}

	if err := tx.Create(&newRefresh).Error; err != nil {
		tx.Rollback()
		return nil, errors.New("failed to store new refresh token")
	}

	tx.Commit()

	return &types.AuthResponse{
		Token: types.TokenPair{
			AccessToken:           tokenPair.AccessToken,
			RefreshToken:          tokenPair.RefreshToken,
			AccessTokenExpiresAt:  tokenPair.AccessTokenExpiresAt,
			RefreshTokenExpiresAt: tokenPair.RefreshTokenExpiresAt,
		},
		User: user,
	}, nil
}


func (s *AuthService) Logout(refreshToken string) error {
	// Revoke the refresh token
	return s.db.Model(&models.RefreshToken{}).
		Where("token = ?", refreshToken).
		Update("is_revoked", true).Error
}

func (s *AuthService) LogoutAll(userID uint) error {
	// Revoke all refresh tokens for the user
	return s.db.Model(&models.RefreshToken{}).
		Where("user_id = ?", userID).
		Update("is_revoked", true).Error
}

func (s *AuthService) GetUserByID(userID uint) (*models.User, error) {
	var user models.User
	if err := s.db.Where("id = ? AND is_active = ?", userID, true).First(&user).Error; err != nil {
		return nil, errors.New("user not found")
	}
	return &user, nil
}



func (s *AuthService) generateSecureToken() (string, error) {
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}

func (s *AuthService) ForgotPassword(req ForgotPasswordRequest) error {
    if !utils.IsValidEmail(req.Email) {
        return errors.New("invalid email format")
    }

    var user models.User
    if err := s.db.Where("email = ? AND is_active = ?", req.Email, true).First(&user).Error; err != nil {
        return nil // Don't reveal if email exists
    }

    resetToken, err := s.generateSecureToken()
    if err != nil {
        return errors.New("failed to generate reset token")
    }

    s.db.Model(&models.PasswordResetToken{}).
        Where("user_id = ? AND is_used = ?", user.ID, false).
        Update("is_used", true)

    passwordResetToken := models.PasswordResetToken{
        UserID:    user.ID,
        Token:     resetToken,
        ExpiresAt: time.Now().Add(1 * time.Hour),
        IsUsed:    false,
    }

    if err := s.db.Create(&passwordResetToken).Error; err != nil {
        return errors.New("failed to create reset token")
    }

    if s.emailService != nil {
        if err := s.emailService.SendPasswordResetEmail(user.Email, resetToken, s.baseURL); err != nil {
            fmt.Printf("Failed to send password reset email: %v\n", err)
        }
    }

    return nil
}

func (s *AuthService) ResetPassword(req ResetPasswordRequest) error {
    if !utils.IsValidPassword(req.NewPassword) {
        return errors.New("password must be at least 8 characters")
    }

    var resetToken models.PasswordResetToken
    if err := s.db.Where("token = ? AND is_used = ? AND expires_at > ?", 
        req.Token, false, time.Now()).First(&resetToken).Error; err != nil {
        return errors.New("invalid or expired reset token")
    }

    var user models.User
    if err := s.db.Where("id = ? AND is_active = ?", resetToken.UserID, true).First(&user).Error; err != nil {
        return errors.New("user not found")
    }

    if err := user.UpdatePassword(req.NewPassword); err != nil {
        return errors.New("failed to update password")
    }

    if err := s.db.Save(&user).Error; err != nil {
        return errors.New("failed to save new password")
    }

    resetToken.IsUsed = true
    s.db.Save(&resetToken)

    s.db.Model(&models.RefreshToken{}).
        Where("user_id = ?", user.ID).
        Update("is_revoked", true)

    return nil
}

func (s *AuthService) ChangePassword(userID uint, req ChangePasswordRequest) error {
    if !utils.IsValidPassword(req.NewPassword) {
        return errors.New("password must be at least 8 characters")
    }

    var user models.User
    if err := s.db.Where("id = ? AND is_active = ?", userID, true).First(&user).Error; err != nil {
        return errors.New("user not found")
    }

    if !user.CheckPassword(req.CurrentPassword) {
        return errors.New("current password is incorrect")
    }

    if err := user.UpdatePassword(req.NewPassword); err != nil {
        return errors.New("failed to update password")
    }

    if err := s.db.Save(&user).Error; err != nil {
        return errors.New("failed to save new password")
    }

    return nil
}

func (s *AuthService) ValidateResetToken(token string) (*models.User, error) {
    var resetToken models.PasswordResetToken
    if err := s.db.Where("token = ? AND is_used = ? AND expires_at > ?", 
        token, false, time.Now()).First(&resetToken).Error; err != nil {
        return nil, errors.New("invalid or expired reset token")
    }

    var user models.User
    if err := s.db.Where("id = ? AND is_active = ?", resetToken.UserID, true).First(&user).Error; err != nil {
        return nil, errors.New("user not found")
    }

    return &user, nil
}




func (s *AuthService) UpdateProfile(userID uint, req UpdateProfileRequest) (*models.User, error) {
	// Validate email format
	if !utils.IsValidEmail(req.Email) && s.validationService != nil {
		// If validation service is available, use it to validate email
		emailValid, err := s.validationService.IsEmailValid(req.Email)
		if err != nil {
			return nil, fmt.Errorf("email validation failed: %v", err)
		}
		if !emailValid {
			return nil, errors.New("invalid email format")
		}
	}

	
	// Validate phone number if provided
	if req.PhoneNumber != "" && s.validationService != nil {
		phoneValid, err := s.validationService.IsPhoneValid(req.PhoneNumber)
		if err != nil {
			return nil, fmt.Errorf("phone validation failed: %v", err)
		}
		if !phoneValid {
			return nil, errors.New("phone number is not valid")
		}
	}

	var user models.User
	if err := s.db.Where("id = ? AND is_active = ?", userID, true).First(&user).Error; err != nil {
		return nil, errors.New("user not found")
	}

	user.FirstName = utils.SanitizeString(req.FirstName)
	user.LastName = utils.SanitizeString(req.LastName)
	user.Email = utils.SanitizeString(req.Email)
	user.PhoneNumber = utils.SanitizeString(req.PhoneNumber)

	if err := s.db.Save(&user).Error; err != nil {
		return nil, errors.New("failed to update profile")
	}

	return &user, nil
}