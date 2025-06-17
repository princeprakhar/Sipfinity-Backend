package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Type   string `json:"type"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	AccessTokenExpiresAt  int64  `json:"access_token_expires_at"`
	RefreshTokenExpiresAt int64  `json:"refresh_token_expires_at"`
}

// Generate access token (short-lived: 15 minutes)
func GenerateAccessToken(userID uint, email, role, jwtSecret string) (string, time.Time, error) {
	expirationTime := time.Now().Add( 15* time.Minute)
	
	claims := &Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		Type:   string(AccessToken),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Subject:   email,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expirationTime, nil
}

// Generate refresh token (long-lived: 7 days)
func GenerateRefreshToken(userID uint, email, role, jwtSecret string) (string, time.Time, error) {
	expirationTime := time.Now().Add(7 * 24 * time.Hour) // 7 days
	
	claims := &Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		Type:   string(RefreshToken),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Subject:   email,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expirationTime, nil
}

// Generate both tokens
func GenerateTokenPair(userID uint, email, role, jwtSecret string) (*TokenPair, error) {
	accessToken, accessExp, err := GenerateAccessToken(userID, email, role, jwtSecret)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshExp, err := GenerateRefreshToken(userID, email, role, jwtSecret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		AccessTokenExpiresAt:  accessExp.Unix(),
		RefreshTokenExpiresAt: refreshExp.Unix(),
	}, nil
}

// Validate token and return claims
func ValidateToken(tokenString, jwtSecret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// Generate random string for additional security
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Legacy function for backward compatibility
func GenerateToken(userID uint, email, role, jwtSecret string) (string, error) {
	token, _, err := GenerateAccessToken(userID, email, role, jwtSecret)
	return token, err
}